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
}

func NewSampleCollector(env *catalog.EnvContext, store *SampleStore) *SampleCollector {
	return &SampleCollector{env: env, store: store, interval: 2 * time.Second}
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
		"seckill_purchase_task_queue_depth", "seckill_purchase_task_queue_cap", "seckill_purchase_task_dropped_total",
	}
	snapshot := catalog.MetricsSnapshot(c.env, metricNames)
	sample.PromQps = extractPromScalar(snapshot, "rpc_request_rate")
	sample.PromErrorRate = extractPromScalar(snapshot, "rpc_error_rate")
	sample.PromP99LatencyMs = extractPromScalar(snapshot, "rpc_p99_latency")
	sample.PurchaseTaskQueueDepth = extractPromScalar(snapshot, "seckill_purchase_task_queue_depth")
	sample.PurchaseTaskQueueCap = extractPromScalar(snapshot, "seckill_purchase_task_queue_cap")
	sample.PurchaseTaskDropped = extractPromScalar(snapshot, "seckill_purchase_task_dropped_total")
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
