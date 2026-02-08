// repository 包包含相关应用代码。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"flashsale/apps/order/rpc/internal/model"
)

const (
	insertOrderSQL = "INSERT INTO orders (id, order_no, user_id, order_source, product_id, seckill_activity_id, seckill_activity_item_id, sku_code, product_name, main_image, unit_price_cent, quantity, total_amount_cent, order_status, payment_status, review_status, shipping_status, refund_status, review_mode) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	insertEventSQL = "INSERT INTO order_events (order_id, event_type, operator_type, operator_id, payload, created_at) VALUES (?, ?, ?, ?, ?, ?)"

	orderSelectColumns = "id, order_no, user_id, order_source, product_id, seckill_activity_id, seckill_activity_item_id, sku_code, product_name, main_image, unit_price_cent, quantity, total_amount_cent, order_status, payment_status, review_status, shipping_status, refund_status, review_mode, paid_at, pay_channel, pay_reference, receiver_name, receiver_phone, receiver_address, buyer_remark, review_due_at, reviewed_at, reviewed_by, review_reason, shipped_at, shipped_by, tracking_no, refund_due_at, refunded_at, stock_released, closed_at, close_reason, created_at, updated_at"
	findByIDSQL        = "SELECT " + orderSelectColumns + " FROM orders WHERE id = ? LIMIT 1"
)

// MySQLOrderRepository 是 OrderRepository 的 MySQL 实现。
type MySQLOrderRepository struct {
	db *sql.DB
}

// NewMySQLOrderRepository 创建 MySQL 订单仓储。
func NewMySQLOrderRepository(db *sql.DB) *MySQLOrderRepository {
	return &MySQLOrderRepository{db: db}
}

// Create 写入订单及初始事件。
func (r *MySQLOrderRepository) Create(ctx context.Context, order *model.Order, event model.OrderEvent) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if order == nil {
		return fmt.Errorf("order is nil")
	}
	return withTx(ctx, r.db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, insertOrderSQL,
			order.ID,
			order.OrderNo,
			order.UserID,
			order.OrderSource,
			order.ProductID,
			sqlNullableInt64(order.SeckillActivityID),
			sqlNullableInt64(order.SeckillActivityItemID),
			order.SkuCode,
			order.ProductName,
			order.MainImage,
			order.UnitPriceCent,
			order.Quantity,
			order.TotalAmountCent,
			order.OrderStatus,
			order.PaymentStatus,
			order.ReviewStatus,
			order.ShippingStatus,
			order.RefundStatus,
			order.ReviewMode,
		)
		if err != nil {
			return err
		}
		return appendEventTx(ctx, tx, event)
	})
}

// FindByID 查询订单详情。
func (r *MySQLOrderRepository) FindByID(ctx context.Context, orderID int64) (*model.Order, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repository db is nil")
	}
	row := r.db.QueryRowContext(ctx, findByIDSQL, orderID)
	m, err := scanOrder(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return m, nil
}

// ListByUser 按用户分页查询订单。
func (r *MySQLOrderRepository) ListByUser(ctx context.Context, query UserListQuery) ([]*model.Order, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("repository db is nil")
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	whereParts := []string{"user_id = ?"}
	args := []any{query.UserID}
	if query.OrderStatus > 0 {
		whereParts = append(whereParts, "order_status = ?")
		args = append(args, query.OrderStatus)
	}
	where := " WHERE " + strings.Join(whereParts, " AND ")
	return r.list(ctx, where, args, query.Page, query.PageSize)
}

// ListAdmin 按管理员条件分页查询订单。
func (r *MySQLOrderRepository) ListAdmin(ctx context.Context, query AdminListQuery) ([]*model.Order, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("repository db is nil")
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	whereParts := make([]string, 0, 4)
	args := make([]any, 0, 4)
	if query.OrderStatus > 0 {
		whereParts = append(whereParts, "order_status = ?")
		args = append(args, query.OrderStatus)
	}
	if query.ReviewStatus > 0 {
		whereParts = append(whereParts, "review_status = ?")
		args = append(args, query.ReviewStatus)
	}
	if query.UserID > 0 {
		whereParts = append(whereParts, "user_id = ?")
		args = append(args, query.UserID)
	}
	if strings.TrimSpace(query.OrderNo) != "" {
		whereParts = append(whereParts, "order_no = ?")
		args = append(args, strings.TrimSpace(query.OrderNo))
	}
	where := ""
	if len(whereParts) > 0 {
		where = " WHERE " + strings.Join(whereParts, " AND ")
	}
	return r.list(ctx, where, args, query.Page, query.PageSize)
}

