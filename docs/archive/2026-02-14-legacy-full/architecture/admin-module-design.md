# 管理员模块设计与运行说明（一期）

更新时间：2026-02-09

## 目标

管理员模块在现有 `user/product/order/seckill + 双网关` 架构上新增 `admin rpc`，将原先“仅靠 JWT claims”的轻量管理能力升级为可管理、可审计、可续期的正式模块。

一期落地目标：

- 管理员认证：登录、刷新、退出、个人资料、个人改密。
- 管理员管理：创建、更新、禁用/启用、删除、重置密码、角色绑定。
- 角色管理：角色 CRUD、角色领域权限绑定。
- 审计：登录与关键管理动作审计落库。
- 兼容现有鉴权：继续复用 claims 中的 `domains + data_scope`。

## 模块边界

一期包含：

- 账号密码登录（bcrypt）。
- 领域 RBAC（domain 维度）。
- `all/self` 两级数据范围。
- Access + Refresh 双 token 会话模型。
- 超级管理员 bootstrap（环境变量 + 幂等初始化）。

一期不包含：

- 组织树/部门权限。
- 菜单级权限。
- MFA/SSO 与外部身份源集成。

## 代码落点

- RPC 服务：`apps/admin/rpc`
- 协议定义：`apps/admin/rpc/admin.proto`
- 网关接入：`apps/gateway/admin/internal/handler/admin_module_handler.go`
- 路由注册：`apps/gateway/admin/internal/handler/handler.go`
- 权限域定义：`apps/gateway/admin/internal/authz/authorizer.go`

## RPC 接口设计

### 认证接口

- `AdminLogin`
- `AdminRefreshToken`
- `AdminLogout`
- `GetMyAdminProfile`
- `ChangeMyPassword`

### 管理员管理接口

- `CreateAdmin`
- `UpdateAdmin`
- `SetAdminStatus`
- `ResetAdminPassword`
- `DeleteAdmin`
- `GetAdmin`
- `ListAdmins`
- `BindAdminRoles`

### 角色与审计接口

- `CreateRole`
- `UpdateRole`
- `DeleteRole`
- `GetRole`
- `ListRoles`
- `SetRoleDomains`
- `ListAdminAuditLogs`

## 网关路由设计

### 匿名可访问

- `POST /api/v1/admin/auth/login`
- `POST /api/v1/admin/auth/refresh`

### 登录后可访问

- `POST /api/v1/admin/auth/logout`
- `GET /api/v1/admin/me`
- `POST /api/v1/admin/me/password`

### 需 `admin_management` 领域权限

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

## 数据模型（flash_admin）

迁移文件：

- `deploy/migrations/admin/000002__admin_core.up.sql`
- `deploy/migrations/admin/000002__admin_core.down.sql`

核心表：

- `admins`：管理员主表（状态、数据范围、锁定、登录信息、超管标志）。
- `admin_roles`：角色主表（编码、名称、状态、系统角色标志）。
- `admin_role_domains`：角色领域权限。
- `admin_role_bindings`：管理员与角色绑定关系。
- `admin_refresh_tokens`：refresh token 持久化（jti/hash/撤销状态）。
- `admin_audit_logs`：关键行为审计日志。

## 认证与授权模型

### 会话策略

- Access Token TTL：15 分钟。
- Refresh Token TTL：7 天。
- Refresh 采用轮换策略：旧 refresh 撤销 + 新 refresh 生成。
- Logout 撤销当前 refresh token。

### RBAC 与数据范围

- 领域权限来自 `admin_role_bindings + admin_role_domains` 聚合。
- claims 继续透传 `domains + data_scope`，下游协议不变。
- `data_scope` 一期仅支持：
  - `all`
  - `self`

当前策略：

- 管理写操作：要求 `admin_management` 且 `data_scope=all`。
- 管理读操作：要求 `admin_management`，不再强制 `all`。

## 安全策略

- 连续密码错误 5 次，锁定 15 分钟。
- 超级管理员保护：
  - 不可删除
  - 不可禁用
  - 不可降权为 `self`
- refresh token 以 hash 存储，避免明文落库。

## 启动初始化（Bootstrap）

当 `ADMIN_BOOTSTRAP_ENABLED=true` 且系统内无管理员时，自动创建超级管理员：

- `ADMIN_BOOTSTRAP_USERNAME`
- `ADMIN_BOOTSTRAP_PASSWORD`
- `ADMIN_BOOTSTRAP_DISPLAY_NAME`

初始化流程包含：

1. 创建超管账号（幂等处理重复用户名）。
2. 创建/复用 `super_admin` 角色。
3. 绑定全领域权限与管理员角色关系。
4. 写入 `bootstrap_super_admin` 审计日志。

说明：初始化链路关键步骤错误不再吞掉，失败会中止启动并显式报错。

## 审计设计

审计覆盖关键动作：

- 登录/退出/刷新
- 管理员创建、更新、状态变更、删除、重置密码、绑定角色
- 角色创建、更新、删除、设置领域权限
- bootstrap 超级管理员初始化

审计字段包含管理员、动作、目标对象、结果、请求标识、IP、UserAgent、明细 JSON、时间戳。

## 与现有模块的兼容关系

- admin-gateway 统一接入 `AdminRPC`，并继续沿用现有中间件链路。
- 领域常量新增 `admin_management`，不影响既有 `user/product/order/seckill` 域。
- 公共错误码映射已补充 admin 领域错误，不影响既有错误语义。

## 后续演进建议

- 增加管理员模块专项单元测试（auth/repository/server 粒度）。
- 引入 refresh token 设备管理与主动踢下线能力。
- 扩展组织树与菜单权限时，保持与当前 domain RBAC 的兼容升级路径。
