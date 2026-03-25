package svc

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/pkg/base/errorx"
)

// RecordTrafficEvent 同步记录埋点并尽力发布 Kafka 事件。
func (s *ServiceContext) RecordTrafficEvent(event *model.TrafficEvent) error {
	if s == nil || event == nil || s.SeckillRepo == nil {
		return nil
	}
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), s.TrafficWriteTimeout())
	defer cancelWrite()
	writeErr := s.SeckillRepo.RecordTraffic(writeCtx, event)

	payload, _ := json.Marshal(event)
	enablePublish := s.Producer != nil && s.AppConfig != nil
	var publishErr error
	if enablePublish {
		pubCtx, cancelPub := context.WithTimeout(context.Background(), s.TrafficPublishTimeout())
		publishErr = s.Producer.Publish(pubCtx,
			s.AppConfig.Kafka.Topics.SeckillTrafficRaw,
			[]byte(strings.TrimSpace(event.IdempotencyKey)),
			payload,
			map[string]string{"event_type": event.EventType},
		)
		cancelPub()
	}
	if !enablePublish {
		if writeErr != nil {
			return errorx.Wrap(errorx.CodeDBError, "记录流量事件失败", writeErr)
		}
		return nil
	}
	if writeErr == nil || publishErr == nil {
		return nil
	}
	return errorx.Wrap(errorx.CodeDBError, "记录流量事件失败", writeErr)
}

// TrafficWriteTimeout 返回秒杀埋点写库超时配置。
func (s *ServiceContext) TrafficWriteTimeout() time.Duration {
	if s == nil || s.trafficWriteTimeout <= 0 {
		return 120 * time.Millisecond
	}
	return s.trafficWriteTimeout
}

// TrafficPublishTimeout 返回秒杀埋点发布超时配置。
func (s *ServiceContext) TrafficPublishTimeout() time.Duration {
	if s == nil || s.trafficPublishTimeout <= 0 {
		return 120 * time.Millisecond
	}
	return s.trafficPublishTimeout
}
