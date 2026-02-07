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

// GetProductPublicLogic 封装公开商品详情逻辑。
type GetProductPublicLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetProductPublicLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProductPublicLogic {
	return &GetProductPublicLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetProductPublicLogic) GetProductPublic(in *pb.GetProductPublicReq) (*pb.GetProductPublicResp, error) {
	if in == nil || in.ProductId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "product_id 非法")
	}
	if l.svcCtx == nil || l.svcCtx.ProductRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}
	m, err := l.svcCtx.ProductRepo.FindByIDPublic(l.ctx, in.ProductId)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, errorx.New(errorx.CodeProductNotFound, "商品不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "查询商品失败", err)
	}
	return &pb.GetProductPublicResp{Product: toPublicProduct(m)}, nil
}

// ListProductsPublicLogic 封装公开商品列表逻辑。
type ListProductsPublicLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListProductsPublicLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListProductsPublicLogic {
	return &ListProductsPublicLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListProductsPublicLogic) ListProductsPublic(in *pb.ListProductsPublicReq) (*pb.ListProductsPublicResp, error) {
	if in == nil {
		in = &pb.ListProductsPublicReq{}
	}
	if l.svcCtx == nil || l.svcCtx.ProductRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}
	page, pageSize := normalizePagination(in.Page, in.PageSize)
	items, total, err := l.svcCtx.ProductRepo.ListPublic(l.ctx, repository.ListQuery{
		Page:     page,
		PageSize: pageSize,
		Keyword:  strings.TrimSpace(in.Keyword),
	})
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询商品列表失败", err)
	}
	list := make([]*pb.PublicProduct, 0, len(items))
	for _, item := range items {
		list = append(list, toPublicProduct(item))
	}
	return &pb.ListProductsPublicResp{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
