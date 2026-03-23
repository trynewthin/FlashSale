package scheduler

import (
	"encoding/json"
	"testing"
	"time"

	"flashsale/ops/backend/model"
	"flashsale/ops/backend/store"
)

func TestMergePerfProgressSamplesUsesNearestMonitorSample(t *testing.T) {
	promQpsA := 1200.0
	publishRateA := 8.0
	consumeRateA := 6.0
	failedA := 1.0

	promQpsB := 1800.0
	publishRateB := 18.0
	consumeRateB := 16.0
	failedB := 3.0

	perfSamples := []model.PerfProgress{
		{Timestamp: 2_100, QPS: 900},
		{Timestamp: 3_800, QPS: 950},
	}
	monitorSamples := []store.MonitorSample{
		{
			Timestamp:                  2_000,
			PromQps:                    &promQpsA,
			PurchaseKafkaPublishRate:   &publishRateA,
			OrderStateConsumeRate:      &consumeRateA,
			PurchaseKafkaPublishFailed: &failedA,
		},
		{
			Timestamp:                  4_000,
			PromQps:                    &promQpsB,
			PurchaseKafkaPublishRate:   &publishRateB,
			OrderStateConsumeRate:      &consumeRateB,
			PurchaseKafkaPublishFailed: &failedB,
		},
	}

	merged := mergePerfProgressSamples(perfSamples, monitorSamples)
	if len(merged) != 2 {
		t.Fatalf("len=%d", len(merged))
	}
	if merged[0].PromQps == nil || *merged[0].PromQps != promQpsA {
		t.Fatalf("sample0 promQps=%v", merged[0].PromQps)
	}
	if merged[0].PurchaseKafkaPublishRate == nil || *merged[0].PurchaseKafkaPublishRate != publishRateA {
		t.Fatalf("sample0 publishRate=%v", merged[0].PurchaseKafkaPublishRate)
	}
	if merged[1].PromQps == nil || *merged[1].PromQps != promQpsB {
		t.Fatalf("sample1 promQps=%v", merged[1].PromQps)
	}
	if merged[1].PurchaseKafkaPublishFailed == nil || *merged[1].PurchaseKafkaPublishFailed != failedB {
		t.Fatalf("sample1 publishFailed=%v", merged[1].PurchaseKafkaPublishFailed)
	}
}

func TestFinalizePerfReportBackfillsMonitorMetrics(t *testing.T) {
	s := New(t.TempDir(), map[string]model.TaskDef{})

	promQps := 1600.0
	promP99 := 220.0
	promErrorRate := 0.8
	publishRate := 12.0
	consumeRate := 11.0
	publishFailed := 2.0

	now := time.Now().UnixMilli()
	if err := s.sampleStore.Append(&store.MonitorSample{
		Timestamp:                  now,
		Label:                      "10:00:00",
		PromQps:                    &promQps,
		PromP99LatencyMs:           &promP99,
		PromErrorRate:              &promErrorRate,
		PurchaseKafkaPublishRate:   &publishRate,
		PurchaseKafkaPublishFailed: &publishFailed,
		OrderStateConsumeRate:      &consumeRate,
	}); err != nil {
		t.Fatalf("append monitor sample: %v", err)
	}
	s.sampleStore.Close()

	rec := &jobRecord{
		perfSamples: []model.PerfProgress{
			{
				Timestamp:        now + 300,
				Label:            "10:00:00",
				QPS:              900,
				P95LatencyMs:     20,
				SuccessRate:      99,
				RejectRate:       1,
				SystemErrorRate:  0,
				NetworkErrorRate: 0,
				StockDeductRate:  98,
			},
		},
		perfReport: &model.PerfReport{GeneratedAt: time.Now().Format(time.RFC3339)},
	}

	s.finalizePerfReport(rec)

	if rec.PerfReport == nil || len(rec.PerfReport.Samples) != 1 {
		t.Fatalf("perf report samples=%v", rec.PerfReport)
	}
	got := rec.PerfReport.Samples[0]
	if got.PromQps == nil || *got.PromQps != promQps {
		t.Fatalf("promQps=%v", got.PromQps)
	}
	if got.PromP99LatencyMs == nil || *got.PromP99LatencyMs != promP99 {
		t.Fatalf("promP99=%v", got.PromP99LatencyMs)
	}
	if got.PromErrorRate == nil || *got.PromErrorRate != promErrorRate {
		t.Fatalf("promErrorRate=%v", got.PromErrorRate)
	}
	if got.PurchaseKafkaPublishRate == nil || *got.PurchaseKafkaPublishRate != publishRate {
		t.Fatalf("publishRate=%v", got.PurchaseKafkaPublishRate)
	}
	if got.OrderStateConsumeRate == nil || *got.OrderStateConsumeRate != consumeRate {
		t.Fatalf("consumeRate=%v", got.OrderStateConsumeRate)
	}
	if got.PurchaseKafkaPublishFailed == nil || *got.PurchaseKafkaPublishFailed != publishFailed {
		t.Fatalf("publishFailed=%v", got.PurchaseKafkaPublishFailed)
	}
}

func TestEnrichPerfRawLineAddsServerAndKafkaFieldsToSummary(t *testing.T) {
	promQps := 1500.0
	promP99 := 180.0
	promErrorRate := 0.5
	publishRate := 10.0
	consumeRate := 8.0
	publishFailed := 2.0

	raw := `{"kind":"progress","generated_at":"2026-03-22T10:23:54Z","summary":{"total":10,"rps":100}}`
	enriched := enrichPerfRawLine(raw, model.PerfProgress{
		PromQps:                    &promQps,
		PromP99LatencyMs:           &promP99,
		PromErrorRate:              &promErrorRate,
		PurchaseKafkaPublishRate:   &publishRate,
		PurchaseKafkaPublishFailed: &publishFailed,
		OrderStateConsumeRate:      &consumeRate,
	})

	var payload struct {
		Summary map[string]float64 `json:"summary"`
	}
	if err := json.Unmarshal([]byte(enriched), &payload); err != nil {
		t.Fatalf("unmarshal enriched line: %v", err)
	}
	if payload.Summary["prom_qps"] != promQps {
		t.Fatalf("prom_qps=%v", payload.Summary["prom_qps"])
	}
	if payload.Summary["prom_p99_latency_ms"] != promP99 {
		t.Fatalf("prom_p99_latency_ms=%v", payload.Summary["prom_p99_latency_ms"])
	}
	if payload.Summary["prom_error_rate"] != promErrorRate {
		t.Fatalf("prom_error_rate=%v", payload.Summary["prom_error_rate"])
	}
	if payload.Summary["purchase_kafka_publish_rate"] != publishRate {
		t.Fatalf("purchase_kafka_publish_rate=%v", payload.Summary["purchase_kafka_publish_rate"])
	}
	if payload.Summary["purchase_kafka_publish_failed"] != publishFailed {
		t.Fatalf("purchase_kafka_publish_failed=%v", payload.Summary["purchase_kafka_publish_failed"])
	}
	if payload.Summary["order_state_consume_rate"] != consumeRate {
		t.Fatalf("order_state_consume_rate=%v", payload.Summary["order_state_consume_rate"])
	}
}
