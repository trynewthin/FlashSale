# 订单 RPC 运行态（As-Is）

更新时间：2026-02-08

## 服务入口

- 启动文件：`apps/order/rpc/order.go`
- 协议文件：`apps/order/rpc/order.proto`
- 默认监听：`0.0.0.0:8085`
- 关键配置：
  - `BaseConfigPath`：基础配置路径
  - `SnowflakeNode`：订单 ID 节点号（默认 `3`）
  - `ProductRPC.Target`：商品 RPC 地址（默认 `127.0.0.1:8084`）

## RPC 方法清单

### 用户侧

- `CreateOrder`
- `ConfirmPaymentAndInfo`
- `CancelOrder`
- `ConfirmReceipt`
- `GetOrderUser`
- `ListOrdersUser`

### 管理侧

- `ReviewOrderAdmin`
- `ShipOrderAdmin`
- `GetOrderAdmin`
- `ListOrdersAdmin`

## 订单状态模型

### 主订单状态（`order_status`）

- `10`：待支付（`pending_pay`）
- `20`：待审核（`pending_review`）
- `30`：待发货（`pending_ship`）
- `40`：已发货（`shipped`）
- `90`：已关闭（`closed`）

### 子状态

- 支付状态（`payment_status`）：`0=unpaid`、`1=paid`
- 审核状态（`review_status`）：`0=none`、`1=manual_pending`、`2=passed`、`3=rejected`、`4=timeout_reject`
- 发货状态（`shipping_status`）：`0=not_shipped`、`1=shipped`、`2=received`
- 退款状态（`refund_status`）：`0=none`、`1=refunding`、`2=refunded`
- 审核模式（`review_mode`）：`1=auto`、`2=manual`
- 库存回补标志（`stock_released`）：`0=pending`、`1=released`

### 订单来源（`order_source`）

- `1`：普通订单（`normal`）
- `2`：秒杀订单（`seckill`）

## 核心流程

1. 用户下单：`CreateOrder`
- 校验用户和商品参数。
- 调用 `product-rpc.GetProductPublic` 获取商品快照。
- 调用 `product-rpc.ReserveStockForOrder` 预扣库存（数量固定为 `1`）。
- 创建订单与初始事件 `order_created`。

2. 用户支付并确认收货信息：`ConfirmPaymentAndInfo`
- 校验支付信息与收货信息。
- 根据审核策略进入：
  - 自动审核通过：直接到待发货；
  - 人工审核：进入待审核并写入 `review_due_at`。

3. 审核：`ReviewOrderAdmin`
- 通过：待发货。
- 拒绝：订单关闭 + 进入退款中 + 回补库存。

4. 发货：`ShipOrderAdmin`
- 仅待发货状态可发货，写入运单号、发货人、发货时间。

5. 收货：`ConfirmReceipt`
- 用户确认收货后订单关闭（`close_reason=completed`）。

## 审核与超时策略

- 人工审核触发条件：
  - `order_source=seckill` 必须人工审核；
  - 或 `total_amount_cent >= 100000`（1000 元）人工审核。
- 超时参数：
  - 支付超时：15 分钟
  - 审核超时：30 分钟
  - 自动收货：7 天
  - 虚拟退款完成：1 分钟
- 超时任务由订单 RPC 进程内 ticker 每分钟推进一轮，处理：
  - 支付超时关单
  - 审核超时拒绝
  - 退款完成
  - 自动收货
  - 库存回补重试

## 库存协同与幂等

1. 预扣/回补全部通过 `product-rpc`，不跨库直写商品表。
2. product 侧库存接口使用 `idempotency_records` 落库幂等：
- scene：`product_stock_reserve` / `product_stock_release`
- 重复请求直接重放 `remain_stock`。
3. 订单侧补偿机制：
- 关单路径回补成功后标记 `stock_released=1`。
- 若回补失败，保留 `stock_released=0`，由超时任务重试直至成功。

## 鉴权与权限

- 用户 RPC 方法：要求 user token，且 `subject == user_id`。
- 管理 RPC 方法：要求 admin token 且具备 `order_management` 领域。
- 服务端会从 token 提取 `admin_id` 并覆盖请求中的 `admin_id`，避免伪造。

## 网关路由映射

### user-gateway（需要用户登录）

- `POST /api/v1/orders` -> `CreateOrder`
- `POST /api/v1/orders/{order_id}/pay-confirm` -> `ConfirmPaymentAndInfo`
- `POST /api/v1/orders/{order_id}/cancel` -> `CancelOrder`
- `POST /api/v1/orders/{order_id}/confirm-receipt` -> `ConfirmReceipt`
- `GET /api/v1/orders/{order_id}` -> `GetOrderUser`
- `GET /api/v1/orders` -> `ListOrdersUser`

### admin-gateway（需要 `order_management`）

- `POST /api/v1/admin/orders/{order_id}/review` -> `ReviewOrderAdmin`
- `POST /api/v1/admin/orders/{order_id}/ship` -> `ShipOrderAdmin`
- `GET /api/v1/admin/orders/{order_id}` -> `GetOrderAdmin`
- `GET /api/v1/admin/orders` -> `ListOrdersAdmin`

## 数据库迁移

- `deploy/migrations/order/000002__orders.up.sql`
- `deploy/migrations/order/000002__orders.down.sql`

关键表：
- `orders`：订单聚合、状态、超时字段、回补标记
- `order_events`：事件流水，记录状态变更轨迹

## 错误码（订单相关）

- `ORDER_NOT_FOUND`
- `ORDER_INVALID_STATE`
- `ORDER_OUT_OF_STOCK`
- `ORDER_PAY_TIMEOUT`
- `ORDER_REVIEW_TIMEOUT`
- `ORDER_REVIEW_REJECTED`
- `ORDER_ALREADY_CLOSED`
- `ORDER_ALREADY_PAID`
- `ORDER_INVALID_RECEIVER_INFO`
