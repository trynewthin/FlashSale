// model 包包含相关应用代码。
package model

import "time"

const (
	// ActivityStatusDraft 表示草稿。
	ActivityStatusDraft int8 = 0
	// ActivityStatusPublished 表示已发布。
	ActivityStatusPublished int8 = 1
	// ActivityStatusOffline 表示已下线。
	ActivityStatusOffline int8 = 2
)

const (
	// ItemStatusDisabled 表示活动商品停用。
	ItemStatusDisabled int8 = 0
	// ItemStatusEnabled 表示活动商品启用。
	ItemStatusEnabled int8 = 1
)

const (
	// UserLimitModeNone 表示不限购。
	UserLimitModeNone int8 = 0
	// UserLimitModeWindowCompleted 表示窗口期按已完成订单限购。
	UserLimitModeWindowCompleted int8 = 1
)

const (
	TrafficEventPV              = "pv"
	TrafficEventClick           = "click"
	TrafficEventPurchaseAttempt = "purchase_attempt"
	TrafficEventPurchaseSuccess = "purchase_success"
	TrafficEventPurchaseFail    = "purchase_fail"
	TrafficEventPaySuccess      = "pay_success"
	TrafficEventOrderClosed     = "order_closed"
)

// Activity 表示秒杀活动聚合。
type Activity struct {
	ID              int64
	Title           string
	Description     string
	StyleConfigJSON string
	StartAt         time.Time
	EndAt           time.Time
	Status          int8
	CreatedBy       int64
	UpdatedBy       int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

// ActivityItem 表示活动商品配置。
type ActivityItem struct {
	ID                 int64
	ActivityID         int64
	ProductID          int64
	SKUCode            string
	SnapshotName       string
	SnapshotMainImage  string
	OriginPriceCent    int64
	SeckillPriceCent   int64
	ReservedStockTotal int64
	AvailableStock     int64
	SoldStock          int64
	UserLimitMode      int8
	UserLimitWindowSec int64
	UserLimitQty       int64
	MaxQtyPerOrder     int64
	Status             int8
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// OrderLink 表示活动订单关联记录。
type OrderLink struct {
	ID             int64
	OrderID        int64
	OrderNo        string
	UserID         int64
	ActivityID     int64
	ActivityItemID int64
	Quantity       int64
	OrderStatus    int8
	PaymentStatus  int8
	CloseReason    string
	LastSyncedAt   time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// OrderStateSync 表示订单状态回流同步模型。
type OrderStateSync struct {
	OrderID        int64
	OrderNo        string
	UserID         int64
	ActivityID     int64
	ActivityItemID int64
	Quantity       int64
	OrderStatus    int8
	PaymentStatus  int8
	CloseReason    string
	LastSyncedAt   time.Time
}

// TrafficBucket 表示分钟级聚合统计。
type TrafficBucket struct {
	BucketMinute    time.Time
	ActivityID      int64
	ActivityItemID  int64
	PV              int64
	UV              int64
	Click           int64
	PurchaseAttempt int64
	PurchaseSuccess int64
	PurchaseFail    int64
	PaySuccess      int64
	OrderClosed     int64
}

// PurchaseReservation 表示抢购预留结果。
type PurchaseReservation struct {
	ActivityID        int64
	ActivityItemID    int64
	ProductID         int64
	Quantity          int64
	RemainStock       int64
	SeckillPriceCent  int64
	SKUCode           string
	SnapshotName      string
	SnapshotMainImage string
}

// TrafficEvent 表示流量事件。
type TrafficEvent struct {
	ActivityID     int64
	ActivityItemID int64
	EventType      string
	UserID         int64
	ClientID       string
	IdempotencyKey string
	OccurredAt     time.Time
}

// PurchaseTask 表示异步建单任务（Redis 预扣成功后入队）。
type PurchaseTask struct {
	ActivityID     int64
	ActivityItemID int64
	UserID         int64
	Quantity       int64
	IdempotencyKey string
	AccessToken    string
	Item           *ActivityItem
	EnqueuedAt     time.Time
}
