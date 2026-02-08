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
	"flashsale/pkg/base/eventx"
	"github.com/zeromicro/go-zero/core/logx"
)

// ShipOrderAdminLogic 封装管理员发货逻辑。
type ShipOrderAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewShipOrderAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ShipOrderAdminLogic {
	return &ShipOrderAdminLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// ShipOrderAdmin 管理员发货。
func (l *ShipOrderAdminLogic) ShipOrderAdmin(in *pb.ShipOrderAdminReq) (*pb.ShipOrderAdminResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.OrderId <= 0 || in.AdminId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "order_id 或 admin_id 非法")
	}
	if l.svcCtx == nil || l.svcCtx.OrderRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}
	trackingNo, err := normalizeTrackingNo(in.TrackingNo)
	if err != nil {
		return nil, err
	}

	order, err := l.svcCtx.OrderRepo.FindByID(l.ctx, in.OrderId)
	if err != nil {
		if err == repository.ErrOrderNotFound {
			return nil, errorx.New(errorx.CodeOrderNotFound, "订单不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "查询订单失败", err)
	}
	if order.OrderStatus != model.OrderStatusPendingShip {
		return nil, errorx.New(errorx.CodeOrderInvalidState, "当前状态不可发货")
	}

	now := time.Now()
	event := model.OrderEvent{
		OrderID:      order.ID,
		EventType:    "order_shipped",
		OperatorType: "admin",
		OperatorID:   in.AdminId,
		Payload:      mustJSON(map[string]any{"tracking_no": trackingNo}),
		CreatedAt:    now,
	}
	if err := l.svcCtx.OrderRepo.Ship(l.ctx, order.ID, in.AdminId, trackingNo, now, event); err != nil {
		if err == repository.ErrOrderStateConflict {
			return nil, errorx.New(errorx.CodeOrderInvalidState, "订单状态已变化，请刷新后重试")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "发货失败", err)
	}

	updated, err := l.svcCtx.OrderRepo.FindByID(l.ctx, order.ID)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询订单失败", err)
	}
	emitSeckillOrderStateEvent(l.ctx, l.svcCtx, updated, eventx.SeckillOrderStateEventTypeShipped)
	return &pb.ShipOrderAdminResp{Order: toOrderView(updated)}, nil
}
