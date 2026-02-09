// model 包包含相关应用代码。
package model

import "time"

const (
	// AdminStatusDisabled 表示管理员禁用。
	AdminStatusDisabled int8 = 0
	// AdminStatusEnabled 表示管理员启用。
	AdminStatusEnabled int8 = 1
)

const (
	// DataScopeAll 表示可访问全量数据。
	DataScopeAll = "all"
	// DataScopeSelf 表示仅可访问本人数据。
	DataScopeSelf = "self"
)

const (
	// RoleStatusDisabled 表示角色禁用。
	RoleStatusDisabled int8 = 0
	// RoleStatusEnabled 表示角色启用。
	RoleStatusEnabled int8 = 1
)

// Admin 表示管理员账号模型。
type Admin struct {
	ID               int64
	Username         string
	DisplayName      string
	PasswordHash     string
	Status           int8
	DataScope        string
	FailedLoginCount int
	LockedUntil      *time.Time
	LastLoginAt      *time.Time
	LastLoginIP      string
	IsSuperAdmin     bool
	DeletedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	RoleIDs          []int64
	Domains          []string
}

// Role 表示管理员角色模型。
type Role struct {
	ID        int64
	RoleCode  string
	RoleName  string
	Status    int8
	IsSystem  bool
	Domains   []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// RefreshToken 表示刷新令牌记录。
type RefreshToken struct {
	ID        int64
	AdminID   int64
	TokenID   string
	TokenHash string
	IssuedAt  time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
	IssuedIP  string
	UserAgent string
}

// AuditLog 表示审计日志记录。
type AuditLog struct {
	ID        int64
	AdminID   int64
	Action    string
	TargetType string
	TargetID  int64
	Result    string
	RequestID string
	IP        string
	UserAgent string
	DetailJSON string
	CreatedAt time.Time
}
