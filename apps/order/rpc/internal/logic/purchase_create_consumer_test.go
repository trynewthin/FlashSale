package logic

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"flashsale/apps/order/rpc/internal/svc"
	baseconfig "flashsale/pkg/base/config"
	"flashsale/pkg/base/eventx"
	"flashsale/pkg/base/kafkax"
	"go.uber.org/zap"
)

type orderProducerMock struct {
	published []orderPublishedMessage
	publishFn func(context.Context, string, []byte, []byte, map[string]string) error
}

type orderPublishedMessage struct {
	topic   string
	key     []byte
	value   []byte
	headers map[string]string
}

func (m *orderProducerMock) Publish(ctx context.Context, topic string, key, value []byte, headers map[string]string) error {
	msg := orderPublishedMessage{
		topic: topic,
		key:   append([]byte(nil), key...),
		value: append([]byte(nil), value...),
	}
	if headers != nil {
		msg.headers = make(map[string]string, len(headers))
		for k, v := range headers {
			msg.headers[k] = v
		}
	}
	m.published = append(m.published, msg)
	if m.publishFn != nil {
		return m.publishFn(ctx, topic, key, value, headers)
	}
	return nil
}

func newTestOrderConsumerSvc(t *testing.T, repo *memoryOrderRepo) *svc.ServiceContext {
	t.Helper()
	svcCtx := newTestOrderSvc(t, repo, nil)
	appCfg := &baseconfig.AppConfig{}
	appCfg.Kafka.Topics.SeckillPurchaseCreateDLQ = "seckill.purchase.create.dlq"
	appCfg.Kafka.Topics.StockCompensate = "stock.compensate"
	svcCtx.AppConfig = appCfg
	svcCtx.Producer = &orderProducerMock{}
	svcCtx.Logger = zap.NewNop()
	return svcCtx
}

