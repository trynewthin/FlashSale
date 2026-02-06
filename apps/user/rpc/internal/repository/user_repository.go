// Package repository 声明用户模块的数据访问抽象。
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

// UserRepository 定义用户持久化访问接口。
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByPhone(ctx context.Context, phone string) (*model.User, error)
	FindByID(ctx context.Context, userID int64) (*model.User, error)
	UpdateNickname(ctx context.Context, userID int64, nickname string) error
	UpdateLoginAudit(ctx context.Context, userID int64, ip string, at time.Time) error
	SoftDelete(ctx context.Context, userID int64, at time.Time) error
}
