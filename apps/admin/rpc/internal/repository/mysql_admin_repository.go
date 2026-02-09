// repository 包包含相关应用代码。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"flashsale/apps/admin/rpc/internal/model"
)

const (
	adminSelectColumns = "id, username, display_name, password_hash, status, data_scope, failed_login_count, locked_until, last_login_at, last_login_ip, is_super_admin, deleted_at, created_at, updated_at"
	roleSelectColumns  = "id, role_code, role_name, status, is_system, created_at, updated_at"
)

// MySQLAdminRepository 是管理员仓储 MySQL 实现。
type MySQLAdminRepository struct {
	db *sql.DB
}

// NewMySQLAdminRepository 创建管理员 MySQL 仓储。
func NewMySQLAdminRepository(db *sql.DB) *MySQLAdminRepository {
	return &MySQLAdminRepository{db: db}
}

func (r *MySQLAdminRepository) CountAdmins(ctx context.Context) (int64, error) {
	if err := r.ensureDB(); err != nil {
		return 0, err
	}
	var total int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM admins WHERE deleted_at IS NULL").Scan(&total)
	return total, err
}

func (r *MySQLAdminRepository) CreateAdmin(ctx context.Context, admin *model.Admin) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	if admin == nil {
		return fmt.Errorf("admin is nil")
	}
	_, err := r.db.ExecContext(ctx, "INSERT INTO admins (id, username, display_name, password_hash, status, data_scope, is_super_admin) VALUES (?, ?, ?, ?, ?, ?, ?)",
		admin.ID, admin.Username, admin.DisplayName, admin.PasswordHash, admin.Status, admin.DataScope, boolToInt(admin.IsSuperAdmin))
	if err != nil {
		if isDuplicateEntry(err) {
			return ErrDuplicateUsername
		}
		return err
	}
	return nil
}

func (r *MySQLAdminRepository) UpdateAdminProfile(ctx context.Context, adminID int64, displayName, dataScope string, updatedAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	ret, err := r.db.ExecContext(ctx, "UPDATE admins SET display_name = ?, data_scope = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL", displayName, dataScope, updatedAt, adminID)
	if err != nil {
		return err
	}
	return ensureRowsAffected(ret, ErrAdminNotFound)
}

func (r *MySQLAdminRepository) SetAdminStatus(ctx context.Context, adminID int64, status int8, updatedAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	ret, err := r.db.ExecContext(ctx, "UPDATE admins SET status = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL", status, updatedAt, adminID)
	if err != nil {
		return err
	}
	return ensureRowsAffected(ret, ErrAdminNotFound)
}

func (r *MySQLAdminRepository) UpdateAdminPassword(ctx context.Context, adminID int64, passwordHash string, updatedAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	ret, err := r.db.ExecContext(ctx, "UPDATE admins SET password_hash = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL", passwordHash, updatedAt, adminID)
	if err != nil {
		return err
	}
	return ensureRowsAffected(ret, ErrAdminNotFound)
}

func (r *MySQLAdminRepository) SoftDeleteAdmin(ctx context.Context, adminID int64, deletedAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	ret, err := r.db.ExecContext(ctx, "UPDATE admins SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL", deletedAt, deletedAt, adminID)
	if err != nil {
		return err
	}
	return ensureRowsAffected(ret, ErrAdminNotFound)
}

func (r *MySQLAdminRepository) FindAdminByID(ctx context.Context, adminID int64) (*model.Admin, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, "SELECT "+adminSelectColumns+" FROM admins WHERE id = ? AND deleted_at IS NULL LIMIT 1", adminID)
	admin, err := scanAdmin(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAdminNotFound
		}
		return nil, err
	}
	return admin, nil
}

func (r *MySQLAdminRepository) FindAdminByUsername(ctx context.Context, username string) (*model.Admin, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, "SELECT "+adminSelectColumns+" FROM admins WHERE username = ? AND deleted_at IS NULL LIMIT 1", strings.TrimSpace(username))
	admin, err := scanAdmin(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAdminNotFound
		}
		return nil, err
	}
	return admin, nil
}

