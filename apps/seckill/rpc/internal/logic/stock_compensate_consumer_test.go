package logic

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/svc"
	"flashsale/pkg/base/eventx"
	"flashsale/pkg/base/kafkax"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func TestConsumeStockCompensateMessage_Success(t *testing.T) {
	mr := miniredis.RunT(t)
	redisCli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisCli.Close()

	item := &model.ActivityItem{
		ID:            41,
		ActivityID:    501,
		UserLimitMode: model.UserLimitModeNone,
	}
	if err := redisCli.Set(context.Background(), cacheKeyItemStock(item.ID), 2, 0).Err(); err != nil {
		t.Fatalf("seed stock failed: %v", err)
	}
	if err := redisCli.Set(context.Background(), cacheKeyActivityUserBought(item.ActivityID, 9201), 1, 0).Err(); err != nil {
		t.Fatalf("seed bought failed: %v", err)
	}
	if err := redisCli.Set(context.Background(), cacheKeyReleaseToken(item.ActivityID, item.ID, 9201, "idem-compensate-success"), "idem-compensate-success", 0).Err(); err != nil {
		t.Fatalf("seed release token failed: %v", err)
	}

	repoMock := &seckillRepoMock{item: item}
	svcCtx := &svc.ServiceContext{
		SeckillRepo: repoMock,
		Redis:       redisCli,
		Logger:      zap.NewNop(),
		Perf:        &svc.PerfStats{},
	}
	payload, err := json.Marshal(eventx.SeckillStockCompensateEvent{
		OrderNo:        "SCKORDER9201",
		UserID:         9201,
		ActivityID:     item.ActivityID,
		ActivityItemID: item.ID,
		Quantity:       1,
		IdempotencyKey: "idem-compensate-success",
		Reason:         eventx.SeckillStockCompensateReasonOrderCreateFailed,
		OccurredAtUnix: time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	if err := consumeStockCompensateMessage(context.Background(), svcCtx, kafkax.Message{Value: payload}); err != nil {
		t.Fatalf("compensate should succeed, got err=%v", err)
	}
	stock, err := redisCli.Get(context.Background(), cacheKeyItemStock(item.ID)).Int64()
	if err != nil {
		t.Fatalf("read stock failed: %v", err)
	}
	if stock != 3 {
		t.Fatalf("stock mismatch: got=%d want=3", stock)
	}
	bought, err := redisCli.Get(context.Background(), cacheKeyActivityUserBought(item.ActivityID, 9201)).Int64()
	if err != nil {
		t.Fatalf("read bought failed: %v", err)
	}
	if bought != 0 {
		t.Fatalf("bought mismatch: got=%d want=0", bought)
	}
	if mr.Exists(cacheKeyReleaseToken(item.ActivityID, item.ID, 9201, "idem-compensate-success")) {
		t.Fatalf("release token should be deleted after compensation")
	}
	if len(repoMock.traffic) != 1 {
		t.Fatalf("compensate should record one traffic event, got=%d", len(repoMock.traffic))
	}
}

func TestConsumeStockCompensateMessage_FindItemFailureReturnsError(t *testing.T) {
	repoMock := &seckillRepoMock{findItemErr: errors.New("db unavailable")}
	svcCtx := &svc.ServiceContext{
		SeckillRepo: repoMock,
		Logger:      zap.NewNop(),
		Perf:        &svc.PerfStats{},
	}
	payload, err := json.Marshal(eventx.SeckillStockCompensateEvent{
		OrderNo:        "SCKORDER9202",
		UserID:         9202,
		ActivityID:     502,
		ActivityItemID: 42,
		Quantity:       1,
		IdempotencyKey: "idem-compensate-find-item",
		Reason:         eventx.SeckillStockCompensateReasonOrderCreateFailed,
		OccurredAtUnix: time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	if err := consumeStockCompensateMessage(context.Background(), svcCtx, kafkax.Message{Value: payload}); err == nil {
		t.Fatalf("find item failure should return error for retry")
	}
}

func TestConsumeStockCompensateMessage_RedisRollbackFailureReturnsError(t *testing.T) {
	redisCli := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  20 * time.Millisecond,
		ReadTimeout:  20 * time.Millisecond,
		WriteTimeout: 20 * time.Millisecond,
	})
	defer redisCli.Close()

	repoMock := &seckillRepoMock{
		item: &model.ActivityItem{
			ID:            43,
			ActivityID:    503,
			UserLimitMode: model.UserLimitModeNone,
		},
	}
	svcCtx := &svc.ServiceContext{
		SeckillRepo: repoMock,
		Redis:       redisCli,
		Logger:      zap.NewNop(),
		Perf:        &svc.PerfStats{},
	}
	payload, err := json.Marshal(eventx.SeckillStockCompensateEvent{
		OrderNo:        "SCKORDER9203",
		UserID:         9203,
		ActivityID:     503,
		ActivityItemID: 43,
		Quantity:       1,
		IdempotencyKey: "idem-compensate-redis",
		Reason:         eventx.SeckillStockCompensateReasonOrderCreateFailed,
		OccurredAtUnix: time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	if err := consumeStockCompensateMessage(context.Background(), svcCtx, kafkax.Message{Value: payload}); err == nil {
		t.Fatalf("redis rollback failure should return error for retry")
	}
}
