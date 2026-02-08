// logic 包包含相关应用代码。
package logic

import (
	"context"
	"errors"
	"strings"

	"flashsale/apps/product/rpc/internal/repository"
	"flashsale/apps/product/rpc/internal/svc"
	"flashsale/apps/product/rpc/pb"
	"flashsale/pkg/base/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

// GetProductAdminLogic 封装管理端商品详情逻辑。
type GetProductAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetProductAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProductAdminLogic {
	return &GetProductAdminLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetProductAdminLogic) GetProductAdmin(in *pb.GetProductAdminReq) (*pb.GetProductAdminResp, error) {
	if in == nil || in.ProductId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "product_id 非法")
	}
	if l.svcCtx == nil || l.svcCtx.ProductRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}
	m, err := l.svcCtx.ProductRepo.FindByIDAdmin(l.ctx, in.ProductId)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, errorx.New(errorx.CodeProductNotFound, "商品不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "查询商品失败", err)
	}
	return &pb.GetProductAdminResp{Product: toAdminProduct(m)}, nil
}

// ListProductsAdminLogic 封装管理端商品列表逻辑。
type ListProductsAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListProductsAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListProductsAdminLogic {
	return &ListProductsAdminLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListProductsAdminLogic) ListProductsAdmin(in *pb.ListProductsAdminReq) (*pb.ListProductsAdminResp, error) {
	if in == nil {
		in = &pb.ListProductsAdminReq{}
	}
	if l.svcCtx == nil || l.svcCtx.ProductRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}
	page, pageSize := normalizePagination(in.Page, in.PageSize)
	items, total, err := l.svcCtx.ProductRepo.ListAdmin(l.ctx, repository.ListQuery{
		Page:           page,
		PageSize:       pageSize,
		Keyword:        strings.TrimSpace(in.Keyword),
		IncludeDeleted: in.IncludeDeleted,
	})
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询商品列表失败", err)
	}
	list := make([]*pb.AdminProduct, 0, len(items))
	for _, item := range items {
		list = append(list, toAdminProduct(item))
	}
	return &pb.ListProductsAdminResp{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
