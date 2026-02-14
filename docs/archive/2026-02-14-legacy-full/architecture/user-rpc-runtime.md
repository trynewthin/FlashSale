# 用户 RPC 运行态说明（As-Is）

更新时间：2026-02-07

## 1. 服务入口

- 启动文件：`apps/user/rpc/user.go`
- 协议定义：`apps/user/rpc/user.proto`
- 默认监听：`0.0.0.0:8081`

## 2. RPC 接口

| 方法 | 说明 |
|---|---|
| `Register` | 注册并签发用户 token |
| `Login` | 登录并签发用户 token |
| `GetProfile` | 获取用户资料 |
| `UpdateNickname` | 更新昵称 |
| `DeleteUser` | 软删除用户 |

## 3. 业务规则（来自逻辑层代码）

文件：`apps/user/rpc/internal/logic/*.go`

- 手机号校验：`^1[3-9]\d{9}$`
- 密码规则：8~32 位，必须同时包含字母和数字
- 昵称默认值：空字符串会归一为`新用户`
- 昵称长度：最多 32 个字符（按 rune 计数）
- 登录失败文案：统一返回`账号或密码错误`
- 软删除行为：写入 `deleted_at`，并将 `status` 置为 `0`

## 4. 权限模型（RPC 侧）

文件：`apps/user/rpc/internal/server/authz.go`

- token 来自 gRPC metadata 键：`x-access-token`
- user token：只允许访问自身 `user_id`
- admin token：必须包含 `user_management` 域
- `data_scope`：
  - `all` 或空：允许
  - `self`：要求 `admin_id == target_user_id`
  - 其他值：拒绝

## 5. 数据层

- 仓储实现：`apps/user/rpc/internal/repository/mysql_user_repository.go`
- 用户表迁移：
  - `deploy/migrations/user/000002__user_accounts.up.sql`
  - `deploy/migrations/user/000002__user_accounts.down.sql`

`users` 表关键字段：`phone`（唯一）、`password_hash`、`nickname`、`status`、`last_login_at`、`last_login_ip`、`deleted_at`。

## 6. 测试覆盖

- 逻辑与校验测试：`apps/user/rpc/internal/logic/*_test.go`
- 鉴权测试：`apps/user/rpc/internal/server/authz_test.go`
- 仓储集成测试：`apps/user/rpc/internal/repository/mysql_user_repository_integration_test.go`
