package logic

import (
	"context"

	"flashsale/apps/product/rpc/internal/model"
	"flashsale/apps/product/rpc/internal/repository"
	"flashsale/apps/product/rpc/internal/svc"
	"flashsale/apps/product/rpc/pb"
	"flashsale/pkg/base/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

// CreateProductLogic 封装商品创建逻辑。
type CreateProductLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateProductLogic {
	return &CreateProductLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// CreateProduct 创建商品。
func (l *CreateProductLogic) CreateProduct(in *pb.CreateProductReq) (*pb.CreateProductResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if l.svcCtx == nil || l.svcCtx.ProductRepo == nil || l.svcCtx.IDNode == nil {
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

	id := l.svcCtx.IDNode.Generate().Int64()
	product := &model.Product{
		ID:          id,
		SkuCode:     generateSKU(id),
		Name:        name,
		MainImage:   mainImage,
		Description: description,
		PriceCent:   priceCent,
		Stock:       stock,
		Status:      status,
	}
	if err := l.svcCtx.ProductRepo.Create(l.ctx, product); err != nil {
		if repository.IsDuplicateEntry(err) {
			return nil, errorx.New(errorx.CodeProductSKUAlreadyExists, "商品编码已存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "创建商品失败", err)
	}

	created, err := l.svcCtx.ProductRepo.FindByIDAdmin(l.ctx, id)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询商品失败", err)
	}
	return &pb.CreateProductResp{Product: toAdminProduct(created)}, nil
}
