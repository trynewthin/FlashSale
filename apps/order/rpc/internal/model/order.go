// model 包包含相关应用代码。
package model

import "time"

const (
	OrderSourceNormal  int8 = 1
	OrderSourceSeckill int8 = 2
)

const (
	OrderStatusPendingPay    int8 = 10
	OrderStatusPendingReview int8 = 20
	OrderStatusPendingShip   int8 = 30
	OrderStatusShipped       int8 = 40
	OrderStatusClosed        int8 = 90
)

const (
	PaymentStatusUnpaid int8 = 0
	PaymentStatusPaid   int8 = 1
)

const (
	ReviewStatusNone          int8 = 0
	ReviewStatusManualPending int8 = 1
	ReviewStatusPassed        int8 = 2
	ReviewStatusRejected      int8 = 3
	ReviewStatusTimeoutReject int8 = 4
)

const (
	ShippingStatusNotShipped int8 = 0
	ShippingStatusShipped    int8 = 1
	ShippingStatusReceived   int8 = 2
)

const (
	RefundStatusNone      int8 = 0
	RefundStatusRefunding int8 = 1
	RefundStatusRefunded  int8 = 2
)

const (
	ReviewModeAuto   int8 = 1
	ReviewModeManual int8 = 2
)

const (
	StockReleasePending int8 = 0
	StockReleased       int8 = 1
)

const (
	CloseReasonCompleted      = "completed"
	CloseReasonAutoCompleted  = "auto_completed"
	CloseReasonUserCancel     = "user_cancel"
	CloseReasonPayTimeout     = "pay_timeout"
	CloseReasonAuditReject    = "audit_reject"
	CloseReasonAuditTimeout   = "audit_timeout"
	CloseReasonRefundComplete = "refund_completed"
)

// Order 定义订单聚合。
type Order struct {
	ID                    int64
	OrderNo               string
	UserID                int64
	OrderSource           int8
	ProductID             int64
	SeckillActivityID     int64
	SeckillActivityItemID int64
	SkuCode               string
	ProductName           string
	MainImage             string
	UnitPriceCent         int64
	Quantity              int64
	TotalAmountCent       int64

	OrderStatus    int8
	PaymentStatus  int8
	ReviewStatus   int8
	ShippingStatus int8
	RefundStatus   int8
	ReviewMode     int8

	PaidAt       *time.Time
	PayChannel   string
	PayReference string

	ReceiverName    string
	ReceiverPhone   string
	ReceiverAddress string
	BuyerRemark     string

	ReviewDueAt  *time.Time
	ReviewedAt   *time.Time
	ReviewedBy   int64
	ReviewReason string

	ShippedAt  *time.Time
	ShippedBy  int64
	TrackingNo string

	RefundDueAt   *time.Time
	RefundedAt    *time.Time
	StockReleased int8

	ClosedAt    *time.Time
	CloseReason string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// OrderEvent 定义订单状态变更事件。
type OrderEvent struct {
	OrderID      int64
	EventType    string
	OperatorType string
	OperatorID   int64
	Payload      string
	CreatedAt    time.Time
}
