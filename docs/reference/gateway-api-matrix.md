# 网关 API 对齐矩阵

> 本文档用于快速核对“HTTP 路由 -> Gateway Handler -> RPC 方法”是否对齐。

## 1. User Gateway 对齐

| HTTP 路由 | Handler | RPC 方法 |
|---|---|---|
| `POST /api/v1/user/register` | `UserHandler.Register` | `UserRpc.Register` |
| `POST /api/v1/user/login` | `UserHandler.Login` | `UserRpc.Login` |
| `GET /api/v1/user/profile` | `UserHandler.GetProfile` | `UserRpc.GetProfile` |
| `PATCH /api/v1/user/nickname` | `UserHandler.UpdateNickname` | `UserRpc.UpdateNickname` |
| `DELETE /api/v1/user` | `UserHandler.DeleteUser` | `UserRpc.DeleteUser` |
| `PUT /api/v1/user/password` | `UserHandler.ChangePassword` | `UserRpc.ChangePassword` |
| `GET /api/v1/products` | `ProductPublicHandler.ListProducts` | `ProductRpc.ListProductsPublic` |
| `GET /api/v1/products/{product_id}` | `ProductPublicHandler.GetProduct` | `ProductRpc.GetProductPublic` |
| `POST /api/v1/orders` | `OrderUserHandler.CreateOrder` | `OrderRpc.CreateOrder` |
| `POST /api/v1/orders/{order_id}/pay-confirm` | `OrderUserHandler.ConfirmPaymentAndInfo` | `OrderRpc.ConfirmPaymentAndInfo` |
| `POST /api/v1/orders/{order_id}/cancel` | `OrderUserHandler.CancelOrder` | `OrderRpc.CancelOrder` |
| `POST /api/v1/orders/{order_id}/confirm-receipt` | `OrderUserHandler.ConfirmReceipt` | `OrderRpc.ConfirmReceipt` |
| `GET /api/v1/orders/{order_id}` | `OrderUserHandler.GetOrder` | `OrderRpc.GetOrderUser` |
| `GET /api/v1/orders` | `OrderUserHandler.ListOrders` | `OrderRpc.ListOrdersUser` |
| `GET /api/v1/seckill/activities` | `SeckillPublicHandler.ListActivities` | `SeckillRpc.ListActivitiesPublic` |
| `GET /api/v1/seckill/activities/{activity_id}` | `SeckillPublicHandler.GetActivity` | `SeckillRpc.GetActivityPublic` |
| `POST /api/v1/seckill/activities/{activity_id}/purchase` | `SeckillPublicHandler.Purchase` | `SeckillRpc.Purchase` |
| `POST /api/v1/seckill/activities/{activity_id}/track` | `SeckillPublicHandler.TrackEvent` | `SeckillRpc.TrackEvent` |

## 2. Admin Gateway 对齐

### 2.1 管理员模块

| HTTP 路由 | Handler | RPC 方法 |
|---|---|---|
| `POST /api/v1/admin/auth/login` | `AdminModuleHandler.Login` | `AdminRpc.AdminLogin` |
| `POST /api/v1/admin/auth/refresh` | `AdminModuleHandler.Refresh` | `AdminRpc.AdminRefreshToken` |
| `POST /api/v1/admin/auth/logout` | `AdminModuleHandler.Logout` | `AdminRpc.AdminLogout` |
| `GET /api/v1/admin/me` | `AdminModuleHandler.GetMyProfile` | `AdminRpc.GetMyAdminProfile` |
| `POST /api/v1/admin/me/password` | `AdminModuleHandler.ChangeMyPassword` | `AdminRpc.ChangeMyPassword` |
| `POST /api/v1/admin/admins` | `AdminModuleHandler.CreateAdmin` | `AdminRpc.CreateAdmin` |
| `PATCH /api/v1/admin/admins/{admin_id}` | `AdminModuleHandler.UpdateAdmin` | `AdminRpc.UpdateAdmin` |
| `POST /api/v1/admin/admins/{admin_id}/status` | `AdminModuleHandler.SetAdminStatus` | `AdminRpc.SetAdminStatus` |
| `POST /api/v1/admin/admins/{admin_id}/reset-password` | `AdminModuleHandler.ResetAdminPassword` | `AdminRpc.ResetAdminPassword` |
| `DELETE /api/v1/admin/admins/{admin_id}` | `AdminModuleHandler.DeleteAdmin` | `AdminRpc.DeleteAdmin` |
| `GET /api/v1/admin/admins/{admin_id}` | `AdminModuleHandler.GetAdmin` | `AdminRpc.GetAdmin` |
| `GET /api/v1/admin/admins` | `AdminModuleHandler.ListAdmins` | `AdminRpc.ListAdmins` |
| `POST /api/v1/admin/admins/{admin_id}/roles` | `AdminModuleHandler.BindAdminRoles` | `AdminRpc.BindAdminRoles` |
| `POST /api/v1/admin/roles` | `AdminModuleHandler.CreateRole` | `AdminRpc.CreateRole` |
| `PATCH /api/v1/admin/roles/{role_id}` | `AdminModuleHandler.UpdateRole` | `AdminRpc.UpdateRole` |
| `DELETE /api/v1/admin/roles/{role_id}` | `AdminModuleHandler.DeleteRole` | `AdminRpc.DeleteRole` |
| `GET /api/v1/admin/roles/{role_id}` | `AdminModuleHandler.GetRole` | `AdminRpc.GetRole` |
| `GET /api/v1/admin/roles` | `AdminModuleHandler.ListRoles` | `AdminRpc.ListRoles` |
| `POST /api/v1/admin/roles/{role_id}/domains` | `AdminModuleHandler.SetRoleDomains` | `AdminRpc.SetRoleDomains` |
| `GET /api/v1/admin/audit-logs` | `AdminModuleHandler.ListAuditLogs` | `AdminRpc.ListAdminAuditLogs` |

