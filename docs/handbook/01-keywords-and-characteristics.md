# 关键词与项目特点

## 1. 架构关键词

| 关键词 | 在本项目中的含义 |
|---|---|
| 微服务架构 | user/product/order/seckill/admin 五个 RPC 独立部署，gateway 统一入口 |
| 双网关 | 用户网关（8082）与管理网关（8083）分离，避免认证模型耦合 |
| 领域权限（Domain RBAC） | 管理端通过 `domains` 做能力授权（user/product/order/seckill/admin） |
| 数据范围（Data Scope） | 管理员 `all/self` 数据可见范围，管理员模块与网关协同控制 |
| 软删除 | 商品、活动等实体使用 `deleted_at`，默认查询过滤已删除记录 |
| 雪花 ID | 主键和业务 ID 使用 int64 分布式 ID，前端按字符串承载 |
| 金额分存储 | 价格字段统一 `*_cent`，避免浮点精度问题 |

## 2. 交易与秒杀关键词

| 关键词 | 在本项目中的含义 |
|---|---|
| 完整交易链路 | 下单 -> 支付确认 -> 审核 -> 发货 -> 收货 -> 关闭 |
| 秒杀活动维度 | 活动可配置多商品，抢购必须显式指定 `activity_item_id` |
| 活动库存预占 | 发布活动时预占商品库存，避免活动期间与普通库存冲突 |
| 幂等键（idempotency_key） | 抢购/埋点/库存操作去重关键参数 |
| 热路径 | 秒杀判定优先 Redis，降低 MySQL 主路径压力 |
| 回流事件 | order 通过 Kafka 回传秒杀订单状态，seckill 更新追溯表 |
| 可追溯性 | seckill_order_links + stock_ledger + traffic_agg 支持运营回看 |

## 3. 工程与运维关键词

| 关键词 | 在本项目中的含义 |
|---|---|
| `cmd/fs` | 一体化命令：环境、运行时、数据、压测、ops 控制 |
| ops-control | 独立运维控制面（任务编排 + Web 可视化 + 日志流） |
| SSE 日志流 | ops 页面实时接收任务日志，不依赖轮询全量日志 |
| Docker 本地环境 | MySQL/Redis/Kafka/Nginx/可观测组件 compose 化管理 |
| Smoke 检查 | 启动后验证 MySQL/Redis/Kafka 连通性 |
| 压测分场景 | purchase-stress、idempotency、track-stress、purchase-open、track-open |

## 4. 该项目的可展示特点

- 业务面：包含用户、商品、订单、秒杀、管理员全链路，不是单点 demo。
- 工程面：网关鉴权、领域权限、统一错误码、统一参数解析、基础库可复用。
- 性能面：有独立压测器和开环/闭环场景，支持直接展示吞吐与稳定性指标。
- 运维面：ops-control + Docker + Nginx，具备可演示的“运行控制台”能力。
