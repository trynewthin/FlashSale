// repository 包包含相关应用代码。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"flashsale/apps/product/rpc/internal/model"
	_ "github.com/go-sql-driver/mysql"
)

func TestMySQLProductRepositoryIntegration(t *testing.T) {
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

	if !tableExists(t, db, "products") {
		t.Skip("products table not found, run migrations first")
	}
	if !tableExists(t, db, "idempotency_records") {
		t.Skip("idempotency_records table not found, run migrations first")
	}

	repo := NewMySQLProductRepository(db)
	now := time.Now().UnixNano()
	pid := now
	sku := fmt.Sprintf("SPU%d", now)
	p := &model.Product{
		ID:          pid,
		SkuCode:     sku,
		Name:        "集成测试商品",
		MainImage:   "https://example.com/a.png",
		Description: "desc",
		PriceCent:   1999,
		Stock:       10,
		Status:      model.StatusOnShelf,
	}
	if err := repo.Create(context.Background(), p); err != nil {
		t.Fatalf("create product failed: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM products WHERE id = ?", pid)
	}()

	// unique sku
	err = repo.Create(context.Background(), &model.Product{
		ID:          pid + 1,
		SkuCode:     sku,
		Name:        "dup",
		MainImage:   "https://example.com/b.png",
		Description: "dup",
		PriceCent:   100,
		Stock:       1,
		Status:      model.StatusOnShelf,
	})
	if err == nil || !IsDuplicateEntry(err) {
		t.Fatalf("expected duplicate entry error, got: %v", err)
	}

	gotAdmin, err := repo.FindByIDAdmin(context.Background(), pid)
	if err != nil {
		t.Fatalf("find admin failed: %v", err)
	}
	if gotAdmin.SkuCode != sku {
		t.Fatalf("sku mismatch: got %s", gotAdmin.SkuCode)
	}

	if err := repo.Update(context.Background(), &model.Product{
		ID:          pid,
		Name:        "已更新商品",
		MainImage:   "https://example.com/c.png",
		Description: "updated",
		PriceCent:   2999,
		Stock:       0,
		Status:      model.StatusOffShelf,
	}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if _, err := repo.FindByIDPublic(context.Background(), pid); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("off shelf product should not be public, got: %v", err)
	}

	items, total, err := repo.ListAdmin(context.Background(), ListQuery{Page: 1, PageSize: 20, Keyword: "更新"})
	if err != nil {
		t.Fatalf("list admin failed: %v", err)
	}
	if total < 1 || len(items) < 1 {
		t.Fatalf("list admin expected data, total=%d len=%d", total, len(items))
	}

	if err := repo.SoftDelete(context.Background(), pid, time.Now()); err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}
	remainAfterRelease, err := repo.ReleaseStock(context.Background(), pid, 1, fmt.Sprintf("ORD%d", pid), fmt.Sprintf("ORD%d:release", pid))
	if err != nil {
		t.Fatalf("release stock after soft delete failed: %v", err)
	}
	if remainAfterRelease != 1 {
		t.Fatalf("release stock after soft delete mismatch: got=%d want=1", remainAfterRelease)
	}
	if _, err := repo.FindByIDAdmin(context.Background(), pid); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound after delete, got: %v", err)
	}

	// idempotent replay for reserve/release should not apply stock change twice.
	pid2 := pid + 100
	p2 := &model.Product{
		ID:          pid2,
		SkuCode:     fmt.Sprintf("SPU%d", pid2),
		Name:        "幂等测试商品",
		MainImage:   "https://example.com/idem.png",
		Description: "idem",
		PriceCent:   199,
		Stock:       5,
		Status:      model.StatusOnShelf,
	}
	if err := repo.Create(context.Background(), p2); err != nil {
		t.Fatalf("create product2 failed: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM idempotency_records WHERE idempotency_key IN (?, ?, ?, ?)",
			fmt.Sprintf("ORD%d:reserve", pid2),
			fmt.Sprintf("ORD%d:release", pid2),
			fmt.Sprintf("ORD%d:reserve:rollback", pid2),
			fmt.Sprintf("ORD%d:release:rollback", pid2),
		)
		_, _ = db.Exec("DELETE FROM products WHERE id = ?", pid2)
	}()

	reserveKey := fmt.Sprintf("ORD%d:reserve", pid2)
	remain1, err := repo.ReserveStock(context.Background(), pid2, 2, fmt.Sprintf("ORD%d", pid2), reserveKey)
	if err != nil {
		t.Fatalf("reserve stock failed: %v", err)
	}
	remain2, err := repo.ReserveStock(context.Background(), pid2, 2, fmt.Sprintf("ORD%d", pid2), reserveKey)
	if err != nil {
		t.Fatalf("reserve replay failed: %v", err)
	}
	if remain1 != 3 || remain2 != 3 {
		t.Fatalf("reserve replay mismatch: first=%d second=%d", remain1, remain2)
	}

	releaseKey := fmt.Sprintf("ORD%d:release", pid2)
	release1, err := repo.ReleaseStock(context.Background(), pid2, 2, fmt.Sprintf("ORD%d", pid2), releaseKey)
	if err != nil {
		t.Fatalf("release stock failed: %v", err)
	}
	release2, err := repo.ReleaseStock(context.Background(), pid2, 2, fmt.Sprintf("ORD%d", pid2), releaseKey)
	if err != nil {
		t.Fatalf("release replay failed: %v", err)
	}
	if release1 != 5 || release2 != 5 {
		t.Fatalf("release replay mismatch: first=%d second=%d", release1, release2)
	}
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
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