// MarkPaidAutoPass 将待支付订单更新为已支付且自动审核通过。
func (r *MySQLOrderRepository) MarkPaidAutoPass(ctx context.Context, orderID, userID int64, paidAt time.Time, payChannel, payRef string, receiverName, receiverPhone, receiverAddress, buyerRemark string, event model.OrderEvent) error {
	sqlText := "UPDATE orders SET payment_status = ?, paid_at = ?, pay_channel = ?, pay_reference = ?, receiver_name = ?, receiver_phone = ?, receiver_address = ?, buyer_remark = ?, order_status = ?, review_status = ?, review_mode = ?, review_due_at = NULL WHERE id = ? AND user_id = ? AND order_status = ? AND payment_status = ?"
	return r.execWithEvent(ctx, sqlText, []any{
		model.PaymentStatusPaid,
		paidAt,
		payChannel,
		payRef,
		receiverName,
		receiverPhone,
		receiverAddress,
		buyerRemark,
		model.OrderStatusPendingShip,
		model.ReviewStatusPassed,
		model.ReviewModeAuto,
		orderID,
		userID,
		model.OrderStatusPendingPay,
		model.PaymentStatusUnpaid,
	}, event)
}

// MarkPaidManualReview 将待支付订单更新为已支付且进入人工审核。
func (r *MySQLOrderRepository) MarkPaidManualReview(ctx context.Context, orderID, userID int64, paidAt time.Time, payChannel, payRef string, receiverName, receiverPhone, receiverAddress, buyerRemark string, reviewDueAt time.Time, event model.OrderEvent) error {
	sqlText := "UPDATE orders SET payment_status = ?, paid_at = ?, pay_channel = ?, pay_reference = ?, receiver_name = ?, receiver_phone = ?, receiver_address = ?, buyer_remark = ?, order_status = ?, review_status = ?, review_mode = ?, review_due_at = ? WHERE id = ? AND user_id = ? AND order_status = ? AND payment_status = ?"
	return r.execWithEvent(ctx, sqlText, []any{
		model.PaymentStatusPaid,
		paidAt,
		payChannel,
		payRef,
		receiverName,
		receiverPhone,
		receiverAddress,
		buyerRemark,
		model.OrderStatusPendingReview,
		model.ReviewStatusManualPending,
		model.ReviewModeManual,
		reviewDueAt,
		orderID,
		userID,
		model.OrderStatusPendingPay,
		model.PaymentStatusUnpaid,
	}, event)
}

// CancelUnpaid 取消未支付订单并写入关闭原因。
func (r *MySQLOrderRepository) CancelUnpaid(ctx context.Context, orderID, userID int64, now time.Time, reason string, event model.OrderEvent) error {
	sqlText := "UPDATE orders SET order_status = ?, close_reason = ?, closed_at = ? WHERE id = ? AND user_id = ? AND order_status = ? AND payment_status = ?"
	return r.execWithEvent(ctx, sqlText, []any{
		model.OrderStatusClosed,
		reason,
		now,
		orderID,
		userID,
		model.OrderStatusPendingPay,
		model.PaymentStatusUnpaid,
	}, event)
}

// CancelPaidPendingReview 取消“已支付待审核”订单并标记退款中。
func (r *MySQLOrderRepository) CancelPaidPendingReview(ctx context.Context, orderID, userID int64, now time.Time, reason string, refundDueAt time.Time, event model.OrderEvent) error {
	sqlText := "UPDATE orders SET order_status = ?, close_reason = ?, closed_at = ?, refund_status = ?, refund_due_at = ? WHERE id = ? AND user_id = ? AND order_status = ? AND payment_status = ?"
	return r.execWithEvent(ctx, sqlText, []any{
		model.OrderStatusClosed,
		reason,
		now,
		model.RefundStatusRefunding,
		refundDueAt,
		orderID,
		userID,
		model.OrderStatusPendingReview,
		model.PaymentStatusPaid,
	}, event)
}

// ReviewApprove 将人工审核中的订单审核通过并推进到待发货。
func (r *MySQLOrderRepository) ReviewApprove(ctx context.Context, orderID, adminID int64, now time.Time, reason string, event model.OrderEvent) error {
	sqlText := "UPDATE orders SET order_status = ?, review_status = ?, reviewed_at = ?, reviewed_by = ?, review_reason = ? WHERE id = ? AND order_status = ? AND review_status = ?"
	return r.execWithEvent(ctx, sqlText, []any{
		model.OrderStatusPendingShip,
		model.ReviewStatusPassed,
		now,
		adminID,
		reason,
		orderID,
		model.OrderStatusPendingReview,
		model.ReviewStatusManualPending,
	}, event)
}

