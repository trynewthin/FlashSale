// model 包包含相关应用代码。
package model

import "time"

const (
	// StatusOffShelf 表示下架。
	StatusOffShelf int8 = 0
	// StatusOnShelf 表示上架。
	StatusOnShelf int8 = 1
)

// Product 表示商品实体（单 SKU）。
type Product struct {
	ID          int64
	SkuCode     string
	Name        string
	MainImage   string
	Description string
	PriceCent   int64
	Stock       int64
	Status      int8
	DeletedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
