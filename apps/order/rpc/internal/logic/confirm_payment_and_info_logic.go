// logic 包包含相关应用代码。
package logic

import (
	"context"
	"time"

	"flashsale/apps/order/rpc/internal/model"
	"flashsale/apps/order/rpc/internal/repository"
	"flashsale/apps/order/rpc/internal/svc"
	"flashsale/apps/order/rpc/pb"
	"flashsale/pkg/base/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

// ConfirmPaymentAndInfoLogic 封装支付确认逻辑。
type ConfirmPaymentAndInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConfirmPaymentAndInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmPaymentAndInfoLogic {
	return &ConfirmPaymentAndInfoLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// ConfirmPaymentAndInfo 支付并确认收货信息。
func (l *ConfirmPaymentAndInfoLogic) ConfirmPaymentAndInfo(in *pb.ConfirmPaymentAndInfoReq) (*pb.ConfirmPaymentAndInfoResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.UserId <= 0 || in.OrderId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "user_id 或 order_id 非法")
	}
	if l.svcCtx == nil || l.svcCtx.OrderRepo == nil {
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
	if order.OrderStatus != model.OrderStatusPendingPay || order.PaymentStatus != model.PaymentStatusUnpaid {
		if order.OrderStatus == model.OrderStatusClosed {
			return nil, errorx.New(errorx.CodeOrderAlreadyClosed, "订单已关闭")
		}
		if order.PaymentStatus == model.PaymentStatusPaid {
			return nil, errorx.New(errorx.CodeOrderAlreadyPaid, "订单已支付")
		}
		return nil, errorx.New(errorx.CodeOrderInvalidState, "当前状态不可支付")
	}

	payChannel, err := normalizePayField(in.PayChannel, "支付渠道", 32)
	if err != nil {
		return nil, err
	}
	payRef, err := normalizePayField(in.PayReference, "支付流水号", 64)
	if err != nil {
		return nil, err
	}
	receiverName, receiverPhone, receiverAddr, buyerRemark, err := normalizeReceiverInfo(in.ReceiverName, in.ReceiverPhone, in.ReceiverAddress, in.BuyerRemark)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	manual := needManualReview(order.OrderSource, order.TotalAmountCent)
	event := model.OrderEvent{
		OrderID:      order.ID,
		EventType:    "order_paid_confirmed",
		OperatorType: "user",
		OperatorID:   order.UserID,
		Payload: mustJSON(map[string]any{
			"manual_review": manual,
			"pay_channel":   payChannel,
		}),
		CreatedAt: now,
	}

	if manual {
		reviewDueAt := now.Add(reviewTimeoutDur)
		err = l.svcCtx.OrderRepo.MarkPaidManualReview(l.ctx, order.ID, order.UserID, now, payChannel, payRef, receiverName, receiverPhone, receiverAddr, buyerRemark, reviewDueAt, event)
	} else {
		err = l.svcCtx.OrderRepo.MarkPaidAutoPass(l.ctx, order.ID, order.UserID, now, payChannel, payRef, receiverName, receiverPhone, receiverAddr, buyerRemark, event)
	}
	if err != nil {
		if err == repository.ErrOrderStateConflict {
			return nil, errorx.New(errorx.CodeOrderInvalidState, "订单状态已变化，请刷新后重试")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "更新订单失败", err)
	}

	updated, err := l.svcCtx.OrderRepo.FindByID(l.ctx, order.ID)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询订单失败", err)
	}
	return &pb.ConfirmPaymentAndInfoResp{Order: toOrderView(updated)}, nil
}
