package svc

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestPurchaseTaskMetricsCollectorExposesKafkaSeries(t *testing.T) {
	t.Parallel()

	svc := &ServiceContext{
		Perf: newPerfStats(),
	}
	svc.Perf.MarkPurchaseKafkaPublished()
	svc.Perf.MarkPurchaseKafkaPublished()
	svc.Perf.MarkPurchaseKafkaPublishFailed()
	svc.Perf.MarkOrderStateConsumed()
	svc.Perf.MarkOrderStateConsumed()
	svc.Perf.MarkOrderStateConsumed()

	reg := prometheus.NewRegistry()
	if _, _, err := registerPurchaseKafkaMetrics(reg, svc); err != nil {
		t.Fatalf("register metrics: %v", err)
	}

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	if got := metricValue(t, families, "seckill_purchase_kafka_published_total"); got != 2 {
		t.Fatalf("published total mismatch: got=%v want=2", got)
	}
	if got := metricValue(t, families, "seckill_purchase_kafka_publish_failed_total"); got != 1 {
		t.Fatalf("publish failed total mismatch: got=%v want=1", got)
	}
	if got := metricValue(t, families, "seckill_order_state_consumed_total"); got != 3 {
		t.Fatalf("order state consumed total mismatch: got=%v want=3", got)
	}
}

func metricValue(t *testing.T, families []*dto.MetricFamily, name string) float64 {
	t.Helper()
	for _, family := range families {
		if family.GetName() != name || len(family.Metric) == 0 {
			continue
		}
		metric := family.Metric[0]
		if metric.Gauge != nil {
			return metric.GetGauge().GetValue()
		}
		if metric.Counter != nil {
			return metric.GetCounter().GetValue()
		}
	}
	t.Fatalf("metric %s not found", name)
	return 0
}

func TestRegisterPurchaseKafkaMetricsReportsOwnership(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	firstCollector, firstOwned, err := registerPurchaseKafkaMetrics(reg, &ServiceContext{Perf: newPerfStats()})
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	if !firstOwned {
		t.Fatal("first register should own collector")
	}

	secondCollector, secondOwned, err := registerPurchaseKafkaMetrics(reg, &ServiceContext{Perf: newPerfStats()})
	if err != nil {
		t.Fatalf("second register: %v", err)
	}
	if secondOwned {
		t.Fatal("duplicate register should not own collector")
	}
	if firstCollector != secondCollector {
		t.Fatal("duplicate register should reuse existing collector")
	}
}
