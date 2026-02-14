# 基础能力层说明（pkg/base）

更新时间：2026-02-07

## 1. 包职责

| 包 | 职责 |
|---|---|
| `pkg/base/config` | 统一加载配置（文件 + 环境变量覆盖） |
| `pkg/base/logx` | zap 日志初始化 |
| `pkg/base/errorx` | 统一错误码与 `AppError` |
| `pkg/base/responsex` | 统一响应结构 `OK/Fail` |
| `pkg/base/middleware` | 通用 HTTP 中间件（Trace/Recover/RequestLogger） |
| `pkg/base/mysqlx` | MySQL DSN 与连接管理 |
| `pkg/base/redisx` | Redis 客户端与分布式锁 |
| `pkg/base/kafkax` | Kafka 生产/消费封装 |
| `pkg/base/tracing` | OTel tracing 初始化 |
| `pkg/base/metrics` | Prometheus 指标注册与 handler |
| `pkg/base/authx` | user/admin 双域 JWT 签发与解析 |
| `pkg/base/grpcerr` | `AppError` 与 gRPC status 双向映射 |
| `pkg/base/rpcmeta` | 网关到 RPC 的 token 元数据透传 |
| `pkg/base/idempotency` | 基于 Redis 的幂等保护 |

## 2. 已被应用层实际复用的能力

- `config`：网关与用户 RPC 启动都依赖 `BaseConfigPath` 加载基础配置。
- `authx`：网关鉴权与 RPC 令牌签发/校验使用同一套 JWT 配置。
- `grpcerr`：用户 RPC 返回统一业务错误码，网关再映射到 HTTP。
- `rpcmeta`：网关将 `x-access-token` 注入 gRPC metadata，RPC 侧读取后做权限判定。
- `mysqlx`：用户 RPC 连接 `flash_user` 并执行业务仓储读写。

## 3. Smoke 程序与基础层关系

- `cmd/smoke/mysqlcheck`：遍历 MySQL 分库执行 `Ping + SELECT 1`
- `cmd/smoke/redischeck`：验证 Redis `Ping/Set/Get/TryLock`
- `cmd/smoke/kafkacheck`：验证 Kafka 生产消费闭环

依赖方向：`cmd/smoke/*` -> `pkg/base/*`。
