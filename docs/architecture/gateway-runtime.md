# 网关运行态（As-Is）

更新时间：2026-02-08

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

## admin-gateway

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

## 权限要点

- admin 路由统一走：`AuthRequired + RequireDomain`。
- 商品管理必须具备 `product_management` 领域角色。
- 订单管理必须具备 `order_management` 领域角色。
