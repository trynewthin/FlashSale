// logic 包包含相关应用代码。
package logic

import (
	"context"

	"flashsale/apps/order/rpc/internal/repository"
	"flashsale/apps/order/rpc/internal/svc"
	"flashsale/apps/order/rpc/pb"
	"flashsale/pkg/base/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetOrderUserLogic 封装用户订单详情查询逻辑。
type GetOrderUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetOrderUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderUserLogic {
	return &GetOrderUserLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// GetOrderUser 查询用户订单详情。
func (l *GetOrderUserLogic) GetOrderUser(in *pb.GetOrderUserReq) (*pb.GetOrderUserResp, error) {
	if in == nil || in.UserId <= 0 || in.OrderId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "user_id 或 order_id 非法")
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
	return &pb.GetOrderUserResp{Order: toOrderView(order)}, nil
}

// ListOrdersUserLogic 封装用户订单列表查询逻辑。
type ListOrdersUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListOrdersUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOrdersUserLogic {
	return &ListOrdersUserLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// ListOrdersUser 查询用户订单列表。
func (l *ListOrdersUserLogic) ListOrdersUser(in *pb.ListOrdersUserReq) (*pb.ListOrdersUserResp, error) {
	if in == nil || in.UserId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "user_id 非法")
	}
	page, pageSize := normalizePagination(in.Page, in.PageSize)
	status := int8(in.OrderStatus)
	if status < 0 {
		status = 0
	}
	items, total, err := l.svcCtx.OrderRepo.ListByUser(l.ctx, repository.UserListQuery{
		UserID:      in.UserId,
		Page:        page,
		PageSize:    pageSize,
		OrderStatus: status,
		OrderNo:     in.OrderNo,
	})
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询订单列表失败", err)
	}
	list := make([]*pb.OrderView, 0, len(items))
	for _, item := range items {
		list = append(list, toOrderView(item))
	}
	return &pb.ListOrdersUserResp{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
