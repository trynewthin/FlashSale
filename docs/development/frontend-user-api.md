# 用户端前端接口文档（基于当前后端实现）

更新时间：2026-02-12  
对应网关：`apps/gateway/user`  
事实来源：`user/product/order/seckill` 网关路由 + handler + RPC proto

## 1. 通用约定

1. 基础路径：`/api/v1`
2. 鉴权方式：`Authorization: Bearer <user_access_token>`（仅需要登录的接口）
3. 响应包裹格式：

```json
{
  "code": "OK",
  "message": "success",
  "data": {}
}
```

失败时：

```json
{
  "code": "业务错误码",
  "message": "错误信息"
}
```

4. JSON 解析严格模式：后端 `DisallowUnknownFields`，请求体含未知字段会直接 `400`
5. 分页默认值：`page=1`、`page_size=20`、`page_size<=100`
6. 金额单位：`*_cent`，单位为分（`int64`）
7. 时间单位：`*_unix`，Unix 秒级时间戳
8. 成功 HTTP 状态：当前统一返回 `200`

## 2. 枚举与状态值

### 2.1 订单字段枚举

- `order_source`：`1=normal`，`2=seckill`
- `order_status`：`10=待支付`，`20=待审核`，`30=待发货`，`40=已发货`，`90=已关闭`
- `payment_status`：`0=未支付`，`1=已支付`
- `review_status`：`0=无`，`1=待人工审核`，`2=审核通过`，`3=审核拒绝`，`4=审核超时拒绝`
- `shipping_status`：`0=未发货`，`1=已发货`，`2=已收货`
- `refund_status`：`0=无`，`1=退款中`，`2=已退款`

### 2.2 秒杀事件类型（`event_type`）

- `pv`
- `click`
- `purchase_attempt`
- `purchase_success`
- `purchase_fail`
- `pay_success`
- `order_closed`

## 3. 接口清单

### 3.1 用户账号

1. `POST /api/v1/user/register`（匿名）
请求体：
- `phone`：手机号，格式 `^1[3-9]\d{9}$`
- `password`：8~32 位，且包含字母+数字
- `nickname`：可空，空值会归一为“新用户”，最大 32 字符

响应 `data`：`AuthResp`
- `user_id`
- `phone`
- `nickname`
- `access_token`
- `expires_in_sec`

2. `POST /api/v1/user/login`（匿名）
请求体：
- `phone`
- `password`

响应 `data`：`AuthResp`

3. `GET /api/v1/user/profile`（需登录）
响应 `data`：`ProfileResp`
- `user_id`
- `phone`
- `nickname`

4. `PATCH /api/v1/user/nickname`（需登录）
请求体：
- `nickname`：最大 32 字符，空值会归一为“新用户”

响应 `data`：`ProfileResp`

5. `DELETE /api/v1/user`（需登录）
响应 `data`：
- `user_id`

### 3.2 商品浏览（公开）

1. `GET /api/v1/products`
Query：
- `page`
- `page_size`
- `keyword`（按名称模糊匹配）

响应 `data`：`ListProductsPublicResp`
- `list[]`：`PublicProduct`
- `total`
- `page`
- `page_size`

`PublicProduct` 字段：
- `product_id`
- `sku_code`
- `name`
- `main_image`
- `description`
- `price_cent`
- `in_stock`（布尔，不返回真实库存数）

2. `GET /api/v1/products/{product_id}`
响应 `data`：`GetProductPublicResp`
- `product`：`PublicProduct`

### 3.3 用户订单（需登录）

1. `POST /api/v1/orders`
请求体：
- `product_id`（>0）
- `order_source`（可传 `1/2`，传 `0` 按普通订单处理）

响应 `data`：`CreateOrderResp`
- `order`：`OrderView`

2. `POST /api/v1/orders/{order_id}/pay-confirm`
请求体：
- `pay_channel`（非空）
- `pay_reference`（非空）
- `receiver_name`（非空，1~64）
- `receiver_phone`（非空，格式校验）
- `receiver_address`（非空，1~512）
- `buyer_remark`（可空，<=200）

响应 `data`：`ConfirmPaymentAndInfoResp`
- `order`：`OrderView`

3. `POST /api/v1/orders/{order_id}/cancel`
请求体：
- `reason`（可空字符串）

响应 `data`：`CancelOrderResp`
- `order`：`OrderView`

4. `POST /api/v1/orders/{order_id}/confirm-receipt`
响应 `data`：`ConfirmReceiptResp`
- `order`：`OrderView`

5. `GET /api/v1/orders/{order_id}`
响应 `data`：`GetOrderUserResp`
- `order`：`OrderView`

6. `GET /api/v1/orders`
Query：
- `page`
- `page_size`
- `order_status`（默认 `0` 表示不过滤）

