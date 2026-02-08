// logic 包包含相关应用代码。
package logic

import (
	"context"
	"sort"
	"testing"
	"time"

	"flashsale/apps/order/rpc/internal/model"
	"flashsale/apps/order/rpc/internal/repository"
	"flashsale/apps/order/rpc/internal/svc"
	"flashsale/apps/order/rpc/pb"
	productpb "flashsale/apps/product/rpc/pb"
	"flashsale/apps/product/rpc/productrpc"
	baseauth "flashsale/pkg/base/authx"
	"flashsale/pkg/base/errorx"
	"github.com/bwmarrin/snowflake"
	"google.golang.org/grpc"
)

type memoryOrderRepo struct {
	byID   map[int64]*model.Order
	events []model.OrderEvent
}

func newMemoryOrderRepo() *memoryOrderRepo {
	return &memoryOrderRepo{
		byID:   make(map[int64]*model.Order),
		events: make([]model.OrderEvent, 0, 16),
	}
}

func (m *memoryOrderRepo) seed(order *model.Order) {
	cp := *order
	m.byID[order.ID] = &cp
}

func (m *memoryOrderRepo) Create(_ context.Context, order *model.Order, event model.OrderEvent) error {
	if order == nil {
		return repository.ErrOrderStateConflict
	}
	for _, existing := range m.byID {
		if existing != nil && existing.OrderNo == order.OrderNo {
			return repository.ErrOrderStateConflict
		}
	}
	now := time.Now()
	cp := *order
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = now
	}
	if cp.UpdatedAt.IsZero() {
		cp.UpdatedAt = cp.CreatedAt
	}
	m.byID[cp.ID] = &cp
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) FindByID(_ context.Context, orderID int64) (*model.Order, error) {
	v, ok := m.byID[orderID]
	if !ok {
		return nil, repository.ErrOrderNotFound
	}
	cp := *v
	return &cp, nil
}

