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

	"flashsale/apps/user/rpc/internal/model"
	_ "github.com/go-sql-driver/mysql"
)

// TestMySQLUserRepositoryIntegration 在真实 MySQL 上验证仓储接口。
func TestMySQLUserRepositoryIntegration(t *testing.T) {
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

	if !tableExists(t, db, "users") {
		t.Skip("users table not found, run migrations first")
	}

	repo := NewMySQLUserRepository(db)
	now := time.Now().UnixNano()
	uid := now
	phone := fmt.Sprintf("13%09d", now%1_000_000_000)
	user := &model.User{
		ID:           uid,
		Phone:        phone,
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuv",
		Nickname:     "集成测试",
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM users WHERE id = ?", uid)
	}()

	gotByPhone, err := repo.FindByPhone(context.Background(), phone)
	if err != nil {
		t.Fatalf("find by phone failed: %v", err)
	}
	if gotByPhone.ID != uid {
		t.Fatalf("find by phone uid mismatch: got %d want %d", gotByPhone.ID, uid)
	}

	if err := repo.UpdateNickname(context.Background(), uid, "新昵称"); err != nil {
		t.Fatalf("update nickname failed: %v", err)
	}
	if err := repo.UpdateLoginAudit(context.Background(), uid, "127.0.0.1", time.Now()); err != nil {
		t.Fatalf("update login audit failed: %v", err)
	}

	gotByID, err := repo.FindByID(context.Background(), uid)
	if err != nil {
		t.Fatalf("find by id failed: %v", err)
	}
	if gotByID.Nickname != "新昵称" {
		t.Fatalf("nickname mismatch: got %s", gotByID.Nickname)
	}
	if gotByID.LastLoginIP != "127.0.0.1" {
		t.Fatalf("last login ip mismatch: got %s", gotByID.LastLoginIP)
	}

	if err := repo.SoftDelete(context.Background(), uid, time.Now()); err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}
	if _, err := repo.FindByID(context.Background(), uid); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("find by id after delete expected ErrUserNotFound, got: %v", err)
	}
}

// tableExists 检查目标表是否存在。
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