func (r *MySQLAdminRepository) ListAdmins(ctx context.Context, query AdminListQuery) ([]*model.Admin, int64, error) {
	if err := r.ensureDB(); err != nil {
		return nil, 0, err
	}
	page, pageSize := normalizePage(query.Page, query.PageSize)
	whereParts := []string{"deleted_at IS NULL"}
	args := make([]any, 0, 4)
	if strings.TrimSpace(query.Keyword) != "" {
		like := "%" + strings.TrimSpace(query.Keyword) + "%"
		whereParts = append(whereParts, "(username LIKE ? OR display_name LIKE ?)")
		args = append(args, like, like)
	}
	if query.Status == model.AdminStatusEnabled || query.Status == model.AdminStatusDisabled {
		whereParts = append(whereParts, "status = ?")
		args = append(args, query.Status)
	}
	where := " WHERE " + strings.Join(whereParts, " AND ")
	total, err := r.count(ctx, "SELECT COUNT(1) FROM admins"+where, args)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.Admin{}, 0, nil
	}
	offset := (page - 1) * pageSize
	listSQL := "SELECT " + adminSelectColumns + " FROM admins" + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	rows, err := r.db.QueryContext(ctx, listSQL, append(append([]any{}, args...), pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]*model.Admin, 0, pageSize)
	for rows.Next() {
		item, scanErr := scanAdmin(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *MySQLAdminRepository) ResetAdminLoginFailures(ctx context.Context, adminID int64, lastLoginAt time.Time, lastLoginIP string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	ret, err := r.db.ExecContext(ctx, "UPDATE admins SET failed_login_count = 0, locked_until = NULL, last_login_at = ?, last_login_ip = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL", lastLoginAt, strings.TrimSpace(lastLoginIP), lastLoginAt, adminID)
	if err != nil {
		return err
	}
	return ensureRowsAffected(ret, ErrAdminNotFound)
}

func (r *MySQLAdminRepository) SetAdminLoginFailure(ctx context.Context, adminID int64, failedCount int, lockedUntil *time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	var locked any
	if lockedUntil != nil {
		locked = *lockedUntil
	}
	ret, err := r.db.ExecContext(ctx, "UPDATE admins SET failed_login_count = ?, locked_until = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL", failedCount, locked, time.Now(), adminID)
	if err != nil {
		return err
	}
	return ensureRowsAffected(ret, ErrAdminNotFound)
}

func (r *MySQLAdminRepository) CreateRole(ctx context.Context, role *model.Role) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	if role == nil {
		return fmt.Errorf("role is nil")
	}
	_, err := r.db.ExecContext(ctx, "INSERT INTO admin_roles (id, role_code, role_name, status, is_system) VALUES (?, ?, ?, ?, ?)", role.ID, role.RoleCode, role.RoleName, role.Status, boolToInt(role.IsSystem))
	if err != nil {
		if isDuplicateEntry(err) {
			return ErrDuplicateRoleCode
		}
		return err
	}
	return nil
}

func (r *MySQLAdminRepository) UpdateRole(ctx context.Context, roleID int64, roleName string, status int8, updatedAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	ret, err := r.db.ExecContext(ctx, "UPDATE admin_roles SET role_name = ?, status = ?, updated_at = ? WHERE id = ?", roleName, status, updatedAt, roleID)
	if err != nil {
		return err
	}
	return ensureRowsAffected(ret, ErrRoleNotFound)
}

func (r *MySQLAdminRepository) DeleteRole(ctx context.Context, roleID int64) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	ret, err := r.db.ExecContext(ctx, "DELETE FROM admin_roles WHERE id = ?", roleID)
	if err != nil {
		return err
	}
	return ensureRowsAffected(ret, ErrRoleNotFound)
}

