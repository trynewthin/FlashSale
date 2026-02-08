// repository 包包含相关应用代码。
package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"flashsale/apps/order/rpc/internal/model"
	"github.com/DATA-DOG/go-sqlmock"
)

func TestMySQLOrderRepository_CompleteRefund_KeepCloseReason(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLOrderRepository(db)
	now := time.Unix(1730000000, 0).UTC()
	event := model.OrderEvent{
		OrderID:      1001,
		EventType:    "order_refund_completed",
		OperatorType: "system",
		OperatorID:   0,
		Payload:      "{}",
		CreatedAt:    now,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE orders SET refund_status = ?, refunded_at = ? WHERE id = ? AND order_status = ? AND refund_status = ?")).
		WithArgs(model.RefundStatusRefunded, now, int64(1001), model.OrderStatusClosed, model.RefundStatusRefunding).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(insertEventSQL)).
		WithArgs(event.OrderID, event.EventType, event.OperatorType, event.OperatorID, event.Payload, event.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := repo.CompleteRefund(context.Background(), 1001, now, event); err != nil {
		t.Fatalf("complete refund failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestMySQLOrderRepository_CompleteRefund_StateConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLOrderRepository(db)
	now := time.Unix(1730000010, 0).UTC()
	event := model.OrderEvent{
		OrderID:      1002,
		EventType:    "order_refund_completed",
		OperatorType: "system",
		OperatorID:   0,
		Payload:      "{}",
		CreatedAt:    now,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE orders SET refund_status = ?, refunded_at = ? WHERE id = ? AND order_status = ? AND refund_status = ?")).
		WithArgs(model.RefundStatusRefunded, now, int64(1002), model.OrderStatusClosed, model.RefundStatusRefunding).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err = repo.CompleteRefund(context.Background(), 1002, now, event)
	if !errors.Is(err, ErrOrderStateConflict) {
		t.Fatalf("expect ErrOrderStateConflict, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestMySQLOrderRepository_ListStockReleasePending_Query(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLOrderRepository(db)

	sqlText := "SELECT " + orderSelectColumns + " FROM orders WHERE order_status = ? AND stock_released = ? AND close_reason IN (?, ?, ?, ?) ORDER BY created_at ASC LIMIT ?"
	mock.ExpectQuery(regexp.QuoteMeta(sqlText)).
		WithArgs(
			model.OrderStatusClosed,
			model.StockReleasePending,
			model.CloseReasonUserCancel,
			model.CloseReasonPayTimeout,
			model.CloseReasonAuditReject,
			model.CloseReasonAuditTimeout,
			5,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	items, err := repo.ListStockReleasePending(context.Background(), 5)
	if err != nil {
		t.Fatalf("list stock release pending failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expect empty list, got=%d", len(items))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