响应 `data`：`ListOrdersUserResp`
- `list[]`：`OrderView`
- `total`
- `page`
- `page_size`

`OrderView` 核心字段：
- `order_id`、`order_no`、`user_id`
- `order_source`、`product_id`、`sku_code`、`product_name`、`main_image`
- `unit_price_cent`、`quantity`、`total_amount_cent`
- `order_status`、`payment_status`、`review_status`、`shipping_status`、`refund_status`、`review_mode`
- `pay_channel`、`pay_reference`
- `receiver_name`、`receiver_phone`、`receiver_address`、`buyer_remark`
- `review_due_at_unix`、`reviewed_at_unix`、`reviewed_by`、`review_reason`
- `shipped_at_unix`、`shipped_by`、`tracking_no`
- `refund_due_at_unix`、`refunded_at_unix`
- `closed_at_unix`、`close_reason`
- `created_at_unix`、`updated_at_unix`
- `seckill_activity_id`、`seckill_activity_item_id`

### 3.4 秒杀活动（公开+登录）

1. `GET /api/v1/seckill/activities`（公开）
Query：
- `page`
- `page_size`

响应 `data`：`ListActivitiesPublicResp`
- `list[]`：`ActivityPublic`
- `total`
- `page`
- `page_size`

2. `GET /api/v1/seckill/activities/{activity_id}`（公开）
响应 `data`：`GetActivityPublicResp`
- `activity`：`ActivityPublic`

`ActivityPublic` 字段：
- `activity_id`
- `title`
- `description`
- `style_config_json`
- `start_at_unix`
- `end_at_unix`
- `status`（`0=draft,1=published,2=offline`）
- `items[]`：`ActivityItemPublic`

`ActivityItemPublic` 字段：
- `item_id`
- `product_id`
- `sku_code`
- `snapshot_name`
- `snapshot_main_image`
- `origin_price_cent`
- `seckill_price_cent`
- `in_stock`
- `max_qty_per_order`

3. `POST /api/v1/seckill/activities/{activity_id}/purchase`（需登录）
请求体：
- `activity_item_id`（>0）
- `quantity`（>0）
- `idempotency_key`（非空，建议 UUID）

响应 `data`：`PurchaseResp`
- `activity_id`
- `activity_item_id`
- `order_id`
- `order_no`

说明：
- 受秒杀限流中间件影响（可由环境变量开启/关闭）
- 同幂等键重复请求会触发冲突保护

4. `POST /api/v1/seckill/activities/{activity_id}/track`（匿名可用）
请求体：
- `activity_item_id`（>0）
- `event_type`（见 2.2）
- `client_id`（匿名端建议传设备标识）
- `idempotency_key`（非空）
- `occurred_at_unix`（可空，空则服务端取当前时间）

响应 `data`：`TrackEventResp`
- `accepted`（`true/false`）

说明：
- 登录态下 `user_id` 由网关从 token 注入，客户端无需也不能指定
- 接口受独立秒杀埋点限流中间件控制（可配置）

## 4. 常见错误码（用户端高频）

- 鉴权类：`AUTH_UNAUTHORIZED`、`AUTH_FORBIDDEN`
- 参数类：`SYS_BAD_REQUEST`
- 用户类：`USER_NOT_FOUND`、`AUTH_INVALID_PHONE`、`AUTH_WEAK_PASSWORD`、`AUTH_INVALID_CREDENTIALS`
- 商品类：`PRODUCT_NOT_FOUND`
- 订单类：`ORDER_NOT_FOUND`、`ORDER_INVALID_STATE`、`ORDER_OUT_OF_STOCK`、`ORDER_ALREADY_PAID`、`ORDER_ALREADY_CLOSED`、`ORDER_PAY_TIMEOUT`、`ORDER_REVIEW_TIMEOUT`、`ORDER_REVIEW_REJECTED`
- 秒杀类：`SECKILL_ACTIVITY_NOT_FOUND`、`SECKILL_ACTIVITY_NOT_PUBLISHED`、`SECKILL_ACTIVITY_NOT_STARTED`、`SECKILL_ACTIVITY_ENDED`、`SECKILL_ITEM_NOT_FOUND`、`SECKILL_OUT_OF_STOCK`、`SECKILL_LIMIT_EXCEEDED`、`SECKILL_PURCHASE_CONFLICT`

## 5. 对齐结论（网关 vs 后端）

1. 用户网关路由与后端 RPC 方法已逐项对齐，未发现调用断链。
2. 用户侧所有 JSON 写接口均为严格解码，前端提交字段必须与文档一致。
3. 秒杀抢购与埋点链路均有幂等键要求，前端需统一生成并复用请求级幂等键。
