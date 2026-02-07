// Package server 负责把商品 RPC 请求分发到对应业务逻辑。
package server

import (
	"context"

	"flashsale/apps/product/rpc/internal/logic"
	"flashsale/apps/product/rpc/internal/svc"
	"flashsale/apps/product/rpc/pb"
	"flashsale/pkg/base/grpcerr"
)

// ProductRpcServer 是商品 RPC 服务端实现。
type ProductRpcServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedProductRpcServer
}

// NewProductRpcServer 创建商品 RPC 服务端。
func NewProductRpcServer(svcCtx *svc.ServiceContext) *ProductRpcServer {
	return &ProductRpcServer{svcCtx: svcCtx}
}

func (s *ProductRpcServer) CreateProduct(ctx context.Context, in *pb.CreateProductReq) (*pb.CreateProductResp, error) {
	if err := authorizeAdminProductDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewCreateProductLogic(ctx, s.svcCtx)
	resp, err := l.CreateProduct(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

func (s *ProductRpcServer) UpdateProduct(ctx context.Context, in *pb.UpdateProductReq) (*pb.UpdateProductResp, error) {
	if err := authorizeAdminProductDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewUpdateProductLogic(ctx, s.svcCtx)
	resp, err := l.UpdateProduct(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

func (s *ProductRpcServer) DeleteProduct(ctx context.Context, in *pb.DeleteProductReq) (*pb.DeleteProductResp, error) {
	if err := authorizeAdminProductDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewDeleteProductLogic(ctx, s.svcCtx)
	resp, err := l.DeleteProduct(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

func (s *ProductRpcServer) GetProductAdmin(ctx context.Context, in *pb.GetProductAdminReq) (*pb.GetProductAdminResp, error) {
	if err := authorizeAdminProductDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewGetProductAdminLogic(ctx, s.svcCtx)
	resp, err := l.GetProductAdmin(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

func (s *ProductRpcServer) ListProductsAdmin(ctx context.Context, in *pb.ListProductsAdminReq) (*pb.ListProductsAdminResp, error) {
	if err := authorizeAdminProductDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewListProductsAdminLogic(ctx, s.svcCtx)
	resp, err := l.ListProductsAdmin(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

func (s *ProductRpcServer) GetProductPublic(ctx context.Context, in *pb.GetProductPublicReq) (*pb.GetProductPublicResp, error) {
	l := logic.NewGetProductPublicLogic(ctx, s.svcCtx)
	resp, err := l.GetProductPublic(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

func (s *ProductRpcServer) ListProductsPublic(ctx context.Context, in *pb.ListProductsPublicReq) (*pb.ListProductsPublicResp, error) {
	l := logic.NewListProductsPublicLogic(ctx, s.svcCtx)
	resp, err := l.ListProductsPublic(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}
