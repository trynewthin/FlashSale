// logic 包包含相关应用代码。
package logic

import (
	"context"
	"fmt"
	"testing"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/svc"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// BenchmarkReservePurchaseInCache 基准测试缓存层 Lua 预扣路径性能。
func BenchmarkReservePurchaseInCache(b *testing.B) {
	mr := miniredis.RunT(b)
	redisCli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	b.Cleanup(func() { _ = redisCli.Close() })

	l := &SeckillLogic{
		svcCtx: &svc.ServiceContext{
			Redis:  redisCli,
			Logger: zap.NewNop(),
		},
	}
	item := &model.ActivityItem{
		ID:                 101,
		ActivityID:         10001,
		AvailableStock:     1_000_000_000,
		MaxQtyPerOrder:     3,
		UserLimitMode:      model.UserLimitModeNone,
		UserLimitQty:       0,
		UserLimitWindowSec: 0,
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idem := fmt.Sprintf("bench-reserve-%d", i)
		handled, err := l.reservePurchaseInCache(ctx, item, 9001, 1, idem, time.Now())
		if err != nil {
			b.Fatalf("reservePurchaseInCache failed: %v", err)
		}
		if !handled {
			b.Fatalf("expected cache handled=true")
		}
	}
}

// BenchmarkReleaseByCloseInCache 基准测试关闭回补 Lua 路径性能。
func BenchmarkReleaseByCloseInCache(b *testing.B) {
	mr := miniredis.RunT(b)
	redisCli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	b.Cleanup(func() { _ = redisCli.Close() })

	l := &SeckillLogic{
		svcCtx: &svc.ServiceContext{
			Redis:  redisCli,
			Logger: zap.NewNop(),
		},
	}
	item := &model.ActivityItem{
		ID:             102,
		ActivityID:     10002,
		UserLimitMode:  model.UserLimitModeNone,
		MaxQtyPerOrder: 3,
	}
	ctx := context.Background()
	if err := redisCli.Set(ctx, cacheKeyItemStock(item.ID), 0, 0).Err(); err != nil {
		b.Fatalf("seed stock failed: %v", err)
	}
	if err := redisCli.Set(ctx, cacheKeyActivityUserBought(item.ActivityID, 9002), 1_000_000_000, 0).Err(); err != nil {
		b.Fatalf("seed bought failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		releaseID := fmt.Sprintf("bench-release-%d", i)
		l.releaseByCloseInCache(ctx, item, 9002, 1, releaseID)
	}
}
