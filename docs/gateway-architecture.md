# FlashSale 网关层架构设计

本文档用于明确 FlashSale 网关层的目标、边界、分层、鉴权模型与落地顺序，作为后续网关开发与评审基线。

## 1. 设计目标

1. 统一对外入口，承接前端与第三方请求。
2. 对内统一转发到各业务 RPC 服务（用户、管理员、商品、订单、秒杀）。
3. 在网关集中处理横切能力：鉴权、限流、审计、追踪、错误码映射。
4. 保证用户端与管理员端策略隔离，避免后续权限模型耦合。

## 2. 关键结论

1. 外部访问协议：HTTP。
2. 微服务内部同步调用：RPC（zrpc/gRPC）为主。
3. 微服务异步事件：Kafka。
4. 网关采用双网关：
- user-gateway（面向普通用户）
- admin-gateway（面向管理员/运营）

## 3. 为什么要双网关

1. 用户端与管理员端鉴权模型不同：
- 用户端：身份校验为主。
- 管理端：身份校验 + 角色权限 + 数据范围。
2. 风险控制与限流规则不同：
- 用户端更偏抗流量冲击与接口防刷。
- 管理端更偏操作审计与高危动作保护。
3. 生命周期不同：
- 管理端权限策略变化更频繁，独立迭代更安全。

## 4. 网关职责边界

### 4.1 网关负责

1. 路由分发（HTTP -> RPC）。
2. Token 校验与 Claims 解析。
3. 权限拦截（尤其 admin-gateway）。
4. 限流、熔断、超时控制。
5. 请求日志、Trace 透传、审计埋点。
6. 统一错误码与响应结构对外输出。

### 4.2 网关不负责

1. 业务核心规则（订单状态机、库存扣减、秒杀算法）。
2. 业务数据持久化。
3. 业务事务编排（由后端服务完成）。

## 5. 鉴权与权限模型

## 5.1 用户端（user-gateway）

1. 校验 user 域 JWT。
2. 透传 `user_id` 到后端服务上下文。
3. 只做基础身份与接口级校验，不承担 RBAC 逻辑。

## 5.2 管理端（admin-gateway）

采用 `RBAC + 数据范围`：

1. 角色按“领域”划分：决定“负责什么业务域”（如 `operations`、`user_management`、`product_management`、`order_management`）。
2. 数据范围：决定“能操作哪些数据”（如 `all/dept/self`）。
3. 默认拒绝：无权限映射一律拒绝。
4. 高危操作可追加二次校验（如验证码/审批流，后续扩展）。

推荐做法：

1. 网关优先校验“领域角色”而不是细粒度动作权限。
2. 领域内的细分操作约束由后端服务二次校验（例如状态机约束、风控约束）。
3. 领域角色命名保持稳定，避免路由级权限点爆炸。

## 6. 推荐代码结构

```text
gateways/
  user-gateway/
    internal/
      config/
      handler/
      logic/
      middleware/
      rpc/
      svc/
    etc/
  admin-gateway/
    internal/
      config/
      handler/
      logic/
      middleware/
      authz/
      rpc/
      svc/
    etc/
pkg/base/
  authx/
  middleware/
  responsex/
  errorx/
  tracing/
  metrics/
```

说明：

1. `handler` 处理 HTTP 参数绑定与响应输出。
2. `logic` 做网关编排（参数校验、调用 RPC、错误映射）。
3. `rpc` 封装下游服务客户端。
4. `middleware` 放通用中间件（trace、recover、rate limit、auth）。
5. `authz` 仅在 admin-gateway 下实现角色和数据范围判定。

## 7. 请求链路（标准流程）

1. 客户端请求进入网关。
2. 中间件链执行：Trace -> Recover -> Auth -> RateLimit -> AccessLog。
3. Handler 解析参数并调用 Logic。
4. Logic 调用下游 RPC 服务。
5. 收敛错误码并返回统一响应。

## 8. 与现有代码的衔接点

1. 复用 `pkg/base/authx`：继续沿用 user/admin 双域 JWT。
2. 复用 `pkg/base/errorx`、`pkg/base/responsex`：统一错误和响应格式。
3. 复用 `pkg/base/tracing`、`pkg/base/metrics`：统一可观测性。
4. 复用现有 `apps/*/rpc` 客户端封装，网关只做协议转换和安全控制。

## 9. 分阶段落地计划

### Phase 1：网关骨架

1. 创建 `user-gateway` 与 `admin-gateway` 工程骨架。
2. 打通 1~2 条核心链路（如用户登录、管理员登录）。
3. 接入 trace/log/error 统一输出。

### Phase 2：鉴权与权限

1. user-gateway：完成 JWT 鉴权中间件。
2. admin-gateway：完成 RBAC + 数据范围校验中间件。
3. 固化领域角色命名规范（如 `operations`、`user_management`、`product_management`）。

### Phase 3：治理能力

1. 限流与熔断策略。
2. 审计日志（管理员关键操作）。
3. 灰度发布与配置热更新（可选）。

## 10. 验收标准

1. 用户端与管理员端网关均可独立启动与健康检查。
2. HTTP 请求可正确转发到 RPC 并返回统一响应。
3. 管理端未授权接口返回一致的权限错误码。
4. Trace 可贯穿网关与下游服务。
5. 关键网关路径具备基础单元测试与集成测试。

## 11. 当前决策记录

1. 网关需要做，且不是可选项。
2. 采用双网关，而非单网关混合策略。
3. 内部通信坚持 RPC 为主，HTTP 仅用于外部入口与少量补充场景。
4. 管理端权限采用 `RBAC + 数据范围`，先不引入 ABAC/策略引擎。
5. 管理端角色先按领域建模，不在网关层做细粒度动作权限拆分。
