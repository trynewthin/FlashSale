// logic 包包含相关应用代码。
package logic

import (
	"context"
	"time"

	"flashsale/apps/order/rpc/internal/model"
	"flashsale/apps/order/rpc/internal/repository"
	"flashsale/apps/order/rpc/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// TimeoutJobLogic 封装订单超时推进任务。
type TimeoutJobLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTimeoutJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TimeoutJobLogic {
	return &TimeoutJobLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// RunOnce 执行一轮超时状态推进。
func (l *TimeoutJobLogic) RunOnce() {
	now := time.Now()
	l.handlePayTimeout(now)
	l.handleReviewTimeout(now)
	l.handleStockReleaseRetry(now)
	l.handleRefundCompletion(now)
	l.handleAutoReceive(now)
}

func (l *TimeoutJobLogic) handlePayTimeout(now time.Time) {
	items, err := l.svcCtx.OrderRepo.ListPayTimeout(l.ctx, now.Add(-payTimeoutDur), backgroundBatchSize)
	if err != nil {
		l.Logger.Errorf("list pay timeout orders failed: %v", err)
		return
	}
	for _, item := range items {
		event := model.OrderEvent{
			OrderID:      item.ID,
			EventType:    "order_closed_by_pay_timeout",
			OperatorType: "system",
			OperatorID:   0,
			Payload:      "{}",
			CreatedAt:    now,
		}
		if err := l.svcCtx.OrderRepo.ClosePayTimeout(l.ctx, item.ID, now, event); err != nil {
			if err != repository.ErrOrderStateConflict {
				l.Logger.Errorf("close pay timeout order failed order_id=%d err=%v", item.ID, err)
			}
			continue
		}
		releaseAt := time.Now()
		remain, err := releaseStockForOrder(l.ctx, l.svcCtx, item.OrderNo, item.ProductID, item.Quantity, "release:pay_timeout")
		if err != nil {
			l.Logger.Errorf("release stock for pay timeout failed order_id=%d err=%v", item.ID, err)
			continue
		}
		if err := markOrderStockReleased(l.ctx, l.svcCtx, item.ID, remain, "pay_timeout", releaseAt); err != nil {
			l.Logger.Errorf("mark stock released for pay timeout failed order_id=%d err=%v", item.ID, err)
		}
	}
}

func (l *TimeoutJobLogic) handleReviewTimeout(now time.Time) {
	items, err := l.svcCtx.OrderRepo.ListReviewTimeout(l.ctx, now, backgroundBatchSize)
	if err != nil {
		l.Logger.Errorf("list review timeout orders failed: %v", err)
		return
	}
	for _, item := range items {
		event := model.OrderEvent{
			OrderID:      item.ID,
			EventType:    "order_review_timeout_rejected",
			OperatorType: "system",
			OperatorID:   0,
			Payload:      "{}",
			CreatedAt:    now,
		}
		if err := l.svcCtx.OrderRepo.CloseReviewTimeoutRefunding(l.ctx, item.ID, now, now.Add(virtualRefundDur), event); err != nil {
			if err != repository.ErrOrderStateConflict {
				l.Logger.Errorf("close review timeout order failed order_id=%d err=%v", item.ID, err)
			}
			continue
		}
		releaseAt := time.Now()
		remain, err := releaseStockForOrder(l.ctx, l.svcCtx, item.OrderNo, item.ProductID, item.Quantity, "release:review_timeout")
		if err != nil {
			l.Logger.Errorf("release stock for review timeout failed order_id=%d err=%v", item.ID, err)
			continue
		}
		if err := markOrderStockReleased(l.ctx, l.svcCtx, item.ID, remain, "review_timeout", releaseAt); err != nil {
			l.Logger.Errorf("mark stock released for review timeout failed order_id=%d err=%v", item.ID, err)
		}
	}
}

func (l *TimeoutJobLogic) handleStockReleaseRetry(now time.Time) {
	items, err := l.svcCtx.OrderRepo.ListStockReleasePending(l.ctx, backgroundBatchSize)
	if err != nil {
		l.Logger.Errorf("list stock release pending orders failed: %v", err)
		return
	}
	for _, item := range items {
		suffix, ok := releaseSuffixFromCloseReason(item.CloseReason)
		if !ok {
			continue
		}
		remain, err := releaseStockForOrder(l.ctx, l.svcCtx, item.OrderNo, item.ProductID, item.Quantity, suffix)
		if err != nil {
			l.Logger.Errorf("retry release stock failed order_id=%d err=%v", item.ID, err)
			continue
		}
		if err := markOrderStockReleased(l.ctx, l.svcCtx, item.ID, remain, "retry:"+item.CloseReason, now); err != nil {
			l.Logger.Errorf("mark stock released on retry failed order_id=%d err=%v", item.ID, err)
		}
	}
}

func (l *TimeoutJobLogic) handleRefundCompletion(now time.Time) {
	items, err := l.svcCtx.OrderRepo.ListRefundingDue(l.ctx, now, backgroundBatchSize)
	if err != nil {
		l.Logger.Errorf("list refunding orders failed: %v", err)
		return
	}
	for _, item := range items {
		event := model.OrderEvent{
			OrderID:      item.ID,
			EventType:    "order_refund_completed",
			OperatorType: "system",
			OperatorID:   0,
			Payload:      "{}",
			CreatedAt:    now,
		}
		if err := l.svcCtx.OrderRepo.CompleteRefund(l.ctx, item.ID, now, event); err != nil && err != repository.ErrOrderStateConflict {
			l.Logger.Errorf("complete refund failed order_id=%d err=%v", item.ID, err)
		}
	}
}

func (l *TimeoutJobLogic) handleAutoReceive(now time.Time) {
	items, err := l.svcCtx.OrderRepo.ListAutoReceiveDue(l.ctx, now.Add(-autoReceiveDur), backgroundBatchSize)
	if err != nil {
		l.Logger.Errorf("list auto receive orders failed: %v", err)
		return
	}
	for _, item := range items {
		event := model.OrderEvent{
			OrderID:      item.ID,
			EventType:    "order_auto_received",
			OperatorType: "system",
			OperatorID:   0,
			Payload:      "{}",
			CreatedAt:    now,
		}
		if err := l.svcCtx.OrderRepo.AutoReceive(l.ctx, item.ID, now, event); err != nil && err != repository.ErrOrderStateConflict {
			l.Logger.Errorf("auto receive failed order_id=%d err=%v", item.ID, err)
		}
	}
}
