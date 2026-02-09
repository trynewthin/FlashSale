// repository 包包含相关应用代码。
package repository

import (
	"context"
	"errors"
	"time"

	"flashsale/apps/admin/rpc/internal/model"
)

var (
	// ErrAdminNotFound 表示管理员不存在。
	ErrAdminNotFound = errors.New("admin not found")
	// ErrRoleNotFound 表示角色不存在。
	ErrRoleNotFound = errors.New("role not found")
	// ErrDuplicateUsername 表示管理员用户名冲突。
	ErrDuplicateUsername = errors.New("duplicate admin username")
	// ErrDuplicateRoleCode 表示角色编码冲突。
	ErrDuplicateRoleCode = errors.New("duplicate role code")
	// ErrRefreshTokenNotFound 表示刷新令牌不存在。
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

// AdminListQuery 表示管理员列表查询条件。
type AdminListQuery struct {
	Page     int64
	PageSize int64
	Keyword  string
	Status   int8
}

// RoleListQuery 表示角色列表查询条件。
type RoleListQuery struct {
	Page     int64
	PageSize int64
	Keyword  string
	Status   int8
}

// AuditLogListQuery 表示审计日志列表查询条件。
type AuditLogListQuery struct {
	Page       int64
	PageSize   int64
	AdminID    int64
	Action     string
	TargetType string
	TargetID   int64
}

// AdminRepository 定义管理员仓储接口。
type AdminRepository interface {
	CountAdmins(ctx context.Context) (int64, error)
	CreateAdmin(ctx context.Context, admin *model.Admin) error
	UpdateAdminProfile(ctx context.Context, adminID int64, displayName, dataScope string, updatedAt time.Time) error
	SetAdminStatus(ctx context.Context, adminID int64, status int8, updatedAt time.Time) error
	UpdateAdminPassword(ctx context.Context, adminID int64, passwordHash string, updatedAt time.Time) error
	SoftDeleteAdmin(ctx context.Context, adminID int64, deletedAt time.Time) error
	FindAdminByID(ctx context.Context, adminID int64) (*model.Admin, error)
	FindAdminByUsername(ctx context.Context, username string) (*model.Admin, error)
	ListAdmins(ctx context.Context, query AdminListQuery) ([]*model.Admin, int64, error)
	ResetAdminLoginFailures(ctx context.Context, adminID int64, lastLoginAt time.Time, lastLoginIP string) error
	SetAdminLoginFailure(ctx context.Context, adminID int64, failedCount int, lockedUntil *time.Time) error

	CreateRole(ctx context.Context, role *model.Role) error
	UpdateRole(ctx context.Context, roleID int64, roleName string, status int8, updatedAt time.Time) error
	DeleteRole(ctx context.Context, roleID int64) error
	FindRoleByID(ctx context.Context, roleID int64) (*model.Role, error)
	FindRoleByCode(ctx context.Context, roleCode string) (*model.Role, error)
	ListRoles(ctx context.Context, query RoleListQuery) ([]*model.Role, int64, error)
	ReplaceRoleDomains(ctx context.Context, roleID int64, domains []string) error
	ListRoleDomains(ctx context.Context, roleID int64) ([]string, error)

	ReplaceAdminRoles(ctx context.Context, adminID int64, roleIDs []int64) error
	ListAdminRoleIDs(ctx context.Context, adminID int64) ([]int64, error)
	ListAdminDomains(ctx context.Context, adminID int64) ([]string, error)
	CountRoleBindings(ctx context.Context, roleID int64) (int64, error)
	CountSuperAdmins(ctx context.Context) (int64, error)

	CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error
	FindRefreshToken(ctx context.Context, tokenID string) (*model.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenID string, revokedAt time.Time) error
	RotateRefreshToken(ctx context.Context, oldTokenID string, revokedAt time.Time, newToken *model.RefreshToken) error
	RevokeRefreshTokensByAdmin(ctx context.Context, adminID int64, revokedAt time.Time) error

	CreateAuditLog(ctx context.Context, log *model.AuditLog) error
	ListAuditLogs(ctx context.Context, query AuditLogListQuery) ([]*model.AuditLog, int64, error)
}