func (r *MySQLAdminRepository) FindRoleByID(ctx context.Context, roleID int64) (*model.Role, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, "SELECT "+roleSelectColumns+" FROM admin_roles WHERE id = ? LIMIT 1", roleID)
	role, err := scanRole(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return role, nil
}

func (r *MySQLAdminRepository) FindRoleByCode(ctx context.Context, roleCode string) (*model.Role, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, "SELECT "+roleSelectColumns+" FROM admin_roles WHERE role_code = ? LIMIT 1", strings.TrimSpace(roleCode))
	role, err := scanRole(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return role, nil
}

func (r *MySQLAdminRepository) ListRoles(ctx context.Context, query RoleListQuery) ([]*model.Role, int64, error) {
	if err := r.ensureDB(); err != nil {
		return nil, 0, err
	}
	page, pageSize := normalizePage(query.Page, query.PageSize)
	whereParts := make([]string, 0, 2)
	args := make([]any, 0, 3)
	if strings.TrimSpace(query.Keyword) != "" {
		like := "%" + strings.TrimSpace(query.Keyword) + "%"
		whereParts = append(whereParts, "(role_code LIKE ? OR role_name LIKE ?)")
		args = append(args, like, like)
	}
	if query.Status == model.RoleStatusEnabled || query.Status == model.RoleStatusDisabled {
		whereParts = append(whereParts, "status = ?")
		args = append(args, query.Status)
	}
	where := ""
	if len(whereParts) > 0 {
		where = " WHERE " + strings.Join(whereParts, " AND ")
	}
	total, err := r.count(ctx, "SELECT COUNT(1) FROM admin_roles"+where, args)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.Role{}, 0, nil
	}
	offset := (page - 1) * pageSize
	listSQL := "SELECT " + roleSelectColumns + " FROM admin_roles" + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	rows, err := r.db.QueryContext(ctx, listSQL, append(append([]any{}, args...), pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]*model.Role, 0, pageSize)
	for rows.Next() {
		item, scanErr := scanRole(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *MySQLAdminRepository) ReplaceRoleDomains(ctx context.Context, roleID int64, domains []string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	return withTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM admin_role_domains WHERE role_id = ?", roleID); err != nil {
			return err
		}
		for idx, domain := range domains {
			domain = strings.TrimSpace(domain)
			if domain == "" {
				continue
			}
			_, err := tx.ExecContext(ctx, "INSERT INTO admin_role_domains (id, role_id, domain_code) VALUES (?, ?, ?)", time.Now().UnixNano()+int64(idx), roleID, domain)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *MySQLAdminRepository) ListRoleDomains(ctx context.Context, roleID int64) ([]string, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, "SELECT domain_code FROM admin_role_domains WHERE role_id = ? ORDER BY domain_code ASC", roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		out = append(out, domain)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *MySQLAdminRepository) ReplaceAdminRoles(ctx context.Context, adminID int64, roleIDs []int64) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	return withTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM admin_role_bindings WHERE admin_id = ?", adminID); err != nil {
			return err
		}
		for idx, roleID := range roleIDs {
			if roleID <= 0 {
				continue
			}
			_, err := tx.ExecContext(ctx, "INSERT INTO admin_role_bindings (id, admin_id, role_id) VALUES (?, ?, ?)", time.Now().UnixNano()+int64(idx), adminID, roleID)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *MySQLAdminRepository) ListAdminRoleIDs(ctx context.Context, adminID int64) ([]int64, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, "SELECT role_id FROM admin_role_bindings WHERE admin_id = ? ORDER BY role_id ASC", adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]int64, 0)
	for rows.Next() {
		var roleID int64
		if err := rows.Scan(&roleID); err != nil {
			return nil, err
		}
		out = append(out, roleID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *MySQLAdminRepository) ListAdminDomains(ctx context.Context, adminID int64) ([]string, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, "SELECT DISTINCT d.domain_code FROM admin_role_bindings b JOIN admin_roles r ON r.id = b.role_id AND r.status = 1 JOIN admin_role_domains d ON d.role_id = b.role_id WHERE b.admin_id = ? ORDER BY d.domain_code ASC", adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		out = append(out, domain)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *MySQLAdminRepository) CountRoleBindings(ctx context.Context, roleID int64) (int64, error) {
	if err := r.ensureDB(); err != nil {
		return 0, err
	}
	var total int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM admin_role_bindings WHERE role_id = ?", roleID).Scan(&total)
	return total, err
}

func (r *MySQLAdminRepository) CountSuperAdmins(ctx context.Context) (int64, error) {
	if err := r.ensureDB(); err != nil {
		return 0, err
	}
	var total int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM admins WHERE is_super_admin = 1 AND status = 1 AND deleted_at IS NULL").Scan(&total)
	return total, err
}

func (r *MySQLAdminRepository) CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	if token == nil {
		return fmt.Errorf("refresh token is nil")
	}
	_, err := r.db.ExecContext(ctx, "INSERT INTO admin_refresh_tokens (id, admin_id, token_id, token_hash, issued_at, expires_at, issued_ip, user_agent) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		token.ID, token.AdminID, token.TokenID, token.TokenHash, token.IssuedAt, token.ExpiresAt, token.IssuedIP, token.UserAgent)
	return err
}

func (r *MySQLAdminRepository) FindRefreshToken(ctx context.Context, tokenID string) (*model.RefreshToken, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, "SELECT id, admin_id, token_id, token_hash, issued_at, expires_at, revoked_at, issued_ip, user_agent FROM admin_refresh_tokens WHERE token_id = ? LIMIT 1", tokenID)
	var (
		token     model.RefreshToken
		revokedAt sql.NullTime
	)
	if err := row.Scan(&token.ID, &token.AdminID, &token.TokenID, &token.TokenHash, &token.IssuedAt, &token.ExpiresAt, &revokedAt, &token.IssuedIP, &token.UserAgent); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, err
	}
	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}
	return &token, nil
}

