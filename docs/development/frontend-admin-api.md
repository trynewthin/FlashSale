# 管理端前端接口文档（基于当前后端实现）

更新时间：2026-02-12  
对应网关：`apps/gateway/admin`  
事实来源：`admin/user/product/order/seckill` 网关路由 + handler + RPC proto

## 1. 通用约定

1. 基础路径：`/api/v1/admin`
2. 鉴权方式：`Authorization: Bearer <admin_access_token>`
3. 响应包裹格式：

```json
{
  "code": "OK",
  "message": "success",
  "data": {}
}
```

4. JSON 解析严格模式：后端禁止未知字段，额外字段会返回 `400`
5. 分页默认值：`page=1`、`page_size=20`、`page_size<=100`
6. 成功 HTTP 状态：当前统一 `200`
7. 时间字段：`*_unix` 为秒级时间戳；金额字段：`*_cent` 为分

## 2. 权限与数据范围

### 2.1 Domain 常量

- `operations`
- `user_management`
- `product_management`
- `order_management`
- `order_review_management`
- `seckill_management`
- `admin_management`

### 2.2 data_scope

- `all`：允许全量数据访问
- `self`：仅允许本人范围

### 2.3 关键权限说明

1. 网关做第一层校验（`AuthRequired + RequireDomain/RequireAnyDomain`）。
2. `admin_management` 相关接口在 Admin RPC 服务端进一步要求 `data_scope=all`（包含读接口）；`self` 会返回 `403`。
3. 删除/状态变更等敏感操作的 `admin_id/operator_admin_id` 最终由服务端从 token 注入覆盖，前端无需传递。

## 3. 路由总览

### 3.1 认证与个人中心

- `POST /api/v1/admin/auth/login`（匿名）
- `POST /api/v1/admin/auth/refresh`（匿名）
- `POST /api/v1/admin/auth/logout`（登录）
- `GET /api/v1/admin/me`（登录）
- `POST /api/v1/admin/me/password`（登录）

### 3.2 管理员与角色（`admin_management`）

- `POST /api/v1/admin/admins`
- `PATCH /api/v1/admin/admins/{admin_id}`
- `POST /api/v1/admin/admins/{admin_id}/status`
- `POST /api/v1/admin/admins/{admin_id}/reset-password`
- `DELETE /api/v1/admin/admins/{admin_id}`
- `GET /api/v1/admin/admins/{admin_id}`
- `GET /api/v1/admin/admins`
- `POST /api/v1/admin/admins/{admin_id}/roles`
- `POST /api/v1/admin/roles`
- `PATCH /api/v1/admin/roles/{role_id}`
- `DELETE /api/v1/admin/roles/{role_id}`
- `GET /api/v1/admin/roles/{role_id}`
- `GET /api/v1/admin/roles`
- `POST /api/v1/admin/roles/{role_id}/domains`
- `GET /api/v1/admin/audit-logs`

### 3.3 用户管理（`user_management`）

- `GET /api/v1/admin/users/{user_id}`
- `PATCH /api/v1/admin/users/{user_id}/nickname`
- `DELETE /api/v1/admin/users/{user_id}`

### 3.4 商品管理（`product_management`）

- `POST /api/v1/admin/products`
- `PATCH /api/v1/admin/products/{product_id}`
- `DELETE /api/v1/admin/products/{product_id}`
- `GET /api/v1/admin/products/{product_id}`
- `GET /api/v1/admin/products`

### 3.5 订单管理

- `POST /api/v1/admin/orders/{order_id}/review`（`order_management` 或 `order_review_management`）
- `POST /api/v1/admin/orders/{order_id}/ship`（仅 `order_management`）
- `GET /api/v1/admin/orders/{order_id}`（`order_management` 或 `order_review_management`）
- `GET /api/v1/admin/orders`（`order_management` 或 `order_review_management`）

### 3.6 秒杀活动管理（`seckill_management`）

