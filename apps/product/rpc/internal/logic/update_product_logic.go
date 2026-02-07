package logic

import (
	"context"
	"errors"

	"flashsale/apps/product/rpc/internal/model"
	"flashsale/apps/product/rpc/internal/repository"
	"flashsale/apps/product/rpc/internal/svc"
	"flashsale/apps/product/rpc/pb"
	"flashsale/pkg/base/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

// UpdateProductLogic 封装商品更新逻辑。
type UpdateProductLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProductLogic {
	return &UpdateProductLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// UpdateProduct 更新商品。
func (l *UpdateProductLogic) UpdateProduct(in *pb.UpdateProductReq) (*pb.UpdateProductResp, error) {
	if in == nil || in.ProductId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "product_id 非法")
	}
	if l.svcCtx == nil || l.svcCtx.ProductRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}

	name, err := normalizeName(in.Name)
	if err != nil {
		return nil, err
	}
	mainImage, err := normalizeMainImage(in.MainImage)
	if err != nil {
		return nil, err
	}
	description, err := normalizeDescription(in.Description)
	if err != nil {
		return nil, err
	}
	priceCent, err := normalizePriceCent(in.PriceCent)
	if err != nil {
		return nil, err
	}
	stock, err := normalizeStock(in.Stock)
	if err != nil {
		return nil, err
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return nil, err
	}

	err = l.svcCtx.ProductRepo.Update(l.ctx, &model.Product{
		ID:          in.ProductId,
		Name:        name,
		MainImage:   mainImage,
		Description: description,
		PriceCent:   priceCent,
		Stock:       stock,
		Status:      status,
	})
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, errorx.New(errorx.CodeProductNotFound, "商品不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "更新商品失败", err)
	}
	product, err := l.svcCtx.ProductRepo.FindByIDAdmin(l.ctx, in.ProductId)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, errorx.New(errorx.CodeProductNotFound, "商品不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "查询商品失败", err)
	}
	return &pb.UpdateProductResp{Product: toAdminProduct(product)}, nil
}
