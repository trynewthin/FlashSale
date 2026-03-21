package eventx

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// SeckillStockCompensateReasonOrderCreateFailed 表示异步建单最终失败。
	SeckillStockCompensateReasonOrderCreateFailed = "order_create_failed"
)

// SeckillPurchaseCreateEvent 定义秒杀预扣成功后发送给订单侧的建单消息。
type SeckillPurchaseCreateEvent struct {
	OrderNo           string `json:"order_no"`
	UserID            int64  `json:"user_id"`
	ActivityID        int64  `json:"activity_id"`
	ActivityItemID    int64  `json:"activity_item_id"`
	ProductID         int64  `json:"product_id"`
	Quantity          int64  `json:"quantity"`
	SeckillPriceCent  int64  `json:"seckill_price_cent"`
	SnapshotName      string `json:"snapshot_name"`
	SnapshotMainImage string `json:"snapshot_main_image"`
	SKUCode           string `json:"sku_code"`
	IdempotencyKey    string `json:"idempotency_key"`
	OccurredAtUnix    int64  `json:"occurred_at_unix"`
}

// SeckillStockCompensateEvent 定义订单侧请求秒杀侧回滚 Redis 预扣的补偿消息。
type SeckillStockCompensateEvent struct {
	OrderNo        string `json:"order_no"`
	UserID         int64  `json:"user_id"`
	ActivityID     int64  `json:"activity_id"`
	ActivityItemID int64  `json:"activity_item_id"`
	Quantity       int64  `json:"quantity"`
	IdempotencyKey string `json:"idempotency_key"`
	Reason         string `json:"reason"`
	OccurredAtUnix int64  `json:"occurred_at_unix"`
}

// BuildSeckillOrderNo 基于用户、活动和幂等键生成稳定订单号。
func BuildSeckillOrderNo(userID, activityID, activityItemID int64, idempotencyKey string) string {
	raw := fmt.Sprintf("%d:%d:%d:%s", userID, activityID, activityItemID, strings.TrimSpace(idempotencyKey))
	sum := sha1.Sum([]byte(raw))
	hexDigest := strings.ToUpper(hex.EncodeToString(sum[:]))
	return "SCK" + hexDigest[:29]
}
