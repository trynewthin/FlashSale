// handler 包包含相关应用代码。
package handler

// productUpsertReq 是管理员商品创建/更新请求体。
type productUpsertReq struct {
	Name        string `json:"name"`
	MainImage   string `json:"main_image"`
	Description string `json:"description"`
	PriceCent   int64  `json:"price_cent"`
	Stock       int64  `json:"stock"`
	Status      int32  `json:"status"`
}

// activityItemUpsertReq 是秒杀活动商品创建/更新请求体。
type activityItemUpsertReq struct {
	ProductID          int64 `json:"product_id"`
	SeckillPriceCent   int64 `json:"seckill_price_cent"`
	ReservedStockTotal int64 `json:"reserved_stock_total"`
	UserLimitMode      int32 `json:"user_limit_mode"`
	UserLimitWindowSec int64 `json:"user_limit_window_sec"`
	UserLimitQty       int64 `json:"user_limit_qty"`
	MaxQtyPerOrder     int64 `json:"max_qty_per_order"`
	Status             int32 `json:"status"`
}
