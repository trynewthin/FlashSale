// logic 包包含相关应用代码。
package logic

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/repository"
	"flashsale/apps/seckill/rpc/internal/svc"
	"flashsale/pkg/base/eventx"
	"flashsale/pkg/base/kafkax"
	"go.uber.org/zap"
)

type seckillRepoMock struct {
	item       *model.ActivityItem
	synced     []*model.OrderStateSync
	released   []string
	traffic    []*model.TrafficEvent
	trafficErr error
}

func (m *seckillRepoMock) CreateActivity(context.Context, *model.Activity) error {
	return nil
}

func (m *seckillRepoMock) UpdateActivity(context.Context, *model.Activity) error {
	return nil
}

func (m *seckillRepoMock) SoftDeleteActivity(context.Context, int64, int64, time.Time) error {
	return nil
}

func (m *seckillRepoMock) FindActivityByID(context.Context, int64, bool) (*model.Activity, error) {
	return nil, repository.ErrActivityNotFound
}

func (m *seckillRepoMock) ListActivities(context.Context, repository.ActivityListQuery) ([]*model.Activity, int64, error) {
	return []*model.Activity{}, 0, nil
}

func (m *seckillRepoMock) UpsertActivityItem(context.Context, *model.ActivityItem, bool) (*model.ActivityItem, error) {
	return nil, nil
}

func (m *seckillRepoMock) RemoveActivityItem(context.Context, int64, int64, int64) error {
	return nil
}

func (m *seckillRepoMock) ListActivityItems(context.Context, int64, bool) ([]*model.ActivityItem, error) {
	return []*model.ActivityItem{}, nil
}

func (m *seckillRepoMock) FindActivityItem(context.Context, int64, int64) (*model.ActivityItem, error) {
	if m.item == nil {
		return nil, repository.ErrActivityItemNotFound
	}
	cp := *m.item
	return &cp, nil
}

func (m *seckillRepoMock) MarkActivityStatus(context.Context, int64, int8, int64, time.Time) error {
	return nil
}

func (m *seckillRepoMock) ResetAvailableStockByReserved(context.Context, int64) error {
	return nil
}

func (m *seckillRepoMock) ReservePurchase(context.Context, int64, int64, int64, int64, string, time.Time) (*model.PurchaseReservation, error) {
	return nil, nil
}

func (m *seckillRepoMock) CompensateReleasePurchase(context.Context, int64, int64, int64, string) error {
	return nil
}

func (m *seckillRepoMock) CreateOrderLink(context.Context, *model.OrderLink) error {
	return nil
}

func (m *seckillRepoMock) SyncOrderLinkState(_ context.Context, sync *model.OrderStateSync) error {
	m.synced = append(m.synced, sync)
	return nil
}

func (m *seckillRepoMock) ReleasePurchaseByOrder(_ context.Context, _ int64, _ int64, _ int64, _ int64, idempotencyKey string) error {
	m.released = append(m.released, idempotencyKey)
	return nil
}

func (m *seckillRepoMock) ListOrderLinks(context.Context, repository.OrderLinkListQuery) ([]*model.OrderLink, int64, error) {
	return []*model.OrderLink{}, 0, nil
}

func (m *seckillRepoMock) RecordTraffic(_ context.Context, event *model.TrafficEvent) error {
	if event != nil {
		cp := *event
		m.traffic = append(m.traffic, &cp)
	}
	return m.trafficErr
}

func (m *seckillRepoMock) ListTraffic(context.Context, int64, int64, time.Time, time.Time) ([]*model.TrafficBucket, error) {
	return []*model.TrafficBucket{}, nil
}

