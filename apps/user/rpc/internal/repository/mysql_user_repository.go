// repository 包包含相关应用代码。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"flashsale/apps/user/rpc/internal/model"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

const (
	insertUserSQL       = "INSERT INTO users (id, phone, password_hash, nickname, status) VALUES (?, ?, ?, ?, 1)"
	findByPhoneSQL      = "SELECT id, phone, password_hash, nickname, status, last_login_at, last_login_ip, deleted_at, created_at, updated_at FROM users WHERE phone = ? AND deleted_at IS NULL LIMIT 1"
	findByIDSQL         = "SELECT id, phone, password_hash, nickname, status, last_login_at, last_login_ip, deleted_at, created_at, updated_at FROM users WHERE id = ? AND deleted_at IS NULL LIMIT 1"
	updateNicknameSQL   = "UPDATE users SET nickname = ? WHERE id = ? AND deleted_at IS NULL"
	updateLoginAuditSQL = "UPDATE users SET last_login_at = ?, last_login_ip = ? WHERE id = ? AND deleted_at IS NULL"
	softDeleteSQL       = "UPDATE users SET deleted_at = ?, status = 0 WHERE id = ? AND deleted_at IS NULL"
)

// MySQLUserRepository 是 UserRepository 的 MySQL 实现。
type MySQLUserRepository struct {
	db *sql.DB
}

// NewMySQLUserRepository 创建 MySQL 用户仓储。
func NewMySQLUserRepository(db *sql.DB) *MySQLUserRepository {
	return &MySQLUserRepository{db: db}
}

// Create 写入新用户记录。
func (r *MySQLUserRepository) Create(ctx context.Context, user *model.User) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if user == nil {
		return fmt.Errorf("user is nil")
	}
	_, err := r.db.ExecContext(ctx, insertUserSQL, user.ID, user.Phone, user.PasswordHash, user.Nickname)
	return err
}

// FindByPhone 根据手机号查询用户。
func (r *MySQLUserRepository) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	return r.findOne(ctx, findByPhoneSQL, phone)
}

// FindByID 根据用户 ID 查询用户。
func (r *MySQLUserRepository) FindByID(ctx context.Context, userID int64) (*model.User, error) {
	return r.findOne(ctx, findByIDSQL, userID)
}

// UpdateNickname 更新用户昵称。
func (r *MySQLUserRepository) UpdateNickname(ctx context.Context, userID int64, nickname string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	ret, err := r.db.ExecContext(ctx, updateNicknameSQL, nickname, userID)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdateLoginAudit 更新登录时间与登录 IP。
func (r *MySQLUserRepository) UpdateLoginAudit(ctx context.Context, userID int64, ip string, at time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	ret, err := r.db.ExecContext(ctx, updateLoginAuditSQL, at, ip, userID)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// SoftDelete 软删除用户。
func (r *MySQLUserRepository) SoftDelete(ctx context.Context, userID int64, at time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	ret, err := r.db.ExecContext(ctx, softDeleteSQL, at, userID)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// IsDuplicateEntry 判断错误是否为 MySQL 唯一键冲突。
func IsDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}

// findOne 执行单行查询并把 SQL 结果转换为领域对象。
func (r *MySQLUserRepository) findOne(ctx context.Context, query string, arg any) (*model.User, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repository db is nil")
	}
	var (
		m           model.User
		lastLoginAt sql.NullTime
		lastLoginIP sql.NullString
		deletedAt   sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&m.ID,
		&m.Phone,
		&m.PasswordHash,
		&m.Nickname,
		&m.Status,
		&lastLoginAt,
		&lastLoginIP,
		&deletedAt,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if lastLoginAt.Valid {
		m.LastLoginAt = &lastLoginAt.Time
	}
	if lastLoginIP.Valid {
		m.LastLoginIP = lastLoginIP.String
	}
	if deletedAt.Valid {
		m.DeletedAt = &deletedAt.Time
	}
	return &m, nil
}
