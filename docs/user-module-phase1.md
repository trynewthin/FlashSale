# 用户模块一期（RPC）

## 1. 模块定位

用户模块一期仅提供普通用户 RPC 能力，不暴露 HTTP 接口。

已实现方法：

- `Register`
- `Login`
- `GetProfile`
- `UpdateNickname`
- `DeleteUser`（软删除）

## 2. 代码结构

- `apps/user/rpc/user.proto`：RPC 契约定义。
- `apps/user/rpc/pb/`：由 `protoc` 生成的 gRPC 代码。
- `apps/user/rpc/internal/config/config.go`：用户 RPC 启动配置。
- `apps/user/rpc/internal/svc/servicecontext.go`：依赖注入（MySQL/JWT/日志/Tracing/雪花节点）。
- `apps/user/rpc/internal/model/user.go`：用户领域模型。
- `apps/user/rpc/internal/repository/`：用户仓储接口与 MySQL 实现。
- `apps/user/rpc/internal/logic/`：五个 RPC 的业务逻辑。
- `apps/user/rpc/internal/server/userrpcserver.go`：RPC Server 到 logic 的分发层。
- `apps/user/rpc/userrpc/userrpc.go`：RPC Client 包装。

## 3. 关键规则

- 手机号：`^1[3-9]\\d{9}$`
- 密码：8~32 位，必须同时包含字母和数字（bcrypt 存储）
- 登录失败文案统一：`账号或密码错误`
- Token：仅签发用户域 AccessToken
- 登录审计：更新 `last_login_at` 和 `last_login_ip`
- 删除语义：软删除，写入 `deleted_at` 并把 `status` 置为 `0`

## 4. 数据库迁移

用户库新增：

- `deploy/migrations/user/000002__user_accounts.up.sql`
- `deploy/migrations/user/000002__user_accounts.down.sql`

新增表：`users`

## 5. 本地启动

1. 启动中间件环境：

```powershell
./scripts/dev/up.ps1
```

2. 执行迁移：

```powershell
./scripts/dev/migrate-up.ps1
```

3. 启动用户 RPC：

```powershell
go run ./apps/user/rpc -f apps/user/rpc/etc/user.yaml
```

## 6. 测试

```powershell
go test ./apps/user/rpc/...
```

若需执行仓储集成测试，请设置：

- `FLASHSALE_TEST_MYSQL_DSN`
