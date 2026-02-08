// logic 包包含相关应用代码。
package logic

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/pkg/base/errorx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	// cacheReserveSuccessCode 表示缓存层扣减成功。
	cacheReserveSuccessCode = int64(1)
	// cacheReserveDuplicateCode 表示请求幂等冲突。
	cacheReserveDuplicateCode = int64(2)
	// cacheReserveOutOfStockCode 表示缓存库存不足。
	cacheReserveOutOfStockCode = int64(3)
	// cacheReserveLimitExceeded 表示限购超限。
	cacheReserveLimitExceeded = int64(4)
	// cacheReserveInvalidRequest 表示参数非法。
	cacheReserveInvalidRequest = int64(5)
	// cacheReserveStockKeyNotReady 表示缓存库存键未就绪。
	cacheReserveStockKeyNotReady = int64(6)
)

// reservePurchaseScript 在 Redis 中原子校验并扣减活动库存。
var reservePurchaseScript = redis.NewScript(`
local qty = tonumber(ARGV[1]) or 0
local maxQty = tonumber(ARGV[2]) or 0
local mode = tonumber(ARGV[3]) or 0
local userLimitQty = tonumber(ARGV[4]) or 0
local windowSec = tonumber(ARGV[5]) or 0
local nowTs = tonumber(ARGV[6]) or 0
local idemTTL = tonumber(ARGV[7]) or 60
local releaseTTL = tonumber(ARGV[8]) or 604800
local reqToken = ARGV[9]

if qty <= 0 then
  return {5, 0, 0}
end
if maxQty > 0 and qty > maxQty then
  return {4, 0, 0}
end
if redis.call("EXISTS", KEYS[4]) == 1 then
  return {2, 0, 0}
end
local stock = tonumber(redis.call("GET", KEYS[1]) or "-1")
if stock < 0 then
  return {6, 0, 0}
end
if stock < qty then
  return {3, stock, 0}
end
if userLimitQty > 0 then
  if mode == 0 then
    local bought = tonumber(redis.call("GET", KEYS[2]) or "0")
    if bought + qty > userLimitQty then
      return {4, stock, bought}
    end
    redis.call("SET", KEYS[2], bought + qty)
  elseif mode == 1 then
    local cutoff = nowTs - windowSec
    redis.call("ZREMRANGEBYSCORE", KEYS[3], "-inf", cutoff)
    local completed = tonumber(redis.call("ZCARD", KEYS[3]) or "0")
    if completed + qty > userLimitQty then
      return {4, stock, completed}
    end
  end
end
local remain = redis.call("DECRBY", KEYS[1], qty)
redis.call("SET", KEYS[4], reqToken, "EX", idemTTL)
redis.call("SET", KEYS[5], reqToken, "EX", releaseTTL)
return {1, remain, 0}
`)

// rollbackReserveScript 回滚缓存预扣结果，保障失败补偿幂等。
var rollbackReserveScript = redis.NewScript(`
local qty = tonumber(ARGV[1]) or 0
local mode = tonumber(ARGV[2]) or 0
if qty <= 0 then
  return 0
end
if redis.call("DEL", KEYS[3]) == 0 then
  return 0
end
redis.call("INCRBY", KEYS[1], qty)
if mode == 0 then
  local bought = tonumber(redis.call("GET", KEYS[2]) or "0")
  local nextBought = bought - qty
  if nextBought < 0 then
    nextBought = 0
  end
  redis.call("SET", KEYS[2], nextBought)
end
return 1
`)

// releaseByCloseScript 在订单关闭时回补缓存库存并回退用户累计件数。
var releaseByCloseScript = redis.NewScript(`
local qty = tonumber(ARGV[1]) or 0
local mode = tonumber(ARGV[2]) or 0
local ttl = tonumber(ARGV[3]) or 604800
if qty <= 0 then
  return 0
end
if redis.call("SETNX", KEYS[3], "1") == 0 then
  return 0
end
redis.call("EXPIRE", KEYS[3], ttl)
redis.call("INCRBY", KEYS[1], qty)
if mode == 0 then
  local bought = tonumber(redis.call("GET", KEYS[2]) or "0")
  local nextBought = bought - qty
  if nextBought < 0 then
    nextBought = 0
  end
  redis.call("SET", KEYS[2], nextBought)
end
return 1
`)

