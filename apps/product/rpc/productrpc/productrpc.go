// Package productrpc 提供商品 RPC 客户端封装。
package productrpc

import (
	"context"

	"flashsale/apps/product/rpc/pb"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type (
	CreateProductReq       = pb.CreateProductReq
	CreateProductResp      = pb.CreateProductResp
	UpdateProductReq       = pb.UpdateProductReq
	UpdateProductResp      = pb.UpdateProductResp
	DeleteProductReq       = pb.DeleteProductReq
	DeleteProductResp      = pb.DeleteProductResp
	GetProductAdminReq     = pb.GetProductAdminReq
	GetProductAdminResp    = pb.GetProductAdminResp
	ListProductsAdminReq   = pb.ListProductsAdminReq
	ListProductsAdminResp  = pb.ListProductsAdminResp
	GetProductPublicReq    = pb.GetProductPublicReq
	GetProductPublicResp   = pb.GetProductPublicResp
	ListProductsPublicReq  = pb.ListProductsPublicReq
	ListProductsPublicResp = pb.ListProductsPublicResp

	// ProductRpc 定义商品 RPC 客户端接口。
	ProductRpc interface {
		CreateProduct(ctx context.Context, in *CreateProductReq, opts ...grpc.CallOption) (*CreateProductResp, error)
		UpdateProduct(ctx context.Context, in *UpdateProductReq, opts ...grpc.CallOption) (*UpdateProductResp, error)
		DeleteProduct(ctx context.Context, in *DeleteProductReq, opts ...grpc.CallOption) (*DeleteProductResp, error)
		GetProductAdmin(ctx context.Context, in *GetProductAdminReq, opts ...grpc.CallOption) (*GetProductAdminResp, error)
		ListProductsAdmin(ctx context.Context, in *ListProductsAdminReq, opts ...grpc.CallOption) (*ListProductsAdminResp, error)
		GetProductPublic(ctx context.Context, in *GetProductPublicReq, opts ...grpc.CallOption) (*GetProductPublicResp, error)
		ListProductsPublic(ctx context.Context, in *ListProductsPublicReq, opts ...grpc.CallOption) (*ListProductsPublicResp, error)
	}

	defaultProductRpc struct {
		cli zrpc.Client
	}
)

// NewProductRpc 创建商品 RPC 客户端。
func NewProductRpc(cli zrpc.Client) ProductRpc {
	return &defaultProductRpc{cli: cli}
}

func (m *defaultProductRpc) CreateProduct(ctx context.Context, in *CreateProductReq, opts ...grpc.CallOption) (*CreateProductResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.CreateProduct(ctx, in, opts...)
}

func (m *defaultProductRpc) UpdateProduct(ctx context.Context, in *UpdateProductReq, opts ...grpc.CallOption) (*UpdateProductResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.UpdateProduct(ctx, in, opts...)
}

func (m *defaultProductRpc) DeleteProduct(ctx context.Context, in *DeleteProductReq, opts ...grpc.CallOption) (*DeleteProductResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.DeleteProduct(ctx, in, opts...)
}

func (m *defaultProductRpc) GetProductAdmin(ctx context.Context, in *GetProductAdminReq, opts ...grpc.CallOption) (*GetProductAdminResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.GetProductAdmin(ctx, in, opts...)
}

func (m *defaultProductRpc) ListProductsAdmin(ctx context.Context, in *ListProductsAdminReq, opts ...grpc.CallOption) (*ListProductsAdminResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.ListProductsAdmin(ctx, in, opts...)
}

func (m *defaultProductRpc) GetProductPublic(ctx context.Context, in *GetProductPublicReq, opts ...grpc.CallOption) (*GetProductPublicResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.GetProductPublic(ctx, in, opts...)
}

func (m *defaultProductRpc) ListProductsPublic(ctx context.Context, in *ListProductsPublicReq, opts ...grpc.CallOption) (*ListProductsPublicResp, error) {
	client := pb.NewProductRpcClient(m.cli.Conn())
	return client.ListProductsPublic(ctx, in, opts...)
}