// ReviewReject 将人工审核中的订单驳回并关闭为退款中。
func (r *MySQLOrderRepository) ReviewReject(ctx context.Context, orderID, adminID int64, now time.Time, reason string, refundDueAt time.Time, event model.OrderEvent) error {
	sqlText := "UPDATE orders SET order_status = ?, review_status = ?, reviewed_at = ?, reviewed_by = ?, review_reason = ?, refund_status = ?, refund_due_at = ?, close_reason = ?, closed_at = ? WHERE id = ? AND order_status = ? AND review_status = ?"
	return r.execWithEvent(ctx, sqlText, []any{
		model.OrderStatusClosed,
		model.ReviewStatusRejected,
		now,
		adminID,
		reason,
		model.RefundStatusRefunding,
		refundDueAt,
		model.CloseReasonAuditReject,
		now,
		orderID,
		model.OrderStatusPendingReview,
		model.ReviewStatusManualPending,
	}, event)
}

// Ship 将待发货订单推进为已发货状态。
func (r *MySQLOrderRepository) Ship(ctx context.Context, orderID, adminID int64, trackingNo string, now time.Time, event model.OrderEvent) error {
	sqlText := "UPDATE orders SET order_status = ?, shipping_status = ?, shipped_at = ?, shipped_by = ?, tracking_no = ? WHERE id = ? AND order_status = ?"
	return r.execWithEvent(ctx, sqlText, []any{
		model.OrderStatusShipped,
		model.ShippingStatusShipped,
		now,
		adminID,
		trackingNo,
		orderID,
		model.OrderStatusPendingShip,
	}, event)
}

// ConfirmReceipt 将已发货订单确认收货并关闭。
func (r *MySQLOrderRepository) ConfirmReceipt(ctx context.Context, orderID, userID int64, now time.Time, event model.OrderEvent) error {
	sqlText := "UPDATE orders SET order_status = ?, shipping_status = ?, close_reason = ?, closed_at = ? WHERE id = ? AND user_id = ? AND order_status = ? AND shipping_status = ?"
	return r.execWithEvent(ctx, sqlText, []any{
		model.OrderStatusClosed,
		model.ShippingStatusReceived,
		model.CloseReasonCompleted,
		now,
		orderID,
		userID,
		model.OrderStatusShipped,
		model.ShippingStatusShipped,
	}, event)
}

// AutoReceive 将超时未确认收货订单自动收货并关闭。
func (r *MySQLOrderRepository) AutoReceive(ctx context.Context, orderID int64, now time.Time, event model.OrderEvent) error {
	sqlText := "UPDATE orders SET order_status = ?, shipping_status = ?, close_reason = ?, closed_at = ? WHERE id = ? AND order_status = ? AND shipping_status = ?"
	return r.execWithEvent(ctx, sqlText, []any{
		model.OrderStatusClosed,
		model.ShippingStatusReceived,
		model.CloseReasonAutoCompleted,
		now,
		orderID,
		model.OrderStatusShipped,
		model.ShippingStatusShipped,
	}, event)
}

// ListPayTimeout 查询支付超时待关闭订单。
func (r *MySQLOrderRepository) ListPayTimeout(ctx context.Context, before time.Time, limit int) ([]*model.Order, error) {
	return r.listByCondition(ctx, "order_status = ? AND payment_status = ? AND created_at <= ?", []any{model.OrderStatusPendingPay, model.PaymentStatusUnpaid, before}, limit)
}

// ClosePayTimeout 关闭支付超时订单。
func (r *MySQLOrderRepository) ClosePayTimeout(ctx context.Context, orderID int64, now time.Time, event model.OrderEvent) error {
	sqlText := "UPDATE orders SET order_status = ?, close_reason = ?, closed_at = ? WHERE id = ? AND order_status = ? AND payment_status = ?"
	return r.execWithEvent(ctx, sqlText, []any{
		model.OrderStatusClosed,
		model.CloseReasonPayTimeout,
		now,
		orderID,
		model.OrderStatusPendingPay,
		model.PaymentStatusUnpaid,
	}, event)
}