- `POST /api/v1/admin/seckill/activities`
- `PATCH /api/v1/admin/seckill/activities/{activity_id}`
- `DELETE /api/v1/admin/seckill/activities/{activity_id}`
- `GET /api/v1/admin/seckill/activities/{activity_id}`
- `GET /api/v1/admin/seckill/activities`
- `POST /api/v1/admin/seckill/activities/{activity_id}/items`
- `PUT /api/v1/admin/seckill/activities/{activity_id}/items/{item_id}`
- `DELETE /api/v1/admin/seckill/activities/{activity_id}/items/{item_id}`
- `POST /api/v1/admin/seckill/activities/{activity_id}/publish`
- `POST /api/v1/admin/seckill/activities/{activity_id}/offline`
- `GET /api/v1/admin/seckill/activities/{activity_id}/traffic`
- `GET /api/v1/admin/seckill/activities/{activity_id}/orders`

### 3.7 运营探针（`operations`）

- `GET /api/v1/admin/ping`

## 4. 请求体定义（前端需按此发送）

### 4.1 认证与个人中心

- `POST /auth/login`
  - `username`
  - `password`
- `POST /auth/refresh`
  - `refresh_token`
- `POST /auth/logout`
  - `refresh_token`
- `POST /me/password`
  - `old_password`
  - `new_password`

### 4.2 管理员与角色

- `POST /admins`
  - `username`（`4~32`，`a-z0-9_`）
  - `display_name`（`1~64`）
  - `password`（`8~32`，字母+数字）
  - `data_scope`（`all/self`）
  - `status`（`0/1`）
- `PATCH /admins/{admin_id}`
  - `display_name`
  - `data_scope`
- `POST /admins/{admin_id}/status`
  - `status`
- `POST /admins/{admin_id}/reset-password`
  - `new_password`
- `POST /admins/{admin_id}/roles`
  - `role_ids[]`
- `POST /roles`
  - `role_code`（`4~32`，`a-z0-9_`）
  - `role_name`
  - `status`（`0/1`）
- `PATCH /roles/{role_id}`
  - `role_name`
  - `status`
- `POST /roles/{role_id}/domains`
  - `domains[]`（需为合法 domain 值）

### 4.3 用户管理

- `PATCH /users/{user_id}/nickname`
  - `nickname`

### 4.4 商品管理

- `POST /products` 与 `PATCH /products/{product_id}` 请求体一致：
  - `name`（1~100）
  - `main_image`（非空，<=512）
  - `description`（<=2000）
  - `price_cent`（>0）
  - `stock`（>=0）
  - `status`（`0=off_shelf,1=on_shelf`）

### 4.5 订单管理

- `POST /orders/{order_id}/review`
  - `approved`（`true/false`）
  - `reason`（可空，建议拒绝时填写）
- `POST /orders/{order_id}/ship`
  - `tracking_no`（非空，1~64）

### 4.6 秒杀管理

- `POST /seckill/activities` 与 `PATCH /seckill/activities/{activity_id}` 请求体一致：
  - `title`（1~120）
  - `description`（<=4000）
  - `style_config_json`（可空；非空需合法 JSON，<=8192）
  - `start_at_unix`
  - `end_at_unix`（必须大于开始时间）

- `POST /seckill/activities/{activity_id}/items` 与 `PUT /.../items/{item_id}` 请求体一致：
  - `product_id`
  - `seckill_price_cent`（>0）
  - `reserved_stock_total`（>0）
  - `user_limit_mode`（`0=none,1=window_completed`）
  - `user_limit_window_sec`（窗口模式需 >0）
  - `user_limit_qty`（窗口模式需 >0）
  - `max_qty_per_order`（1~100）
  - `status`（`0=disabled,1=enabled`）

### 4.7 列表与查询类接口参数

1. `GET /api/v1/admin/admins`
- `page`
- `page_size`
- `keyword`（用户名/显示名模糊匹配）
- `status`（`-1` 不过滤，`0/1` 过滤）

2. `GET /api/v1/admin/roles`
- `page`
- `page_size`
- `keyword`（角色编码/名称模糊匹配）
- `status`（`-1` 不过滤，`0/1` 过滤）

3. `GET /api/v1/admin/audit-logs`
- `page`
- `page_size`
- `admin_id`
- `action`
- `target_type`
- `target_id`

4. `GET /api/v1/admin/products`
- `page`
- `page_size`
- `keyword`
- `include_deleted`（`true/false`）

