// logic 包包含相关应用代码。
package logic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/repository"
	"flashsale/apps/seckill/rpc/internal/svc"
	"flashsale/pkg/base/eventx"
	"flashsale/pkg/base/kafkax"
	"go.uber.org/zap"
)

const seckillOrderConsumerGroupSuffix = "seckill-order-state"

// RunOrderStateConsumer 启动订单状态回流消费者。
func RunOrderStateConsumer(ctx context.Context, svcCtx *svc.ServiceContext) {
	if svcCtx == nil || svcCtx.Consumer == nil || svcCtx.AppConfig == nil {
		return
	}
	topic := strings.TrimSpace(svcCtx.AppConfig.Kafka.Topics.SeckillOrderState)
	if topic == "" {
		return
	}
	group := strings.TrimSpace(svcCtx.AppConfig.Kafka.GroupID)
	if group == "" {
		group = "flashsale"
	}
	group = group + "-" + seckillOrderConsumerGroupSuffix

	for {
		if ctx.Err() != nil {
			return
		}
		err := svcCtx.Consumer.Consume(ctx, group, []string{topic}, func(consumeCtx context.Context, msg kafkax.Message) error {
			return consumeOrderStateMessage(consumeCtx, svcCtx, msg)
		})
		if err == nil || ctx.Err() != nil {
			return
		}
		if svcCtx.Perf != nil {
			svcCtx.Perf.MarkOrderStateLoopError()
		}
		svcCtx.Logger.Warn("consume seckill order state failed", zap.Error(err), zap.String("topic", topic), zap.String("group", group))
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

// consumeOrderStateMessage 消费单条订单状态事件并驱动秒杀侧同步。
func consumeOrderStateMessage(ctx context.Context, svcCtx *svc.ServiceContext, msg kafkax.Message) error {
	if svcCtx == nil {
		return nil
	}
	if svcCtx.Perf != nil {
		svcCtx.Perf.MarkOrderStateConsumed()
	}
	var evt eventx.SeckillOrderStateEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		if svcCtx.Perf != nil {
			svcCtx.Perf.MarkOrderStateDecodeFailed()
		}
		svcCtx.Logger.Warn("decode seckill order state payload failed", zap.Error(err))
		return nil
	}
	if evt.ActivityID <= 0 || evt.ActivityItemID <= 0 || evt.OrderID <= 0 || evt.Quantity <= 0 {
		return nil
	}

	logic := NewSeckillLogic(ctx, svcCtx)
	syncModel := &model.OrderStateSync{
		OrderID:        evt.OrderID,
		OrderNo:        evt.OrderNo,
		UserID:         evt.UserID,
		ActivityID:     evt.ActivityID,
		ActivityItemID: evt.ActivityItemID,
		Quantity:       evt.Quantity,
		OrderStatus:    int8(evt.OrderStatus),
		PaymentStatus:  int8(evt.PaymentStatus),
		CloseReason:    strings.TrimSpace(evt.CloseReason),
		LastSyncedAt:   evt.OccurredAtTime(),
	}
	if err := svcCtx.SeckillRepo.SyncOrderLinkState(ctx, syncModel); err != nil {
		if svcCtx.Perf != nil {
			svcCtx.Perf.MarkOrderStateSyncFailed()
		}
		return err
	}

	if evt.PaymentStatus > 0 {
		_ = logic.recordTraffic(&model.TrafficEvent{
			ActivityID:     evt.ActivityID,
			ActivityItemID: evt.ActivityItemID,
			EventType:      model.TrafficEventPaySuccess,
			UserID:         evt.UserID,
			IdempotencyKey: evt.OrderNo + ":pay_success",
			OccurredAt:     evt.OccurredAtTime(),
		})
	}
	if isOrderClosed(evt.OrderStatus) {
		_ = logic.recordTraffic(&model.TrafficEvent{
			ActivityID:     evt.ActivityID,
			ActivityItemID: evt.ActivityItemID,
			EventType:      model.TrafficEventOrderClosed,
			UserID:         evt.UserID,
			IdempotencyKey: evt.OrderNo + ":order_closed",
			OccurredAt:     evt.OccurredAtTime(),
		})
	}

	item, err := svcCtx.SeckillRepo.FindActivityItem(ctx, evt.ActivityID, evt.ActivityItemID)
	if err != nil {
		if isOrderStateSkipErr(err) {
			if svcCtx.Perf != nil {
				svcCtx.Perf.MarkOrderStateSkippedMissing()
			}
			svcCtx.Logger.Debug("skip seckill order state event: activity item missing",
				zap.Int64("order_id", evt.OrderID),
				zap.Int64("activity_id", evt.ActivityID),
				zap.Int64("activity_item_id", evt.ActivityItemID),
				zap.Error(err),
			)
			return nil
		}
		return err
	}
	if shouldReleaseActivityStock(evt.CloseReason) {
		releaseID := evt.OrderNo + ":close_release"
		if err := svcCtx.SeckillRepo.ReleasePurchaseByOrder(ctx, evt.ActivityID, evt.ActivityItemID, evt.OrderID, evt.Quantity, releaseID); err != nil {
			if svcCtx.Perf != nil {
				svcCtx.Perf.MarkOrderStateReleaseFailed()
			}
			return err
		}
		if svcCtx.Perf != nil {
			svcCtx.Perf.MarkOrderStateReleaseSuccess()
		}
		logic.releaseByCloseInCache(ctx, item, evt.UserID, evt.Quantity, releaseID)
	}
	if shouldRecordWindowCompleted(evt.CloseReason) {
		if svcCtx.Perf != nil {
			svcCtx.Perf.MarkOrderStateWindowRecorded()
		}
		logic.recordCompletedWindow(ctx, item, evt.UserID, evt.Quantity, evt.OrderID, evt.OccurredAtTime())
	}
	return nil
}

// shouldReleaseActivityStock 判断关闭原因是否需要回补活动库存。
func shouldReleaseActivityStock(closeReason string) bool {
	switch strings.TrimSpace(closeReason) {
	case eventx.CloseReasonUserCancel,
		eventx.CloseReasonPayTimeout,
		eventx.CloseReasonAuditReject,
		eventx.CloseReasonAuditTimeout:
		return true
	default:
		return false
	}
}

// shouldRecordWindowCompleted 判断是否应记入窗口限购已完成计数。
func shouldRecordWindowCompleted(closeReason string) bool {
	switch strings.TrimSpace(closeReason) {
	case eventx.CloseReasonCompleted, eventx.CloseReasonAutoCompleted:
		return true
	default:
		return false
	}
}

// isOrderClosed 判断订单状态是否为已关闭。
func isOrderClosed(orderStatus int32) bool {
	return orderStatus == 90
}

// isOrderStateSkipErr 判断订单状态回流是否可跳过该错误。
func isOrderStateSkipErr(err error) bool {
	return errors.Is(err, repository.ErrActivityNotFound) ||
		errors.Is(err, repository.ErrActivityItemNotFound)
}
