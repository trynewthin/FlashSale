// logic 包包含相关应用代码。
package logic

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"flashsale/apps/order/rpc/internal/model"
	"flashsale/apps/order/rpc/internal/svc"
	"flashsale/pkg/base/eventx"
	"go.uber.org/zap"
)

// emitSeckillOrderStateEvent 发布秒杀订单状态事件到 Kafka。
func emitSeckillOrderStateEvent(ctx context.Context, svcCtx *svc.ServiceContext, order *model.Order, eventType string) {
	if svcCtx == nil || svcCtx.AppConfig == nil || svcCtx.Producer == nil || order == nil {
		return
	}
	if order.OrderSource != model.OrderSourceSeckill || order.SeckillActivityID <= 0 || order.SeckillActivityItemID <= 0 {
		return
	}
	topic := strings.TrimSpace(svcCtx.AppConfig.Kafka.Topics.SeckillOrderState)
	if topic == "" {
		return
	}
	if strings.TrimSpace(eventType) == "" {
		eventType = eventx.SeckillOrderStateEventTypeClosed
	}
	evt := eventx.SeckillOrderStateEvent{
		EventType:            eventType,
		OrderID:              order.ID,
		OrderNo:              order.OrderNo,
		UserID:               order.UserID,
		ProductID:            order.ProductID,
		ActivityID:           order.SeckillActivityID,
		ActivityItemID:       order.SeckillActivityItemID,
		Quantity:             order.Quantity,
		OrderStatus:          int32(order.OrderStatus),
		PaymentStatus:        int32(order.PaymentStatus),
		ReviewStatus:         int32(order.ReviewStatus),
		ShippingStatus:       int32(order.ShippingStatus),
		RefundStatus:         int32(order.RefundStatus),
		CloseReason:          strings.TrimSpace(order.CloseReason),
		OccurredAtUnixSecond: time.Now().Unix(),
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		svcCtx.Logger.Warn("marshal seckill order state event failed",
			zap.Error(err),
			zap.Int64("order_id", order.ID),
			zap.String("event_type", eventType),
		)
		return
	}
	if err := svcCtx.Producer.Publish(ctx, topic, []byte(order.OrderNo), payload, map[string]string{
		"event_type": eventType,
	}); err != nil {
		svcCtx.Logger.Warn("publish seckill order state event failed",
			zap.Error(err),
			zap.Int64("order_id", order.ID),
			zap.String("event_type", eventType),
			zap.String("topic", topic),
		)
	}
}
