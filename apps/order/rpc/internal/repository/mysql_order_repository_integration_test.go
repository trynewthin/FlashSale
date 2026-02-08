// repository 包包含相关应用代码。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"flashsale/apps/order/rpc/internal/model"
	_ "github.com/go-sql-driver/mysql"
)

func TestMySQLOrderRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("FLASHSALE_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("FLASHSALE_TEST_MYSQL_DSN not set")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql failed: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping mysql failed: %v", err)
	}
	if !orderTableExists(t, db, "orders") || !orderTableExists(t, db, "order_events") {
		t.Skip("orders/order_events table not found, run migrations first")
	}
	if !orderColumnExists(t, db, "orders", "stock_released") {
		t.Skip("orders.stock_released column not found, run latest migrations first")
	}
	if !orderColumnExists(t, db, "orders", "seckill_activity_id") {
		t.Skip("orders.seckill_activity_id column not found, run latest migrations first")
	}

	repo := NewMySQLOrderRepository(db)
	now := time.Now().UnixNano()
	orderID := now
	orderNo := fmt.Sprintf("ORD%d", now)
	order := &model.Order{
		ID:              orderID,
		OrderNo:         orderNo,
		UserID:          10001,
		OrderSource:     model.OrderSourceNormal,
		ProductID:       20001,
		SkuCode:         "SPU20001",
		ProductName:     "集成测试订单商品",
		MainImage:       "https://example.com/a.png",
		UnitPriceCent:   1299,
		Quantity:        1,
		TotalAmountCent: 1299,
		OrderStatus:     model.OrderStatusPendingPay,
		PaymentStatus:   model.PaymentStatusUnpaid,
		ReviewStatus:    model.ReviewStatusNone,
		ShippingStatus:  model.ShippingStatusNotShipped,
		RefundStatus:    model.RefundStatusNone,
		ReviewMode:      model.ReviewModeAuto,
	}
	createEvent := model.OrderEvent{
		OrderID:      orderID,
		EventType:    "order_created",
		OperatorType: "user",
		OperatorID:   10001,
		Payload:      "{}",
		CreatedAt:    time.Now(),
	}
	if err := repo.Create(context.Background(), order, createEvent); err != nil {
		t.Fatalf("create order failed: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM order_events WHERE order_id = ?", orderID)
		_, _ = db.Exec("DELETE FROM orders WHERE id = ?", orderID)
	}()

	got, err := repo.FindByID(context.Background(), orderID)
	if err != nil {
		t.Fatalf("find order failed: %v", err)
	}
	if got.OrderNo != orderNo {
		t.Fatalf("order no mismatch: got=%s want=%s", got.OrderNo, orderNo)
	}

	cancelEvent := model.OrderEvent{
		OrderID:      orderID,
		EventType:    "order_canceled_by_user",
		OperatorType: "user",
		OperatorID:   10001,
		Payload:      "{}",
		CreatedAt:    time.Now(),
	}
	if err := repo.CancelUnpaid(context.Background(), orderID, 10001, time.Now(), model.CloseReasonUserCancel, cancelEvent); err != nil {
		t.Fatalf("cancel unpaid failed: %v", err)
	}

	got, err = repo.FindByID(context.Background(), orderID)
	if err != nil {
		t.Fatalf("find order after cancel failed: %v", err)
	}
	if got.OrderStatus != model.OrderStatusClosed {
		t.Fatalf("order status mismatch after cancel: got=%d want=%d", got.OrderStatus, model.OrderStatusClosed)
	}

	pending, err := repo.ListStockReleasePending(context.Background(), 10)
	if err != nil {
		t.Fatalf("list stock release pending failed: %v", err)
	}
	if len(pending) < 1 {
		t.Fatal("expected at least one stock release pending order")
	}
	markEvent := model.OrderEvent{
		OrderID:      orderID,
		EventType:    "order_stock_released",
		OperatorType: "system",
		OperatorID:   0,
		Payload:      "{}",
		CreatedAt:    time.Now(),
	}
	if err := repo.MarkStockReleased(context.Background(), orderID, time.Now(), markEvent); err != nil {
		t.Fatalf("mark stock released failed: %v", err)
	}
	got, err = repo.FindByID(context.Background(), orderID)
	if err != nil {
		t.Fatalf("find order after stock release mark failed: %v", err)
	}
	if got.StockReleased != model.StockReleased {
		t.Fatalf("stock released flag mismatch: got=%d want=%d", got.StockReleased, model.StockReleased)
	}
}

func orderTableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var name string
	err := db.QueryRow("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? LIMIT 1", table).Scan(&name)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("query table exists failed: %v", err)
	}
	return name == table
}

func orderColumnExists(t *testing.T, db *sql.DB, table string, column string) bool {
	t.Helper()
	var name string
	err := db.QueryRow("SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1", table, column).Scan(&name)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("query column exists failed: %v", err)
	}
	return name == column
}