func (m *memoryOrderRepo) ListByUser(_ context.Context, query repository.UserListQuery) ([]*model.Order, int64, error) {
	items := make([]*model.Order, 0, len(m.byID))
	for _, v := range m.byID {
		if v.UserID != query.UserID {
			continue
		}
		if query.OrderStatus > 0 && v.OrderStatus != query.OrderStatus {
			continue
		}
		cp := *v
		items = append(items, &cp)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return paginateOrders(items, query.Page, query.PageSize)
}

func (m *memoryOrderRepo) ListAdmin(_ context.Context, query repository.AdminListQuery) ([]*model.Order, int64, error) {
	items := make([]*model.Order, 0, len(m.byID))
	for _, v := range m.byID {
		if query.OrderStatus > 0 && v.OrderStatus != query.OrderStatus {
			continue
		}
		if query.ReviewStatus > 0 && v.ReviewStatus != query.ReviewStatus {
			continue
		}
		if query.UserID > 0 && v.UserID != query.UserID {
			continue
		}
		if query.OrderNo != "" && v.OrderNo != query.OrderNo {
			continue
		}
		cp := *v
		items = append(items, &cp)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return paginateOrders(items, query.Page, query.PageSize)
}

func paginateOrders(items []*model.Order, page, pageSize int64) ([]*model.Order, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	total := int64(len(items))
	start := (page - 1) * pageSize
	if start >= total {
		return []*model.Order{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return items[start:end], total, nil
}

func (m *memoryOrderRepo) MarkPaidAutoPass(_ context.Context, orderID, userID int64, paidAt time.Time, payChannel, payRef string, receiverName, receiverPhone, receiverAddress, buyerRemark string, event model.OrderEvent) error {
	o, err := m.mustPendingPay(orderID, userID)
	if err != nil {
		return err
	}
	o.PaymentStatus = model.PaymentStatusPaid
	o.PaidAt = &paidAt
	o.PayChannel = payChannel
	o.PayReference = payRef
	o.ReceiverName = receiverName
	o.ReceiverPhone = receiverPhone
	o.ReceiverAddress = receiverAddress
	o.BuyerRemark = buyerRemark
	o.OrderStatus = model.OrderStatusPendingShip
	o.ReviewStatus = model.ReviewStatusPassed
	o.ReviewMode = model.ReviewModeAuto
	o.ReviewDueAt = nil
	o.UpdatedAt = paidAt
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) MarkPaidManualReview(_ context.Context, orderID, userID int64, paidAt time.Time, payChannel, payRef string, receiverName, receiverPhone, receiverAddress, buyerRemark string, reviewDueAt time.Time, event model.OrderEvent) error {
	o, err := m.mustPendingPay(orderID, userID)
	if err != nil {
		return err
	}
	o.PaymentStatus = model.PaymentStatusPaid
	o.PaidAt = &paidAt
	o.PayChannel = payChannel
	o.PayReference = payRef
	o.ReceiverName = receiverName
	o.ReceiverPhone = receiverPhone
	o.ReceiverAddress = receiverAddress
	o.BuyerRemark = buyerRemark
	o.OrderStatus = model.OrderStatusPendingReview
	o.ReviewStatus = model.ReviewStatusManualPending
	o.ReviewMode = model.ReviewModeManual
	o.ReviewDueAt = &reviewDueAt
	o.UpdatedAt = paidAt
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) CancelUnpaid(_ context.Context, orderID, userID int64, now time.Time, reason string, event model.OrderEvent) error {
	o, err := m.mustPendingPay(orderID, userID)
	if err != nil {
		return err
	}
	o.OrderStatus = model.OrderStatusClosed
	o.CloseReason = reason
	o.ClosedAt = &now
	o.UpdatedAt = now
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) CancelPaidPendingReview(_ context.Context, orderID, userID int64, now time.Time, reason string, refundDueAt time.Time, event model.OrderEvent) error {
	o, ok := m.byID[orderID]
	if !ok {
		return repository.ErrOrderNotFound
	}
	if o.UserID != userID || o.OrderStatus != model.OrderStatusPendingReview || o.PaymentStatus != model.PaymentStatusPaid {
		return repository.ErrOrderStateConflict
	}
	o.OrderStatus = model.OrderStatusClosed
	o.CloseReason = reason
	o.ClosedAt = &now
	o.RefundStatus = model.RefundStatusRefunding
	o.RefundDueAt = &refundDueAt
	o.UpdatedAt = now
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) ReviewApprove(_ context.Context, orderID, adminID int64, now time.Time, reason string, event model.OrderEvent) error {
	o, ok := m.byID[orderID]
	if !ok {
		return repository.ErrOrderNotFound
	}
	if o.OrderStatus != model.OrderStatusPendingReview || o.ReviewStatus != model.ReviewStatusManualPending {
		return repository.ErrOrderStateConflict
	}
	o.OrderStatus = model.OrderStatusPendingShip
	o.ReviewStatus = model.ReviewStatusPassed
	o.ReviewedBy = adminID
	o.ReviewReason = reason
	o.ReviewedAt = &now
	o.UpdatedAt = now
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) ReviewReject(_ context.Context, orderID, adminID int64, now time.Time, reason string, refundDueAt time.Time, event model.OrderEvent) error {
	o, ok := m.byID[orderID]
	if !ok {
		return repository.ErrOrderNotFound
	}
	if o.OrderStatus != model.OrderStatusPendingReview || o.ReviewStatus != model.ReviewStatusManualPending {
		return repository.ErrOrderStateConflict
	}
	o.OrderStatus = model.OrderStatusClosed
	o.ReviewStatus = model.ReviewStatusRejected
	o.ReviewedBy = adminID
	o.ReviewReason = reason
	o.ReviewedAt = &now
	o.RefundStatus = model.RefundStatusRefunding
	o.RefundDueAt = &refundDueAt
	o.CloseReason = model.CloseReasonAuditReject
	o.ClosedAt = &now
	o.UpdatedAt = now
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) Ship(_ context.Context, orderID, adminID int64, trackingNo string, now time.Time, event model.OrderEvent) error {
	o, ok := m.byID[orderID]
	if !ok {
		return repository.ErrOrderNotFound
	}
	if o.OrderStatus != model.OrderStatusPendingShip {
		return repository.ErrOrderStateConflict
	}
	o.OrderStatus = model.OrderStatusShipped
	o.ShippingStatus = model.ShippingStatusShipped
	o.ShippedAt = &now
	o.ShippedBy = adminID
	o.TrackingNo = trackingNo
	o.UpdatedAt = now
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) ConfirmReceipt(_ context.Context, orderID, userID int64, now time.Time, event model.OrderEvent) error {
	o, ok := m.byID[orderID]
	if !ok {
		return repository.ErrOrderNotFound
	}
	if o.UserID != userID || o.OrderStatus != model.OrderStatusShipped || o.ShippingStatus != model.ShippingStatusShipped {
		return repository.ErrOrderStateConflict
	}
	o.OrderStatus = model.OrderStatusClosed
	o.ShippingStatus = model.ShippingStatusReceived
	o.CloseReason = model.CloseReasonCompleted
	o.ClosedAt = &now
	o.UpdatedAt = now
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) AutoReceive(_ context.Context, orderID int64, now time.Time, event model.OrderEvent) error {
	o, ok := m.byID[orderID]
	if !ok {
		return repository.ErrOrderNotFound
	}
	if o.OrderStatus != model.OrderStatusShipped || o.ShippingStatus != model.ShippingStatusShipped {
		return repository.ErrOrderStateConflict
	}
	o.OrderStatus = model.OrderStatusClosed
	o.ShippingStatus = model.ShippingStatusReceived
	o.CloseReason = model.CloseReasonAutoCompleted
	o.ClosedAt = &now
	o.UpdatedAt = now
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) ListPayTimeout(_ context.Context, before time.Time, limit int) ([]*model.Order, error) {
	out := make([]*model.Order, 0, limit)
	for _, o := range m.byID {
		if o.OrderStatus == model.OrderStatusPendingPay && o.PaymentStatus == model.PaymentStatusUnpaid && !o.CreatedAt.After(before) {
			cp := *o
			out = append(out, &cp)
		}
	}
	if len(out) > limit && limit > 0 {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryOrderRepo) ClosePayTimeout(_ context.Context, orderID int64, now time.Time, event model.OrderEvent) error {
	o, ok := m.byID[orderID]
	if !ok {
		return repository.ErrOrderNotFound
	}
	if o.OrderStatus != model.OrderStatusPendingPay || o.PaymentStatus != model.PaymentStatusUnpaid {
		return repository.ErrOrderStateConflict
	}
	o.OrderStatus = model.OrderStatusClosed
	o.CloseReason = model.CloseReasonPayTimeout
	o.ClosedAt = &now
	o.UpdatedAt = now
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) ListReviewTimeout(_ context.Context, before time.Time, limit int) ([]*model.Order, error) {
	out := make([]*model.Order, 0, limit)
	for _, o := range m.byID {
		if o.OrderStatus == model.OrderStatusPendingReview &&
			o.ReviewStatus == model.ReviewStatusManualPending &&
			o.ReviewDueAt != nil &&
			!o.ReviewDueAt.After(before) {
			cp := *o
			out = append(out, &cp)
		}
	}
	if len(out) > limit && limit > 0 {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryOrderRepo) ListStockReleasePending(_ context.Context, limit int) ([]*model.Order, error) {
	out := make([]*model.Order, 0, limit)
	for _, o := range m.byID {
		if o.OrderStatus != model.OrderStatusClosed || o.StockReleased != model.StockReleasePending {
			continue
		}
		if o.CloseReason != model.CloseReasonUserCancel &&
			o.CloseReason != model.CloseReasonPayTimeout &&
			o.CloseReason != model.CloseReasonAuditReject &&
			o.CloseReason != model.CloseReasonAuditTimeout {
			continue
		}
		cp := *o
		out = append(out, &cp)
	}
	if len(out) > limit && limit > 0 {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryOrderRepo) MarkStockReleased(_ context.Context, orderID int64, now time.Time, event model.OrderEvent) error {
	o, ok := m.byID[orderID]
	if !ok {
		return repository.ErrOrderNotFound
	}
	if o.StockReleased == model.StockReleased {
		return nil
	}
	o.StockReleased = model.StockReleased
	o.UpdatedAt = now
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) CloseReviewTimeoutRefunding(_ context.Context, orderID int64, now time.Time, refundDueAt time.Time, event model.OrderEvent) error {
	o, ok := m.byID[orderID]
	if !ok {
		return repository.ErrOrderNotFound
	}
	if o.OrderStatus != model.OrderStatusPendingReview || o.ReviewStatus != model.ReviewStatusManualPending {
		return repository.ErrOrderStateConflict
	}
	o.OrderStatus = model.OrderStatusClosed
	o.ReviewStatus = model.ReviewStatusTimeoutReject
	o.RefundStatus = model.RefundStatusRefunding
	o.RefundDueAt = &refundDueAt
	o.CloseReason = model.CloseReasonAuditTimeout
	o.ClosedAt = &now
	o.UpdatedAt = now
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) ListRefundingDue(_ context.Context, before time.Time, limit int) ([]*model.Order, error) {
	out := make([]*model.Order, 0, limit)
	for _, o := range m.byID {
		if o.OrderStatus == model.OrderStatusClosed &&
			o.RefundStatus == model.RefundStatusRefunding &&
			o.RefundDueAt != nil &&
			!o.RefundDueAt.After(before) {
			cp := *o
			out = append(out, &cp)
		}
	}
	if len(out) > limit && limit > 0 {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryOrderRepo) CompleteRefund(_ context.Context, orderID int64, now time.Time, event model.OrderEvent) error {
	o, ok := m.byID[orderID]
	if !ok {
		return repository.ErrOrderNotFound
	}
	if o.OrderStatus != model.OrderStatusClosed || o.RefundStatus != model.RefundStatusRefunding {
		return repository.ErrOrderStateConflict
	}
	o.RefundStatus = model.RefundStatusRefunded
	o.RefundedAt = &now
	o.UpdatedAt = now
	m.events = append(m.events, event)
	return nil
}

func (m *memoryOrderRepo) ListAutoReceiveDue(_ context.Context, before time.Time, limit int) ([]*model.Order, error) {
	out := make([]*model.Order, 0, limit)
	for _, o := range m.byID {
		if o.OrderStatus == model.OrderStatusShipped &&
			o.ShippingStatus == model.ShippingStatusShipped &&
			o.ShippedAt != nil &&
			!o.ShippedAt.After(before) {
			cp := *o
			out = append(out, &cp)
		}
	}
	if len(out) > limit && limit > 0 {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryOrderRepo) mustPendingPay(orderID, userID int64) (*model.Order, error) {
	o, ok := m.byID[orderID]
	if !ok {
		return nil, repository.ErrOrderNotFound
	}
	if o.UserID != userID || o.OrderStatus != model.OrderStatusPendingPay || o.PaymentStatus != model.PaymentStatusUnpaid {
		return nil, repository.ErrOrderStateConflict
	}
	return o, nil
}

type fakeProductRPC struct {
	product   *productpb.PublicProduct
	stockByID map[int64]int64
}

func newFakeProductRPC(productID int64, sku string, name string, price int64, stock int64) *fakeProductRPC {
	return &fakeProductRPC{
		product: &productpb.PublicProduct{
			ProductId:   productID,
			SkuCode:     sku,
			Name:        name,
			MainImage:   "https://img/a.png",
			Description: "desc",
			PriceCent:   price,
			InStock:     stock > 0,
		},
		stockByID: map[int64]int64{productID: stock},
	}
}

func (f *fakeProductRPC) CreateProduct(context.Context, *productrpc.CreateProductReq, ...grpc.CallOption) (*productrpc.CreateProductResp, error) {
	return nil, nil
}
func (f *fakeProductRPC) UpdateProduct(context.Context, *productrpc.UpdateProductReq, ...grpc.CallOption) (*productrpc.UpdateProductResp, error) {
	return nil, nil
}
func (f *fakeProductRPC) DeleteProduct(context.Context, *productrpc.DeleteProductReq, ...grpc.CallOption) (*productrpc.DeleteProductResp, error) {
	return nil, nil
}
func (f *fakeProductRPC) GetProductAdmin(context.Context, *productrpc.GetProductAdminReq, ...grpc.CallOption) (*productrpc.GetProductAdminResp, error) {
	return nil, nil
}
func (f *fakeProductRPC) ListProductsAdmin(context.Context, *productrpc.ListProductsAdminReq, ...grpc.CallOption) (*productrpc.ListProductsAdminResp, error) {
	return nil, nil
}
func (f *fakeProductRPC) ListProductsPublic(context.Context, *productrpc.ListProductsPublicReq, ...grpc.CallOption) (*productrpc.ListProductsPublicResp, error) {
	return nil, nil
}
func (f *fakeProductRPC) GetProductPublic(_ context.Context, req *productrpc.GetProductPublicReq, _ ...grpc.CallOption) (*productrpc.GetProductPublicResp, error) {
	if req == nil || f.product == nil || req.ProductId != f.product.ProductId {
		return nil, errorx.New(errorx.CodeProductNotFound, "not found")
	}
	return &productpb.GetProductPublicResp{
		Product: &productpb.PublicProduct{
			ProductId:   f.product.ProductId,
			SkuCode:     f.product.SkuCode,
			Name:        f.product.Name,
			MainImage:   f.product.MainImage,
			Description: f.product.Description,
			PriceCent:   f.product.PriceCent,
			InStock:     f.stockByID[req.ProductId] > 0,
		},
	}, nil
}
func (f *fakeProductRPC) ReserveStockForOrder(_ context.Context, req *productrpc.ReserveStockForOrderReq, _ ...grpc.CallOption) (*productrpc.ReserveStockForOrderResp, error) {
	if req == nil || req.ProductId <= 0 || req.Quantity <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "invalid reserve")
	}
	cur := f.stockByID[req.ProductId]
	if cur < req.Quantity {
		return nil, errorx.New(errorx.CodeOrderOutOfStock, "out of stock")
	}
	cur -= req.Quantity
	f.stockByID[req.ProductId] = cur
	return &productpb.ReserveStockForOrderResp{ProductId: req.ProductId, RemainStock: cur}, nil
}
func (f *fakeProductRPC) ReleaseStockForOrder(_ context.Context, req *productrpc.ReleaseStockForOrderReq, _ ...grpc.CallOption) (*productrpc.ReleaseStockForOrderResp, error) {
	if req == nil || req.ProductId <= 0 || req.Quantity <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "invalid release")
	}
	cur := f.stockByID[req.ProductId] + req.Quantity
	f.stockByID[req.ProductId] = cur
	return &productpb.ReleaseStockForOrderResp{ProductId: req.ProductId, RemainStock: cur}, nil
}

func (f *fakeProductRPC) ReserveStockForActivity(_ context.Context, req *productrpc.ReserveStockForActivityReq, _ ...grpc.CallOption) (*productrpc.ReserveStockForActivityResp, error) {
	if req == nil || req.ProductId <= 0 || req.Quantity <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "invalid reserve")
	}
	cur := f.stockByID[req.ProductId]
	if cur < req.Quantity {
		return nil, errorx.New(errorx.CodeOrderOutOfStock, "out of stock")
	}
	cur -= req.Quantity
	f.stockByID[req.ProductId] = cur
	return &productpb.ReserveStockForActivityResp{ProductId: req.ProductId, RemainStock: cur}, nil
}

func (f *fakeProductRPC) ReleaseStockForActivity(_ context.Context, req *productrpc.ReleaseStockForActivityReq, _ ...grpc.CallOption) (*productrpc.ReleaseStockForActivityResp, error) {
	if req == nil || req.ProductId <= 0 || req.Quantity <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "invalid release")
	}
	cur := f.stockByID[req.ProductId] + req.Quantity
	f.stockByID[req.ProductId] = cur
	return &productpb.ReleaseStockForActivityResp{ProductId: req.ProductId, RemainStock: cur}, nil
}

func initOrderAuthForTest(t *testing.T) {
	t.Helper()
	if err := baseauth.Init(baseauth.JWTConfig{
		User:  baseauth.JWTDomainConfig{Secret: "order-test-user-secret", Issuer: "order-test-user-iss", Audience: "order-test-user-aud", TTL: time.Hour},
		Admin: baseauth.JWTDomainConfig{Secret: "order-test-admin-secret", Issuer: "order-test-admin-iss", Audience: "order-test-admin-aud", TTL: time.Hour},
	}); err != nil {
		t.Fatalf("init auth failed: %v", err)
	}
}

func newTestOrderSvc(t *testing.T, repo *memoryOrderRepo, p *fakeProductRPC) *svc.ServiceContext {
	t.Helper()
	node, err := snowflake.NewNode(31)
	if err != nil {
		t.Fatalf("new snowflake node failed: %v", err)
	}
	return &svc.ServiceContext{
		OrderRepo:     repo,
		ProductRPCCli: p,
		IDNode:        node,
	}
}

func TestCreateOrderAndConfirmPaymentAutoReview(t *testing.T) {
	initOrderAuthForTest(t)
	repo := newMemoryOrderRepo()
	product := newFakeProductRPC(1001, "SPU1001", "可乐", 399, 10)
	svcCtx := newTestOrderSvc(t, repo, product)
	ctx := context.Background()

	create := NewCreateOrderLogic(ctx, svcCtx)
	cResp, err := create.CreateOrder(&pb.CreateOrderReq{
		UserId:      2001,
		ProductId:   1001,
		OrderSource: int32(model.OrderSourceNormal),
	})
	if err != nil {
		t.Fatalf("create order failed: %v", err)
	}
	if got := cResp.GetOrder().GetOrderStatus(); got != int32(model.OrderStatusPendingPay) {
		t.Fatalf("create status mismatch: got=%d", got)
	}
	if stock := product.stockByID[1001]; stock != 9 {
		t.Fatalf("reserve stock mismatch: got=%d want=9", stock)
	}

	pay := NewConfirmPaymentAndInfoLogic(ctx, svcCtx)
	pResp, err := pay.ConfirmPaymentAndInfo(&pb.ConfirmPaymentAndInfoReq{
		UserId:          2001,
		OrderId:         cResp.Order.OrderId,
		PayChannel:      "mock",
		PayReference:    "pay-001",
		ReceiverName:    "张三",
		ReceiverPhone:   "13800138000",
		ReceiverAddress: "上海市浦东新区",
		BuyerRemark:     "尽快",
	})
	if err != nil {
		t.Fatalf("confirm payment failed: %v", err)
	}
	if pResp.Order.OrderStatus != int32(model.OrderStatusPendingShip) {
		t.Fatalf("order status mismatch: got=%d want=%d", pResp.Order.OrderStatus, model.OrderStatusPendingShip)
	}
	if pResp.Order.ReviewStatus != int32(model.ReviewStatusPassed) {
		t.Fatalf("review status mismatch: got=%d want=%d", pResp.Order.ReviewStatus, model.ReviewStatusPassed)
	}
}

func TestSeckillManualReviewRejectAndReleaseStock(t *testing.T) {
	initOrderAuthForTest(t)
	repo := newMemoryOrderRepo()
	product := newFakeProductRPC(2001, "SPU2001", "秒杀商品", 999, 8)
	svcCtx := newTestOrderSvc(t, repo, product)
	ctx := context.Background()

	create := NewCreateOrderLogic(ctx, svcCtx)
	cResp, err := create.CreateOrder(&pb.CreateOrderReq{
		UserId:      3001,
		ProductId:   2001,
		OrderSource: int32(model.OrderSourceSeckill),
	})
	if err != nil {
		t.Fatalf("create order failed: %v", err)
	}
	if stock := product.stockByID[2001]; stock != 7 {
		t.Fatalf("reserve stock mismatch: got=%d want=7", stock)
	}

	pay := NewConfirmPaymentAndInfoLogic(ctx, svcCtx)
	pResp, err := pay.ConfirmPaymentAndInfo(&pb.ConfirmPaymentAndInfoReq{
		UserId:          3001,
		OrderId:         cResp.Order.OrderId,
		PayChannel:      "mock",
		PayReference:    "pay-002",
		ReceiverName:    "李四",
		ReceiverPhone:   "13900139000",
		ReceiverAddress: "深圳市南山区",
		BuyerRemark:     "",
	})
	if err != nil {
		t.Fatalf("confirm payment failed: %v", err)
	}
	if pResp.Order.ReviewStatus != int32(model.ReviewStatusManualPending) {
		t.Fatalf("review status mismatch: got=%d want=%d", pResp.Order.ReviewStatus, model.ReviewStatusManualPending)
	}

	review := NewReviewOrderAdminLogic(ctx, svcCtx)
	rResp, err := review.ReviewOrderAdmin(&pb.ReviewOrderAdminReq{
		OrderId:  cResp.Order.OrderId,
		AdminId:  9001,
		Approved: false,
		Reason:   "风控拒绝",
	})
	if err != nil {
		t.Fatalf("review reject failed: %v", err)
	}
	if rResp.Order.OrderStatus != int32(model.OrderStatusClosed) {
		t.Fatalf("closed status mismatch: got=%d", rResp.Order.OrderStatus)
	}
	if rResp.Order.RefundStatus != int32(model.RefundStatusRefunding) {
		t.Fatalf("refund status mismatch: got=%d", rResp.Order.RefundStatus)
	}
	if stock := product.stockByID[2001]; stock != 7 {
		t.Fatalf("seckill order should not release product stock: got=%d want=7", stock)
	}
}

func TestCancelUnpaidOrderReleaseStock(t *testing.T) {
	initOrderAuthForTest(t)
	repo := newMemoryOrderRepo()
	product := newFakeProductRPC(3001, "SPU3001", "取消商品", 599, 5)
	svcCtx := newTestOrderSvc(t, repo, product)
	ctx := context.Background()

	create := NewCreateOrderLogic(ctx, svcCtx)
	cResp, err := create.CreateOrder(&pb.CreateOrderReq{
		UserId:      4001,
		ProductId:   3001,
		OrderSource: int32(model.OrderSourceNormal),
	})
	if err != nil {
		t.Fatalf("create order failed: %v", err)
	}
	if stock := product.stockByID[3001]; stock != 4 {
		t.Fatalf("reserve stock mismatch: got=%d want=4", stock)
	}

	cancel := NewCancelOrderLogic(ctx, svcCtx)
	xResp, err := cancel.CancelOrder(&pb.CancelOrderReq{
		UserId:  4001,
		OrderId: cResp.Order.OrderId,
		Reason:  "不想买了",
	})
	if err != nil {
		t.Fatalf("cancel order failed: %v", err)
	}
	if xResp.Order.OrderStatus != int32(model.OrderStatusClosed) {
		t.Fatalf("cancel status mismatch: got=%d", xResp.Order.OrderStatus)
	}
	if stock := product.stockByID[3001]; stock != 5 {
		t.Fatalf("release stock mismatch: got=%d want=5", stock)
	}
}

func TestTimeoutJobFlow(t *testing.T) {
	initOrderAuthForTest(t)
	repo := newMemoryOrderRepo()
	product := newFakeProductRPC(5001, "SPU5001", "超时商品", 699, 0)
	svcCtx := newTestOrderSvc(t, repo, product)
	now := time.Now()

	repo.seed(&model.Order{
		ID:             1,
		OrderNo:        "ORD1",
		UserID:         7001,
		ProductID:      5001,
		Quantity:       1,
		OrderStatus:    model.OrderStatusPendingPay,
		PaymentStatus:  model.PaymentStatusUnpaid,
		ReviewStatus:   model.ReviewStatusNone,
		ShippingStatus: model.ShippingStatusNotShipped,
		RefundStatus:   model.RefundStatusNone,
		CreatedAt:      now.Add(-20 * time.Minute),
		UpdatedAt:      now.Add(-20 * time.Minute),
	})
	repo.seed(&model.Order{
		ID:             2,
		OrderNo:        "ORD2",
		UserID:         7002,
		ProductID:      5001,
		Quantity:       1,
		OrderStatus:    model.OrderStatusPendingReview,
		PaymentStatus:  model.PaymentStatusPaid,
		ReviewStatus:   model.ReviewStatusManualPending,
		ShippingStatus: model.ShippingStatusNotShipped,
		RefundStatus:   model.RefundStatusNone,
		ReviewDueAt:    ptrTime(now.Add(-2 * time.Minute)),
		CreatedAt:      now.Add(-40 * time.Minute),
		UpdatedAt:      now.Add(-2 * time.Minute),
	})
	repo.seed(&model.Order{
		ID:             3,
		OrderNo:        "ORD3",
		UserID:         7003,
		ProductID:      5001,
		Quantity:       1,
		OrderStatus:    model.OrderStatusClosed,
		PaymentStatus:  model.PaymentStatusPaid,
		ReviewStatus:   model.ReviewStatusRejected,
		ShippingStatus: model.ShippingStatusNotShipped,
		RefundStatus:   model.RefundStatusRefunding,
		RefundDueAt:    ptrTime(now.Add(-2 * time.Minute)),
		CloseReason:    model.CloseReasonAuditReject,
		CreatedAt:      now.Add(-60 * time.Minute),
		UpdatedAt:      now.Add(-2 * time.Minute),
	})
	repo.seed(&model.Order{
		ID:             4,
		OrderNo:        "ORD4",
		UserID:         7004,
		ProductID:      5001,
		Quantity:       1,
		OrderStatus:    model.OrderStatusShipped,
		PaymentStatus:  model.PaymentStatusPaid,
		ReviewStatus:   model.ReviewStatusPassed,
		ShippingStatus: model.ShippingStatusShipped,
		RefundStatus:   model.RefundStatusNone,
		ShippedAt:      ptrTime(now.Add(-8 * 24 * time.Hour)),
		CreatedAt:      now.Add(-9 * 24 * time.Hour),
		UpdatedAt:      now.Add(-8 * 24 * time.Hour),
	})

	job := NewTimeoutJobLogic(context.Background(), svcCtx)
	job.RunOnce()

	o1, _ := repo.FindByID(context.Background(), 1)
	if o1.OrderStatus != model.OrderStatusClosed || o1.CloseReason != model.CloseReasonPayTimeout {
		t.Fatalf("pay-timeout close mismatch: status=%d reason=%s", o1.OrderStatus, o1.CloseReason)
	}
	o2, _ := repo.FindByID(context.Background(), 2)
	if o2.OrderStatus != model.OrderStatusClosed || o2.ReviewStatus != model.ReviewStatusTimeoutReject {
		t.Fatalf("review-timeout close mismatch: status=%d review=%d", o2.OrderStatus, o2.ReviewStatus)
	}
	o3, _ := repo.FindByID(context.Background(), 3)
	if o3.RefundStatus != model.RefundStatusRefunded || o3.CloseReason != model.CloseReasonAuditReject {
		t.Fatalf("refund complete mismatch: refund=%d reason=%s", o3.RefundStatus, o3.CloseReason)
	}
	o4, _ := repo.FindByID(context.Background(), 4)
	if o4.OrderStatus != model.OrderStatusClosed || o4.CloseReason != model.CloseReasonAutoCompleted {
		t.Fatalf("auto-receive close mismatch: status=%d reason=%s", o4.OrderStatus, o4.CloseReason)
	}
	if got := product.stockByID[5001]; got != 3 {
		t.Fatalf("timeout release stock mismatch: got=%d want=3", got)
	}
}

func TestCreateOrderFromSeckillIdempotentReplay(t *testing.T) {
	initOrderAuthForTest(t)
	repo := newMemoryOrderRepo()
	svcCtx := newTestOrderSvc(t, repo, nil)
	ctx := context.Background()

	logic := NewCreateOrderFromSeckillLogic(ctx, svcCtx)
	req := &pb.CreateOrderFromSeckillReq{
		UserId:            8001,
		ActivityId:        901,
		ActivityItemId:    902,
		ProductId:         3001,
		Quantity:          2,
		SeckillPriceCent:  199,
		SnapshotName:      "秒杀可乐",
		SnapshotMainImage: "https://img/seckill-cola.png",
		SkuCode:           "SPU3001",
		IdempotencyKey:    "idem-replay-001",
	}

	first, err := logic.CreateOrderFromSeckill(req)
	if err != nil {
		t.Fatalf("first create from seckill failed: %v", err)
	}
	second, err := logic.CreateOrderFromSeckill(req)
	if err != nil {
		t.Fatalf("second create from seckill failed: %v", err)
	}
	if first.GetOrder().GetOrderId() != second.GetOrder().GetOrderId() {
		t.Fatalf("idempotent order id mismatch: first=%d second=%d", first.GetOrder().GetOrderId(), second.GetOrder().GetOrderId())
	}
	if first.GetOrder().GetOrderNo() != second.GetOrder().GetOrderNo() {
		t.Fatalf("idempotent order no mismatch: first=%s second=%s", first.GetOrder().GetOrderNo(), second.GetOrder().GetOrderNo())
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
