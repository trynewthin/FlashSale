# FlashSale 代码架构说明

本文档只描述**代码架构**，聚焦“代码分层、包职责、文件职责、调用关系”，不展开项目目录细节。

## 1. 架构定位

当前仓库处于“基础层阶段”，核心目标是为后续业务服务（用户、管理员、商品、订单、秒杀）提供可复用的通用能力：

- 统一配置加载
- 统一日志与错误模型
- 统一响应结构
- 中间件能力
- MySQL / Redis / Kafka 客户端封装
- JWT 双域鉴权
- Redis 幂等执行保护
- Trace / Metrics 可观测性基线
- 本地连通性 smoke 程序

## 2. 代码分层

代码可抽象为三层：

1. 基础能力层（`pkg/base/*`）
2. 验证入口层（`cmd/smoke/*`）
3. 测试保障层（`*_test.go`）

依赖方向为：`cmd/smoke` -> `pkg/base`，`pkg/base` 内部尽量低耦合，按领域拆包。

## 3. 包职责总览

| 包 | 职责 |
|---|---|
| `pkg/base/config` | 统一加载配置（文件 + 环境变量覆盖） |
| `pkg/base/logx` | 统一日志初始化与 trace 字段注入 |
| `pkg/base/errorx` | 统一错误码与错误结构 |
| `pkg/base/responsex` | 统一 API 响应结构 |
| `pkg/base/middleware` | HTTP 中间件（trace、recover、请求日志） |
| `pkg/base/mysqlx` | MySQL DSN 构建、连接池初始化、连通性检查 |
| `pkg/base/redisx` | Redis 客户端封装与分布式锁 |
| `pkg/base/kafkax` | Kafka 生产/消费接口与实现 |
| `pkg/base/tracing` | OpenTelemetry Trace Provider 初始化 |
| `pkg/base/metrics` | Prometheus 指标注册与暴露 |
| `pkg/base/authx` | 双域 JWT（user/admin）签发与校验 |
| `pkg/base/idempotency` | 基于 Redis 的幂等执行保护 |

## 4. 文件职责说明（逐文件）

### 4.1 smoke 入口文件

| 文件 | 职责 |
|---|---|
| `cmd/smoke/mysqlcheck/main.go` | 遍历 MySQL 分库，执行 `Ping` 和 `SELECT 1` 验证 |
| `cmd/smoke/redischeck/main.go` | 校验 Redis Ping、读写、分布式锁流程 |
| `cmd/smoke/kafkacheck/main.go` | 校验 Kafka 生产与消费闭环 |

### 4.2 基础能力文件

| 文件 | 职责 |
|---|---|
| `pkg/base/config/config.go` | 定义总配置模型，加载配置并做显式环境变量覆盖 |
| `pkg/base/logx/log.go` | 创建统一格式 zap logger，附加 `trace_id` |
| `pkg/base/errorx/error.go` | 定义错误码 `Code` 与统一错误对象 `AppError` |
| `pkg/base/responsex/response.go` | 定义统一响应体 `Envelope` 与 `OK/Fail` 构造函数 |
| `pkg/base/middleware/http.go` | 提供 Trace、Recover、RequestLogger 中间件 |
| `pkg/base/mysqlx/mysql.go` | 构建 DSN、初始化 MySQL 连接、应用连接池默认值 |
| `pkg/base/redisx/redis.go` | 初始化 Redis 客户端，提供 `TryLock` 分布式锁 |
| `pkg/base/kafkax/kafka.go` | 定义 Producer/Consumer 接口并实现消息生产消费 |
| `pkg/base/tracing/tracing.go` | 初始化 OTel exporter / provider 并注册全局 tracer |
| `pkg/base/metrics/metrics.go` | 初始化 Prometheus registry 并提供 HTTP handler |
| `pkg/base/authx/auth.go` | JWT 双域配置初始化、签发 token、解析 token |
| `pkg/base/idempotency/idempotency.go` | 用 `doneKey + lockKey` 机制实现幂等保护 |

### 4.3 测试文件

| 文件 | 职责 |
|---|---|
| `pkg/base/config/config_test.go` | 验证配置加载和环境变量覆盖行为 |
| `pkg/base/authx/auth_test.go` | 验证 user/admin 令牌域隔离 |
| `pkg/base/idempotency/idempotency_test.go` | 验证并发场景下幂等只执行一次 |

## 5. 关键调用链

### 5.1 配置驱动链路

`cmd/smoke/*` 入口先调用 `config.Load`，再把配置分发给 `mysqlx/redisx/kafkax` 等基础包。

### 5.2 异常与响应链路

`middleware.Recover` 捕获 panic -> `errorx` 统一错误模型 -> `responsex.Fail` 输出统一响应结构。

### 5.3 幂等执行链路

`idempotency.Guard` 通过 Redis：

1. 先查 `doneKey`（是否已经成功执行）
2. 再抢 `lockKey`（是否获得执行权）
3. 执行业务回调
4. 写入 `doneKey`
5. 安全释放 `lockKey`

该机制用于后续下单等“可能重复提交”的关键路径。

## 6. 当前架构边界

当前代码不包含具体业务服务（如 user/admin/product/order/seckill 的业务 API 与数据库表），仅提供底座能力。后续业务模块应以 `pkg/base/*` 为统一复用层，避免重复造轮子和能力分散。
