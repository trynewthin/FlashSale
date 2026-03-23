package scheduler

import (
	"encoding/json"

	"flashsale/ops/backend/model"
	"flashsale/ops/backend/store"
)

const (
	perfMonitorLookbackMs  = int64(5000)
	perfMonitorLookaheadMs = int64(1500)
)

func (s *Scheduler) enrichPerfProgress(sample model.PerfProgress) model.PerfProgress {
	if s == nil || s.sampleStore == nil || sample.Timestamp <= 0 {
		return sample
	}
	startMs := sample.Timestamp - perfMonitorLookbackMs
	if startMs < 0 {
		startMs = 0
	}
	monitorSamples, err := s.sampleStore.Query(startMs, sample.Timestamp+perfMonitorLookaheadMs, 0)
	if err != nil || len(monitorSamples) == 0 {
		return sample
	}
	merged := mergePerfProgressSamples([]model.PerfProgress{sample}, monitorSamples)
	if len(merged) == 0 {
		return sample
	}
	return merged[0]
}

func (s *Scheduler) enrichPerfProgressSeries(samples []model.PerfProgress) []model.PerfProgress {
	if s == nil || s.sampleStore == nil || len(samples) == 0 {
		return samples
	}
	startMs := samples[0].Timestamp - perfMonitorLookbackMs
	if startMs < 0 {
		startMs = 0
	}
	endMs := samples[len(samples)-1].Timestamp + perfMonitorLookaheadMs
	if endMs < startMs {
		endMs = startMs
	}
	monitorSamples, err := s.sampleStore.Query(startMs, endMs, 0)
	if err != nil || len(monitorSamples) == 0 {
		return samples
	}
	return mergePerfProgressSamples(samples, monitorSamples)
}

func mergePerfProgressSamples(perfSamples []model.PerfProgress, monitorSamples []store.MonitorSample) []model.PerfProgress {
	if len(perfSamples) == 0 || len(monitorSamples) == 0 {
		return perfSamples
	}

	out := make([]model.PerfProgress, len(perfSamples))
	monitorIndex := 0
	var previous *store.MonitorSample

	for i, perfSample := range perfSamples {
		for ; monitorIndex < len(monitorSamples); monitorIndex++ {
			if monitorSamples[monitorIndex].Timestamp <= perfSample.Timestamp {
				previous = &monitorSamples[monitorIndex]
				continue
			}
			break
		}
		var next *store.MonitorSample
		if monitorIndex < len(monitorSamples) {
			next = &monitorSamples[monitorIndex]
		}
		matched := chooseNearestMonitorSample(perfSample.Timestamp, previous, next)
		if matched == nil {
			out[i] = perfSample
			continue
		}
		out[i] = attachMonitorSample(perfSample, *matched)
	}

	return out
}

func chooseNearestMonitorSample(timestamp int64, previous, next *store.MonitorSample) *store.MonitorSample {
	if previous == nil {
		return next
	}
	if next == nil {
		return previous
	}
	if absInt64(previous.Timestamp-timestamp) <= absInt64(next.Timestamp-timestamp) {
		return previous
	}
	return next
}

func attachMonitorSample(sample model.PerfProgress, monitor store.MonitorSample) model.PerfProgress {
	sample.PromQps = preferFloatPtr(sample.PromQps, monitor.PromQps)
	sample.PromP99LatencyMs = preferFloatPtr(sample.PromP99LatencyMs, monitor.PromP99LatencyMs)
	sample.PromErrorRate = preferFloatPtr(sample.PromErrorRate, monitor.PromErrorRate)
	sample.PurchaseKafkaPublishRate = preferFloatPtr(sample.PurchaseKafkaPublishRate, monitor.PurchaseKafkaPublishRate)
	sample.PurchaseKafkaPublishFailed = preferFloatPtr(sample.PurchaseKafkaPublishFailed, monitor.PurchaseKafkaPublishFailed)
	sample.OrderStateConsumeRate = preferFloatPtr(sample.OrderStateConsumeRate, monitor.OrderStateConsumeRate)
	return sample
}

func enrichPerfRawLine(raw string, sample model.PerfProgress) string {
	if raw == "" {
		return raw
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw
	}
	summary, ok := payload["summary"].(map[string]any)
	if !ok || summary == nil {
		return raw
	}
	setSummaryScalar(summary, "prom_qps", sample.PromQps)
	setSummaryScalar(summary, "prom_p99_latency_ms", sample.PromP99LatencyMs)
	setSummaryScalar(summary, "prom_error_rate", sample.PromErrorRate)
	setSummaryScalar(summary, "purchase_kafka_publish_rate", sample.PurchaseKafkaPublishRate)
	setSummaryScalar(summary, "purchase_kafka_publish_failed", sample.PurchaseKafkaPublishFailed)
	setSummaryScalar(summary, "order_state_consume_rate", sample.OrderStateConsumeRate)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return string(encoded)
}

func enrichPerfReportSummary(report *model.PerfReport, sample model.PerfProgress) {
	if report == nil || len(report.Summary) == 0 {
		return
	}
	report.Summary = json.RawMessage(enrichPerfRawSummary(string(report.Summary), sample))
}

func enrichPerfRawSummary(raw string, sample model.PerfProgress) string {
	if raw == "" {
		return raw
	}
	var summary map[string]any
	if err := json.Unmarshal([]byte(raw), &summary); err != nil {
		return raw
	}
	setSummaryScalar(summary, "prom_qps", sample.PromQps)
	setSummaryScalar(summary, "prom_p99_latency_ms", sample.PromP99LatencyMs)
	setSummaryScalar(summary, "prom_error_rate", sample.PromErrorRate)
	setSummaryScalar(summary, "purchase_kafka_publish_rate", sample.PurchaseKafkaPublishRate)
	setSummaryScalar(summary, "purchase_kafka_publish_failed", sample.PurchaseKafkaPublishFailed)
	setSummaryScalar(summary, "order_state_consume_rate", sample.OrderStateConsumeRate)
	encoded, err := json.Marshal(summary)
	if err != nil {
		return raw
	}
	return string(encoded)
}

func setSummaryScalar(summary map[string]any, key string, value *float64) {
	if value == nil {
		return
	}
	summary[key] = *value
}

func preferFloatPtr(current, fallback *float64) *float64 {
	if current != nil {
		return cloneFloatPtr(current)
	}
	return cloneFloatPtr(fallback)
}

func cloneFloatPtr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
