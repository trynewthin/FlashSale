# 用户端 Hooks 设计文档

更新时间：2026-02-12  
适用模块：`frontend-user`  
接口事实源：`docs/development/frontend-user-api.md`

## 1. 目标与边界

1. 目标：把用户端后端能力抽象成稳定、可复用的 Hooks 层，统一处理鉴权、请求、缓存、错误与幂等。
2. 范围：用户账号、商品浏览、用户订单、秒杀活动。
3. 非范围：UI 组件实现、样式体系、路由方案细节。

## 2. 设计前提

1. 技术栈假设：React + TypeScript + TanStack Query（或同类 query 库）。
2. 接口返回统一为 `{ code, message, data }`，`code=OK` 才视为成功。
3. 用户端无 refresh 接口；access token 过期后走重新登录流程。
4. 秒杀抢购与埋点接口要求幂等键。

## 3. 分层结构建议

```text
src/
  api/
    client.ts
    user.ts
    product.ts
    order.ts
    seckill.ts
  hooks/
    common/
      useApiError.ts
      useIdempotencyKey.ts
    user/
      useAuthHooks.ts
      useProductHooks.ts
      useOrderHooks.ts
      useSeckillHooks.ts
    queryKeys.ts
  stores/
    authStore.ts
```

## 4. 公共 Hooks 设计

1. `useApiError()`
- 作用：把后端 `code/message` 映射为前端错误对象与提示文案。
- 输出：`toUserMessage(err)`、`isCode(err, code)`。

2. `useIdempotencyKey()`
- 作用：生成请求级幂等键（推荐 `uuid`）。
- 用法：用于 `purchase`、`track`。
- 约束：重试同一次业务动作必须复用同一 key。

3. `useAuthedRequest()`
- 作用：为需登录接口自动注入 `Authorization`。
- 行为：遇到 `AUTH_UNAUTHORIZED` 清理本地会话并触发登录态失效回调。
- 约束：登录/退出时必须执行会话级缓存清理，避免跨账号缓存串读。

## 5. 业务 Hooks 设计

## 5.1 账号模块 Hooks

1. `useRegisterMutation()`
- 接口：`POST /api/v1/user/register`
- 入参：`phone/password/nickname`
- 成功：写入 `access_token`，更新 `authStore`。

2. `useLoginMutation()`
- 接口：`POST /api/v1/user/login`
- 入参：`phone/password`
- 成功：写入 `access_token` 与用户基础信息。

3. `useUserProfileQuery()`
- 接口：`GET /api/v1/user/profile`
- QueryKey：`['user', userId, 'profile']`

4. `useUpdateNicknameMutation()`
- 接口：`PATCH /api/v1/user/nickname`
- 成功失效：`['user','profile']`

5. `useDeleteUserMutation()`
- 接口：`DELETE /api/v1/user`
- 成功：清理本地会话、跳转登录页。

## 5.2 商品模块 Hooks

1. `useProductsQuery(params)`
- 接口：`GET /api/v1/products`
- 入参：`page/page_size/keyword`
- QueryKey：`['products','list',params]`

2. `useProductDetailQuery(productId)`
- 接口：`GET /api/v1/products/{product_id}`
- QueryKey：`['products','detail',productId]`

## 5.3 订单模块 Hooks

1. `useCreateOrderMutation()`
- 接口：`POST /api/v1/orders`
- 入参：`product_id/order_source`
- 成功失效：`['orders','list']`

2. `useOrderListQuery(params)`
- 接口：`GET /api/v1/orders`
- 入参：`page/page_size/order_status`
- QueryKey：`['orders', userId, 'list', params]`

3. `useOrderDetailQuery(orderId)`
- 接口：`GET /api/v1/orders/{order_id}`
- QueryKey：`['orders', userId, 'detail', orderId]`

4. `useConfirmPaymentAndInfoMutation(orderId)`
- 接口：`POST /api/v1/orders/{order_id}/pay-confirm`
- 成功失效：`['orders', userId, 'detail', orderId]`、`['orders', userId, 'list']`

5. `useCancelOrderMutation(orderId)`
- 接口：`POST /api/v1/orders/{order_id}/cancel`
- 成功失效：`['orders', userId, 'detail', orderId]`、`['orders', userId, 'list']`

6. `useConfirmReceiptMutation(orderId)`
- 接口：`POST /api/v1/orders/{order_id}/confirm-receipt`
- 成功失效：`['orders', userId, 'detail', orderId]`、`['orders', userId, 'list']`

## 5.4 秒杀模块 Hooks

1. `useSeckillActivitiesQuery(params)`
- 接口：`GET /api/v1/seckill/activities`
- QueryKey：`['seckill','activities',params]`

2. `useSeckillActivityDetailQuery(activityId)`
- 接口：`GET /api/v1/seckill/activities/{activity_id}`
- QueryKey：`['seckill','activity',activityId]`

3. `useSeckillPurchaseMutation(activityId)`
- 接口：`POST /api/v1/seckill/activities/{activity_id}/purchase`
- 入参：`activity_item_id/quantity`
- 内部：自动注入 `idempotency_key`
- 成功失效：`['orders', userId, 'list']`、`['seckill','activity',activityId]`

4. `useSeckillTrackMutation(activityId)`
- 接口：`POST /api/v1/seckill/activities/{activity_id}/track`
- 入参：`activity_item_id/event_type/client_id/occurred_at_unix`
- 内部：自动注入 `idempotency_key`
- 建议：失败不打断主流程（降级埋点）。

## 6. Query Key 规范

1. `['user', userId, 'profile']`
2. `['products','list',params]`
3. `['products','detail',productId]`
4. `['orders', userId, 'list', params]`
5. `['orders', userId, 'detail', orderId]`
6. `['seckill','activities',params]`
7. `['seckill','activity',activityId]`

## 7. 错误与重试策略

1. 查询类接口：可启用有限重试（建议 `1~2` 次）。
2. 变更类接口：默认不自动重试，避免重复提交风险。
3. 秒杀冲突码（如 `SECKILL_PURCHASE_CONFLICT`）：提示“请求处理中/请稍后重试”，并引导用户去订单页查询结果。
4. 鉴权错误：统一触发会话失效处理。
5. 登录成功、退出登录、注销账号时，必须清理当前主体相关缓存（建议 `queryClient.clear()` 或按 `userId` 批量移除）。

## 8. 实现要点（强约束）

1. 写接口请求体必须严格对齐文档字段，禁止额外字段。
2. 订单状态展示必须按后端枚举渲染，不在前端自定义状态值。
3. 秒杀购买和埋点必须携带幂等键。
4. 用户端不要传递 `user_id`（后端通过 token 识别）。
5. 所有私有态 Query Key 必须包含 `userId`，避免同端多账号切换导致旧缓存串读。

## 9. 验收清单

1. 所有 Hooks 都有对应类型定义（请求、响应、错误码）。
2. 关键 mutation 均实现 cache 失效。
3. 鉴权失效后能统一回收会话。
4. 秒杀购买重复点击不会因前端重试导致重复下单。
