package svc

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

type purchaseKafkaMetricsSource interface {
	PurchaseKafkaPublishedTotal() float64
	PurchaseKafkaPublishFailedTotal() float64
	OrderStateConsumedTotal() float64
}

type purchaseKafkaMetricsCollector struct {
	source            purchaseKafkaMetricsSource
	publishedDesc     *prometheus.Desc
	publishFailedDesc *prometheus.Desc
	orderStateDesc    *prometheus.Desc
}

func newPurchaseKafkaMetricsCollector(source purchaseKafkaMetricsSource) *purchaseKafkaMetricsCollector {
	return &purchaseKafkaMetricsCollector{
		source: source,
		publishedDesc: prometheus.NewDesc(
			"seckill_purchase_kafka_published_total",
			"Total Kafka purchase-create events published by seckill-rpc",
			nil,
			nil,
		),
		publishFailedDesc: prometheus.NewDesc(
			"seckill_purchase_kafka_publish_failed_total",
			"Total Kafka purchase-create publish failures in seckill-rpc",
			nil,
			nil,
		),
		orderStateDesc: prometheus.NewDesc(
			"seckill_order_state_consumed_total",
			"Total Kafka order-state events consumed by seckill-rpc",
			nil,
			nil,
		),
	}
}

func (c *purchaseKafkaMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.publishedDesc
	ch <- c.publishFailedDesc
	ch <- c.orderStateDesc
}

func (c *purchaseKafkaMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	if c == nil || c.source == nil {
		return
	}
	ch <- prometheus.MustNewConstMetric(c.publishedDesc, prometheus.CounterValue, c.source.PurchaseKafkaPublishedTotal())
	ch <- prometheus.MustNewConstMetric(c.publishFailedDesc, prometheus.CounterValue, c.source.PurchaseKafkaPublishFailedTotal())
	ch <- prometheus.MustNewConstMetric(c.orderStateDesc, prometheus.CounterValue, c.source.OrderStateConsumedTotal())
}

func registerPurchaseKafkaMetrics(reg prometheus.Registerer, source purchaseKafkaMetricsSource) (prometheus.Collector, bool, error) {
	if reg == nil {
		return nil, false, fmt.Errorf("prometheus registerer is nil")
	}
	collector := newPurchaseKafkaMetricsCollector(source)
	if err := reg.Register(collector); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return already.ExistingCollector, false, nil
		}
		return nil, false, err
	}
	return collector, true, nil
}
