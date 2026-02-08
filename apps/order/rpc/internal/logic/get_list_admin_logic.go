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

// GetOrderAdminLogic 封装管理员订单详情查询逻辑。
type GetOrderAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetOrderAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderAdminLogic {
	return &GetOrderAdminLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// GetOrderAdmin 查询订单详情。
func (l *GetOrderAdminLogic) GetOrderAdmin(in *pb.GetOrderAdminReq) (*pb.GetOrderAdminResp, error) {
	if in == nil || in.OrderId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "order_id 非法")
	}
	order, err := l.svcCtx.OrderRepo.FindByID(l.ctx, in.OrderId)
	if err != nil {
		if err == repository.ErrOrderNotFound {
			return nil, errorx.New(errorx.CodeOrderNotFound, "订单不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "查询订单失败", err)
	}
	return &pb.GetOrderAdminResp{Order: toOrderView(order)}, nil
}

// ListOrdersAdminLogic 封装管理员订单列表查询逻辑。
type ListOrdersAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListOrdersAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOrdersAdminLogic {
	return &ListOrdersAdminLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// ListOrdersAdmin 查询订单列表。
func (l *ListOrdersAdminLogic) ListOrdersAdmin(in *pb.ListOrdersAdminReq) (*pb.ListOrdersAdminResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	page, pageSize := normalizePagination(in.Page, in.PageSize)
	items, total, err := l.svcCtx.OrderRepo.ListAdmin(l.ctx, repository.AdminListQuery{
		Page:         page,
		PageSize:     pageSize,
		OrderStatus:  int8(in.OrderStatus),
		ReviewStatus: int8(in.ReviewStatus),
		UserID:       in.UserId,
		OrderNo:      in.OrderNo,
	})
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询订单列表失败", err)
	}
	list := make([]*pb.OrderView, 0, len(items))
	for _, item := range items {
		list = append(list, toOrderView(item))
	}
	return &pb.ListOrdersAdminResp{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
