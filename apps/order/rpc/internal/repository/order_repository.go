// repository 包包含相关应用代码。
package repository

import (
	"context"
	"errors"
	"time"

	"flashsale/apps/order/rpc/internal/model"
)

var (
	// ErrOrderNotFound 表示目标订单不存在。
	ErrOrderNotFound = errors.New("order not found")
	// ErrOrderStateConflict 表示订单状态不满足流转前置条件。
	ErrOrderStateConflict = errors.New("order state conflict")
)

// UserListQuery 表示用户订单列表查询条件。
type UserListQuery struct {
	UserID      int64
	Page        int64
	PageSize    int64
	OrderStatus int8
}

// AdminListQuery 表示管理侧订单列表查询条件。
type AdminListQuery struct {
	Page         int64
	PageSize     int64
	OrderStatus  int8
	ReviewStatus int8
	UserID       int64
	OrderNo      string
}

// OrderRepository 定义订单存储访问接口。
type OrderRepository interface {
	Create(ctx context.Context, order *model.Order, event model.OrderEvent) error
	FindByID(ctx context.Context, orderID int64) (*model.Order, error)
	FindByOrderNo(ctx context.Context, orderNo string) (*model.Order, error)
	ListByUser(ctx context.Context, query UserListQuery) ([]*model.Order, int64, error)
	ListAdmin(ctx context.Context, query AdminListQuery) ([]*model.Order, int64, error)

	MarkPaidAutoPass(ctx context.Context, orderID, userID int64, paidAt time.Time, payChannel, payRef string, receiverName, receiverPhone, receiverAddress, buyerRemark string, event model.OrderEvent) error
	MarkPaidManualReview(ctx context.Context, orderID, userID int64, paidAt time.Time, payChannel, payRef string, receiverName, receiverPhone, receiverAddress, buyerRemark string, reviewDueAt time.Time, event model.OrderEvent) error
	CancelUnpaid(ctx context.Context, orderID, userID int64, now time.Time, reason string, event model.OrderEvent) error
	CancelPaidPendingReview(ctx context.Context, orderID, userID int64, now time.Time, reason string, refundDueAt time.Time, event model.OrderEvent) error
	ReviewApprove(ctx context.Context, orderID, adminID int64, now time.Time, reason string, event model.OrderEvent) error
	ReviewReject(ctx context.Context, orderID, adminID int64, now time.Time, reason string, refundDueAt time.Time, event model.OrderEvent) error
	Ship(ctx context.Context, orderID, adminID int64, trackingNo string, now time.Time, event model.OrderEvent) error
	ConfirmReceipt(ctx context.Context, orderID, userID int64, now time.Time, event model.OrderEvent) error
	AutoReceive(ctx context.Context, orderID int64, now time.Time, event model.OrderEvent) error

	ListPayTimeout(ctx context.Context, before time.Time, limit int) ([]*model.Order, error)
	ClosePayTimeout(ctx context.Context, orderID int64, now time.Time, event model.OrderEvent) error
	ListReviewTimeout(ctx context.Context, before time.Time, limit int) ([]*model.Order, error)
	CloseReviewTimeoutRefunding(ctx context.Context, orderID int64, now time.Time, refundDueAt time.Time, event model.OrderEvent) error
	ListStockReleasePending(ctx context.Context, limit int) ([]*model.Order, error)
	MarkStockReleased(ctx context.Context, orderID int64, now time.Time, event model.OrderEvent) error
	ListRefundingDue(ctx context.Context, before time.Time, limit int) ([]*model.Order, error)
	CompleteRefund(ctx context.Context, orderID int64, now time.Time, event model.OrderEvent) error
	ListAutoReceiveDue(ctx context.Context, before time.Time, limit int) ([]*model.Order, error)
}
