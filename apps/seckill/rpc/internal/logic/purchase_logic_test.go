// logic 包包含相关应用代码。
package logic

import (
	"context"
	"errors"
	"testing"
	"time"

	orderpb "flashsale/apps/order/rpc/pb"
	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/repository"
	"flashsale/apps/seckill/rpc/internal/svc"
	"flashsale/apps/seckill/rpc/pb"
	"flashsale/pkg/base/errorx"
	"github.com/bwmarrin/snowflake"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// purchaseRepoMock 用于覆盖购买链路中的仓储行为。
type purchaseRepoMock struct {
	*seckillRepoMock

	reserveErr           error
	reservation          *model.PurchaseReservation
	reserveCalls         int
	compensateCalls      int
	createOrderLinkCalls int
}

func (m *purchaseRepoMock) ReservePurchase(context.Context, int64, int64, int64, int64, string, time.Time) (*model.PurchaseReservation, error) {
	m.reserveCalls++
	if m.reserveErr != nil {
		return nil, m.reserveErr
	}
	if m.reservation == nil {
		return nil, errors.New("reservation is nil")
	}
	cp := *m.reservation
	return &cp, nil
}

func (m *purchaseRepoMock) CompensateReleasePurchase(context.Context, int64, int64, int64, string) error {
	m.compensateCalls++
	return nil
}

func (m *purchaseRepoMock) CreateOrderLink(context.Context, *model.OrderLink) error {
	m.createOrderLinkCalls++
	return nil
}

// orderRPCMock 用于模拟秒杀建单 RPC。
type orderRPCMock struct {
	createFromSeckillFn    func(context.Context, *orderpb.CreateOrderFromSeckillReq) (*orderpb.CreateOrderFromSeckillResp, error)
	createFromSeckillCalls int
	lastCreateReq          *orderpb.CreateOrderFromSeckillReq
}

func (m *orderRPCMock) CreateOrder(context.Context, *orderpb.CreateOrderReq, ...grpc.CallOption) (*orderpb.CreateOrderResp, error) {
	return nil, errors.New("not implemented")
}

func (m *orderRPCMock) CreateOrderFromSeckill(ctx context.Context, in *orderpb.CreateOrderFromSeckillReq, _ ...grpc.CallOption) (*orderpb.CreateOrderFromSeckillResp, error) {
	m.createFromSeckillCalls++
	if in != nil {
		m.lastCreateReq = &orderpb.CreateOrderFromSeckillReq{
			UserId:            in.UserId,
			ActivityId:        in.ActivityId,
			ActivityItemId:    in.ActivityItemId,
			ProductId:         in.ProductId,
			Quantity:          in.Quantity,
			SeckillPriceCent:  in.SeckillPriceCent,
			SnapshotName:      in.SnapshotName,
			SnapshotMainImage: in.SnapshotMainImage,
			SkuCode:           in.SkuCode,
			IdempotencyKey:    in.IdempotencyKey,
		}
	}
	if m.createFromSeckillFn == nil {
		return nil, errors.New("createFromSeckillFn is nil")
	}
	return m.createFromSeckillFn(ctx, in)
}

func (m *orderRPCMock) ConfirmPaymentAndInfo(context.Context, *orderpb.ConfirmPaymentAndInfoReq, ...grpc.CallOption) (*orderpb.ConfirmPaymentAndInfoResp, error) {
	return nil, errors.New("not implemented")
}

func (m *orderRPCMock) CancelOrder(context.Context, *orderpb.CancelOrderReq, ...grpc.CallOption) (*orderpb.CancelOrderResp, error) {
	return nil, errors.New("not implemented")
}

func (m *orderRPCMock) ConfirmReceipt(context.Context, *orderpb.ConfirmReceiptReq, ...grpc.CallOption) (*orderpb.ConfirmReceiptResp, error) {
	return nil, errors.New("not implemented")
}

func (m *orderRPCMock) GetOrderUser(context.Context, *orderpb.GetOrderUserReq, ...grpc.CallOption) (*orderpb.GetOrderUserResp, error) {
	return nil, errors.New("not implemented")
}

func (m *orderRPCMock) ListOrdersUser(context.Context, *orderpb.ListOrdersUserReq, ...grpc.CallOption) (*orderpb.ListOrdersUserResp, error) {
	return nil, errors.New("not implemented")
}

func (m *orderRPCMock) ReviewOrderAdmin(context.Context, *orderpb.ReviewOrderAdminReq, ...grpc.CallOption) (*orderpb.ReviewOrderAdminResp, error) {
	return nil, errors.New("not implemented")
}

func (m *orderRPCMock) ShipOrderAdmin(context.Context, *orderpb.ShipOrderAdminReq, ...grpc.CallOption) (*orderpb.ShipOrderAdminResp, error) {
	return nil, errors.New("not implemented")
}

func (m *orderRPCMock) GetOrderAdmin(context.Context, *orderpb.GetOrderAdminReq, ...grpc.CallOption) (*orderpb.GetOrderAdminResp, error) {
	return nil, errors.New("not implemented")
}

func (m *orderRPCMock) ListOrdersAdmin(context.Context, *orderpb.ListOrdersAdminReq, ...grpc.CallOption) (*orderpb.ListOrdersAdminResp, error) {
	return nil, errors.New("not implemented")
}

func TestPurchase_IdempotencyConflictRecoversByOrderReplay(t *testing.T) {
	item := &model.ActivityItem{
		ID:                21,
		ActivityID:        301,
		ProductID:         9101,
		SKUCode:           "SPU9101",
		SnapshotName:      "test-product",
		SnapshotMainImage: "https://img.test/p.png",
		SeckillPriceCent:  990,
	}
	repoMock := &purchaseRepoMock{
		seckillRepoMock: &seckillRepoMock{item: item},
		reserveErr:      repository.ErrIdempotencyConflict,
	}
	orderMock := &orderRPCMock{
		createFromSeckillFn: func(_ context.Context, _ *orderpb.CreateOrderFromSeckillReq) (*orderpb.CreateOrderFromSeckillResp, error) {
			return &orderpb.CreateOrderFromSeckillResp{
				Order: &orderpb.OrderView{
					OrderId:       70001,
					OrderNo:       "SCKORDER70001",
					OrderStatus:   10,
					PaymentStatus: 0,
				},
			}, nil
		},
	}
	node, err := snowflake.NewNode(1)
	if err != nil {
		t.Fatalf("new snowflake node failed: %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-access-token", "token-1"))
	logic := NewSeckillLogic(ctx, &svc.ServiceContext{
		SeckillRepo:              repoMock,
		OrderRPCCli:              orderMock,
		IDNode:                   node,
		OrderLinkWriteOnPurchase: true,
		OrderLinkSyncFallback:    true,
	})

	resp, err := logic.Purchase(&pb.PurchaseReq{
		UserId:         9001,
		ActivityId:     301,
		ActivityItemId: 21,
		Quantity:       1,
		IdempotencyKey: "idem-301-21-9001",
	})
	if err != nil {
		t.Fatalf("purchase should recover successfully, got err=%v", err)
	}
	if resp == nil || resp.OrderId != 70001 {
		t.Fatalf("purchase response mismatch: %+v", resp)
	}
	if repoMock.compensateCalls != 0 {
		t.Fatalf("idempotency conflict path should not compensate immediately, got=%d", repoMock.compensateCalls)
	}
	if repoMock.createOrderLinkCalls != 1 {
		t.Fatalf("order link should be created once, got=%d", repoMock.createOrderLinkCalls)
	}
	if orderMock.createFromSeckillCalls != 1 {
		t.Fatalf("order rpc call count mismatch: got=%d want=1", orderMock.createFromSeckillCalls)
	}
	if orderMock.lastCreateReq == nil || orderMock.lastCreateReq.ProductId != item.ProductID || orderMock.lastCreateReq.SeckillPriceCent != item.SeckillPriceCent {
		t.Fatalf("order request should fallback to item snapshot, got=%+v", orderMock.lastCreateReq)
	}
}

func TestPurchase_IdempotencyConflictPendingWithoutImmediateCompensate(t *testing.T) {
	item := &model.ActivityItem{
		ID:                22,
		ActivityID:        302,
		ProductID:         9102,
		SKUCode:           "SPU9102",
		SnapshotName:      "test-product-2",
		SnapshotMainImage: "https://img.test/p2.png",
		SeckillPriceCent:  1990,
	}
	repoMock := &purchaseRepoMock{
		seckillRepoMock: &seckillRepoMock{item: item},
		reserveErr:      repository.ErrIdempotencyConflict,
	}
	orderMock := &orderRPCMock{
		createFromSeckillFn: func(_ context.Context, _ *orderpb.CreateOrderFromSeckillReq) (*orderpb.CreateOrderFromSeckillResp, error) {
			return nil, status.Error(codes.Unavailable, "rpc unavailable")
		},
	}
	node, err := snowflake.NewNode(2)
	if err != nil {
		t.Fatalf("new snowflake node failed: %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-access-token", "token-2"))
	logic := NewSeckillLogic(ctx, &svc.ServiceContext{
		SeckillRepo: repoMock,
		OrderRPCCli: orderMock,
		IDNode:      node,
	})

	_, err = logic.Purchase(&pb.PurchaseReq{
		UserId:         9002,
		ActivityId:     302,
		ActivityItemId: 22,
		Quantity:       1,
		IdempotencyKey: "idem-302-22-9002",
	})
	appErr := errorx.FromError(err)
	if appErr == nil || appErr.Code != errorx.CodeSeckillPurchaseConflict {
		t.Fatalf("expected purchase pending conflict, got err=%v", err)
	}
	if repoMock.compensateCalls != 0 {
		t.Fatalf("idempotency conflict pending path should not compensate, got=%d", repoMock.compensateCalls)
	}
	if repoMock.createOrderLinkCalls != 0 {
		t.Fatalf("pending path should not create order link, got=%d", repoMock.createOrderLinkCalls)
	}
	if orderMock.createFromSeckillCalls != 1 {
		t.Fatalf("unavailable should not replay immediately, got=%d", orderMock.createFromSeckillCalls)
	}
}

func TestPurchase_IdempotencyConflictReplayOnDeadlineExceeded(t *testing.T) {
	item := &model.ActivityItem{
		ID:                23,
		ActivityID:        303,
		ProductID:         9103,
		SKUCode:           "SPU9103",
		SnapshotName:      "test-product-3",
		SnapshotMainImage: "https://img.test/p3.png",
		SeckillPriceCent:  2990,
	}
	repoMock := &purchaseRepoMock{
		seckillRepoMock: &seckillRepoMock{item: item},
		reserveErr:      repository.ErrIdempotencyConflict,
	}
	call := 0
	orderMock := &orderRPCMock{
		createFromSeckillFn: func(_ context.Context, _ *orderpb.CreateOrderFromSeckillReq) (*orderpb.CreateOrderFromSeckillResp, error) {
			call++
			if call == 1 {
				return nil, status.Error(codes.DeadlineExceeded, "timeout")
			}
			return &orderpb.CreateOrderFromSeckillResp{
				Order: &orderpb.OrderView{
					OrderId:       70003,
					OrderNo:       "SCKORDER70003",
					OrderStatus:   10,
					PaymentStatus: 0,
				},
			}, nil
		},
	}
	node, err := snowflake.NewNode(3)
	if err != nil {
		t.Fatalf("new snowflake node failed: %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-access-token", "token-3"))
	logic := NewSeckillLogic(ctx, &svc.ServiceContext{
		SeckillRepo: repoMock,
		OrderRPCCli: orderMock,
		IDNode:      node,
	})

	resp, err := logic.Purchase(&pb.PurchaseReq{
		UserId:         9003,
		ActivityId:     303,
		ActivityItemId: 23,
		Quantity:       1,
		IdempotencyKey: "idem-303-23-9003",
	})
	if err != nil {
		t.Fatalf("deadline replay should recover, got err=%v", err)
	}
	if resp == nil || resp.OrderId != 70003 {
		t.Fatalf("response mismatch: %+v", resp)
	}
	if orderMock.createFromSeckillCalls != 2 {
		t.Fatalf("deadline path should replay once, got=%d", orderMock.createFromSeckillCalls)
	}
}

func TestPurchase_OrderCreateOverloadedCompensates(t *testing.T) {
	item := &model.ActivityItem{
		ID:                24,
		ActivityID:        304,
		ProductID:         9104,
		SKUCode:           "SPU9104",
		SnapshotName:      "test-product-4",
		SnapshotMainImage: "https://img.test/p4.png",
		SeckillPriceCent:  3990,
		MaxQtyPerOrder:    2,
	}
	repoMock := &purchaseRepoMock{
		seckillRepoMock: &seckillRepoMock{item: item},
		reservation: &model.PurchaseReservation{
			ActivityID:        304,
			ActivityItemID:    24,
			ProductID:         9104,
			Quantity:          1,
			RemainStock:       99,
			SeckillPriceCent:  3990,
			SKUCode:           "SPU9104",
			SnapshotName:      "test-product-4",
			SnapshotMainImage: "https://img.test/p4.png",
		},
	}
	orderMock := &orderRPCMock{
		createFromSeckillFn: func(_ context.Context, _ *orderpb.CreateOrderFromSeckillReq) (*orderpb.CreateOrderFromSeckillResp, error) {
			return &orderpb.CreateOrderFromSeckillResp{
				Order: &orderpb.OrderView{OrderId: 70004, OrderNo: "SCKORDER70004"},
			}, nil
		},
	}
	node, err := snowflake.NewNode(4)
	if err != nil {
		t.Fatalf("new snowflake node failed: %v", err)
	}
	limiter := make(chan struct{}, 1)
	limiter <- struct{}{}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-access-token", "token-4"))
	logic := NewSeckillLogic(ctx, &svc.ServiceContext{
		SeckillRepo:               repoMock,
		OrderRPCCli:               orderMock,
		IDNode:                    node,
		OrderCreateLimiter:        limiter,
		OrderCreateAcquireTimeout: time.Millisecond,
	})

	_, err = logic.Purchase(&pb.PurchaseReq{
		UserId:         9004,
		ActivityId:     304,
		ActivityItemId: 24,
		Quantity:       1,
		IdempotencyKey: "idem-304-24-9004",
	})
	appErr := errorx.FromError(err)
	if appErr == nil || appErr.Code != errorx.CodeSeckillPurchaseConflict {
		t.Fatalf("expected purchase conflict on overload, got err=%v", err)
	}
	if orderMock.createFromSeckillCalls != 0 {
		t.Fatalf("overload should fail fast before order rpc call, got=%d", orderMock.createFromSeckillCalls)
	}
	if repoMock.compensateCalls != 1 {
		t.Fatalf("overload should compensate reserved stock, got=%d", repoMock.compensateCalls)
	}
}

func TestPurchase_OrderCreateConflictCompensates(t *testing.T) {
	item := &model.ActivityItem{
		ID:                25,
		ActivityID:        305,
		ProductID:         9105,
		SKUCode:           "SPU9105",
		SnapshotName:      "test-product-5",
		SnapshotMainImage: "https://img.test/p5.png",
		SeckillPriceCent:  4990,
		MaxQtyPerOrder:    2,
	}
	repoMock := &purchaseRepoMock{
		seckillRepoMock: &seckillRepoMock{item: item},
		reservation: &model.PurchaseReservation{
			ActivityID:        305,
			ActivityItemID:    25,
			ProductID:         9105,
			Quantity:          1,
			RemainStock:       99,
			SeckillPriceCent:  4990,
			SKUCode:           "SPU9105",
			SnapshotName:      "test-product-5",
			SnapshotMainImage: "https://img.test/p5.png",
		},
	}
	orderMock := &orderRPCMock{
		createFromSeckillFn: func(_ context.Context, _ *orderpb.CreateOrderFromSeckillReq) (*orderpb.CreateOrderFromSeckillResp, error) {
			return nil, errorx.New(errorx.CodeSeckillPurchaseConflict, "busy")
		},
	}
	node, err := snowflake.NewNode(5)
	if err != nil {
		t.Fatalf("new snowflake node failed: %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-access-token", "token-5"))
	logic := NewSeckillLogic(ctx, &svc.ServiceContext{
		SeckillRepo: repoMock,
		OrderRPCCli: orderMock,
		IDNode:      node,
	})

	_, err = logic.Purchase(&pb.PurchaseReq{
		UserId:         9005,
		ActivityId:     305,
		ActivityItemId: 25,
		Quantity:       1,
		IdempotencyKey: "idem-305-25-9005",
	})
	appErr := errorx.FromError(err)
	if appErr == nil || appErr.Code != errorx.CodeSeckillPurchaseConflict {
		t.Fatalf("expected purchase conflict on order busy, got err=%v", err)
	}
	if repoMock.compensateCalls != 1 {
		t.Fatalf("order busy should compensate reserved stock, got=%d", repoMock.compensateCalls)
	}
	if repoMock.createOrderLinkCalls != 0 {
		t.Fatalf("order busy should not create order link, got=%d", repoMock.createOrderLinkCalls)
	}
}

func TestTrackEvent_DegradeOnRecordFailure(t *testing.T) {
	repoMock := &seckillRepoMock{trafficErr: errors.New("mock traffic write failed")}
	logic := NewSeckillLogic(context.Background(), &svc.ServiceContext{SeckillRepo: repoMock})
	resp, err := logic.TrackEvent(&pb.TrackEventReq{
		ActivityId:     401,
		ActivityItemId: 41,
		EventType:      model.TrafficEventClick,
		ClientId:       "client-1",
		IdempotencyKey: "track-401-41-1",
		OccurredAtUnix: time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("track event should degrade without rpc error, got=%v", err)
	}
	if resp == nil || resp.Accepted {
		t.Fatalf("expected accepted=false when record fails, got=%+v", resp)
	}
}
