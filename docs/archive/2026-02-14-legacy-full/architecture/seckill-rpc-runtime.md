# 秒杀 RPC 运行态（As-Is）

更新时间：2026-02-09

## 服务入口

- 启动文件：`apps/seckill/rpc/seckill.go`
- 协议文件：`apps/seckill/rpc/seckill.proto`
- 默认监听：`0.0.0.0:8086`
- 关键配置：
  - `BaseConfigPath`：基础配置路径
  - `SnowflakeNode`：节点号（默认 `6`）
  - `ProductRPC.Target`：商品 RPC 地址（默认 `127.0.0.1:8084`）
  - `OrderRPC.Target`：订单 RPC 地址（默认 `127.0.0.1:8085`）

## RPC 方法清单

### 管理侧

- `CreateActivity`
- `UpdateActivity`
- `DeleteActivity`
- `GetActivityAdmin`
- `ListActivitiesAdmin`
- `UpsertActivityItem`
- `RemoveActivityItem`
- `PublishActivity`
- `OfflineActivity`
- `GetActivityTraffic`
- `ListActivityOrders`

### 用户侧

- `ListActivitiesPublic`
- `GetActivityPublic`
- `Purchase`
- `TrackEvent`

## 鉴权模型

- 管理侧：要求 admin token + `seckill_management` 域。
- 用户抢购：要求 user token，且 `user_id == token.subject`。
- 公开接口：活动列表/详情与埋点允许匿名调用。

## 活动与商品模型

活动状态：

- `0`：草稿（draft）
- `1`：已发布（published）
- `2`：已下线（offline）

活动商品状态：

- `0`：禁用
- `1`：启用

限购模式：

- `0`：不限购（`NONE`）
- `1`：窗口期已完成限购（`WINDOW_COMPLETED`）

## 核心流程

### 1) 发布活动（库存预占）

1. 校验活动配置与时间窗口。
2. 遍历启用商品，调用 `product.ReserveStockForActivity` 预占库存。
3. 刷新 `available_stock` 并预热 Redis 库存。
4. 标记活动为 `published`。
5. 任一预占失败时执行已预占项回滚。

### 2) 下线活动（库存释放）

1. 仅对 `published` 活动执行库存释放。
2. 对每个商品释放未售库存：`product.ReleaseStockForActivity`。
3. 任一释放失败则整体返回错误，活动状态不切换。
4. 全量释放成功后标记 `offline`。

### 3) 用户抢购

1. 网关校验 `activity_id/activity_item_id/quantity/idempotency_key`。
2. Redis Lua 做热路径原子校验与扣减（库存、限购、幂等）。
3. 调 `order.CreateOrderFromSeckill` 建单。
4. 建单失败按策略补偿（可判定失败时回补，不确定错误返回“处理中”）。
5. 记录 `purchase_attempt/success/fail` 流量事件。

### 4) 订单状态回流

1. 消费 `seckill.order.state` 事件。
2. 同步 `seckill_order_links` 状态。
3. 对关闭订单按原因执行活动库存回补。
4. 对“已收货关闭”写入窗口限购完成计数。

## 数据与中间件

### MySQL（flash_seckill）

- `seckill_activities`
- `seckill_activity_items`
- `seckill_order_links`
- `seckill_traffic_raw_events`
- `seckill_traffic_agg_minute`
- `seckill_stock_ledger`
- `outbox_events`（基础设施表）
- `idempotency_records`（基础设施表）

### Redis（热点键）

- 商品库存键：`seckill:item:{item_id}:stock`
- 用户活动累计：`seckill:item:{item_id}:user:{uid}:bought`
- 用户窗口完成集：`seckill:item:{item_id}:user:{uid}:completed:zset`
- 请求幂等键：`seckill:req:{activity_id}:{item_id}:{uid}:{idempotency_key}`

### Kafka

- 流量事件：`seckill.traffic.raw`
- 订单状态回流：`seckill.order.state`
- 死信：`seckill.traffic.dlq`

## 网关路由映射

### user-gateway

- `GET /api/v1/seckill/activities` -> `ListActivitiesPublic`
- `GET /api/v1/seckill/activities/{activity_id}` -> `GetActivityPublic`
- `POST /api/v1/seckill/activities/{activity_id}/purchase` -> `Purchase`
- `POST /api/v1/seckill/activities/{activity_id}/track` -> `TrackEvent`

### admin-gateway（`seckill_management`）

- `POST /api/v1/admin/seckill/activities` -> `CreateActivity`
- `PATCH /api/v1/admin/seckill/activities/{activity_id}` -> `UpdateActivity`
- `DELETE /api/v1/admin/seckill/activities/{activity_id}` -> `DeleteActivity`
- `GET /api/v1/admin/seckill/activities/{activity_id}` -> `GetActivityAdmin`
- `GET /api/v1/admin/seckill/activities` -> `ListActivitiesAdmin`
- `POST /api/v1/admin/seckill/activities/{activity_id}/items` -> `UpsertActivityItem(item_id=0)`
- `PUT /api/v1/admin/seckill/activities/{activity_id}/items/{item_id}` -> `UpsertActivityItem`
- `DELETE /api/v1/admin/seckill/activities/{activity_id}/items/{item_id}` -> `RemoveActivityItem`
- `POST /api/v1/admin/seckill/activities/{activity_id}/publish` -> `PublishActivity`
- `POST /api/v1/admin/seckill/activities/{activity_id}/offline` -> `OfflineActivity`
- `GET /api/v1/admin/seckill/activities/{activity_id}/traffic` -> `GetActivityTraffic`
- `GET /api/v1/admin/seckill/activities/{activity_id}/orders` -> `ListActivityOrders`

## 迁移文件

- `deploy/migrations/seckill/000001__infra_tables.up.sql`
- `deploy/migrations/seckill/000001__infra_tables.down.sql`
- `deploy/migrations/seckill/000002__seckill_core.up.sql`
- `deploy/migrations/seckill/000002__seckill_core.down.sql`

关联迁移（订单侧）：

- `deploy/migrations/order/000003__orders_seckill_fields.up.sql`
- `deploy/migrations/order/000003__orders_seckill_fields.down.sql`

## 已知边界

- 审核管理员尚未独立权限域（当前仍复用订单管理域）。
- 匿名埋点接口可传 `user_id`，建议后续收敛为“匿名仅 client_id，登录态由服务端补 user_id”。
