// logic 包包含相关应用代码。
package logic

import (
	"context"
	"strings"
	"time"

	"flashsale/apps/order/rpc/internal/model"
	"flashsale/apps/order/rpc/internal/repository"
	"flashsale/apps/order/rpc/internal/svc"
	"flashsale/apps/order/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/eventx"
	"github.com/zeromicro/go-zero/core/logx"
)

// CancelOrderLogic 封装取消订单逻辑。
type CancelOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelOrderLogic {
	return &CancelOrderLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// CancelOrder 用户取消订单。
func (l *CancelOrderLogic) CancelOrder(in *pb.CancelOrderReq) (*pb.CancelOrderResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.UserId <= 0 || in.OrderId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "user_id 或 order_id 非法")
	}
	if l.svcCtx == nil || l.svcCtx.OrderRepo == nil || l.svcCtx.ProductRPCCli == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}

	order, err := l.svcCtx.OrderRepo.FindByID(l.ctx, in.OrderId)
	if err != nil {
		if err == repository.ErrOrderNotFound {
			return nil, errorx.New(errorx.CodeOrderNotFound, "订单不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "查询订单失败", err)
	}
	if order.UserID != in.UserId {
		return nil, errorx.New(errorx.CodeAuthForbidden, "无权限访问该订单")
	}
	if order.OrderStatus == model.OrderStatusClosed {
		return nil, errorx.New(errorx.CodeOrderAlreadyClosed, "订单已关闭")
	}

	userReason := strings.TrimSpace(in.Reason)

	now := time.Now()
	event := model.OrderEvent{
		OrderID:      order.ID,
		EventType:    "order_canceled_by_user",
		OperatorType: "user",
		OperatorID:   order.UserID,
		Payload: mustJSON(map[string]any{
			"user_reason": userReason,
		}),
		CreatedAt: now,
	}

	switch {
	case order.OrderStatus == model.OrderStatusPendingPay && order.PaymentStatus == model.PaymentStatusUnpaid:
		if err := l.svcCtx.OrderRepo.CancelUnpaid(l.ctx, order.ID, order.UserID, now, model.CloseReasonUserCancel, event); err != nil {
			return nil, mapStateConflict(err, "取消订单失败")
		}
	case isManualPendingForCancel(order):
		refundDueAt := now.Add(virtualRefundDur)
		if err := l.svcCtx.OrderRepo.CancelPaidPendingReview(l.ctx, order.ID, order.UserID, now, model.CloseReasonUserCancel, refundDueAt, event); err != nil {
			return nil, mapStateConflict(err, "取消订单失败")
		}
	default:
		return nil, errorx.New(errorx.CodeOrderInvalidState, "当前状态不可取消")
	}

	releaseAt := time.Now()
	remain := int64(0)
	if order.OrderSource != model.OrderSourceSeckill {
		remain, err = releaseStockForOrder(l.ctx, l.svcCtx, order.OrderNo, order.ProductID, order.Quantity, "release:cancel")
		if err != nil {
			logStockReleaseErr(l.Logger, "release stock after cancel failed", err)
			return nil, err
		}
	}
	if err := markOrderStockReleased(l.ctx, l.svcCtx, order.ID, remain, "user_cancel", releaseAt); err != nil {
		logStockReleaseErr(l.Logger, "mark stock released after cancel failed", err)
		return nil, errorx.Wrap(errorx.CodeDBError, "标记库存回补状态失败", err)
	}

	updated, err := l.svcCtx.OrderRepo.FindByID(l.ctx, order.ID)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询订单失败", err)
	}
	emitSeckillOrderStateEvent(l.ctx, l.svcCtx, updated, eventx.SeckillOrderStateEventTypeClosed)
	return &pb.CancelOrderResp{Order: toOrderView(updated)}, nil
}

func mapStateConflict(err error, msg string) error {
	if err == nil {
		return nil
	}
	if err == repository.ErrOrderStateConflict {
		return errorx.New(errorx.CodeOrderInvalidState, "订单状态已变化，请刷新后重试")
	}
	return errorx.Wrap(errorx.CodeDBError, msg, err)
}