func (r *MySQLAdminRepository) RevokeRefreshToken(ctx context.Context, tokenID string, revokedAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	ret, err := r.db.ExecContext(ctx, "UPDATE admin_refresh_tokens SET revoked_at = ?, updated_at = ? WHERE token_id = ? AND revoked_at IS NULL", revokedAt, revokedAt, tokenID)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrRefreshTokenNotFound
	}
	return nil
}

func (r *MySQLAdminRepository) RotateRefreshToken(ctx context.Context, oldTokenID string, revokedAt time.Time, newToken *model.RefreshToken) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	if newToken == nil {
		return fmt.Errorf("new refresh token is nil")
	}
	return withTx(ctx, r.db, func(tx *sql.Tx) error {
		ret, err := tx.ExecContext(ctx, "UPDATE admin_refresh_tokens SET revoked_at = ?, updated_at = ? WHERE token_id = ? AND revoked_at IS NULL", revokedAt, revokedAt, oldTokenID)
		if err != nil {
			return err
		}
		rows, err := ret.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrRefreshTokenNotFound
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO admin_refresh_tokens (id, admin_id, token_id, token_hash, issued_at, expires_at, issued_ip, user_agent) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			newToken.ID, newToken.AdminID, newToken.TokenID, newToken.TokenHash, newToken.IssuedAt, newToken.ExpiresAt, newToken.IssuedIP, newToken.UserAgent)
		return err
	})
}

func (r *MySQLAdminRepository) RevokeRefreshTokensByAdmin(ctx context.Context, adminID int64, revokedAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, "UPDATE admin_refresh_tokens SET revoked_at = ?, updated_at = ? WHERE admin_id = ? AND revoked_at IS NULL", revokedAt, revokedAt, adminID)
	return err
}

