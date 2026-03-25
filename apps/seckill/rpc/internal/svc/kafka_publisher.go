package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"flashsale/pkg/base/eventx"
	"flashsale/pkg/base/kafkax"
)

// PublishSeckillPurchaseCreateEvent 发布秒杀建单消息。
func (s *ServiceContext) PublishSeckillPurchaseCreateEvent(event *eventx.SeckillPurchaseCreateEvent) error {
	if s == nil || event == nil {
		return fmt.Errorf("service context or event is nil")
	}
	if s.Producer == nil || s.AppConfig == nil {
		return fmt.Errorf("kafka producer not initialized")
	}
	payload, err := json.Marshal(event)
	if err != nil {
		if s.Perf != nil {
			s.Perf.MarkPurchaseKafkaPublishFailed()
		}
		return fmt.Errorf("marshal purchase create event: %w", err)
	}
	pubCtx, cancel := context.WithTimeout(context.Background(), s.TrafficPublishTimeout())
	defer cancel()
	err = s.Producer.Publish(pubCtx,
		s.AppConfig.Kafka.Topics.SeckillPurchaseCreate,
		[]byte(strings.TrimSpace(event.OrderNo)),
		payload,
		map[string]string{kafkax.IdempotencyHeader: strings.TrimSpace(event.IdempotencyKey)},
	)
	if err != nil {
		if s.Perf != nil {
			s.Perf.MarkPurchaseKafkaPublishFailed()
		}
		return err
	}
	if s.Perf != nil {
		s.Perf.MarkPurchaseKafkaPublished()
	}
	return nil
}

// PurchaseKafkaPublishedTotal 返回已发布 Kafka 消息总数（Prometheus metrics 接口）。
func (s *ServiceContext) PurchaseKafkaPublishedTotal() float64 {
	if s == nil || s.Perf == nil {
		return 0
	}
	return float64(s.Perf.PurchaseKafkaPublished())
}

// PurchaseKafkaPublishFailedTotal 返回发布失败 Kafka 消息总数（Prometheus metrics 接口）。
func (s *ServiceContext) PurchaseKafkaPublishFailedTotal() float64 {
	if s == nil || s.Perf == nil {
		return 0
	}
	return float64(s.Perf.PurchaseKafkaPublishFailed())
}

// OrderStateConsumedTotal 返回已消费订单状态消息总数（Prometheus metrics 接口）。
func (s *ServiceContext) OrderStateConsumedTotal() float64 {
	if s == nil || s.Perf == nil {
		return 0
	}
	return float64(s.Perf.OrderStateConsumed())
}
