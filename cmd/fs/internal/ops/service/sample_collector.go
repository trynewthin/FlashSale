package service

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"time"
)

// SampleCollector 后台采样协程：定期采集系统指标并写入 SampleStore。
type SampleCollector struct {
	env      *EnvContext
	store    *SampleStore
	interval time.Duration
}

// NewSampleCollector 创建采样收集器。
func NewSampleCollector(env *EnvContext, store *SampleStore) *SampleCollector {
	return &SampleCollector{
		env:      env,
		store:    store,
		interval: 2 * time.Second,
	}
}

// Run 启动后台采样循环，ctx 取消时停止。
func (c *SampleCollector) Run(ctx context.Context) {
	log.Printf("[sample_collector] started, interval=%s, store=%s", c.interval, c.store.baseDir)

	// 启动时清理过期数据
	c.store.Cleanup()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// 每小时清理一次过期数据
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
	sample := &MonitorSample{
		Timestamp: now.UnixMilli(),
		Label:     now.Format("15:04:05"),
	}

	// 采集系统状态
	status := BuildStatusSnapshot(c.env)
	portsTotal := len(status.Ports)
	portsUp := 0
	for _, p := range status.Ports {
		if p.OK {
			portsUp++
		}
	}
	httpTotal := len(status.HTTP)
	httpUp := 0
	for _, h := range status.HTTP {
		if h.OK {
			httpUp++
		}
	}
	sample.PortRate = safeRateFloat(portsUp, portsTotal)
	sample.HttpRate = safeRateFloat(httpUp, httpTotal)

	// 采集容器状态
	containers := BuildContainerRuntimeSnapshot(c.env.RepoRoot)
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

	// 采集 Prometheus 指标
	metricNames := []string{"rpc_request_rate", "rpc_error_rate", "rpc_p99_latency"}
	snapshot := MetricsSnapshot(c.env, metricNames)
	sample.PromQps = extractPromScalar(snapshot, "rpc_request_rate")
	sample.PromErrorRate = extractPromScalar(snapshot, "rpc_error_rate")
	sample.PromP99LatencyMs = extractPromScalar(snapshot, "rpc_p99_latency")

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

// extractPromScalar 从 MetricsSnapshot 结果中提取标量值。
func extractPromScalar(snapshot map[string]any, metricName string) *float64 {
	raw, ok := snapshot[metricName]
	if !ok {
		return nil
	}

	// 检查错误情况
	if errMap, ok := raw.(map[string]any); ok {
		if _, hasErr := errMap["error"]; hasErr {
			return nil
		}
	}

	// raw 是 json.RawMessage (Prometheus 响应)
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
	if len(promResp.Data.Result) == 0 {
		return nil
	}
	vals := promResp.Data.Result[0].Value
	if len(vals) < 2 {
		return nil
	}

	var valStr string
	if err := json.Unmarshal(vals[1], &valStr); err != nil {
		return nil
	}

	f, err := json.Number(valStr).Float64()
	if err != nil {
		return nil
	}

	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil
	}
	return &f
}
