package logic

import (
	"context"
	"errors"
	"time"

	"flashsale/apps/product/rpc/internal/repository"
	"flashsale/apps/product/rpc/internal/svc"
	"flashsale/apps/product/rpc/pb"
	"flashsale/pkg/base/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeleteProductLogic 封装商品删除逻辑。
type DeleteProductLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteProductLogic {
	return &DeleteProductLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// DeleteProduct 软删除商品。
func (l *DeleteProductLogic) DeleteProduct(in *pb.DeleteProductReq) (*pb.DeleteProductResp, error) {
	if in == nil || in.ProductId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "product_id 非法")
	}
	if l.svcCtx == nil || l.svcCtx.ProductRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}
	if err := l.svcCtx.ProductRepo.SoftDelete(l.ctx, in.ProductId, time.Now()); err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, errorx.New(errorx.CodeProductNotFound, "商品不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "删除商品失败", err)
	}
	return &pb.DeleteProductResp{ProductId: in.ProductId}, nil
}
