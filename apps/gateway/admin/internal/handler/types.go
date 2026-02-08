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