func TestConsumeSeckillPurchaseCreateMessage_SuccessAndIdempotent(t *testing.T) {
	initOrderAuthForTest(t)
	repo := newMemoryOrderRepo()
	svcCtx := newTestOrderConsumerSvc(t, repo)
	orderNo := eventx.BuildSeckillOrderNo(8001, 901, 902, "idem-consume-success")
	payload, err := json.Marshal(eventx.SeckillPurchaseCreateEvent{
		OrderNo:           orderNo,
		UserID:            8001,
		ActivityID:        901,
		ActivityItemID:    902,
		ProductID:         3001,
		Quantity:          2,
		SeckillPriceCent:  199,
		SnapshotName:      "秒杀可乐",
		SnapshotMainImage: "https://img/seckill-cola.png",
		SKUCode:           "SPU3001",
		IdempotencyKey:    "idem-consume-success",
		OccurredAtUnix:    time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	msg := kafkax.Message{Topic: "seckill.purchase.create", Key: []byte(orderNo), Value: payload}
	if err := consumeSeckillPurchaseCreateMessage(context.Background(), svcCtx, msg); err != nil {
		t.Fatalf("first consume failed: %v", err)
	}
	if err := consumeSeckillPurchaseCreateMessage(context.Background(), svcCtx, msg); err != nil {
		t.Fatalf("second consume failed: %v", err)
	}
	if len(repo.byID) != 1 {
		t.Fatalf("duplicate consume should keep one order, got=%d", len(repo.byID))
	}
	order, err := repo.FindByOrderNo(context.Background(), orderNo)
	if err != nil {
		t.Fatalf("find order by orderNo failed: %v", err)
	}
	if order.OrderNo != orderNo || order.ProductID != 3001 || order.Quantity != 2 {
		t.Fatalf("created order mismatch: %+v", order)
	}
	producer := svcCtx.Producer.(*orderProducerMock)
	if len(producer.published) != 0 {
		t.Fatalf("success path should not publish dlq/compensate, got=%d", len(producer.published))
	}
}

func TestConsumeSeckillPurchaseCreateMessage_DecodeFailureReturnsErrorAfterDLQ(t *testing.T) {
	initOrderAuthForTest(t)
	repo := newMemoryOrderRepo()
	svcCtx := newTestOrderConsumerSvc(t, repo)
	producer := svcCtx.Producer.(*orderProducerMock)

	msg := kafkax.Message{
		Topic: "seckill.purchase.create",
		Key:   []byte("bad-message"),
		Value: []byte("{bad-json"),
	}
	if err := consumeSeckillPurchaseCreateMessage(context.Background(), svcCtx, msg); err == nil {
		t.Fatalf("decode failure should return error to avoid committing offset")
	}
	if len(repo.byID) != 0 {
		t.Fatalf("invalid payload should not create order, got=%d", len(repo.byID))
	}
	if len(producer.published) != 1 {
		t.Fatalf("invalid payload should publish one dlq message, got=%d", len(producer.published))
	}
	dlq := producer.published[0]
	if dlq.topic != "seckill.purchase.create.dlq" {
		t.Fatalf("dlq topic mismatch: got=%s", dlq.topic)
	}
	if dlq.headers["reason"] != "decode_failed" {
		t.Fatalf("dlq reason mismatch: got=%s", dlq.headers["reason"])
	}
}

func TestConsumeSeckillPurchaseCreateMessage_InvalidEventPublishesDLQAndCompensate(t *testing.T) {
	initOrderAuthForTest(t)
	repo := newMemoryOrderRepo()
	svcCtx := newTestOrderConsumerSvc(t, repo)
	producer := svcCtx.Producer.(*orderProducerMock)

	orderNo := eventx.BuildSeckillOrderNo(8101, 911, 912, "idem-invalid-event")
	payload, err := json.Marshal(eventx.SeckillPurchaseCreateEvent{
		OrderNo:           orderNo,
		UserID:            8101,
		ActivityID:        911,
		ActivityItemID:    912,
		ProductID:         3003,
		Quantity:          1,
		SeckillPriceCent:  299,
		SnapshotName:      "",
		SnapshotMainImage: "https://img/invalid.png",
		SKUCode:           "SPU3003",
		IdempotencyKey:    "idem-invalid-event",
		OccurredAtUnix:    time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	msg := kafkax.Message{Topic: "seckill.purchase.create", Key: []byte(orderNo), Value: payload}
	if err := consumeSeckillPurchaseCreateMessage(context.Background(), svcCtx, msg); err != nil {
		t.Fatalf("invalid event should end with dlq+compensate commit, got err=%v", err)
	}
	if len(repo.byID) != 0 {
		t.Fatalf("invalid event should not create order, got=%d", len(repo.byID))
	}
	if len(producer.published) != 2 {
		t.Fatalf("invalid event should publish dlq and compensate, got=%d", len(producer.published))
	}
	if producer.published[0].topic != "seckill.purchase.create.dlq" {
		t.Fatalf("first publish should be dlq, got=%s", producer.published[0].topic)
	}
	if producer.published[0].headers["reason"] != "invalid_payload" {
		t.Fatalf("dlq reason mismatch: got=%s", producer.published[0].headers["reason"])
	}
	if producer.published[1].topic != "stock.compensate" {
		t.Fatalf("second publish should be stock compensate, got=%s", producer.published[1].topic)
	}
}

func TestConsumeSeckillPurchaseCreateMessage_RetryExhaustedPublishesDLQAndCompensate(t *testing.T) {
	initOrderAuthForTest(t)
	repo := newMemoryOrderRepo()
	svcCtx := newTestOrderConsumerSvc(t, repo)
	producer := svcCtx.Producer.(*orderProducerMock)
	limiter := make(chan struct{}, 1)
	limiter <- struct{}{}
	svcCtx.SeckillCreateLimiter = limiter
	svcCtx.SeckillCreateAcquireTimeoutDur = time.Millisecond

	orderNo := eventx.BuildSeckillOrderNo(8002, 903, 904, "idem-consume-retry")
	payload, err := json.Marshal(eventx.SeckillPurchaseCreateEvent{
		OrderNo:           orderNo,
		UserID:            8002,
		ActivityID:        903,
		ActivityItemID:    904,
		ProductID:         3002,
		Quantity:          1,
		SeckillPriceCent:  299,
		SnapshotName:      "秒杀雪碧",
		SnapshotMainImage: "https://img/seckill-sprite.png",
		SKUCode:           "SPU3002",
		IdempotencyKey:    "idem-consume-retry",
		OccurredAtUnix:    time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	msg := kafkax.Message{Topic: "seckill.purchase.create", Key: []byte(orderNo), Value: payload}
	if err := consumeSeckillPurchaseCreateMessage(context.Background(), svcCtx, msg); err != nil {
		t.Fatalf("retry exhaustion should end with dlq+compensate commit, got err=%v", err)
	}
	if len(repo.byID) != 0 {
		t.Fatalf("retry exhaustion should not create order, got=%d", len(repo.byID))
	}
	if len(producer.published) != 2 {
		t.Fatalf("retry exhaustion should publish dlq and compensate, got=%d", len(producer.published))
	}
	if producer.published[0].topic != "seckill.purchase.create.dlq" {
		t.Fatalf("first publish should be dlq, got=%s", producer.published[0].topic)
	}
	if producer.published[0].headers["reason"] != "retries_exhausted" {
		t.Fatalf("dlq reason mismatch: got=%s", producer.published[0].headers["reason"])
	}
	if producer.published[1].topic != "stock.compensate" {
		t.Fatalf("second publish should be stock compensate, got=%s", producer.published[1].topic)
	}
	if got := producer.published[1].headers[kafkax.IdempotencyHeader]; got != "idem-consume-retry" {
		t.Fatalf("compensate header mismatch: got=%s", got)
	}
	var compensate eventx.SeckillStockCompensateEvent
	if err := json.Unmarshal(producer.published[1].value, &compensate); err != nil {
		t.Fatalf("unmarshal compensate payload failed: %v", err)
	}
	if compensate.OrderNo != orderNo || compensate.Reason != eventx.SeckillStockCompensateReasonOrderCreateFailed {
		t.Fatalf("compensate payload mismatch: %+v", compensate)
	}
}

func TestConsumeSeckillPurchaseCreateMessage_DLQPublishFailureReturnsError(t *testing.T) {
	initOrderAuthForTest(t)
	repo := newMemoryOrderRepo()
	svcCtx := newTestOrderConsumerSvc(t, repo)
	svcCtx.Producer = &orderProducerMock{
		publishFn: func(_ context.Context, topic string, _ []byte, _ []byte, _ map[string]string) error {
			if topic == "seckill.purchase.create.dlq" {
				return errors.New("dlq down")
			}
			return nil
		},
	}

	msg := kafkax.Message{
		Topic: "seckill.purchase.create",
		Key:   []byte("bad-message"),
		Value: []byte("{bad-json"),
	}
	if err := consumeSeckillPurchaseCreateMessage(context.Background(), svcCtx, msg); err == nil {
		t.Fatalf("dlq publish failure should bubble up error")
	}
}
