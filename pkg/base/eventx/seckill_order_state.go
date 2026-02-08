// eventx 包提供跨服务事件模型。
package eventx

import "time"

const (
	// SeckillOrderStateEventTypeCreated 表示秒杀订单创建。
	SeckillOrderStateEventTypeCreated = "order_created"
	// SeckillOrderStateEventTypePaid 表示秒杀订单支付确认。
	SeckillOrderStateEventTypePaid = "order_paid_confirmed"
	// SeckillOrderStateEventTypeReviewApproved 表示秒杀订单审核通过。
	SeckillOrderStateEventTypeReviewApproved = "order_review_approved"
	// SeckillOrderStateEventTypeReviewRejected 表示秒杀订单审核拒绝。
	SeckillOrderStateEventTypeReviewRejected = "order_review_rejected"
	// SeckillOrderStateEventTypeShipped 表示秒杀订单已发货。
	SeckillOrderStateEventTypeShipped = "order_shipped"
	// SeckillOrderStateEventTypeReceived 表示秒杀订单已收货。
	SeckillOrderStateEventTypeReceived = "order_received"
	// SeckillOrderStateEventTypeClosed 表示秒杀订单已关闭。
	SeckillOrderStateEventTypeClosed = "order_closed"
	// SeckillOrderStateEventTypeRefundCompleted 表示秒杀订单退款完成。
	SeckillOrderStateEventTypeRefundCompleted = "order_refund_completed"
)

const (
	// CloseReasonCompleted 表示用户确认收货关闭。
	CloseReasonCompleted = "completed"
	// CloseReasonAutoCompleted 表示系统自动收货关闭。
	CloseReasonAutoCompleted = "auto_completed"
	// CloseReasonUserCancel 表示用户取消关闭。
	CloseReasonUserCancel = "user_cancel"
	// CloseReasonPayTimeout 表示支付超时关闭。
	CloseReasonPayTimeout = "pay_timeout"
	// CloseReasonAuditReject 表示审核拒绝关闭。
	CloseReasonAuditReject = "audit_reject"
	// CloseReasonAuditTimeout 表示审核超时关闭。
	CloseReasonAuditTimeout = "audit_timeout"
	// CloseReasonRefundComplete 表示退款完成事件。
	CloseReasonRefundComplete = "refund_completed"
)

// SeckillOrderStateEvent 定义订单服务回流到秒杀服务的状态事件。
type SeckillOrderStateEvent struct {
	EventType            string `json:"event_type"`
	OrderID              int64  `json:"order_id"`
	OrderNo              string `json:"order_no"`
	UserID               int64  `json:"user_id"`
	ProductID            int64  `json:"product_id"`
	ActivityID           int64  `json:"activity_id"`
	ActivityItemID       int64  `json:"activity_item_id"`
	Quantity             int64  `json:"quantity"`
	OrderStatus          int32  `json:"order_status"`
	PaymentStatus        int32  `json:"payment_status"`
	ReviewStatus         int32  `json:"review_status"`
	ShippingStatus       int32  `json:"shipping_status"`
	RefundStatus         int32  `json:"refund_status"`
	CloseReason          string `json:"close_reason"`
	OccurredAtUnixSecond int64  `json:"occurred_at_unix_second"`
}

// OccurredAtTime 返回事件时间；无效值回退为当前时间。
func (e SeckillOrderStateEvent) OccurredAtTime() time.Time {
	if e.OccurredAtUnixSecond <= 0 {
		return time.Now()
	}
	return time.Unix(e.OccurredAtUnixSecond, 0)
}
