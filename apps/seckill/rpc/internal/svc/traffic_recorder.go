package svc

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/repository"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/kafkax"

	baseconfig "flashsale/pkg/base/config"
)

// TrafficRecorder 封装流量埋点的同步/异步写入和 Kafka 发布。
type TrafficRecorder struct {
	writeTimeout   time.Duration
	publishTimeout time.Duration
	repo           repository.SeckillRepository
	producer       kafkax.Producer
	appConfig      *baseconfig.AppConfig
	perf           *PerfStats
}

// NewTrafficRecorder 创建流量记录器。
func NewTrafficRecorder(writeTimeout, publishTimeout time.Duration, repo repository.SeckillRepository, producer kafkax.Producer, appConfig *baseconfig.AppConfig, perf *PerfStats) *TrafficRecorder {
	return &TrafficRecorder{
		writeTimeout:   writeTimeout,
		publishTimeout: publishTimeout,
		repo:           repo,
		producer:       producer,
		appConfig:      appConfig,
		perf:           perf,
	}
}

// Record 同步记录埋点并尽力发布 Kafka 事件。
func (t *TrafficRecorder) Record(event *model.TrafficEvent) error {
	if t == nil || event == nil || t.repo == nil {
		return nil
	}
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), t.WriteTimeout())
	defer cancelWrite()
	writeErr := t.repo.RecordTraffic(writeCtx, event)

	payload, _ := json.Marshal(event)
	enablePublish := t.producer != nil && t.appConfig != nil
	var publishErr error
	if enablePublish {
		pubCtx, cancelPub := context.WithTimeout(context.Background(), t.PublishTimeout())
		publishErr = t.producer.Publish(pubCtx,
			t.appConfig.Kafka.Topics.SeckillTrafficRaw,
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

// WriteTimeout 返回秒杀埋点写库超时配置。
func (t *TrafficRecorder) WriteTimeout() time.Duration {
	if t == nil || t.writeTimeout <= 0 {
		return 120 * time.Millisecond
	}
	return t.writeTimeout
}

// PublishTimeout 返回秒杀埋点发布超时配置。
func (t *TrafficRecorder) PublishTimeout() time.Duration {
	if t == nil || t.publishTimeout <= 0 {
		return 120 * time.Millisecond
	}
	return t.publishTimeout
}
