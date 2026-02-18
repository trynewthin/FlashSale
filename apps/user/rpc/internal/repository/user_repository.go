// repository 包包含相关应用代码。
package repository

import (
	"context"
	"errors"
	"time"

	"flashsale/apps/user/rpc/internal/model"
)

var (
	// ErrUserNotFound 表示目标用户不存在或已被软删除。
	ErrUserNotFound = errors.New("user not found")
)

// UserListQuery 表示用户列表查询条件。
type UserListQuery struct {
	Page     int64
	PageSize int64
	Keyword  string // 手机号或昵称模糊搜索
	Status   int8   // -1 表示不过滤
}

// UserRepository 定义用户持久化访问接口。
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByPhone(ctx context.Context, phone string) (*model.User, error)
	FindByID(ctx context.Context, userID int64) (*model.User, error)
	UpdateNickname(ctx context.Context, userID int64, nickname string) error
	UpdateLoginAudit(ctx context.Context, userID int64, ip string, at time.Time) error
	SoftDelete(ctx context.Context, userID int64, at time.Time) error
	UpdatePasswordHash(ctx context.Context, userID int64, passwordHash string) error
	ListUsers(ctx context.Context, query UserListQuery) ([]*model.User, int64, error)
}