// ListReviewTimeout 查询审核超时待处理订单。
func (r *MySQLOrderRepository) ListReviewTimeout(ctx context.Context, before time.Time, limit int) ([]*model.Order, error) {
	return r.listByCondition(ctx, "order_status = ? AND review_status = ? AND review_due_at IS NOT NULL AND review_due_at <= ?", []any{model.OrderStatusPendingReview, model.ReviewStatusManualPending, before}, limit)
}

// CloseReviewTimeoutRefunding 将审核超时订单关闭并标记退款中。
func (r *MySQLOrderRepository) CloseReviewTimeoutRefunding(ctx context.Context, orderID int64, now time.Time, refundDueAt time.Time, event model.OrderEvent) error {
	sqlText := "UPDATE orders SET order_status = ?, review_status = ?, refund_status = ?, refund_due_at = ?, close_reason = ?, closed_at = ? WHERE id = ? AND order_status = ? AND review_status = ?"
	return r.execWithEvent(ctx, sqlText, []any{
		model.OrderStatusClosed,
		model.ReviewStatusTimeoutReject,
		model.RefundStatusRefunding,
		refundDueAt,
		model.CloseReasonAuditTimeout,
		now,
		orderID,
		model.OrderStatusPendingReview,
		model.ReviewStatusManualPending,
	}, event)
}

// ListStockReleasePending 查询待执行库存回补的已关闭订单。
func (r *MySQLOrderRepository) ListStockReleasePending(ctx context.Context, limit int) ([]*model.Order, error) {
	condition := "order_status = ? AND stock_released = ? AND close_reason IN (?, ?, ?, ?)"
	args := []any{
		model.OrderStatusClosed,
		model.StockReleasePending,
		model.CloseReasonUserCancel,
		model.CloseReasonPayTimeout,
		model.CloseReasonAuditReject,
		model.CloseReasonAuditTimeout,
	}
	return r.listByCondition(ctx, condition, args, limit)
}

// MarkStockReleased 标记订单库存已回补，并写入事件。
func (r *MySQLOrderRepository) MarkStockReleased(ctx context.Context, orderID int64, now time.Time, event model.OrderEvent) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	return withTx(ctx, r.db, func(tx *sql.Tx) error {
		ret, err := tx.ExecContext(ctx, "UPDATE orders SET stock_released = ?, updated_at = ? WHERE id = ? AND stock_released = ?",
			model.StockReleased, now, orderID, model.StockReleasePending)
		if err != nil {
			return err
		}
		rows, err := ret.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			var exists int
			if err := tx.QueryRowContext(ctx, "SELECT 1 FROM orders WHERE id = ? LIMIT 1", orderID).Scan(&exists); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ErrOrderNotFound
				}
				return err
			}
			return nil
		}
		return appendEventTx(ctx, tx, event)
	})
}

// ListRefundingDue 查询到达退款完成时间点的订单。
func (r *MySQLOrderRepository) ListRefundingDue(ctx context.Context, before time.Time, limit int) ([]*model.Order, error) {
	return r.listByCondition(ctx, "order_status = ? AND refund_status = ? AND refund_due_at IS NOT NULL AND refund_due_at <= ?", []any{model.OrderStatusClosed, model.RefundStatusRefunding, before}, limit)
}

// CompleteRefund 将退款中订单推进为退款完成，保留既有关闭原因用于补偿重试审计。
func (r *MySQLOrderRepository) CompleteRefund(ctx context.Context, orderID int64, now time.Time, event model.OrderEvent) error {
	sqlText := "UPDATE orders SET refund_status = ?, refunded_at = ? WHERE id = ? AND order_status = ? AND refund_status = ?"
	return r.execWithEvent(ctx, sqlText, []any{
		model.RefundStatusRefunded,
		now,
		orderID,
		model.OrderStatusClosed,
		model.RefundStatusRefunding,
	}, event)
}

// ListAutoReceiveDue 查询满足自动收货条件的订单。
func (r *MySQLOrderRepository) ListAutoReceiveDue(ctx context.Context, before time.Time, limit int) ([]*model.Order, error) {
	return r.listByCondition(ctx, "order_status = ? AND shipping_status = ? AND shipped_at IS NOT NULL AND shipped_at <= ?", []any{model.OrderStatusShipped, model.ShippingStatusShipped, before}, limit)
}

func (r *MySQLOrderRepository) execWithEvent(ctx context.Context, updateSQL string, args []any, event model.OrderEvent) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	return withTx(ctx, r.db, func(tx *sql.Tx) error {
		ret, err := tx.ExecContext(ctx, updateSQL, args...)
		if err != nil {
			return err
		}
		rows, err := ret.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrOrderStateConflict
		}
		return appendEventTx(ctx, tx, event)
	})
}

