// handler 包包含相关应用代码。
package handler

import "flashsale/pkg/base/handlerx"

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
	ProductID          handlerx.JSONInt64 `json:"product_id"`
	SeckillPriceCent   int64              `json:"seckill_price_cent"`
	ReservedStockTotal int64              `json:"reserved_stock_total"`
	UserLimitMode      int32              `json:"user_limit_mode"`
	UserLimitWindowSec int64              `json:"user_limit_window_sec"`
	UserLimitQty       int64              `json:"user_limit_qty"`
	MaxQtyPerOrder     int64              `json:"max_qty_per_order"`
	Status             int32              `json:"status"`
}

// adminLoginReq 是管理员登录请求体。
type adminLoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// adminRefreshReq 是管理员刷新令牌请求体。
type adminRefreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

// changeMyPasswordReq 是管理员修改本人密码请求体。
type changeMyPasswordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// adminCreateReq 是管理员创建请求体。
type adminCreateReq struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	DataScope   string `json:"data_scope"`
	Status      int32  `json:"status"`
}

// adminUpdateReq 是管理员更新请求体。
type adminUpdateReq struct {
	DisplayName string `json:"display_name"`
	DataScope   string `json:"data_scope"`
}

// adminStatusReq 是管理员状态设置请求体。
type adminStatusReq struct {
	Status int32 `json:"status"`
}

// adminResetPasswordReq 是管理员重置密码请求体。
type adminResetPasswordReq struct {
	NewPassword string `json:"new_password"`
}

// adminBindRolesReq 是管理员角色绑定请求体。
type adminBindRolesReq struct {
	RoleIDs []handlerx.JSONInt64 `json:"role_ids"`
}

// roleCreateReq 是角色创建请求体。
type roleCreateReq struct {
	RoleCode string `json:"role_code"`
	RoleName string `json:"role_name"`
	Status   int32  `json:"status"`
}

// roleUpdateReq 是角色更新请求体。
type roleUpdateReq struct {
	RoleName string `json:"role_name"`
	Status   int32  `json:"status"`
}

// roleDomainsReq 是角色领域绑定请求体。
type roleDomainsReq struct {
	Domains []string `json:"domains"`
}