func (r *MySQLAdminRepository) CreateAuditLog(ctx context.Context, log *model.AuditLog) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	if log == nil {
		return fmt.Errorf("audit log is nil")
	}
	if strings.TrimSpace(log.DetailJSON) == "" {
		log.DetailJSON = "{}"
	}
	_, err := r.db.ExecContext(ctx, "INSERT INTO admin_audit_logs (id, admin_id, action, target_type, target_id, result, request_id, ip, user_agent, detail_json, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		log.ID, log.AdminID, log.Action, log.TargetType, log.TargetID, log.Result, log.RequestID, log.IP, log.UserAgent, log.DetailJSON, log.CreatedAt)
	return err
}

func (r *MySQLAdminRepository) ListAuditLogs(ctx context.Context, query AuditLogListQuery) ([]*model.AuditLog, int64, error) {
	if err := r.ensureDB(); err != nil {
		return nil, 0, err
	}
	page, pageSize := normalizePage(query.Page, query.PageSize)
	whereParts := make([]string, 0, 4)
	args := make([]any, 0, 4)
	if query.AdminID > 0 {
		whereParts = append(whereParts, "admin_id = ?")
		args = append(args, query.AdminID)
	}
	if strings.TrimSpace(query.Action) != "" {
		whereParts = append(whereParts, "action = ?")
		args = append(args, strings.TrimSpace(query.Action))
	}
	if strings.TrimSpace(query.TargetType) != "" {
		whereParts = append(whereParts, "target_type = ?")
		args = append(args, strings.TrimSpace(query.TargetType))
	}
	if query.TargetID > 0 {
		whereParts = append(whereParts, "target_id = ?")
		args = append(args, query.TargetID)
	}
	where := ""
	if len(whereParts) > 0 {
		where = " WHERE " + strings.Join(whereParts, " AND ")
	}
	total, err := r.count(ctx, "SELECT COUNT(1) FROM admin_audit_logs"+where, args)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.AuditLog{}, 0, nil
	}
	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx, "SELECT id, admin_id, action, target_type, target_id, result, request_id, ip, user_agent, detail_json, created_at FROM admin_audit_logs"+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?", append(append([]any{}, args...), pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]*model.AuditLog, 0, pageSize)
	for rows.Next() {
		item := &model.AuditLog{}
		if err := rows.Scan(&item.ID, &item.AdminID, &item.Action, &item.TargetType, &item.TargetID, &item.Result, &item.RequestID, &item.IP, &item.UserAgent, &item.DetailJSON, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *MySQLAdminRepository) ensureDB() error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	return nil
}

func (r *MySQLAdminRepository) count(ctx context.Context, sqlText string, args []any) (int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, sqlText, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func ensureRowsAffected(ret sql.Result, notFound error) error {
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return notFound
	}
	return nil
}

func normalizePage(page, pageSize int64) (int64, int64) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func isDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "1062")
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

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAdmin(s scanner) (*model.Admin, error) {
	var (
		item        model.Admin
		lockedUntil sql.NullTime
		lastLoginAt sql.NullTime
		deletedAt   sql.NullTime
		isSuper     int
	)
	if err := s.Scan(&item.ID, &item.Username, &item.DisplayName, &item.PasswordHash, &item.Status, &item.DataScope, &item.FailedLoginCount, &lockedUntil, &lastLoginAt, &item.LastLoginIP, &isSuper, &deletedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	if lockedUntil.Valid {
		item.LockedUntil = &lockedUntil.Time
	}
	if lastLoginAt.Valid {
		item.LastLoginAt = &lastLoginAt.Time
	}
	if deletedAt.Valid {
		item.DeletedAt = &deletedAt.Time
	}
	item.IsSuperAdmin = isSuper == 1
	return &item, nil
}

func scanRole(s scanner) (*model.Role, error) {
	var (
		item     model.Role
		isSystem int
	)
	if err := s.Scan(&item.ID, &item.RoleCode, &item.RoleName, &item.Status, &isSystem, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.IsSystem = isSystem == 1
	return &item, nil
}