func appendEventTx(ctx context.Context, tx *sql.Tx, event model.OrderEvent) error {
	if strings.TrimSpace(event.Payload) == "" {
		event.Payload = "{}"
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	_, err := tx.ExecContext(ctx, insertEventSQL,
		event.OrderID,
		event.EventType,
		event.OperatorType,
		event.OperatorID,
		event.Payload,
		event.CreatedAt,
	)
	return err
}

func withTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *MySQLOrderRepository) list(ctx context.Context, where string, args []any, page, pageSize int64) ([]*model.Order, int64, error) {
	countSQL := "SELECT COUNT(1) FROM orders" + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.Order{}, 0, nil
	}
	offset := (page - 1) * pageSize
	listSQL := "SELECT " + orderSelectColumns + " FROM orders" + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	listArgs := append(append([]any{}, args...), pageSize, offset)
	rows, err := r.db.QueryContext(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]*model.Order, 0, pageSize)
	for rows.Next() {
		m, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *MySQLOrderRepository) listByCondition(ctx context.Context, condition string, args []any, limit int) ([]*model.Order, error) {
	if limit <= 0 {
		limit = 100
	}
	sqlText := "SELECT " + orderSelectColumns + " FROM orders WHERE " + condition + " ORDER BY created_at ASC LIMIT ?"
	queryArgs := append(append([]any{}, args...), limit)
	rows, err := r.db.QueryContext(ctx, sqlText, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*model.Order, 0, limit)
	for rows.Next() {
		m, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func sqlNullableInt64(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}

type scanner interface {
	Scan(dest ...any) error
}

func scanOrder(s scanner) (*model.Order, error) {
	var (
		m                     model.Order
		seckillActivityID     sql.NullInt64
		seckillActivityItemID sql.NullInt64
		paidAt                sql.NullTime
		reviewDueAt           sql.NullTime
		reviewedAt            sql.NullTime
		shippedAt             sql.NullTime
		refundDueAt           sql.NullTime
		refundedAt            sql.NullTime
		closedAt              sql.NullTime
		reviewedBy            sql.NullInt64
		shippedBy             sql.NullInt64
	)
	if err := s.Scan(
		&m.ID,
		&m.OrderNo,
		&m.UserID,
		&m.OrderSource,
		&m.ProductID,
		&seckillActivityID,
		&seckillActivityItemID,
		&m.SkuCode,
		&m.ProductName,
		&m.MainImage,
		&m.UnitPriceCent,
		&m.Quantity,
		&m.TotalAmountCent,
		&m.OrderStatus,
		&m.PaymentStatus,
		&m.ReviewStatus,
		&m.ShippingStatus,
		&m.RefundStatus,
		&m.ReviewMode,
		&paidAt,
		&m.PayChannel,
		&m.PayReference,
		&m.ReceiverName,
		&m.ReceiverPhone,
		&m.ReceiverAddress,
		&m.BuyerRemark,
		&reviewDueAt,
		&reviewedAt,
		&reviewedBy,
		&m.ReviewReason,
		&shippedAt,
		&shippedBy,
		&m.TrackingNo,
		&refundDueAt,
		&refundedAt,
		&m.StockReleased,
		&closedAt,
		&m.CloseReason,
		&m.CreatedAt,
		&m.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if paidAt.Valid {
		m.PaidAt = &paidAt.Time
	}
	if seckillActivityID.Valid {
		m.SeckillActivityID = seckillActivityID.Int64
	}
	if seckillActivityItemID.Valid {
		m.SeckillActivityItemID = seckillActivityItemID.Int64
	}
	if reviewDueAt.Valid {
		m.ReviewDueAt = &reviewDueAt.Time
	}
	if reviewedAt.Valid {
		m.ReviewedAt = &reviewedAt.Time
	}
	if reviewedBy.Valid {
		m.ReviewedBy = reviewedBy.Int64
	}
	if shippedAt.Valid {
		m.ShippedAt = &shippedAt.Time
	}
	if shippedBy.Valid {
		m.ShippedBy = shippedBy.Int64
	}
	if refundDueAt.Valid {
		m.RefundDueAt = &refundDueAt.Time
	}
	if refundedAt.Valid {
		m.RefundedAt = &refundedAt.Time
	}
	if closedAt.Valid {
		m.ClosedAt = &closedAt.Time
	}
	return &m, nil
}
