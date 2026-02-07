# 网关运行态说明（As-Is）

更新时间：2026-02-07

## 1. 模块与职责

- `apps/gateway/user`：用户侧网关
- `apps/gateway/admin`：管理员侧网关

两者均为 `net/http` 服务，复用 `pkg/base/middleware` 的 Trace/Recover/RequestLogger。

## 2. 用户网关路由

文件：`apps/gateway/user/internal/handler/handler.go`

| 方法 | 路径 | 说明 | 中间件 |
|---|---|---|---|
| GET | `/healthz` | 健康检查 | 无 |
| POST | `/api/v1/user/register` | 用户注册 | 注册限流 |
| POST | `/api/v1/user/login` | 用户登录 | 登录限流 |
| GET | `/api/v1/user/profile` | 当前用户资料 | 用户 JWT |
| PATCH | `/api/v1/user/nickname` | 修改昵称 | 用户 JWT |
| DELETE | `/api/v1/user` | 软删除自己 | 用户 JWT |

## 3. 管理员网关路由

文件：`apps/gateway/admin/internal/handler/handler.go`

| 方法 | 路径 | 说明 | 中间件 |
|---|---|---|---|
| GET | `/healthz` | 健康检查 | 无 |
| GET | `/api/v1/admin/ping` | 鉴权链路联调 | 管理员 JWT + `operations` 域 |
| GET | `/api/v1/admin/users/{user_id}` | 查用户资料 | 管理员 JWT + `user_management` 域 |
| PATCH | `/api/v1/admin/users/{user_id}/nickname` | 改用户昵称 | 管理员 JWT + `user_management` 域 |
| DELETE | `/api/v1/admin/users/{user_id}` | 软删用户 | 管理员 JWT + `user_management` 域 |

## 4. 鉴权与授权语义

- 用户网关：只接受 user 域 token（`authx.Parse(TokenTypeUser, ...)`）。
- 管理员网关：只接受 admin 域 token，并解析 `domains`、`data_scope`。
- 管理员领域授权由 `apps/gateway/admin/internal/authz/authorizer.go` 判定（静态领域匹配）。

## 5. RPC 转发与错误映射

- 网关通过 `apps/user/rpc/userrpc` 调用下游用户 RPC。
- 请求 token 会通过 `pkg/base/rpcmeta.WithAccessToken` 透传到 gRPC metadata。
- gRPC 错误经 `pkg/base/grpcerr.FromStatus` 转换为统一 `AppError`，再输出统一 HTTP 响应。

## 6. 当前实现边界

- 已实现用户领域相关接口转发。
- 尚未实现商品/订单/秒杀等业务网关路由。
