// logic 包包含相关应用代码。
package logic

import (
	"context"
	"testing"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/svc"
	"flashsale/pkg/base/errorx"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func TestReservePurchaseInCacheAndRollback(t *testing.T) {
	mr := miniredis.RunT(t)
	redisCli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisCli.Close()

	l := &SeckillLogic{
		svcCtx: &svc.ServiceContext{
			Redis:  redisCli,
			Logger: zap.NewNop(),
		},
	}
	item := &model.ActivityItem{
		ID:                 11,
		ActivityID:         101,
		AvailableStock:     5,
		MaxQtyPerOrder:     2,
		UserLimitMode:      model.UserLimitModeNone,
		UserLimitQty:       3,
		UserLimitWindowSec: 0,
	}
	ctx := context.Background()

	// 首次预扣应成功，库存从 5 降为 3。
	handled, err := l.reservePurchaseInCache(ctx, item, 9001, 2, "idem-a", time.Now())
	if err != nil {
		t.Fatalf("reserve in cache failed: %v", err)
	}
	if !handled {
		t.Fatalf("reserve should be handled by cache")
	}

	stock, err := redisCli.Get(ctx, cacheKeyItemStock(item.ID)).Int64()
	if err != nil {
		t.Fatalf("read stock failed: %v", err)
	}
	if stock != 3 {
		t.Fatalf("stock mismatch: got=%d want=3", stock)
	}
	bought, err := redisCli.Get(ctx, cacheKeyActivityUserBought(item.ActivityID, 9001)).Int64()
	if err != nil {
		t.Fatalf("read bought failed: %v", err)
	}
	if bought != 2 {
		t.Fatalf("bought mismatch: got=%d want=2", bought)
	}

	// 相同幂等键重复请求应返回冲突。
	handled, err = l.reservePurchaseInCache(ctx, item, 9001, 1, "idem-a", time.Now())
	if !handled {
		t.Fatalf("duplicate request should still be handled by cache")
	}
	appErr := errorx.FromError(err)
	if appErr == nil || appErr.Code != errorx.CodeSeckillPurchaseConflict {
		t.Fatalf("duplicate request should return conflict, got=%v", err)
	}

	// 回滚后库存与累计购买件数都应恢复。
	l.rollbackReserveInCache(ctx, item, 9001, 2, "idem-a")
	stock, err = redisCli.Get(ctx, cacheKeyItemStock(item.ID)).Int64()
	if err != nil {
		t.Fatalf("read stock after rollback failed: %v", err)
	}
	if stock != 5 {
		t.Fatalf("stock after rollback mismatch: got=%d want=5", stock)
	}
	bought, err = redisCli.Get(ctx, cacheKeyActivityUserBought(item.ActivityID, 9001)).Int64()
	if err != nil {
		t.Fatalf("read bought after rollback failed: %v", err)
	}
	if bought != 0 {
		t.Fatalf("bought after rollback mismatch: got=%d want=0", bought)
	}

	// 第二次回滚应幂等无副作用。
	l.rollbackReserveInCache(ctx, item, 9001, 2, "idem-a")
	stock, _ = redisCli.Get(ctx, cacheKeyItemStock(item.ID)).Int64()
	if stock != 5 {
		t.Fatalf("idempotent rollback mismatch: got=%d want=5", stock)
	}
}

func TestReleaseByCloseInCacheIdempotent(t *testing.T) {
	mr := miniredis.RunT(t)
	redisCli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisCli.Close()

	l := &SeckillLogic{
		svcCtx: &svc.ServiceContext{
			Redis:  redisCli,
			Logger: zap.NewNop(),
		},
	}
	item := &model.ActivityItem{
		ID:             12,
		ActivityID:     102,
		UserLimitMode:  model.UserLimitModeNone,
		UserLimitQty:   10,
		MaxQtyPerOrder: 5,
	}
	ctx := context.Background()
	if err := redisCli.Set(ctx, cacheKeyItemStock(item.ID), 1, 0).Err(); err != nil {
		t.Fatalf("seed stock failed: %v", err)
	}
	if err := redisCli.Set(ctx, cacheKeyActivityUserBought(item.ActivityID, 9002), 2, 0).Err(); err != nil {
		t.Fatalf("seed bought failed: %v", err)
	}

	// 同一 releaseID 连续调用两次，只应生效一次。
	l.releaseByCloseInCache(ctx, item, 9002, 2, "ord-1")
	l.releaseByCloseInCache(ctx, item, 9002, 2, "ord-1")

	stock, err := redisCli.Get(ctx, cacheKeyItemStock(item.ID)).Int64()
	if err != nil {
		t.Fatalf("read stock failed: %v", err)
	}
	if stock != 3 {
		t.Fatalf("stock after idempotent release mismatch: got=%d want=3", stock)
	}
	bought, err := redisCli.Get(ctx, cacheKeyActivityUserBought(item.ActivityID, 9002)).Int64()
	if err != nil {
		t.Fatalf("read bought failed: %v", err)
	}
	if bought != 0 {
		t.Fatalf("bought after idempotent release mismatch: got=%d want=0", bought)
	}
}
