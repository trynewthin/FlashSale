// logic 包包含相关应用代码。
package logic

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"flashsale/apps/order/rpc/internal/model"
	"flashsale/apps/order/rpc/internal/repository"
	"flashsale/apps/order/rpc/internal/svc"
	"flashsale/apps/order/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/eventx"
	"github.com/zeromicro/go-zero/core/logx"
)

// CreateOrderFromSeckillLogic 封装秒杀建单逻辑。
type CreateOrderFromSeckillLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateOrderFromSeckillLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrderFromSeckillLogic {
	return &CreateOrderFromSeckillLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// CreateOrderFromSeckill 创建秒杀订单（库存由秒杀服务预占，不重复扣减商品库存）。
func (l *CreateOrderFromSeckillLogic) CreateOrderFromSeckill(in *pb.CreateOrderFromSeckillReq) (*pb.CreateOrderFromSeckillResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.UserId <= 0 || in.ActivityId <= 0 || in.ActivityItemId <= 0 || in.ProductId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "user_id 或活动参数非法")
	}
	if in.Quantity <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "quantity 非法")
	}
	if in.SeckillPriceCent <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "秒杀价格非法")
	}
	if strings.TrimSpace(in.SnapshotName) == "" || strings.TrimSpace(in.SnapshotMainImage) == "" || strings.TrimSpace(in.SkuCode) == "" {
		return nil, errorx.New(errorx.CodeSysBadRequest, "商品快照信息不能为空")
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		return nil, errorx.New(errorx.CodeSysBadRequest, "idempotency_key 不能为空")
	}
	idempotencyKey := strings.TrimSpace(in.IdempotencyKey)
	if l.svcCtx == nil || l.svcCtx.OrderRepo == nil || l.svcCtx.IDNode == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}

	orderID := l.svcCtx.IDNode.Generate().Int64()
	orderNo := buildSeckillOrderNo(in.UserId, idempotencyKey)
	now := time.Now()
	unitPrice := in.SeckillPriceCent
	totalAmount := unitPrice * in.Quantity

	order := &model.Order{
		ID:                    orderID,
		OrderNo:               orderNo,
		UserID:                in.UserId,
		OrderSource:           model.OrderSourceSeckill,
		ProductID:             in.ProductId,
		SeckillActivityID:     in.ActivityId,
		SeckillActivityItemID: in.ActivityItemId,
		SkuCode:               strings.TrimSpace(in.SkuCode),
		ProductName:           strings.TrimSpace(in.SnapshotName),
		MainImage:             strings.TrimSpace(in.SnapshotMainImage),
		UnitPriceCent:         unitPrice,
		Quantity:              in.Quantity,
		TotalAmountCent:       totalAmount,
		OrderStatus:           model.OrderStatusPendingPay,
		PaymentStatus:         model.PaymentStatusUnpaid,
		ReviewStatus:          model.ReviewStatusNone,
		ShippingStatus:        model.ShippingStatusNotShipped,
		RefundStatus:          model.RefundStatusNone,
		ReviewMode:            model.ReviewModeManual,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	createEvent := model.OrderEvent{
		OrderID:      orderID,
		EventType:    "order_created_from_seckill",
		OperatorType: "user",
		OperatorID:   in.UserId,
		Payload: mustJSON(map[string]any{
			"activity_id":      in.ActivityId,
			"activity_item_id": in.ActivityItemId,
			"product_id":       in.ProductId,
			"quantity":         in.Quantity,
			"idempotency_key":  idempotencyKey,
		}),
		CreatedAt: now,
	}
	if err := l.svcCtx.OrderRepo.Create(l.ctx, order, createEvent); err != nil {
		existing, ok, lookupErr := l.findIdempotentSeckillOrder(orderNo, in)
		if lookupErr != nil {
			return nil, errorx.Wrap(errorx.CodeDBError, "查询秒杀幂等订单失败", lookupErr)
		}
		if ok {
			return &pb.CreateOrderFromSeckillResp{Order: toOrderView(existing)}, nil
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "创建秒杀订单失败", err)
	}
	created, err := l.svcCtx.OrderRepo.FindByID(l.ctx, orderID)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询秒杀订单失败", err)
	}
	emitSeckillOrderStateEvent(l.ctx, l.svcCtx, created, eventx.SeckillOrderStateEventTypeCreated)
	return &pb.CreateOrderFromSeckillResp{Order: toOrderView(created)}, nil
}

// buildSeckillOrderNo 基于用户与幂等键构造稳定订单号，用于秒杀建单幂等重放。
func buildSeckillOrderNo(userID int64, idempotencyKey string) string {
	raw := fmt.Sprintf("%d:%s", userID, strings.TrimSpace(idempotencyKey))
	sum := sha1.Sum([]byte(raw))
	hexDigest := strings.ToUpper(hex.EncodeToString(sum[:]))
	// orders.order_no 长度上限 32，固定前缀 + 截断哈希。
	return "SCK" + hexDigest[:29]
}

// findIdempotentSeckillOrder 查询幂等键对应的已存在秒杀订单并校验核心参数一致性。
func (l *CreateOrderFromSeckillLogic) findIdempotentSeckillOrder(orderNo string, in *pb.CreateOrderFromSeckillReq) (*model.Order, bool, error) {
	existing, err := l.svcCtx.OrderRepo.FindByOrderNo(l.ctx, orderNo)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if existing.OrderSource != model.OrderSourceSeckill ||
		existing.UserID != in.UserId ||
		existing.SeckillActivityID != in.ActivityId ||
		existing.SeckillActivityItemID != in.ActivityItemId {
		return nil, false, nil
	}
	return existing, true, nil
}
