// repository 包包含相关应用代码。
package repository

import (
	"context"
	"errors"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
)

var (
	// ErrActivityNotFound 表示活动不存在。
	ErrActivityNotFound = errors.New("activity not found")
	// ErrActivityItemNotFound 表示活动商品不存在。
	ErrActivityItemNotFound = errors.New("activity item not found")
	// ErrActivityStateConflict 表示活动状态不允许当前操作。
	ErrActivityStateConflict = errors.New("activity state conflict")
	// ErrActivityOutOfStock 表示活动商品库存不足。
	ErrActivityOutOfStock = errors.New("activity out of stock")
	// ErrActivityLimitExceeded 表示活动限购超限。
	ErrActivityLimitExceeded = errors.New("activity limit exceeded")
	// ErrIdempotencyConflict 表示幂等冲突。
	ErrIdempotencyConflict = errors.New("idempotency conflict")
)

// ActivityListQuery 表示活动列表查询条件。
type ActivityListQuery struct {
	Page           int64
	PageSize       int64
	Keyword        string
	Status         int8
	IncludeDeleted bool
	PublicOnly     bool
}

// OrderLinkListQuery 表示活动订单查询条件。
type OrderLinkListQuery struct {
	ActivityID    int64
	Page          int64
	PageSize      int64
	OrderStatus   int8
	PaymentStatus int8
	UserID        int64
}

// SeckillRepository 定义秒杀存储访问接口。
type SeckillRepository interface {
	CreateActivity(ctx context.Context, activity *model.Activity) error
	UpdateActivity(ctx context.Context, activity *model.Activity) error
	SoftDeleteActivity(ctx context.Context, activityID, adminID int64, at time.Time) error
	FindActivityByID(ctx context.Context, activityID int64, includeDeleted bool) (*model.Activity, error)
	ListActivities(ctx context.Context, query ActivityListQuery) ([]*model.Activity, int64, error)

	UpsertActivityItem(ctx context.Context, item *model.ActivityItem, isCreate bool) (*model.ActivityItem, error)
	RemoveActivityItem(ctx context.Context, activityID, itemID, adminID int64) error
	ListActivityItems(ctx context.Context, activityID int64, publicOnly bool) ([]*model.ActivityItem, error)
	FindActivityItem(ctx context.Context, activityID, itemID int64) (*model.ActivityItem, error)

	MarkActivityStatus(ctx context.Context, activityID int64, status int8, adminID int64, now time.Time) error
	ResetAvailableStockByReserved(ctx context.Context, activityID int64) error

	ReservePurchase(ctx context.Context, activityID, itemID, userID, quantity int64, idempotencyKey string, now time.Time) (*model.PurchaseReservation, error)
	CompensateReleasePurchase(ctx context.Context, activityID, itemID, quantity int64, idempotencyKey string) error

	CreateOrderLink(ctx context.Context, link *model.OrderLink) error
	SyncOrderLinkState(ctx context.Context, sync *model.OrderStateSync) error
	ReleasePurchaseByOrder(ctx context.Context, activityID, itemID, orderID, quantity int64, idempotencyKey string) error
	ListOrderLinks(ctx context.Context, query OrderLinkListQuery) ([]*model.OrderLink, int64, error)

	RecordTraffic(ctx context.Context, event *model.TrafficEvent) error
	ListTraffic(ctx context.Context, activityID, activityItemID int64, from, to time.Time) ([]*model.TrafficBucket, error)
}
