// server 包包含相关应用代码。
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

// CreateProduct 处理管理侧创建商品请求。
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

// UpdateProduct 处理管理侧更新商品请求。
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

// DeleteProduct 处理管理侧删除商品请求。
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

// GetProductAdmin 处理管理侧商品详情请求。
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

// ListProductsAdmin 处理管理侧商品列表请求。
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

// GetProductPublic 处理用户侧商品详情请求。
func (s *ProductRpcServer) GetProductPublic(ctx context.Context, in *pb.GetProductPublicReq) (*pb.GetProductPublicResp, error) {
	l := logic.NewGetProductPublicLogic(ctx, s.svcCtx)
	resp, err := l.GetProductPublic(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ListProductsPublic 处理用户侧商品列表请求。
func (s *ProductRpcServer) ListProductsPublic(ctx context.Context, in *pb.ListProductsPublicReq) (*pb.ListProductsPublicResp, error) {
	l := logic.NewListProductsPublicLogic(ctx, s.svcCtx)
	resp, err := l.ListProductsPublic(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ReserveStockForOrder 处理订单库存预扣请求。
func (s *ProductRpcServer) ReserveStockForOrder(ctx context.Context, in *pb.ReserveStockForOrderReq) (*pb.ReserveStockForOrderResp, error) {
	if err := authorizeAdminProductDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewReserveStockForOrderLogic(ctx, s.svcCtx)
	resp, err := l.ReserveStockForOrder(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ReleaseStockForOrder 处理订单库存回补请求。
func (s *ProductRpcServer) ReleaseStockForOrder(ctx context.Context, in *pb.ReleaseStockForOrderReq) (*pb.ReleaseStockForOrderResp, error) {
	if err := authorizeAdminProductDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewReleaseStockForOrderLogic(ctx, s.svcCtx)
	resp, err := l.ReleaseStockForOrder(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ReserveStockForActivity 处理活动库存预占请求。
func (s *ProductRpcServer) ReserveStockForActivity(ctx context.Context, in *pb.ReserveStockForActivityReq) (*pb.ReserveStockForActivityResp, error) {
	if err := authorizeAdminProductDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewReserveStockForActivityLogic(ctx, s.svcCtx)
	resp, err := l.ReserveStockForActivity(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ReleaseStockForActivity 处理活动库存释放请求。
func (s *ProductRpcServer) ReleaseStockForActivity(ctx context.Context, in *pb.ReleaseStockForActivityReq) (*pb.ReleaseStockForActivityResp, error) {
	if err := authorizeAdminProductDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewReleaseStockForActivityLogic(ctx, s.svcCtx)
	resp, err := l.ReleaseStockForActivity(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}
