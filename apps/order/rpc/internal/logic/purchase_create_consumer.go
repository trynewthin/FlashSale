package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"flashsale/apps/order/rpc/internal/svc"
	"flashsale/apps/order/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/eventx"
	"flashsale/pkg/base/kafkax"
	"go.uber.org/zap"
)

const (
	seckillPurchaseCreateConsumerGroupSuffix = "seckill-purchase-create"
	seckillPurchaseCreateRetryCount          = 2
	seckillPurchaseCreateRetryBackoff        = 30 * time.Millisecond
)

// RunSeckillPurchaseCreateConsumer 启动秒杀异步建单消费者。
func RunSeckillPurchaseCreateConsumer(ctx context.Context, svcCtx *svc.ServiceContext) {
	if svcCtx == nil || svcCtx.Consumer == nil || svcCtx.AppConfig == nil {
		return
	}
	topic := strings.TrimSpace(svcCtx.AppConfig.Kafka.Topics.SeckillPurchaseCreate)
	if topic == "" {
		return
	}
	group := strings.TrimSpace(svcCtx.AppConfig.Kafka.GroupID)
	if group == "" {
		group = "flashsale"
	}
	group = group + "-" + seckillPurchaseCreateConsumerGroupSuffix

	for {
		if ctx.Err() != nil {
			return
		}
		err := svcCtx.Consumer.Consume(ctx, group, []string{topic}, func(consumeCtx context.Context, msg kafkax.Message) error {
			return consumeSeckillPurchaseCreateMessage(consumeCtx, svcCtx, msg)
		})
		if err == nil || ctx.Err() != nil {
			return
		}
		svcCtx.Logger.Warn("consume seckill purchase create failed", zap.Error(err), zap.String("topic", topic), zap.String("group", group))
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func consumeSeckillPurchaseCreateMessage(ctx context.Context, svcCtx *svc.ServiceContext, msg kafkax.Message) error {
	if svcCtx == nil {
		return nil
	}
	var evt eventx.SeckillPurchaseCreateEvent
	if decodeErr := json.Unmarshal(msg.Value, &evt); decodeErr != nil {
		svcCtx.Logger.Warn("decode seckill purchase create payload failed", zap.Error(decodeErr))
		if err := publishPurchaseCreateDLQ(ctx, svcCtx, msg, "decode_failed"); err != nil {
			return err
		}
		// 无法可靠提取补偿字段时，保留消息重试，避免直接提交 offset 造成库存泄漏。
		return fmt.Errorf("decode seckill purchase create payload: %w", decodeErr)
	}
	if err := validateSeckillPurchaseCreateEvent(&evt); err != nil {
		svcCtx.Logger.Warn("invalid seckill purchase create payload", zap.Error(err), zap.String("order_no", evt.OrderNo))
		if dlqErr := publishPurchaseCreateDLQ(ctx, svcCtx, msg, "invalid_payload"); dlqErr != nil {
			return dlqErr
		}
		return publishStockCompensate(ctx, svcCtx, &evt)
	}

	var lastErr error
	for attempt := 0; attempt <= seckillPurchaseCreateRetryCount; attempt++ {
		_, lastErr = NewCreateOrderFromSeckillLogic(ctx, svcCtx).CreateOrderFromSeckill(&pb.CreateOrderFromSeckillReq{
			UserId:            evt.UserID,
			ActivityId:        evt.ActivityID,
			ActivityItemId:    evt.ActivityItemID,
			ProductId:         evt.ProductID,
			Quantity:          evt.Quantity,
			SeckillPriceCent:  evt.SeckillPriceCent,
			SnapshotName:      evt.SnapshotName,
			SnapshotMainImage: evt.SnapshotMainImage,
			SkuCode:           evt.SKUCode,
			IdempotencyKey:    evt.IdempotencyKey,
		})
		if lastErr == nil {
			return nil
		}
		appErr := errorx.FromError(lastErr)
		if isNonRetryablePurchaseCreateError(appErr) {
			if err := publishPurchaseCreateDLQ(ctx, svcCtx, msg, "non_retryable"); err != nil {
				return err
			}
			return publishStockCompensate(ctx, svcCtx, &evt)
		}
		if attempt < seckillPurchaseCreateRetryCount {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(seckillPurchaseCreateRetryBackoff):
			}
		}
	}
	svcCtx.Logger.Warn("seckill purchase create exhausted retries",
		zap.Error(lastErr),
		zap.String("order_no", evt.OrderNo),
		zap.Int64("activity_id", evt.ActivityID),
		zap.Int64("activity_item_id", evt.ActivityItemID),
	)
	if err := publishPurchaseCreateDLQ(ctx, svcCtx, msg, "retries_exhausted"); err != nil {
		return err
	}
	return publishStockCompensate(ctx, svcCtx, &evt)
}

func validateSeckillPurchaseCreateEvent(evt *eventx.SeckillPurchaseCreateEvent) error {
	if evt == nil {
		return errors.New("event is nil")
	}
	if strings.TrimSpace(evt.OrderNo) == "" || strings.TrimSpace(evt.IdempotencyKey) == "" {
		return errors.New("order_no or idempotency_key is empty")
	}
	if evt.UserID <= 0 || evt.ActivityID <= 0 || evt.ActivityItemID <= 0 || evt.ProductID <= 0 || evt.Quantity <= 0 || evt.SeckillPriceCent <= 0 {
		return errors.New("numeric fields are invalid")
	}
	if strings.TrimSpace(evt.SnapshotName) == "" || strings.TrimSpace(evt.SnapshotMainImage) == "" || strings.TrimSpace(evt.SKUCode) == "" {
		return errors.New("snapshot fields are empty")
	}
	return nil
}

func isNonRetryablePurchaseCreateError(appErr *errorx.AppError) bool {
	if appErr == nil {
		return false
	}
	switch appErr.Code {
	case errorx.CodeSysBadRequest, errorx.CodeAuthUnauthorized, errorx.CodeAuthForbidden:
		return true
	default:
		return false
	}
}

func publishPurchaseCreateDLQ(ctx context.Context, svcCtx *svc.ServiceContext, msg kafkax.Message, reason string) error {
	if svcCtx == nil || svcCtx.Producer == nil || svcCtx.AppConfig == nil {
		return nil
	}
	dlqTopic := strings.TrimSpace(svcCtx.AppConfig.Kafka.Topics.SeckillPurchaseCreateDLQ)
	if dlqTopic == "" {
		return nil
	}
	pubCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	return svcCtx.Producer.Publish(pubCtx, dlqTopic, msg.Key, msg.Value, map[string]string{"reason": strings.TrimSpace(reason)})
}

func publishStockCompensate(ctx context.Context, svcCtx *svc.ServiceContext, evt *eventx.SeckillPurchaseCreateEvent) error {
	if svcCtx == nil || svcCtx.Producer == nil || svcCtx.AppConfig == nil || evt == nil {
		return nil
	}
	payload, err := json.Marshal(&eventx.SeckillStockCompensateEvent{
		OrderNo:        evt.OrderNo,
		UserID:         evt.UserID,
		ActivityID:     evt.ActivityID,
		ActivityItemID: evt.ActivityItemID,
		Quantity:       evt.Quantity,
		IdempotencyKey: evt.IdempotencyKey,
		Reason:         eventx.SeckillStockCompensateReasonOrderCreateFailed,
		OccurredAtUnix: time.Now().Unix(),
	})
	if err != nil {
		return err
	}
	pubCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	return svcCtx.Producer.Publish(
		pubCtx,
		svcCtx.AppConfig.Kafka.Topics.StockCompensate,
		[]byte(strings.TrimSpace(evt.OrderNo)),
		payload,
		map[string]string{kafkax.IdempotencyHeader: strings.TrimSpace(evt.IdempotencyKey)},
	)
}