func TestConsumeOrderStateMessage_ReleaseAndTraffic(t *testing.T) {
	repoMock := &seckillRepoMock{
		item: &model.ActivityItem{
			ID:                 22,
			ActivityID:         202,
			UserLimitMode:      model.UserLimitModeNone,
			UserLimitQty:       10,
			MaxQtyPerOrder:     5,
			UserLimitWindowSec: 0,
		},
	}
	svcCtx := &svc.ServiceContext{
		SeckillRepo: repoMock,
		Logger:      zap.NewNop(),
	}
	evt := eventx.SeckillOrderStateEvent{
		EventType:            eventx.SeckillOrderStateEventTypeClosed,
		OrderID:              501,
		OrderNo:              "SCKORD501",
		UserID:               7001,
		ActivityID:           202,
		ActivityItemID:       22,
		Quantity:             2,
		OrderStatus:          90,
		PaymentStatus:        1,
		CloseReason:          eventx.CloseReasonAuditReject,
		OccurredAtUnixSecond: time.Now().Unix(),
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal event failed: %v", err)
	}

	// 审核拒绝关闭：应同步订单、回补活动库存，并记录支付成功/订单关闭流量事件。
	err = consumeOrderStateMessage(context.Background(), svcCtx, kafkax.Message{Value: payload})
	if err != nil {
		t.Fatalf("consume order state failed: %v", err)
	}
	if got := len(repoMock.synced); got != 1 {
		t.Fatalf("sync order link count mismatch: got=%d want=1", got)
	}
	if got := len(repoMock.released); got != 1 {
		t.Fatalf("release activity stock count mismatch: got=%d want=1", got)
	}
	var paySuccess, orderClosed int
	for _, event := range repoMock.traffic {
		switch event.EventType {
		case model.TrafficEventPaySuccess:
			paySuccess++
		case model.TrafficEventOrderClosed:
			orderClosed++
		}
	}
	if paySuccess != 1 || orderClosed != 1 {
		t.Fatalf("traffic event mismatch: pay_success=%d order_closed=%d", paySuccess, orderClosed)
	}
}

func TestConsumeOrderStateMessage_NoReleaseOnCompleted(t *testing.T) {
	repoMock := &seckillRepoMock{
		item: &model.ActivityItem{
			ID:                 23,
			ActivityID:         203,
			UserLimitMode:      model.UserLimitModeWindowCompleted,
			UserLimitWindowSec: 3600,
			UserLimitQty:       3,
			MaxQtyPerOrder:     1,
		},
	}
	svcCtx := &svc.ServiceContext{
		SeckillRepo: repoMock,
		Logger:      zap.NewNop(),
	}
	evt := eventx.SeckillOrderStateEvent{
		EventType:            eventx.SeckillOrderStateEventTypeClosed,
		OrderID:              601,
		OrderNo:              "SCKORD601",
		UserID:               7002,
		ActivityID:           203,
		ActivityItemID:       23,
		Quantity:             1,
		OrderStatus:          90,
		PaymentStatus:        1,
		CloseReason:          eventx.CloseReasonCompleted,
		OccurredAtUnixSecond: time.Now().Unix(),
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal event failed: %v", err)
	}

	// 已完成关闭：应同步订单，但不触发库存回补。
	err = consumeOrderStateMessage(context.Background(), svcCtx, kafkax.Message{Value: payload})
	if err != nil {
		t.Fatalf("consume order state failed: %v", err)
	}
	if got := len(repoMock.released); got != 0 {
		t.Fatalf("completed order should not release activity stock, got=%d", got)
	}
	if got := len(repoMock.synced); got != 1 {
		t.Fatalf("sync order link count mismatch: got=%d want=1", got)
	}
}

func TestConsumeOrderStateMessage_SkipWhenActivityItemMissing(t *testing.T) {
	repoMock := &seckillRepoMock{}
	svcCtx := &svc.ServiceContext{
		SeckillRepo: repoMock,
		Logger:      zap.NewNop(),
	}
	evt := eventx.SeckillOrderStateEvent{
		EventType:            eventx.SeckillOrderStateEventTypeClosed,
		OrderID:              701,
		OrderNo:              "SCKORD701",
		UserID:               7003,
		ActivityID:           204,
		ActivityItemID:       24,
		Quantity:             1,
		OrderStatus:          90,
		PaymentStatus:        1,
		CloseReason:          eventx.CloseReasonAuditReject,
		OccurredAtUnixSecond: time.Now().Unix(),
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal event failed: %v", err)
	}

	// 活动商品缺失时应跳过并继续消费，避免同一消息阻塞消费组。
	err = consumeOrderStateMessage(context.Background(), svcCtx, kafkax.Message{Value: payload})
	if err != nil {
		t.Fatalf("consume order state should skip missing item, got err=%v", err)
	}
	if got := len(repoMock.synced); got != 1 {
		t.Fatalf("sync order link count mismatch: got=%d want=1", got)
	}
	if got := len(repoMock.released); got != 0 {
		t.Fatalf("missing item should not release stock, got=%d", got)
	}
}
