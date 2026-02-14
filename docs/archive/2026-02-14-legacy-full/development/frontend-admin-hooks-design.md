# 管理端 Hooks 设计文档

更新时间：2026-02-12  
适用模块：`frontend-ops`  
接口事实源：`docs/development/frontend-admin-api.md`

## 1. 目标与边界

1. 目标：构建管理端统一 Hooks 层，承载认证、RBAC、管理员中心与业务管理能力。
2. 范围：管理员认证、管理员与角色管理、用户管理、商品管理、订单管理、秒杀活动管理。
3. 非范围：页面组件、UI 设计系统、菜单视觉实现。

## 2. 关键约束（必须遵守）

1. `admin_id/operator_admin_id` 由后端从 token 注入，前端不要传。
2. `admin_management` 接口在后端要求 `data_scope=all`，前端需处理 `403`。
3. 订单审核和发货权限不同：
- 审核：`order_management` 或 `order_review_management`
- 发货：仅 `order_management`
4. 所有写接口为严格 JSON 解码，字段必须精确对齐。

## 3. 分层结构建议

```text
src/
  api/
    adminClient.ts
    adminAuth.ts
    adminUser.ts
    adminRole.ts
    userMgmt.ts
    productMgmt.ts
    orderMgmt.ts
    seckillMgmt.ts
  hooks/
    common/
      useApiError.ts
      usePageQuery.ts
    auth/
      useAdminSession.ts
      useAdminPermission.ts
    admin/
      useAdminAccountHooks.ts
      useRoleHooks.ts
      useAuditHooks.ts
    biz/
      useUserMgmtHooks.ts
      useProductMgmtHooks.ts
      useOrderMgmtHooks.ts
      useSeckillMgmtHooks.ts
    queryKeys.ts
  stores/
    adminAuthStore.ts
```

## 4. 基础 Hooks 设计

1. `useAdminSession()`
- 职责：登录、刷新、退出、读取当前管理员信息、改密。
- 对应接口：
  - `POST /api/v1/admin/auth/login`
  - `POST /api/v1/admin/auth/refresh`
  - `POST /api/v1/admin/auth/logout`
  - `GET /api/v1/admin/me`
  - `POST /api/v1/admin/me/password`
- 关键状态：
  - `accessToken`
  - `refreshToken`
  - `adminProfile`
  - `domains`
  - `dataScope`

2. `useAdminPermission()`
- 输入：`domains[]`、`dataScope`
- 输出：
  - `can(domain)`
  - `canAny(domains[])`
  - `isDataScopeAll()`
- 用途：菜单、按钮、路由守卫统一判定。

3. `useAdminApiGuard()`
- 作用：统一处理 `AUTH_UNAUTHORIZED/AUTH_FORBIDDEN`。
- 建议：
  - `401`：先尝试一次 refresh，再重放原请求；refresh 失败才登出
  - `403`：保留会话并提示“权限不足”
- 约束：
  - refresh 必须做单飞（singleflight）防并发竞态
  - 重放仅执行一次，避免无限重试

## 5. 管理员中心 Hooks

## 5.1 管理员账号 Hooks

1. `useAdminListQuery(params)`
- 接口：`GET /api/v1/admin/admins`
- 参数：`page/page_size/keyword/status`
- Key：`['admin', adminId, 'admins', params]`

2. `useAdminDetailQuery(adminId)`
- 接口：`GET /api/v1/admin/admins/{admin_id}`
- Key：`['admin', operatorAdminId, 'detail', adminId]`

3. `useCreateAdminMutation()`
- 接口：`POST /api/v1/admin/admins`
- 成功失效：`['admin', operatorAdminId, 'admins']`

4. `useUpdateAdminMutation(adminId)`
- 接口：`PATCH /api/v1/admin/admins/{admin_id}`
- 成功失效：`['admin', operatorAdminId, 'detail', adminId]`、`['admin', operatorAdminId, 'admins']`

5. `useSetAdminStatusMutation(adminId)`
- 接口：`POST /api/v1/admin/admins/{admin_id}/status`

6. `useResetAdminPasswordMutation(adminId)`
- 接口：`POST /api/v1/admin/admins/{admin_id}/reset-password`

7. `useDeleteAdminMutation(adminId)`
- 接口：`DELETE /api/v1/admin/admins/{admin_id}`

8. `useBindAdminRolesMutation(adminId)`
- 接口：`POST /api/v1/admin/admins/{admin_id}/roles`

## 5.2 角色与审计 Hooks

1. `useRoleListQuery(params)` -> `GET /api/v1/admin/roles`
2. `useRoleDetailQuery(roleId)` -> `GET /api/v1/admin/roles/{role_id}`
3. `useCreateRoleMutation()` -> `POST /api/v1/admin/roles`
4. `useUpdateRoleMutation(roleId)` -> `PATCH /api/v1/admin/roles/{role_id}`
5. `useDeleteRoleMutation(roleId)` -> `DELETE /api/v1/admin/roles/{role_id}`
6. `useSetRoleDomainsMutation(roleId)` -> `POST /api/v1/admin/roles/{role_id}/domains`
7. `useAuditLogsQuery(params)` -> `GET /api/v1/admin/audit-logs`
8. 以上 QueryKey 均建议包含 `adminId` 维度，避免跨管理员会话缓存串读。

## 6. 业务管理 Hooks

## 6.1 用户管理 Hooks

