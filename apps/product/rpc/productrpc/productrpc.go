// productrpc 包包含相关应用代码。
package productrpc

import (
	"context"

	"flashsale/apps/product/rpc/pb"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type (
	CreateProductReq         = pb.CreateProductReq
	CreateProductResp        = pb.CreateProductResp
	UpdateProductReq         = pb.UpdateProductReq
	UpdateProductResp        = pb.UpdateProductResp
	DeleteProductReq         = pb.DeleteProductReq
	DeleteProductResp        = pb.DeleteProductResp
	GetProductAdminReq       = pb.GetProductAdminReq
	GetProductAdminResp      = pb.GetProductAdminResp
	ListProductsAdminReq     = pb.ListProductsAdminReq
	ListProductsAdminResp    = pb.ListProductsAdminResp
	GetProductPublicReq      = pb.GetProductPublicReq
	GetProductPublicResp     = pb.GetProductPublicResp
	ListProductsPublicReq    = pb.ListProductsPublicReq
	ListProductsPublicResp   = pb.ListProductsPublicResp
	ReserveStockForOrderReq  = pb.ReserveStockForOrderReq
	ReserveStockForOrderResp = pb.ReserveStockForOrderResp
	ReleaseStockForOrderReq  = pb.ReleaseStockForOrderReq
	ReleaseStockForOrderResp = pb.ReleaseStockForOrderResp

	// ProductRpc 定义商品 RPC 客户端接口。
	ProductRpc interface {
		CreateProduct(ctx context.Context, in *CreateProductReq, opts ...grpc.CallOption) (*CreateProductResp, error)
		UpdateProduct(ctx context.Context, in *UpdateProductReq, opts ...grpc.CallOption) (*UpdateProductResp, error)
		DeleteProduct(ctx context.Context, in *DeleteProductReq, opts ...grpc.CallOption) (*DeleteProductResp, error)
		GetProductAdmin(ctx context.Context, in *GetProductAdminReq, opts ...grpc.CallOption) (*GetProductAdminResp, error)
		ListProductsAdmin(ctx context.Context, in *ListProductsAdminReq, opts ...grpc.CallOption) (*ListProductsAdminResp, error)
		GetProductPublic(ctx context.Context, in *GetProductPublicReq, opts ...grpc.CallOption) (*GetProductPublicResp, error)
		ListProductsPublic(ctx context.Context, in *ListProductsPublicReq, opts ...grpc.CallOption) (*ListProductsPublicResp, error)
		ReserveStockForOrder(ctx context.Context, in *ReserveStockForOrderReq, opts ...grpc.CallOption) (*ReserveStockForOrderResp, error)
		ReleaseStockForOrder(ctx context.Context, in *ReleaseStockForOrderReq, opts ...grpc.CallOption) (*ReleaseStockForOrderResp, error)
	}

	defaultProductRpc struct {
		cli zrpc.Client
	}
)

// NewProductRpc 创建商品 RPC 客户端。
func NewProductRpc(cli zrpc.Client) ProductRpc {
	return &defaultProductRpc{cli: cli}
}

// CreateProduct 调用商品服务创建接口。
func (m *defaultProductRpc) CreateProduct(ctx context.Context, in *CreateProductReq, opts ...grpc.CallOption) (*CreateProductResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.CreateProduct(ctx, in, opts...)
}

// UpdateProduct 调用商品服务更新接口。
func (m *defaultProductRpc) UpdateProduct(ctx context.Context, in *UpdateProductReq, opts ...grpc.CallOption) (*UpdateProductResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.UpdateProduct(ctx, in, opts...)
}

// DeleteProduct 调用商品服务删除接口。
func (m *defaultProductRpc) DeleteProduct(ctx context.Context, in *DeleteProductReq, opts ...grpc.CallOption) (*DeleteProductResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.DeleteProduct(ctx, in, opts...)
}

// GetProductAdmin 调用商品服务管理侧详情接口。
func (m *defaultProductRpc) GetProductAdmin(ctx context.Context, in *GetProductAdminReq, opts ...grpc.CallOption) (*GetProductAdminResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.GetProductAdmin(ctx, in, opts...)
}

// ListProductsAdmin 调用商品服务管理侧列表接口。
func (m *defaultProductRpc) ListProductsAdmin(ctx context.Context, in *ListProductsAdminReq, opts ...grpc.CallOption) (*ListProductsAdminResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.ListProductsAdmin(ctx, in, opts...)
}

// GetProductPublic 调用商品服务用户侧详情接口。
func (m *defaultProductRpc) GetProductPublic(ctx context.Context, in *GetProductPublicReq, opts ...grpc.CallOption) (*GetProductPublicResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.GetProductPublic(ctx, in, opts...)
}

// ListProductsPublic 调用商品服务用户侧列表接口。
func (m *defaultProductRpc) ListProductsPublic(ctx context.Context, in *ListProductsPublicReq, opts ...grpc.CallOption) (*ListProductsPublicResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.ListProductsPublic(ctx, in, opts...)
}

// ReserveStockForOrder 调用商品服务库存预扣接口。
func (m *defaultProductRpc) ReserveStockForOrder(ctx context.Context, in *ReserveStockForOrderReq, opts ...grpc.CallOption) (*ReserveStockForOrderResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.ReserveStockForOrder(ctx, in, opts...)
}

// ReleaseStockForOrder 调用商品服务库存回补接口。
func (m *defaultProductRpc) ReleaseStockForOrder(ctx context.Context, in *ReleaseStockForOrderReq, opts ...grpc.CallOption) (*ReleaseStockForOrderResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.ReleaseStockForOrder(ctx, in, opts...)
}