// reservePurchaseInCache 执行缓存层抢购预判，返回是否已由缓存处理。
func (l *SeckillLogic) reservePurchaseInCache(ctx context.Context, item *model.ActivityItem, userID, quantity int64, idempotencyKey string, now time.Time) (bool, error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.Redis == nil || item == nil {
		return false, nil
	}
	if item.ID <= 0 || item.ActivityID <= 0 || userID <= 0 || quantity <= 0 {
		return false, nil
	}
	if err := l.warmCacheStock(ctx, item); err != nil {
		l.svcCtx.Logger.Warn("warm cache stock failed", zap.Error(err), zap.Int64("item_id", item.ID))
		return false, nil
	}
	keys := []string{
		cacheKeyItemStock(item.ID),
		cacheKeyActivityUserBought(item.ActivityID, userID),
		cacheKeyItemUserCompleted(item.ID, userID),
		cacheKeyRequest(item.ActivityID, item.ID, userID, idempotencyKey),
		cacheKeyReleaseToken(item.ActivityID, item.ID, userID, idempotencyKey),
	}
	raw, err := reservePurchaseScript.Run(ctx, l.svcCtx.Redis, keys,
		quantity,
		item.MaxQtyPerOrder,
		item.UserLimitMode,
		item.UserLimitQty,
		item.UserLimitWindowSec,
		now.Unix(),
		60,
		7*24*3600,
		strings.TrimSpace(idempotencyKey),
	).Result()
	if err != nil {
		l.svcCtx.Logger.Warn("reserve cache stock failed", zap.Error(err), zap.Int64("item_id", item.ID))
		return false, nil
	}
	result, ok := raw.([]interface{})
	if !ok || len(result) == 0 {
		return false, nil
	}
	code := toInt64(result[0])
	// handled=true 表示缓存层已给出最终判定；
	// handled=false 表示回退到数据库事务链路。
	switch code {
	case cacheReserveSuccessCode:
		return true, nil
	case cacheReserveDuplicateCode:
		// 幂等键已存在时不直接失败，回退 DB 并触发建单幂等恢复链路。
		return false, nil
	case cacheReserveOutOfStockCode:
		return true, errorx.New(errorx.CodeSeckillOutOfStock, "库存不足")
	case cacheReserveLimitExceeded:
		return true, errorx.New(errorx.CodeSeckillLimitExceeded, "限购超限")
	case cacheReserveInvalidRequest:
		return true, errorx.New(errorx.CodeSysBadRequest, "抢购参数非法")
	case cacheReserveStockKeyNotReady:
		return false, nil
	default:
		return false, nil
	}
}

// rollbackReserveInCache 在建单失败场景回滚缓存预扣。
func (l *SeckillLogic) rollbackReserveInCache(ctx context.Context, item *model.ActivityItem, userID, quantity int64, idempotencyKey string) {
	if l == nil || l.svcCtx == nil || l.svcCtx.Redis == nil || item == nil {
		return
	}
	keys := []string{
		cacheKeyItemStock(item.ID),
		cacheKeyActivityUserBought(item.ActivityID, userID),
		cacheKeyRequest(item.ActivityID, item.ID, userID, idempotencyKey),
	}
	_, err := rollbackReserveScript.Run(ctx, l.svcCtx.Redis, keys, quantity, item.UserLimitMode).Result()
	if err != nil {
		l.svcCtx.Logger.Warn("rollback reserve in cache failed", zap.Error(err), zap.Int64("item_id", item.ID))
	}
}

// releaseByCloseInCache 在关闭类事件中回补缓存库存。
func (l *SeckillLogic) releaseByCloseInCache(ctx context.Context, item *model.ActivityItem, userID, quantity int64, releaseID string) {
	if l == nil || l.svcCtx == nil || l.svcCtx.Redis == nil || item == nil {
		return
	}
	keys := []string{
		cacheKeyItemStock(item.ID),
		cacheKeyActivityUserBought(item.ActivityID, userID),
		cacheKeyCloseRelease(item.ActivityID, item.ID, userID, releaseID),
	}
	_, err := releaseByCloseScript.Run(ctx, l.svcCtx.Redis, keys, quantity, item.UserLimitMode, 7*24*3600).Result()
	if err != nil {
		l.svcCtx.Logger.Warn("release by close in cache failed", zap.Error(err), zap.Int64("item_id", item.ID))
	}
}

// warmCacheStock 在首次访问时把数据库库存写入缓存。
func (l *SeckillLogic) warmCacheStock(ctx context.Context, item *model.ActivityItem) error {
	if l == nil || l.svcCtx == nil || l.svcCtx.Redis == nil || item == nil || item.ID <= 0 {
		return nil
	}
	key := cacheKeyItemStock(item.ID)
	ok, err := l.svcCtx.Redis.SetNX(ctx, key, item.AvailableStock, 0).Result()
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return nil
}

