// orderrpc 包包含相关应用代码。
package orderrpc

import (
	"context"

	"flashsale/apps/order/rpc/pb"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type (
	CreateOrderReq             = pb.CreateOrderReq
	CreateOrderResp            = pb.CreateOrderResp
	CreateOrderFromSeckillReq  = pb.CreateOrderFromSeckillReq
	CreateOrderFromSeckillResp = pb.CreateOrderFromSeckillResp
	ConfirmPaymentAndInfoReq   = pb.ConfirmPaymentAndInfoReq
	ConfirmPaymentAndInfoResp  = pb.ConfirmPaymentAndInfoResp
	CancelOrderReq             = pb.CancelOrderReq
	CancelOrderResp            = pb.CancelOrderResp
	ConfirmReceiptReq          = pb.ConfirmReceiptReq
	ConfirmReceiptResp         = pb.ConfirmReceiptResp
	GetOrderUserReq            = pb.GetOrderUserReq
	GetOrderUserResp           = pb.GetOrderUserResp
	ListOrdersUserReq          = pb.ListOrdersUserReq
	ListOrdersUserResp         = pb.ListOrdersUserResp
	ReviewOrderAdminReq        = pb.ReviewOrderAdminReq
	ReviewOrderAdminResp       = pb.ReviewOrderAdminResp
	ShipOrderAdminReq          = pb.ShipOrderAdminReq
	ShipOrderAdminResp         = pb.ShipOrderAdminResp
	GetOrderAdminReq           = pb.GetOrderAdminReq
	GetOrderAdminResp          = pb.GetOrderAdminResp
	ListOrdersAdminReq         = pb.ListOrdersAdminReq
	ListOrdersAdminResp        = pb.ListOrdersAdminResp

	// OrderRpc 定义订单 RPC 客户端接口。
	OrderRpc interface {
		CreateOrder(ctx context.Context, in *CreateOrderReq, opts ...grpc.CallOption) (*CreateOrderResp, error)
		CreateOrderFromSeckill(ctx context.Context, in *CreateOrderFromSeckillReq, opts ...grpc.CallOption) (*CreateOrderFromSeckillResp, error)
		ConfirmPaymentAndInfo(ctx context.Context, in *ConfirmPaymentAndInfoReq, opts ...grpc.CallOption) (*ConfirmPaymentAndInfoResp, error)
		CancelOrder(ctx context.Context, in *CancelOrderReq, opts ...grpc.CallOption) (*CancelOrderResp, error)
		ConfirmReceipt(ctx context.Context, in *ConfirmReceiptReq, opts ...grpc.CallOption) (*ConfirmReceiptResp, error)
		GetOrderUser(ctx context.Context, in *GetOrderUserReq, opts ...grpc.CallOption) (*GetOrderUserResp, error)
		ListOrdersUser(ctx context.Context, in *ListOrdersUserReq, opts ...grpc.CallOption) (*ListOrdersUserResp, error)
		ReviewOrderAdmin(ctx context.Context, in *ReviewOrderAdminReq, opts ...grpc.CallOption) (*ReviewOrderAdminResp, error)
		ShipOrderAdmin(ctx context.Context, in *ShipOrderAdminReq, opts ...grpc.CallOption) (*ShipOrderAdminResp, error)
		GetOrderAdmin(ctx context.Context, in *GetOrderAdminReq, opts ...grpc.CallOption) (*GetOrderAdminResp, error)
		ListOrdersAdmin(ctx context.Context, in *ListOrdersAdminReq, opts ...grpc.CallOption) (*ListOrdersAdminResp, error)
	}

	defaultOrderRpc struct {
		cli zrpc.Client
	}
)

// NewOrderRpc 创建订单 RPC 客户端。
func NewOrderRpc(cli zrpc.Client) OrderRpc {
	return &defaultOrderRpc{cli: cli}
}

// CreateOrder 调用订单服务创建订单接口。
func (m *defaultOrderRpc) CreateOrder(ctx context.Context, in *CreateOrderReq, opts ...grpc.CallOption) (*CreateOrderResp, error) {
	return pb.NewOrderRpcClient(m.cli.Conn()).CreateOrder(ctx, in, opts...)
}

// CreateOrderFromSeckill 调用订单服务秒杀专用建单接口。
func (m *defaultOrderRpc) CreateOrderFromSeckill(ctx context.Context, in *CreateOrderFromSeckillReq, opts ...grpc.CallOption) (*CreateOrderFromSeckillResp, error) {
	return pb.NewOrderRpcClient(m.cli.Conn()).CreateOrderFromSeckill(ctx, in, opts...)
}

// ConfirmPaymentAndInfo 调用订单服务支付确认接口。
func (m *defaultOrderRpc) ConfirmPaymentAndInfo(ctx context.Context, in *ConfirmPaymentAndInfoReq, opts ...grpc.CallOption) (*ConfirmPaymentAndInfoResp, error) {
	return pb.NewOrderRpcClient(m.cli.Conn()).ConfirmPaymentAndInfo(ctx, in, opts...)
}

// CancelOrder 调用订单服务取消订单接口。
func (m *defaultOrderRpc) CancelOrder(ctx context.Context, in *CancelOrderReq, opts ...grpc.CallOption) (*CancelOrderResp, error) {
	return pb.NewOrderRpcClient(m.cli.Conn()).CancelOrder(ctx, in, opts...)
}

// ConfirmReceipt 调用订单服务确认收货接口。
func (m *defaultOrderRpc) ConfirmReceipt(ctx context.Context, in *ConfirmReceiptReq, opts ...grpc.CallOption) (*ConfirmReceiptResp, error) {
	return pb.NewOrderRpcClient(m.cli.Conn()).ConfirmReceipt(ctx, in, opts...)
}

// GetOrderUser 调用订单服务用户侧详情接口。
func (m *defaultOrderRpc) GetOrderUser(ctx context.Context, in *GetOrderUserReq, opts ...grpc.CallOption) (*GetOrderUserResp, error) {
	return pb.NewOrderRpcClient(m.cli.Conn()).GetOrderUser(ctx, in, opts...)
}

// ListOrdersUser 调用订单服务用户侧列表接口。
func (m *defaultOrderRpc) ListOrdersUser(ctx context.Context, in *ListOrdersUserReq, opts ...grpc.CallOption) (*ListOrdersUserResp, error) {
	return pb.NewOrderRpcClient(m.cli.Conn()).ListOrdersUser(ctx, in, opts...)
}

// ReviewOrderAdmin 调用订单服务管理侧审核接口。
func (m *defaultOrderRpc) ReviewOrderAdmin(ctx context.Context, in *ReviewOrderAdminReq, opts ...grpc.CallOption) (*ReviewOrderAdminResp, error) {
	return pb.NewOrderRpcClient(m.cli.Conn()).ReviewOrderAdmin(ctx, in, opts...)
}

// ShipOrderAdmin 调用订单服务管理侧发货接口。
func (m *defaultOrderRpc) ShipOrderAdmin(ctx context.Context, in *ShipOrderAdminReq, opts ...grpc.CallOption) (*ShipOrderAdminResp, error) {
	return pb.NewOrderRpcClient(m.cli.Conn()).ShipOrderAdmin(ctx, in, opts...)
}

// GetOrderAdmin 调用订单服务管理侧详情接口。
func (m *defaultOrderRpc) GetOrderAdmin(ctx context.Context, in *GetOrderAdminReq, opts ...grpc.CallOption) (*GetOrderAdminResp, error) {
	return pb.NewOrderRpcClient(m.cli.Conn()).GetOrderAdmin(ctx, in, opts...)
}

// ListOrdersAdmin 调用订单服务管理侧列表接口。
func (m *defaultOrderRpc) ListOrdersAdmin(ctx context.Context, in *ListOrdersAdminReq, opts ...grpc.CallOption) (*ListOrdersAdminResp, error) {
	return pb.NewOrderRpcClient(m.cli.Conn()).ListOrdersAdmin(ctx, in, opts...)
}
