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

// ConfirmReceiptLogic 封装确认收货逻辑。
type ConfirmReceiptLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConfirmReceiptLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmReceiptLogic {
	return &ConfirmReceiptLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// ConfirmReceipt 用户确认收货。
func (l *ConfirmReceiptLogic) ConfirmReceipt(in *pb.ConfirmReceiptReq) (*pb.ConfirmReceiptResp, error) {
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
	if order.OrderStatus != model.OrderStatusShipped || order.ShippingStatus != model.ShippingStatusShipped {
		return nil, errorx.New(errorx.CodeOrderInvalidState, "当前状态不可确认收货")
	}

	now := time.Now()
	event := model.OrderEvent{
		OrderID:      order.ID,
		EventType:    "order_received_by_user",
		OperatorType: "user",
		OperatorID:   order.UserID,
		Payload:      "{}",
		CreatedAt:    now,
	}
	if err := l.svcCtx.OrderRepo.ConfirmReceipt(l.ctx, order.ID, order.UserID, now, event); err != nil {
		if err == repository.ErrOrderStateConflict {
			return nil, errorx.New(errorx.CodeOrderInvalidState, "订单状态已变化，请刷新后重试")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "确认收货失败", err)
	}

	updated, err := l.svcCtx.OrderRepo.FindByID(l.ctx, order.ID)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询订单失败", err)
	}
	return &pb.ConfirmReceiptResp{Order: toOrderView(updated)}, nil
}
