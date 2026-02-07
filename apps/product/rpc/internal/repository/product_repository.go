// Package repository 声明商品模块的数据访问抽象。
package repository

import (
	"context"
	"errors"
	"time"

	"flashsale/apps/product/rpc/internal/model"
)

var (
	// ErrProductNotFound 表示目标商品不存在或已被软删除。
	ErrProductNotFound = errors.New("product not found")
)

// ListQuery 表示列表查询条件。
type ListQuery struct {
	Page           int64
	PageSize       int64
	Keyword        string
	IncludeDeleted bool
}

// ProductRepository 定义商品持久化访问接口。
type ProductRepository interface {
	Create(ctx context.Context, product *model.Product) error
	Update(ctx context.Context, product *model.Product) error
	SoftDelete(ctx context.Context, productID int64, at time.Time) error
	FindByIDAdmin(ctx context.Context, productID int64) (*model.Product, error)
	FindByIDPublic(ctx context.Context, productID int64) (*model.Product, error)
	ListAdmin(ctx context.Context, query ListQuery) ([]*model.Product, int64, error)
	ListPublic(ctx context.Context, query ListQuery) ([]*model.Product, int64, error)
}
