// svc 包包含相关应用代码。
package svc

import (
	"testing"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
)

func TestActivityItemCache_SetGetClone(t *testing.T) {
	t.Parallel()
	s := &ServiceContext{
		ActivityItemCacheTTL: 2 * time.Second,
		Perf:                 newPerfStats(),
	}
	src := &model.ActivityItem{
		ID:               1001,
		ActivityID:       2002,
		ProductID:        3003,
		MaxQtyPerOrder:   5,
		SeckillPriceCent: 9900,
	}
	s.SetCachedActivityItem(src)

	got, ok := s.GetCachedActivityItem(src.ActivityID, src.ID)
	if !ok || got == nil {
		t.Fatalf("expected cache hit")
	}
	if got.ProductID != src.ProductID || got.SeckillPriceCent != src.SeckillPriceCent {
		t.Fatalf("unexpected cached value: %+v", got)
	}

	// 修改返回对象不应污染缓存副本。
	got.ProductID = 9999
	got2, ok := s.GetCachedActivityItem(src.ActivityID, src.ID)
	if !ok || got2 == nil {
		t.Fatalf("expected second cache hit")
	}
	if got2.ProductID != src.ProductID {
		t.Fatalf("cache should return cloned value, got product_id=%d", got2.ProductID)
	}
}

func TestActivityItemCache_Expire(t *testing.T) {
	t.Parallel()
	s := &ServiceContext{
		ActivityItemCacheTTL: 30 * time.Millisecond,
		Perf:                 newPerfStats(),
	}
	s.SetCachedActivityItem(&model.ActivityItem{
		ID:         11,
		ActivityID: 22,
	})
	if _, ok := s.GetCachedActivityItem(22, 11); !ok {
		t.Fatalf("expected cache hit before expire")
	}
	time.Sleep(80 * time.Millisecond)
	if _, ok := s.GetCachedActivityItem(22, 11); ok {
		t.Fatalf("expected cache miss after expire")
	}
}