5. `GET /api/v1/admin/orders`
- `page`
- `page_size`
- `order_status`（默认 `0` 不过滤）
- `review_status`（默认 `0` 不过滤）
- `user_id`
- `order_no`

6. `GET /api/v1/admin/seckill/activities`
- `page`
- `page_size`
- `keyword`
- `status`（`-1` 不过滤；`0=draft,1=published,2=offline`）
- `include_deleted`（`true/false`）

7. `GET /api/v1/admin/seckill/activities/{activity_id}/traffic`
- `activity_item_id`（可选；`0` 表示活动整体）
- `from_minute_unix`
- `to_minute_unix`

8. `GET /api/v1/admin/seckill/activities/{activity_id}/orders`
- `page`
- `page_size`
- `order_status`（默认 `0` 不过滤）
- `payment_status`（默认 `0` 不过滤）
- `user_id`

## 5. 核心响应模型

### 5.1 管理员模块

- `AdminAuthResp`
  - `admin`（`AdminView`）
  - `access_token`
  - `access_expires_in_sec`
  - `refresh_token`
  - `refresh_expires_in_sec`

- `AdminView`
  - `admin_id`
  - `username`
  - `display_name`
  - `status`
  - `data_scope`
  - `is_super_admin`
  - `last_login_at_unix`
  - `last_login_ip`
  - `role_ids[]`
  - `domains[]`
  - `created_at_unix`
  - `updated_at_unix`

- `RoleView`
  - `role_id`
  - `role_code`
  - `role_name`
  - `status`
  - `is_system`
  - `domains[]`
  - `created_at_unix`
  - `updated_at_unix`

- `AdminAuditLogView`
  - `log_id`
  - `admin_id`
  - `action`
  - `target_type`
  - `target_id`
  - `result`
  - `request_id`
  - `ip`
  - `user_agent`
  - `detail_json`
  - `created_at_unix`

### 5.2 商品、订单、秒杀模型

- 商品管理返回：`AdminProduct`（含 `stock/status`）
- 订单管理返回：`OrderView`（状态字段定义同用户端文档）
- 秒杀管理返回：`ActivityAdmin`、`ActivityItemAdmin`、`TrafficBucket`、`ActivityOrderView`

列表响应统一包含：
- `items` 或 `list`（数组）
- `total`
- `page`
- `page_size`

秒杀状态值：
- 活动 `status`：`0=draft,1=published,2=offline`
- 活动商品 `status`：`0=disabled,1=enabled`
- 限购模式 `user_limit_mode`：`0=none,1=window_completed`

## 6. 常见错误码（管理端高频）

- 鉴权：`AUTH_UNAUTHORIZED`、`AUTH_FORBIDDEN`
- 通用参数：`SYS_BAD_REQUEST`
- 管理员模块：`ADMIN_NOT_FOUND`、`ADMIN_USERNAME_ALREADY_EXISTS`、`ADMIN_ACCOUNT_DISABLED`、`ADMIN_ACCOUNT_LOCKED`、`ADMIN_ROLE_NOT_FOUND`、`ADMIN_ROLE_CODE_ALREADY_EXISTS`、`ADMIN_INVALID_SCOPE`、`ADMIN_REFRESH_TOKEN_INVALID`
- 用户模块：`USER_NOT_FOUND`
- 商品模块：`PRODUCT_NOT_FOUND`、`PRODUCT_INVALID_STATUS`、`PRODUCT_SKU_ALREADY_EXISTS`
- 订单模块：`ORDER_NOT_FOUND`、`ORDER_INVALID_STATE`、`ORDER_REVIEW_REJECTED` 等
- 秒杀模块：`SECKILL_ACTIVITY_NOT_FOUND`、`SECKILL_INVALID_CONFIG`、`SECKILL_ITEM_NOT_FOUND` 等

## 7. 对齐结论（网关 vs 后端）

1. 管理网关路由与后端 RPC 接口已对齐，未发现断链。
2. `admin_id/operator_admin_id` 在后端服务端按 token 覆盖，前端不需要也不应自行构造。
3. 订单审核权限已拆分：审核接口支持 `order_review_management`，发货仍需 `order_management`。
4. `admin_management` 接口除网关 domain 校验外，RPC 侧还要求 `data_scope=all`，前端需要针对 403 做明确提示。
