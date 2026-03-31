package store

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"time"

	"flashsale/ops/backend/catalog"
)

type SampleCollector struct {
	env      *catalog.EnvContext
	store    *SampleStore
	interval time.Duration
	trigger  chan struct{}
}

func NewSampleCollector(env *catalog.EnvContext, store *SampleStore) *SampleCollector {
	return &SampleCollector{
		env:      env,
		store:    store,
		interval: 15 * time.Second,
		trigger:  make(chan struct{}, 1),
	}
}

func (c *SampleCollector) TriggerNow() {
	select {
	case c.trigger <- struct{}{}:
	default:
	}
}

func (c *SampleCollector) Run(ctx context.Context) {
	log.Printf("[sample_collector] started, interval=%s, store=%s", c.interval, c.store.baseDir)
	c.store.Cleanup()
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	cleanupTicker := time.NewTicker(time.Hour)
	defer cleanupTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("[sample_collector] stopped")
			c.store.Close()
			return
		case <-ticker.C:
			c.collectOnce()
		case <-c.trigger:
			c.collectOnce()
			ticker.Reset(c.interval)
		case <-cleanupTicker.C:
			c.store.Cleanup()
		}
	}
}

func (c *SampleCollector) collectOnce() {
	now := time.Now()
	sample := &MonitorSample{Timestamp: now.UnixMilli(), Label: now.Format("15:04:05")}
	status := catalog.BuildStatusSnapshot(c.env)
	portsUp := 0
	for _, p := range status.Ports {
		if p.OK {
			portsUp++
		}
	}
	httpUp := 0
	for _, h := range status.HTTP {
		if h.OK {
			httpUp++
		}
	}
	sample.PortRate = safeRateFloat(portsUp, len(status.Ports))
	sample.HttpRate = safeRateFloat(httpUp, len(status.HTTP))
	containers := catalog.BuildContainerRuntimeSnapshot(c.env.RepoRoot)
	running := 0
	for _, ct := range containers.Containers {
		if ct.Running {
			running++
		}
	}
	sample.RunningContainers = running
	sample.TotalContainers = len(containers.Containers)
	runRep := 0
	totalRep := 0
	for _, svc := range containers.Services {
		runRep += svc.RunningReplicas
		totalRep += svc.Replicas
	}
	sample.RunningReplicas = runRep
	sample.TotalReplicas = totalRep
	sample.ReplicaRate = safeRateFloat(runRep, totalRep)
	metricNames := []string{
		"rpc_request_rate", "rpc_error_rate", "rpc_p99_latency",
		"seckill_purchase_kafka_publish_rate", "seckill_purchase_kafka_publish_failed", "seckill_order_state_consume_rate",
	}
	snapshot := catalog.MetricsSnapshot(c.env, metricNames, catalog.MetricProfileOverview)
	sample.PromQps = extractPromScalar(snapshot, "rpc_request_rate")
	sample.PromErrorRate = extractPromScalar(snapshot, "rpc_error_rate")
	sample.PromP99LatencyMs = extractPromScalar(snapshot, "rpc_p99_latency")
	sample.PurchaseKafkaPublishRate = extractPromScalar(snapshot, "seckill_purchase_kafka_publish_rate")
	sample.PurchaseKafkaPublishFailed = extractPromScalar(snapshot, "seckill_purchase_kafka_publish_failed")
	sample.OrderStateConsumeRate = extractPromScalar(snapshot, "seckill_order_state_consume_rate")
	// P99 降噪：采样间隔现为 15s，对应查询窗口约 300s（overview profile），
	// 若窗口内总请求 < 100 则 P99 统计不可靠，置空。
	maskP99ByQPS(sample, 300)
	if err := c.store.Append(sample); err != nil {
		log.Printf("[sample_collector] write failed: %v", err)
	}
}

func safeRateFloat(ok, total int) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(float64(ok)/float64(total)*10000) / 100
}

func extractPromScalar(snapshot map[string]any, metricName string) *float64 {
	raw, ok := snapshot[metricName]
	if !ok {
		return nil
	}
	if errMap, ok := raw.(map[string]any); ok {
		if _, hasErr := errMap["error"]; hasErr {
			return nil
		}
	}
	rawMsg, ok := raw.(json.RawMessage)
	if !ok {
		return nil
	}
	var promResp struct {
		Data struct {
			Result []struct {
				Value []json.RawMessage `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rawMsg, &promResp); err != nil {
		return nil
	}
	if len(promResp.Data.Result) == 0 || len(promResp.Data.Result[0].Value) < 2 {
		return nil
	}
	var valStr string
	if err := json.Unmarshal(promResp.Data.Result[0].Value[1], &valStr); err != nil {
		return nil
	}
	f, err := json.Number(valStr).Float64()
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return nil
	}
	return &f
}

// maskP99ByQPS 在 QPS 过低时将 P99 延迟置空，避免噪声干扰图表。
// minRequests 为统计窗口内最小请求数阈值（默认 100）。
func maskP99ByQPS(s *MonitorSample, lookbackSeconds float64) {
	if s.PromP99LatencyMs == nil {
		return
	}
	if s.PromQps == nil || *s.PromQps*lookbackSeconds < 100 {
		s.PromP99LatencyMs = nil
	}
}
