// logic 包包含相关应用代码。
package logic

import (
	"context"
	"fmt"
	"time"

	"flashsale/apps/order/rpc/internal/model"
	"flashsale/apps/order/rpc/internal/svc"
	"flashsale/apps/order/rpc/pb"
	productpb "flashsale/apps/product/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/grpcerr"
	"flashsale/pkg/base/rpcmeta"
	"github.com/zeromicro/go-zero/core/logx"
)

// CreateOrderLogic 封装创建订单逻辑。
type CreateOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrderLogic {
	return &CreateOrderLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// CreateOrder 创建订单并执行库存预扣。
func (l *CreateOrderLogic) CreateOrder(in *pb.CreateOrderReq) (*pb.CreateOrderResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.UserId <= 0 || in.ProductId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "user_id 或 product_id 非法")
	}
	if l.svcCtx == nil || l.svcCtx.OrderRepo == nil || l.svcCtx.ProductRPCCli == nil || l.svcCtx.IDNode == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}

	orderSource, err := normalizeOrderSource(in.OrderSource)
	if err != nil {
		return nil, err
	}

	productResp, err := l.svcCtx.ProductRPCCli.GetProductPublic(l.ctx, &productpb.GetProductPublicReq{ProductId: in.ProductId})
	if err != nil {
		return nil, grpcerr.FromStatus(err)
	}
	if productResp == nil || productResp.Product == nil {
		return nil, errorx.New(errorx.CodeProductNotFound, "商品不存在")
	}

	orderID := l.svcCtx.IDNode.Generate().Int64()
	orderNo := fmt.Sprintf("ORD%d", orderID)

	productToken, err := l.svcCtx.IssueProductManagementToken()
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeSysInternal, "生成内部鉴权令牌失败", err)
	}
	reserveCtx := rpcmeta.WithAccessToken(l.ctx, productToken)
	_, err = l.svcCtx.ProductRPCCli.ReserveStockForOrder(reserveCtx, &productpb.ReserveStockForOrderReq{
		ProductId:      in.ProductId,
		Quantity:       1,
		BizOrderNo:     orderNo,
		IdempotencyKey: orderNo + ":reserve",
	})
	if err != nil {
		appErr := grpcerr.FromStatus(err)
		if appErr != nil {
			return nil, appErr
		}
		return nil, errorx.Wrap(errorx.CodeSysInternal, "库存预扣失败", err)
	}

	now := time.Now()
	order := &model.Order{
		ID:              orderID,
		OrderNo:         orderNo,
		UserID:          in.UserId,
		OrderSource:     orderSource,
		ProductID:       productResp.Product.ProductId,
		SkuCode:         productResp.Product.SkuCode,
		ProductName:     productResp.Product.Name,
		MainImage:       productResp.Product.MainImage,
		UnitPriceCent:   productResp.Product.PriceCent,
		Quantity:        1,
		TotalAmountCent: productResp.Product.PriceCent,
		OrderStatus:     model.OrderStatusPendingPay,
		PaymentStatus:   model.PaymentStatusUnpaid,
		ReviewStatus:    model.ReviewStatusNone,
		ShippingStatus:  model.ShippingStatusNotShipped,
		RefundStatus:    model.RefundStatusNone,
		ReviewMode:      model.ReviewModeAuto,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	createEvent := model.OrderEvent{
		OrderID:      orderID,
		EventType:    "order_created",
		OperatorType: "user",
		OperatorID:   in.UserId,
		Payload: mustJSON(map[string]any{
			"product_id":        in.ProductId,
			"order_source":      orderSource,
			"total_amount_cent": order.TotalAmountCent,
		}),
		CreatedAt: now,
	}
	if err := l.svcCtx.OrderRepo.Create(l.ctx, order, createEvent); err != nil {
		l.compensateReleaseStock(orderNo, in.ProductId)
		return nil, errorx.Wrap(errorx.CodeDBError, "创建订单失败", err)
	}

	created, err := l.svcCtx.OrderRepo.FindByID(l.ctx, orderID)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询订单失败", err)
	}
	return &pb.CreateOrderResp{Order: toOrderView(created)}, nil
}

func (l *CreateOrderLogic) compensateReleaseStock(orderNo string, productID int64) {
	productToken, err := l.svcCtx.IssueProductManagementToken()
	if err != nil {
		l.Logger.Errorf("issue token for compensate release failed: %v", err)
		return
	}
	compensateCtx := rpcmeta.WithAccessToken(context.Background(), productToken)
	_, err = l.svcCtx.ProductRPCCli.ReleaseStockForOrder(compensateCtx, &productpb.ReleaseStockForOrderReq{
		ProductId:      productID,
		Quantity:       1,
		BizOrderNo:     orderNo,
		IdempotencyKey: orderNo + ":release:create_fail",
	})
	if err != nil {
		l.Logger.Errorf("compensate release stock failed: %v", err)
	}
}
