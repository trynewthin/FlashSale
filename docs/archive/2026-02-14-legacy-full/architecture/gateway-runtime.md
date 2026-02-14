# 网关运行态（As-Is）

更新时间：2026-02-09

## user-gateway

- 用户接口：
  - `POST /api/v1/user/register`
  - `POST /api/v1/user/login`
  - `GET /api/v1/user/profile`
  - `PATCH /api/v1/user/nickname`
  - `DELETE /api/v1/user`
- 商品公开接口（匿名可访问）：
  - `GET /api/v1/products`
  - `GET /api/v1/products/{product_id}`
- 订单接口（登录用户）：
  - `POST /api/v1/orders`
  - `POST /api/v1/orders/{order_id}/pay-confirm`
  - `POST /api/v1/orders/{order_id}/cancel`
  - `POST /api/v1/orders/{order_id}/confirm-receipt`
  - `GET /api/v1/orders/{order_id}`
  - `GET /api/v1/orders`
- 秒杀接口：
  - `GET /api/v1/seckill/activities`（匿名）
  - `GET /api/v1/seckill/activities/{activity_id}`（匿名）
  - `POST /api/v1/seckill/activities/{activity_id}/purchase`（登录用户）
  - `POST /api/v1/seckill/activities/{activity_id}/track`（匿名）

## admin-gateway

- 管理员认证与自助接口：
  - `POST /api/v1/admin/auth/login`（匿名）
  - `POST /api/v1/admin/auth/refresh`（匿名）
  - `POST /api/v1/admin/auth/logout`
  - `GET /api/v1/admin/me`
  - `POST /api/v1/admin/me/password`
- 管理员与角色管理接口（`admin_management`）：
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
- 用户管理接口（`user_management`）：
  - `GET /api/v1/admin/users/{user_id}`
  - `PATCH /api/v1/admin/users/{user_id}/nickname`
  - `DELETE /api/v1/admin/users/{user_id}`
- 商品管理接口（`product_management`）：
  - `POST /api/v1/admin/products`
  - `PATCH /api/v1/admin/products/{product_id}`
  - `DELETE /api/v1/admin/products/{product_id}`
  - `GET /api/v1/admin/products/{product_id}`
  - `GET /api/v1/admin/products`
- 订单管理接口（`order_management`）：
  - `POST /api/v1/admin/orders/{order_id}/review`
  - `POST /api/v1/admin/orders/{order_id}/ship`
  - `GET /api/v1/admin/orders/{order_id}`
  - `GET /api/v1/admin/orders`
- 秒杀管理接口（`seckill_management`）：
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

## 权限要点

- admin 匿名登录/刷新接口不经过 `AuthRequired`。
- 其余 admin 路由统一走：`AuthRequired + RequireDomain`。
- 领域权限包含：`operations`、`user_management`、`product_management`、`order_management`、`seckill_management`、`admin_management`。
- 管理员模块读写细分策略在 `admin rpc` 服务端执行：
  - 写操作要求 `data_scope=all`
  - 读操作要求具备 `admin_management` 域