1. `useManagedUserProfileQuery(userId)` -> `GET /api/v1/admin/users/{user_id}`
2. `useManagedUserNicknameMutation(userId)` -> `PATCH /api/v1/admin/users/{user_id}/nickname`
3. `useManagedUserDeleteMutation(userId)` -> `DELETE /api/v1/admin/users/{user_id}`

## 6.2 商品管理 Hooks

1. `useAdminProductListQuery(params)` -> `GET /api/v1/admin/products`
2. `useAdminProductDetailQuery(productId)` -> `GET /api/v1/admin/products/{product_id}`
3. `useCreateProductMutation()` -> `POST /api/v1/admin/products`
4. `useUpdateProductMutation(productId)` -> `PATCH /api/v1/admin/products/{product_id}`
5. `useDeleteProductMutation(productId)` -> `DELETE /api/v1/admin/products/{product_id}`

## 6.3 订单管理 Hooks

1. `useAdminOrderListQuery(params)` -> `GET /api/v1/admin/orders`
2. `useAdminOrderDetailQuery(orderId)` -> `GET /api/v1/admin/orders/{order_id}`
3. `useReviewOrderMutation(orderId)` -> `POST /api/v1/admin/orders/{order_id}/review`
4. `useShipOrderMutation(orderId)` -> `POST /api/v1/admin/orders/{order_id}/ship`

## 6.4 秒杀管理 Hooks

1. `useSeckillActivityListQuery(params)` -> `GET /api/v1/admin/seckill/activities`
2. `useSeckillActivityDetailQuery(activityId)` -> `GET /api/v1/admin/seckill/activities/{activity_id}`
3. `useCreateSeckillActivityMutation()` -> `POST /api/v1/admin/seckill/activities`
4. `useUpdateSeckillActivityMutation(activityId)` -> `PATCH /api/v1/admin/seckill/activities/{activity_id}`
5. `useDeleteSeckillActivityMutation(activityId)` -> `DELETE /api/v1/admin/seckill/activities/{activity_id}`
6. `useCreateSeckillItemMutation(activityId)` -> `POST /api/v1/admin/seckill/activities/{activity_id}/items`
7. `useUpdateSeckillItemMutation(activityId,itemId)` -> `PUT /api/v1/admin/seckill/activities/{activity_id}/items/{item_id}`
8. `useRemoveSeckillItemMutation(activityId,itemId)` -> `DELETE /api/v1/admin/seckill/activities/{activity_id}/items/{item_id}`
9. `usePublishSeckillActivityMutation(activityId)` -> `POST /api/v1/admin/seckill/activities/{activity_id}/publish`
10. `useOfflineSeckillActivityMutation(activityId)` -> `POST /api/v1/admin/seckill/activities/{activity_id}/offline`
11. `useSeckillTrafficQuery(activityId,params)` -> `GET /api/v1/admin/seckill/activities/{activity_id}/traffic`
12. `useSeckillOrdersQuery(activityId,params)` -> `GET /api/v1/admin/seckill/activities/{activity_id}/orders`

## 6.5 运营探针 Hooks

1. `useOpsPingQuery()`
- 接口：`GET /api/v1/admin/ping`
- 用途：登录后权限链路自检、环境联调探针。

## 7. Query Key 与失效策略

1. 管理员中心：
- `['admin', adminId, 'admins', params]`
- `['admin', adminId, 'detail', targetAdminId]`
- `['admin', adminId, 'roles', params]`
- `['admin', adminId, 'role', roleId]`
- `['admin', adminId, 'audit', params]`

2. 业务域：
- `['mgmt', adminId, 'users', userId]`
- `['mgmt', adminId, 'products', 'list', params]`
- `['mgmt', adminId, 'products', 'detail', productId]`
- `['mgmt', adminId, 'orders', 'list', params]`
- `['mgmt', adminId, 'orders', 'detail', orderId]`
- `['mgmt', adminId, 'seckill', 'activities', params]`
- `['mgmt', adminId, 'seckill', 'activity', activityId]`
- `['mgmt', adminId, 'seckill', 'traffic', activityId, params]`
- `['mgmt', adminId, 'seckill', 'orders', activityId, params]`
- `['ops', adminId, 'ping']`

3. mutation 成功后至少失效当前详情与所属列表。
4. 登录成功、切换账号、退出登录时，必须清理当前管理员缓存（推荐 `queryClient.clear()`）。

## 8. 典型实现建议

1. 菜单渲染基于 `useAdminPermission()`：
- 无 domain 时菜单不渲染；
- 有 domain 但 `data_scope != all` 时，`admin_management` 页面进入后给出只读/禁止提示。

2. 表单层复用后端约束：
- 商品价格/库存、秒杀活动时间、角色编码规则等在前端同步校验，减少无效请求。

3. 对敏感写操作统一二次确认：
- 删除管理员、删除角色、删除活动、下线活动、删除商品等。
4. 会话策略：
- access 过期后统一走 `401 -> refresh(单飞) -> 请求重放`。
- refresh 失败时执行登出与缓存清理。

## 9. 测试清单

1. Hook 单测：成功、参数错误、401、403、409 分支。
2. 权限回归：
- `order_review_management` 能审核不能发货；
- `data_scope=self` 无法访问 `admin_management` 列表与详情。
3. 缓存回归：变更后列表与详情是否一致刷新。
4. 会话回归：refresh 失败后是否正确登出并跳转。