// setCacheStock 强制刷新缓存库存值。
func (l *SeckillLogic) setCacheStock(ctx context.Context, itemID, stock int64) {
	if l == nil || l.svcCtx == nil || l.svcCtx.Redis == nil || itemID <= 0 {
		return
	}
	if err := l.svcCtx.Redis.Set(ctx, cacheKeyItemStock(itemID), stock, 0).Err(); err != nil {
		l.svcCtx.Logger.Warn("set cache stock failed", zap.Error(err), zap.Int64("item_id", itemID), zap.Int64("stock", stock))
	}
}

// deleteCacheStock 删除活动商品库存缓存键。
func (l *SeckillLogic) deleteCacheStock(ctx context.Context, itemID int64) {
	if l == nil || l.svcCtx == nil || l.svcCtx.Redis == nil || itemID <= 0 {
		return
	}
	if err := l.svcCtx.Redis.Del(ctx, cacheKeyItemStock(itemID)).Err(); err != nil {
		l.svcCtx.Logger.Warn("delete cache stock failed", zap.Error(err), zap.Int64("item_id", itemID))
	}
}

// recordCompletedWindow 记录窗口限购模式下的“已完成订单”计数样本。
func (l *SeckillLogic) recordCompletedWindow(ctx context.Context, item *model.ActivityItem, userID, quantity, orderID int64, occurredAt time.Time) {
	if l == nil || l.svcCtx == nil || l.svcCtx.Redis == nil || item == nil {
		return
	}
	if item.UserLimitMode != model.UserLimitModeWindowCompleted || item.UserLimitWindowSec <= 0 || item.UserLimitQty <= 0 {
		return
	}
	if userID <= 0 || quantity <= 0 || orderID <= 0 {
		return
	}
	key := cacheKeyItemUserCompleted(item.ID, userID)
	pipe := l.svcCtx.Redis.TxPipeline()
	for i := int64(0); i < quantity; i++ {
		member := fmt.Sprintf("%d:%d:%d", orderID, i, occurredAt.UnixNano())
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(occurredAt.Unix()), Member: member})
	}
	cutoff := occurredAt.Unix() - item.UserLimitWindowSec
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(cutoff, 10))
	pipe.Expire(ctx, key, time.Duration(item.UserLimitWindowSec*3)*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		l.svcCtx.Logger.Warn("record completed window failed", zap.Error(err), zap.Int64("item_id", item.ID), zap.Int64("user_id", userID))
	}
}

// cacheKeyItemStock 返回活动商品库存键。
func cacheKeyItemStock(itemID int64) string {
	return fmt.Sprintf("seckill:item:%d:stock", itemID)
}

// cacheKeyActivityUserBought 返回活动维度用户累计购买键。
func cacheKeyActivityUserBought(activityID, userID int64) string {
	return fmt.Sprintf("seckill:activity:%d:user:%d:bought", activityID, userID)
}

// cacheKeyItemUserCompleted 返回窗口限购完成计数 zset 键。
func cacheKeyItemUserCompleted(itemID, userID int64) string {
	return fmt.Sprintf("seckill:item:%d:user:%d:completed:zset", itemID, userID)
}

// cacheKeyRequest 返回抢购请求幂等键。
func cacheKeyRequest(activityID, itemID, userID int64, idempotencyKey string) string {
	return fmt.Sprintf("seckill:req:%d:%d:%d:%s", activityID, itemID, userID, strings.TrimSpace(idempotencyKey))
}

// cacheKeyReleaseToken 返回释放令牌缓存键。
func cacheKeyReleaseToken(activityID, itemID, userID int64, idempotencyKey string) string {
	return fmt.Sprintf("seckill:req:%d:%d:%d:%s:release-token", activityID, itemID, userID, strings.TrimSpace(idempotencyKey))
}

// cacheKeyCloseRelease 返回关闭回补幂等键。
func cacheKeyCloseRelease(activityID, itemID, userID int64, releaseID string) string {
	return fmt.Sprintf("seckill:close-release:%d:%d:%d:%s", activityID, itemID, userID, strings.TrimSpace(releaseID))
}

// toInt64 把 Lua 结果统一转换为 int64。
func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case int:
		return int64(n)
	case uint64:
		return int64(n)
	case string:
		out, _ := strconv.ParseInt(n, 10, 64)
		return out
	default:
		return 0
	}
}
