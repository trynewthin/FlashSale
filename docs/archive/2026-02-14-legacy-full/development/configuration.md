# 配置说明

更新时间：2026-02-10

## 1. 配置来源

1. 基础配置文件：`configs/local/dev.yaml`
2. 本地环境变量文件：`configs/local/dev.env`（由 `scripts/dev/common.ps1` 自动加载）
3. 服务配置文件：
  - `apps/admin/rpc/etc/admin.yaml`
  - `apps/seckill/rpc/etc/seckill.yaml`
  - `apps/order/rpc/etc/order.yaml`
  - `apps/product/rpc/etc/product.yaml`
  - `apps/user/rpc/etc/user.yaml`
  - `apps/gateway/user/etc/user-gateway.yaml`
  - `apps/gateway/admin/etc/admin-gateway.yaml`

## 2. 基础配置结构

`configs/local/dev.yaml` 包含：

- `log`
- `mysql`
- `redis`
- `kafka`
- `jwt.user` / `jwt.admin`
- `otel`
- `prometheus`

加载入口：`pkg/base/config/config.go`。

## 3. 脚本层环境变量

开发脚本通过 `scripts/dev/common.ps1` 加载 `dev.env`，并做以下兜底：

- `FLASHSALE_MYSQL_HOST` 默认 `localhost`
- 若未设置 `FLASHSALE_MYSQL_PORT` 且已设置 `FLASH_MYSQL_PORT`，自动回填
- 若未设置 `FLASHSALE_REDIS_ADDR` 且已设置 `FLASH_REDIS_PORT`，自动回填
- 若未设置 `FLASHSALE_KAFKA_BROKERS` 且已设置 `FLASH_KAFKA_PORT`，自动回填

基础配置还支持显式环境变量覆盖以下 MySQL 连接池参数：

- `FLASHSALE_MYSQL_MAX_OPEN_CONNS`
- `FLASHSALE_MYSQL_MAX_IDLE_CONNS`
- `FLASHSALE_MYSQL_CONN_MAX_LIFETIME`（如 `30m`）

## 4. 服务监听地址覆盖

- `FLASHSALE_USER_RPC_LISTEN_ON` -> user RPC 监听地址
- `FLASHSALE_PRODUCT_RPC_LISTEN_ON` -> product RPC 监听地址
- `FLASHSALE_ORDER_RPC_LISTEN_ON` -> order RPC 监听地址
- `FLASHSALE_SECKILL_RPC_LISTEN_ON` -> seckill RPC 监听地址
- `FLASHSALE_ADMIN_RPC_LISTEN_ON` -> admin RPC 监听地址
- `FLASHSALE_USER_GATEWAY_LISTEN_ON` -> user gateway 监听地址
- `FLASHSALE_ADMIN_GATEWAY_LISTEN_ON` -> admin gateway 监听地址

对应代码：

- `apps/admin/rpc/internal/config/config.go`
- `apps/seckill/rpc/internal/config/config.go`
- `apps/order/rpc/internal/config/config.go`
- `apps/product/rpc/internal/config/config.go`
- `apps/user/rpc/internal/config/config.go`
- `apps/gateway/user/internal/config/config.go`
- `apps/gateway/admin/internal/config/config.go`

## 5. 管理员 bootstrap 环境变量

管理员 RPC 启动时支持首个超级管理员幂等初始化：

- `ADMIN_BOOTSTRAP_ENABLED`
- `ADMIN_BOOTSTRAP_USERNAME`
- `ADMIN_BOOTSTRAP_PASSWORD`
- `ADMIN_BOOTSTRAP_DISPLAY_NAME`

对应代码：`apps/admin/rpc/internal/svc/servicecontext.go`。

## 6. 用户网关限流环境变量（压测可调）

用户网关注册/登录限流支持环境变量覆盖（默认启用）：

- `FLASHSALE_RATE_LIMIT_ENABLED`
  - `true/false`，默认 `true`
- `FLASHSALE_REGISTER_RATE_LIMIT_RPS`
  - 注册接口每来源令牌速率，默认 `5`
- `FLASHSALE_REGISTER_RATE_LIMIT_BURST`
  - 注册接口突发令牌，默认 `10`
- `FLASHSALE_REGISTER_RATE_LIMIT_TTL_SEC`
  - 注册来源键保留时间（秒），默认 `600`
- `FLASHSALE_LOGIN_RATE_LIMIT_RPS`
  - 登录接口每来源令牌速率，默认 `5`
- `FLASHSALE_LOGIN_RATE_LIMIT_BURST`
  - 登录接口突发令牌，默认 `10`
- `FLASHSALE_LOGIN_RATE_LIMIT_TTL_SEC`
  - 登录来源键保留时间（秒），默认 `600`

建议：

- 压测环境可设置 `FLASHSALE_RATE_LIMIT_ENABLED=false`，避免登录/注册限流干扰压测链路。
- 生产环境建议保持启用，仅按业务峰值调高 `RPS/BURST`。

用户网关秒杀 RPC 超时支持环境变量覆盖（用于压测调参与链路保护）：

- `FLASHSALE_USER_GATEWAY_SECKILL_RPC_TIMEOUT_MS`
  - 用户网关 SeckillRPC 客户端基础超时（毫秒），默认 `8000`
- `FLASHSALE_USER_GATEWAY_SECKILL_PURCHASE_RPC_TIMEOUT_MS`
  - 用户网关调用秒杀购买 RPC 超时（毫秒），默认 `8000`