### 2.2 业务管理模块

| HTTP 路由 | Handler | RPC 方法 |
|---|---|---|
| `GET /api/v1/admin/users/{user_id}` | `AdminHandler.GetUserProfile` | `UserRpc.GetProfile` |
| `PATCH /api/v1/admin/users/{user_id}/nickname` | `AdminHandler.UpdateUserNickname` | `UserRpc.UpdateNickname` |
| `DELETE /api/v1/admin/users/{user_id}` | `AdminHandler.DeleteUser` | `UserRpc.DeleteUser` |
| `GET /api/v1/admin/users` | `AdminHandler.ListUsers` | `UserRpc.ListUsers` |
| `POST /api/v1/admin/users/{user_id}/reset-password` | `AdminHandler.ResetUserPassword` | `UserRpc.ResetUserPassword` |
| `POST /api/v1/admin/products` | `ProductAdminHandler.CreateProduct` | `ProductRpc.CreateProduct` |
| `PATCH /api/v1/admin/products/{product_id}` | `ProductAdminHandler.UpdateProduct` | `ProductRpc.UpdateProduct` |
| `DELETE /api/v1/admin/products/{product_id}` | `ProductAdminHandler.DeleteProduct` | `ProductRpc.DeleteProduct` |
| `GET /api/v1/admin/products/{product_id}` | `ProductAdminHandler.GetProduct` | `ProductRpc.GetProductAdmin` |
| `GET /api/v1/admin/products` | `ProductAdminHandler.ListProducts` | `ProductRpc.ListProductsAdmin` |
| `POST /api/v1/admin/orders/{order_id}/review` | `OrderAdminHandler.ReviewOrder` | `OrderRpc.ReviewOrderAdmin` |
| `POST /api/v1/admin/orders/{order_id}/ship` | `OrderAdminHandler.ShipOrder` | `OrderRpc.ShipOrderAdmin` |
| `GET /api/v1/admin/orders/{order_id}` | `OrderAdminHandler.GetOrder` | `OrderRpc.GetOrderAdmin` |
| `GET /api/v1/admin/orders` | `OrderAdminHandler.ListOrders` | `OrderRpc.ListOrdersAdmin` |
| `POST /api/v1/admin/seckill/activities` | `SeckillAdminHandler.CreateActivity` | `SeckillRpc.CreateActivity` |
| `PATCH /api/v1/admin/seckill/activities/{activity_id}` | `SeckillAdminHandler.UpdateActivity` | `SeckillRpc.UpdateActivity` |
| `DELETE /api/v1/admin/seckill/activities/{activity_id}` | `SeckillAdminHandler.DeleteActivity` | `SeckillRpc.DeleteActivity` |
| `GET /api/v1/admin/seckill/activities/{activity_id}` | `SeckillAdminHandler.GetActivity` | `SeckillRpc.GetActivityAdmin` |
| `GET /api/v1/admin/seckill/activities` | `SeckillAdminHandler.ListActivities` | `SeckillRpc.ListActivitiesAdmin` |
| `POST /api/v1/admin/seckill/activities/{activity_id}/items` | `SeckillAdminHandler.CreateActivityItem` | `SeckillRpc.UpsertActivityItem(item_id=0)` |
| `PUT /api/v1/admin/seckill/activities/{activity_id}/items/{item_id}` | `SeckillAdminHandler.UpsertActivityItem` | `SeckillRpc.UpsertActivityItem` |
| `DELETE /api/v1/admin/seckill/activities/{activity_id}/items/{item_id}` | `SeckillAdminHandler.RemoveActivityItem` | `SeckillRpc.RemoveActivityItem` |
| `POST /api/v1/admin/seckill/activities/{activity_id}/publish` | `SeckillAdminHandler.PublishActivity` | `SeckillRpc.PublishActivity` |
| `POST /api/v1/admin/seckill/activities/{activity_id}/offline` | `SeckillAdminHandler.OfflineActivity` | `SeckillRpc.OfflineActivity` |
| `GET /api/v1/admin/seckill/activities/{activity_id}/traffic` | `SeckillAdminHandler.GetTraffic` | `SeckillRpc.GetActivityTraffic` |
| `GET /api/v1/admin/seckill/activities/{activity_id}/orders` | `SeckillAdminHandler.ListOrders` | `SeckillRpc.ListActivityOrders` |

### 2.3 工作台聚合接口

> 以下接口在网关内部聚合多个 RPC 调用，不直接映射到单个 RPC 方法。

| HTTP 路由 | Handler | 聚合调用 |
|---|---|---|
| `GET /api/v1/admin/dashboard/stats` | `DashboardHandler.GetStats` | `UserRpc.ListUsers` + `ProductRpc.ListProductsAdmin` + `OrderRpc.ListOrdersAdmin` × 6 + `SeckillRpc.ListActivitiesAdmin` × 3 |

## 3. 对齐结论

- 当前路由注册与 RPC 调用方法可一一对应，未发现“网关调用无 proto 方法”的断链情况。
- Dashboard 聚合接口在网关层并发扇出多个 RPC，用于统计面板展示。
- 管理端鉴权由网关前置，敏感字段（如 `admin_id`）由服务端上下文生成/覆盖，避免前端伪造。
