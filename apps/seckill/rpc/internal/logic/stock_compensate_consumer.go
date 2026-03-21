package logic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/svc"
	"flashsale/pkg/base/eventx"
	"flashsale/pkg/base/kafkax"
	"go.uber.org/zap"
)

const seckillStockCompensateConsumerGroupSuffix = "seckill-stock-compensate"

// RunStockCompensateConsumer 启动库存补偿消费者。
func RunStockCompensateConsumer(ctx context.Context, svcCtx *svc.ServiceContext) {
	if svcCtx == nil || svcCtx.Consumer == nil || svcCtx.AppConfig == nil {
		return
	}
	topic := strings.TrimSpace(svcCtx.AppConfig.Kafka.Topics.StockCompensate)
	if topic == "" {
		return
	}
	group := strings.TrimSpace(svcCtx.AppConfig.Kafka.GroupID)
	if group == "" {
		group = "flashsale"
	}
	group = group + "-" + seckillStockCompensateConsumerGroupSuffix

	for {
		if ctx.Err() != nil {
			return
		}
		err := svcCtx.Consumer.Consume(ctx, group, []string{topic}, func(consumeCtx context.Context, msg kafkax.Message) error {
			return consumeStockCompensateMessage(consumeCtx, svcCtx, msg)
		})
		if err == nil || ctx.Err() != nil {
			return
		}
		if svcCtx.Perf != nil {
			svcCtx.Perf.MarkStockCompensateLoopError()
		}
		svcCtx.Logger.Warn("consume stock compensate failed", zap.Error(err), zap.String("topic", topic), zap.String("group", group))
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func consumeStockCompensateMessage(ctx context.Context, svcCtx *svc.ServiceContext, msg kafkax.Message) error {
	if svcCtx == nil {
		return nil
	}
	if svcCtx.Perf != nil {
		svcCtx.Perf.MarkStockCompensateConsumed()
	}
	var evt eventx.SeckillStockCompensateEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		if svcCtx.Perf != nil {
			svcCtx.Perf.MarkStockCompensateDecodeFailed()
		}
		svcCtx.Logger.Warn("decode stock compensate payload failed", zap.Error(err))
		return err
	}
	if evt.ActivityID <= 0 || evt.ActivityItemID <= 0 || evt.UserID <= 0 || evt.Quantity <= 0 || strings.TrimSpace(evt.IdempotencyKey) == "" {
		if svcCtx.Perf != nil {
			svcCtx.Perf.MarkStockCompensateFailed()
		}
		return errors.New("invalid stock compensate payload")
	}

	item, err := svcCtx.SeckillRepo.FindActivityItem(ctx, evt.ActivityID, evt.ActivityItemID)
	if err != nil {
		if svcCtx.Perf != nil {
			svcCtx.Perf.MarkStockCompensateFailed()
		}
		svcCtx.Logger.Warn("find activity item for compensate failed",
			zap.Error(err),
			zap.Int64("activity_id", evt.ActivityID),
			zap.Int64("activity_item_id", evt.ActivityItemID),
			zap.String("order_no", evt.OrderNo),
		)
		return err
	}
	keys := []string{
		cacheKeyItemStock(item.ID),
		cacheKeyActivityUserBought(item.ActivityID, evt.UserID),
		cacheKeyReleaseToken(evt.ActivityID, evt.ActivityItemID, evt.UserID, evt.IdempotencyKey),
	}
	if _, err := rollbackReserveScript.Run(ctx, svcCtx.Redis, keys, evt.Quantity, item.UserLimitMode).Result(); err != nil {
		if svcCtx.Perf != nil {
			svcCtx.Perf.MarkStockCompensateFailed()
		}
		svcCtx.Logger.Warn("rollback reserve in cache by compensate failed",
			zap.Error(err),
			zap.String("order_no", evt.OrderNo),
			zap.Int64("activity_id", evt.ActivityID),
			zap.Int64("activity_item_id", evt.ActivityItemID),
			zap.Int64("user_id", evt.UserID),
		)
		return err
	}
	if svcCtx.Perf != nil {
		svcCtx.Perf.MarkStockCompensateSuccess()
	}
	logic := NewSeckillLogic(ctx, svcCtx)
	occurredAt := time.Now()
	if evt.OccurredAtUnix > 0 {
		occurredAt = time.Unix(evt.OccurredAtUnix, 0)
	}
	_ = logic.recordTraffic(&model.TrafficEvent{
		ActivityID:     evt.ActivityID,
		ActivityItemID: evt.ActivityItemID,
		EventType:      model.TrafficEventPurchaseFail,
		UserID:         evt.UserID,
		IdempotencyKey: strings.TrimSpace(evt.OrderNo) + ":compensate",
		OccurredAt:     occurredAt,
	})
	return nil
}