- `FLASHSALE_USER_GATEWAY_SECKILL_TRACK_RPC_TIMEOUT_MS`
  - 用户网关调用秒杀埋点 RPC 超时（毫秒），默认 `4000`

## 7. 秒杀链路性能调优环境变量

秒杀 RPC 支持以下性能参数环境变量覆盖（用于压测与容量调优）：

- `FLASHSALE_SECKILL_ACTIVITY_ITEM_CACHE_TTL_MS`
  - 秒杀活动商品元数据进程内缓存 TTL（毫秒），默认 `1500`
- `FLASHSALE_SECKILL_RESERVE_PURCHASE_TIMEOUT_MS`
  - 秒杀预扣库存数据库事务超时（毫秒），默认 `5000`
- `FLASHSALE_SECKILL_RESERVE_DB_USER_LIMIT_CHECK`
  - 是否在数据库预扣事务内执行用户限购聚合校验，默认 `false`（最终一致口径，推荐压测与高峰流量使用；需要更强一致性时可显式设为 `true`）
- `FLASHSALE_SECKILL_ORDER_CREATE_RPC_TIMEOUT_MS`
  - 秒杀调用 `order.CreateOrderFromSeckill` 的 RPC 超时（毫秒），默认 `2500`
- `FLASHSALE_SECKILL_ORDER_CREATE_MAX_IN_FLIGHT`
  - 秒杀建单并发闸门上限，默认 `128`
- `FLASHSALE_SECKILL_ORDER_CREATE_ACQUIRE_TIMEOUT_MS`
  - 获取建单并发闸门等待超时（毫秒），默认 `80`
- `FLASHSALE_SECKILL_ORDER_LINK_WRITE_ON_PURCHASE`
  - 是否在 `Purchase` 成功后立即写 `seckill_order_links`，默认 `false`（推荐关闭，依赖订单状态回流写入，减少热路径写库压力）
- `FLASHSALE_SECKILL_ORDER_LINK_SYNC_FALLBACK`
  - 当 `order_link` 异步队列不可用/已满时，是否启用同步兜底写库，默认 `false`（压测场景建议关闭）
- `FLASHSALE_SECKILL_ORDER_LINK_ASYNC_WORKERS`
  - 异步写 `order_link` 工作协程数，默认 `4`（仅当 `WRITE_ON_PURCHASE=true` 时生效）
- `FLASHSALE_SECKILL_ORDER_LINK_ASYNC_QUEUE_SIZE`
  - 异步写 `order_link` 队列长度，默认 `2048`（仅当 `WRITE_ON_PURCHASE=true` 时生效）
- `FLASHSALE_SECKILL_ORDER_LINK_WRITE_TIMEOUT_MS`
  - `order_link` 写库超时（毫秒），默认 `600`
- `FLASHSALE_SECKILL_TRAFFIC_ASYNC_WORKERS`
  - 异步写埋点 worker 数，默认 `8`
- `FLASHSALE_SECKILL_TRAFFIC_ASYNC_QUEUE_SIZE`
  - 异步写埋点队列长度，默认 `8192`
- `FLASHSALE_SECKILL_TRAFFIC_WRITE_TIMEOUT_MS`
  - 秒杀埋点写库超时（毫秒），默认 `120`
- `FLASHSALE_SECKILL_TRAFFIC_PUBLISH_TIMEOUT_MS`
  - 秒杀埋点发布超时（毫秒），默认 `120`
- `FLASHSALE_SECKILL_PERF_LOG_INTERVAL_SEC`
  - 秒杀性能快照日志间隔（秒），默认 `30`；设置 `0` 可关闭

说明：

- 以上变量定义于 `apps/seckill/rpc/internal/config/config.go`。
- 运行期会周期输出性能快照日志，包含建单超时、熔断命中、order_link/traffic 队列指标，以及订单状态回流消费（跳过缺失活动商品、回补成功/失败等）指标。

## 8. 订单链路性能调优环境变量

订单 RPC 支持以下秒杀建单相关参数（用于高并发限舱和事件发布解耦）：

- `FLASHSALE_ORDER_SECKILL_CREATE_MAX_IN_FLIGHT`
  - 秒杀建单并发闸门上限，默认 `128`
- `FLASHSALE_ORDER_SECKILL_CREATE_ACQUIRE_TIMEOUT_MS`
  - 获取建单闸门等待超时（毫秒），默认 `80`
- `FLASHSALE_ORDER_SECKILL_ORDER_EVENT_ASYNC_WORKERS`
  - 秒杀订单状态事件异步发布 worker 数，默认 `8`
- `FLASHSALE_ORDER_SECKILL_ORDER_EVENT_ASYNC_QUEUE_SIZE`
  - 秒杀订单状态事件异步发布队列长度，默认 `8192`
- `FLASHSALE_ORDER_SECKILL_ORDER_EVENT_PUBLISH_TIMEOUT_MS`
  - 秒杀订单状态事件单次发布超时（毫秒），默认 `120`
- `FLASHSALE_ORDER_SECKILL_ORDER_EVENT_ENABLED`
  - 是否启用秒杀订单状态事件发布，默认 `true`；压测核心交易链路时可临时设为 `false`

说明：

- 以上变量定义于 `apps/order/rpc/internal/config/config.go`。
- 事件发布采用“异步队列优先 + 同步兜底”，用于降低建单主路径受 Kafka 抖动影响。
