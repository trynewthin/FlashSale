// svc 包包含相关应用代码。
package svc

import (
	"fmt"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
)

// cachedActivityItem 保存活动商品短期缓存条目。
type cachedActivityItem struct {
	item     model.ActivityItem
	expireAt time.Time
}

// GetCachedActivityItem 返回活动商品缓存命中结果。
func (s *ServiceContext) GetCachedActivityItem(activityID, itemID int64) (*model.ActivityItem, bool) {
	if s == nil || activityID <= 0 || itemID <= 0 || s.ActivityItemCacheTTL <= 0 {
		if s != nil && s.Perf != nil {
			s.Perf.MarkActivityItemCacheMiss()
		}
		return nil, false
	}
	key := activityItemCacheKey(activityID, itemID)
	raw, ok := s.activityItemCache.Load(key)
	if !ok {
		if s.Perf != nil {
			s.Perf.MarkActivityItemCacheMiss()
		}
		return nil, false
	}
	entry, ok := raw.(*cachedActivityItem)
	if !ok || entry == nil {
		s.activityItemCache.Delete(key)
		if s.Perf != nil {
			s.Perf.MarkActivityItemCacheMiss()
		}
		return nil, false
	}
	if time.Now().After(entry.expireAt) {
		s.activityItemCache.Delete(key)
		if s.Perf != nil {
			s.Perf.MarkActivityItemCacheMiss()
		}
		return nil, false
	}
	cp := entry.item
	if s.Perf != nil {
		s.Perf.MarkActivityItemCacheHit()
	}
	return &cp, true
}

// SetCachedActivityItem 写入活动商品短期缓存。
func (s *ServiceContext) SetCachedActivityItem(item *model.ActivityItem) {
	if s == nil || item == nil || item.ActivityID <= 0 || item.ID <= 0 || s.ActivityItemCacheTTL <= 0 {
		return
	}
	cp := *item
	s.activityItemCache.Store(activityItemCacheKey(item.ActivityID, item.ID), &cachedActivityItem{
		item:     cp,
		expireAt: time.Now().Add(s.ActivityItemCacheTTL),
	})
}

func activityItemCacheKey(activityID, itemID int64) string {
	return fmt.Sprintf("%d:%d", activityID, itemID)
}
