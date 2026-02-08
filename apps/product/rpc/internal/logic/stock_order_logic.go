// logic 包包含相关应用代码。
package logic

import (
	"context"
	"strings"

	"flashsale/apps/product/rpc/internal/repository"
	"flashsale/apps/product/rpc/internal/svc"
	"flashsale/apps/product/rpc/pb"
	"flashsale/pkg/base/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

// ReserveStockForOrderLogic 封装订单库存预扣逻辑。
type ReserveStockForOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReserveStockForOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReserveStockForOrderLogic {
	return &ReserveStockForOrderLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// ReserveStockForOrder 执行库存预扣。
func (l *ReserveStockForOrderLogic) ReserveStockForOrder(in *pb.ReserveStockForOrderReq) (*pb.ReserveStockForOrderResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.ProductId <= 0 || in.Quantity <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "product_id 或 quantity 非法")
	}
	if strings.TrimSpace(in.BizOrderNo) == "" || strings.TrimSpace(in.IdempotencyKey) == "" {
		return nil, errorx.New(errorx.CodeSysBadRequest, "biz_order_no 或 idempotency_key 不能为空")
	}
	remain, err := l.svcCtx.ProductRepo.ReserveStock(l.ctx, in.ProductId, in.Quantity, in.BizOrderNo, in.IdempotencyKey)
	if err != nil {
		switch err {
		case repository.ErrProductNotFound:
			return nil, errorx.New(errorx.CodeProductNotFound, "商品不存在")
		case repository.ErrStockNotEnough:
			return nil, errorx.New(errorx.CodeOrderOutOfStock, "库存不足")
		case repository.ErrIdempotencyInProgress:
			return nil, errorx.New(errorx.CodeSysInternal, "库存预扣处理中，请稍后重试")
		default:
			return nil, errorx.Wrap(errorx.CodeDBError, "库存预扣失败", err)
		}
	}
	return &pb.ReserveStockForOrderResp{
		ProductId:   in.ProductId,
		RemainStock: remain,
	}, nil
}

// ReleaseStockForOrderLogic 封装订单库存回补逻辑。
type ReleaseStockForOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReleaseStockForOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReleaseStockForOrderLogic {
	return &ReleaseStockForOrderLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// ReleaseStockForOrder 执行库存回补。
func (l *ReleaseStockForOrderLogic) ReleaseStockForOrder(in *pb.ReleaseStockForOrderReq) (*pb.ReleaseStockForOrderResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.ProductId <= 0 || in.Quantity <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "product_id 或 quantity 非法")
	}
	if strings.TrimSpace(in.BizOrderNo) == "" || strings.TrimSpace(in.IdempotencyKey) == "" {
		return nil, errorx.New(errorx.CodeSysBadRequest, "biz_order_no 或 idempotency_key 不能为空")
	}
	remain, err := l.svcCtx.ProductRepo.ReleaseStock(l.ctx, in.ProductId, in.Quantity, in.BizOrderNo, in.IdempotencyKey)
	if err != nil {
		switch err {
		case repository.ErrProductNotFound:
			return nil, errorx.New(errorx.CodeProductNotFound, "商品不存在")
		case repository.ErrIdempotencyInProgress:
			return nil, errorx.New(errorx.CodeSysInternal, "库存回补处理中，请稍后重试")
		default:
			return nil, errorx.Wrap(errorx.CodeDBError, "库存回补失败", err)
		}
	}
	return &pb.ReleaseStockForOrderResp{
		ProductId:   in.ProductId,
		RemainStock: remain,
	}, nil
}
