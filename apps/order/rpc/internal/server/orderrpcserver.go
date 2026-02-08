// server 包包含相关应用代码。
package server

import (
	"context"

	"flashsale/apps/order/rpc/internal/logic"
	"flashsale/apps/order/rpc/internal/svc"
	"flashsale/apps/order/rpc/pb"
	"flashsale/pkg/base/grpcerr"
)

// OrderRpcServer 是订单 RPC 服务端实现。
type OrderRpcServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedOrderRpcServer
}

// NewOrderRpcServer 创建订单 RPC 服务端。
func NewOrderRpcServer(svcCtx *svc.ServiceContext) *OrderRpcServer {
	return &OrderRpcServer{svcCtx: svcCtx}
}

// CreateOrder 处理用户下单请求。
func (s *OrderRpcServer) CreateOrder(ctx context.Context, in *pb.CreateOrderReq) (*pb.CreateOrderResp, error) {
	if err := authorizeUser(ctx, in.GetUserId()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewCreateOrderLogic(ctx, s.svcCtx)
	resp, err := l.CreateOrder(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// CreateOrderFromSeckill 处理秒杀专用建单请求。
func (s *OrderRpcServer) CreateOrderFromSeckill(ctx context.Context, in *pb.CreateOrderFromSeckillReq) (*pb.CreateOrderFromSeckillResp, error) {
	if err := authorizeUser(ctx, in.GetUserId()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewCreateOrderFromSeckillLogic(ctx, s.svcCtx)
	resp, err := l.CreateOrderFromSeckill(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ConfirmPaymentAndInfo 处理用户支付与收货信息确认请求。
func (s *OrderRpcServer) ConfirmPaymentAndInfo(ctx context.Context, in *pb.ConfirmPaymentAndInfoReq) (*pb.ConfirmPaymentAndInfoResp, error) {
	if err := authorizeUser(ctx, in.GetUserId()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewConfirmPaymentAndInfoLogic(ctx, s.svcCtx)
	resp, err := l.ConfirmPaymentAndInfo(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// CancelOrder 处理用户取消订单请求。
func (s *OrderRpcServer) CancelOrder(ctx context.Context, in *pb.CancelOrderReq) (*pb.CancelOrderResp, error) {
	if err := authorizeUser(ctx, in.GetUserId()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewCancelOrderLogic(ctx, s.svcCtx)
	resp, err := l.CancelOrder(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ConfirmReceipt 处理用户确认收货请求。
func (s *OrderRpcServer) ConfirmReceipt(ctx context.Context, in *pb.ConfirmReceiptReq) (*pb.ConfirmReceiptResp, error) {
	if err := authorizeUser(ctx, in.GetUserId()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewConfirmReceiptLogic(ctx, s.svcCtx)
	resp, err := l.ConfirmReceipt(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// GetOrderUser 处理用户订单详情查询请求。
func (s *OrderRpcServer) GetOrderUser(ctx context.Context, in *pb.GetOrderUserReq) (*pb.GetOrderUserResp, error) {
	if err := authorizeUser(ctx, in.GetUserId()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewGetOrderUserLogic(ctx, s.svcCtx)
	resp, err := l.GetOrderUser(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ListOrdersUser 处理用户订单列表查询请求。
func (s *OrderRpcServer) ListOrdersUser(ctx context.Context, in *pb.ListOrdersUserReq) (*pb.ListOrdersUserResp, error) {
	if err := authorizeUser(ctx, in.GetUserId()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewListOrdersUserLogic(ctx, s.svcCtx)
	resp, err := l.ListOrdersUser(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ReviewOrderAdmin 处理管理侧审核订单请求。
func (s *OrderRpcServer) ReviewOrderAdmin(ctx context.Context, in *pb.ReviewOrderAdminReq) (*pb.ReviewOrderAdminResp, error) {
	adminID, err := authorizeAdminOrderDomain(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.ReviewOrderAdminReq{
			OrderId:  req.OrderId,
			AdminId:  adminID,
			Approved: req.Approved,
			Reason:   req.Reason,
		}
	}
	l := logic.NewReviewOrderAdminLogic(ctx, s.svcCtx)
	resp, err := l.ReviewOrderAdmin(req)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ShipOrderAdmin 处理管理侧发货请求。
func (s *OrderRpcServer) ShipOrderAdmin(ctx context.Context, in *pb.ShipOrderAdminReq) (*pb.ShipOrderAdminResp, error) {
	adminID, err := authorizeAdminOrderDomain(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.ShipOrderAdminReq{
			OrderId:    req.OrderId,
			AdminId:    adminID,
			TrackingNo: req.TrackingNo,
		}
	}
	l := logic.NewShipOrderAdminLogic(ctx, s.svcCtx)
	resp, err := l.ShipOrderAdmin(req)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// GetOrderAdmin 处理管理侧订单详情查询请求。
func (s *OrderRpcServer) GetOrderAdmin(ctx context.Context, in *pb.GetOrderAdminReq) (*pb.GetOrderAdminResp, error) {
	if _, err := authorizeAdminOrderDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewGetOrderAdminLogic(ctx, s.svcCtx)
	resp, err := l.GetOrderAdmin(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ListOrdersAdmin 处理管理侧订单列表查询请求。
func (s *OrderRpcServer) ListOrdersAdmin(ctx context.Context, in *pb.ListOrdersAdminReq) (*pb.ListOrdersAdminResp, error) {
	if _, err := authorizeAdminOrderDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewListOrdersAdminLogic(ctx, s.svcCtx)
	resp, err := l.ListOrdersAdmin(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}
