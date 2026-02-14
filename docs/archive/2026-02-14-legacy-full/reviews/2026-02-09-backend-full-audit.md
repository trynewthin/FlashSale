# 后端全量逐文件排查总报告（已完成）

更新时间：2026-02-09
状态：阶段性完成（问题已闭环）

## 1. 目标

对后端代码执行逐文件排查，覆盖：

- `pkg/base`
- `apps/gateway/admin`
- `apps/gateway/user`
- `apps/*/rpc`（`user/product/order/seckill/admin`）

产出可执行的问题清单、修复优先级与回归结果。

## 2. 审查范围体量

- `apps/admin/rpc`：19 文件
- `apps/user/rpc`：24 文件
- `apps/product/rpc`：26 文件
- `apps/order/rpc`：30 文件
- `apps/seckill/rpc`：21 文件
- `apps/gateway/admin`：16 文件
- `apps/gateway/user`：15 文件
- `pkg/base`：22 文件

## 3. 审查维度

- 功能正确性：状态流转、边界条件、错误路径。
- 安全与权限：认证、鉴权、数据越权、伪造防护。
- 一致性：网关与 RPC 对齐、跨服务幂等与补偿一致性。
- 可维护性：重复代码、抽象边界、注释与命名。
- 可测试性：单测/集成测试覆盖缺口与不可测点。
- 文档对齐：实现与 `docs` 描述一致性。

## 4. 阶段计划

1. Phase 1：基线冻结（规则、模板、严重度）
2. Phase 2：`pkg/base` 逐文件排查
3. Phase 3：gateway 逐文件排查
4. Phase 4：业务 RPC 逐文件排查
5. Phase 5：测试与文档一致性排查
6. Phase 6：分批修复与全量回归

## 5. 输出规范

每条问题记录包含：

- 严重度：`Critical/High/Medium/Low`
- 文件：精确到路径
- 位置：函数或关键代码段
- 现象：可复现描述
- 风险：影响范围
- 修复建议：最小可落地方案
- 状态：`Open/In Progress/Fixed/Verified`

## 6. 当前进展

- 已完成：Phase 1 基线冻结。
- 已完成：Phase 2 `pkg/base` 逐文件排查（22 文件）。
- 已完成：Phase 3 `apps/gateway/*` 逐文件排查（31 文件）。
- 已完成：Phase 4 业务 RPC 逐文件排查（`user/product/order/seckill/admin`）。
- 已完成：Phase 5 全量回归与文档口径复核（`go test ./... -short`、`go vet ./...`）。
- 下一步：可按新需求启动下一轮专项审查（当前 12 条已完成修复闭环）。

## 7. Phase 2 结果（pkg/base）

### 7.1 执行记录

- 已检查目录：`pkg/base/*`
- 已执行：
  - `go test ./pkg/base/... -short`
  - `go vet ./pkg/base/...`
- 结果：全部通过。

### 7.2 Findings（按严重度）

#### Finding-B-01（High）幂等成功后“完成标记”使用业务上下文，可能导致重复执行

- 文件：`pkg/base/idempotency/idempotency.go`
- 位置：`Guard`（`pkg/base/idempotency/idempotency.go:104`）
- 现象：
  - `run()` 成功后，`doneKey` 写入使用原始 `ctx`；
  - 若此时 `ctx` 已超时/取消，`doneKey` 写入失败并返回错误；
  - 调用方可能重试，从而重复执行已成功的业务副作用。
- 风险：
  - 破坏“最多一次成功执行”语义；
  - 在订单/库存等副作用链路中可能造成重复扣减或重复写入。
- 建议修复：
  - `doneKey` 写入改为独立短超时上下文（如 `context.WithTimeout(context.Background(), 2s)`）；
  - 对“业务已成功但标记失败”定义专门错误码，避免上层盲目重试。
- 状态：Verified（2026-02-09，已修复为独立短超时上下文写入 doneKey，`go test ./... -short` / `go vet ./...` 通过）

#### Finding-B-02（Medium）锁 TTL 固定且不续租，长耗时临界区存在并发重入窗口

- 文件：
  - `pkg/base/idempotency/idempotency.go`
  - `pkg/base/redisx/redis.go`
- 位置：
  - `pkg/base/idempotency/idempotency.go:67`
  - `pkg/base/idempotency/idempotency.go:88`
  - `pkg/base/redisx/redis.go:70`
  - `pkg/base/redisx/redis.go:75`
- 现象：
  - 锁仅通过固定 TTL 保持，执行期间无续租机制；
  - 当业务执行时间超过 TTL 时，其他请求可重新拿锁进入临界区。
- 风险：
  - 并发重复执行，影响一致性；
  - 热点场景下会放大重复请求冲突。
- 建议修复：
  - 增加 watchdog 续租机制，或提供“最大执行时长必须小于 TTL”的强约束并在调用侧统一校验。
- 状态：Verified（2026-02-09，已加入锁续租 watchdog，新增 `TestGuardRenewsLockTTL`，并通过 `go test ./pkg/base/idempotency -v`）

#### Finding-B-03（Low）指标配置 `Namespace` 未生效

- 文件：`pkg/base/metrics/metrics.go`
- 位置：`MetricsConfig` 与 `Init`（`pkg/base/metrics/metrics.go:14`、`pkg/base/metrics/metrics.go:24`）
- 现象：
  - `MetricsConfig.Namespace` 定义后未被使用；
  - 当前仅注册默认 collector，未体现命名空间策略。
- 风险：
  - 配置项名实不符，运维预期与实现不一致。
- 建议修复：
  - 若无需命名空间则移除字段；
  - 若保留字段则在业务指标注册路径统一使用该 namespace。
- 状态：Verified（2026-02-09，`metrics.Init` 已使用 `Namespace` 前缀注册默认 collector）

### 7.3 正向结论

- `errorx` 与 `grpcerr` 映射体系完整，覆盖当前业务错误码。
- `authx` 双域 token 隔离实现与测试一致（user/admin 分域成功）。
- `rpcmeta`、`handlerx` 工具函数职责清晰，调用侧复用度良好。

## 8. Phase 3 结果（gateway）

### 8.1 执行记录

- 已检查目录：
  - `apps/gateway/admin/*`
  - `apps/gateway/user/*`
- 已执行：
  - `go test ./apps/gateway/... -short`
  - `go vet ./apps/gateway/...`
- 结果：全部通过。

### 8.2 Findings（按严重度）

#### Finding-G-01（Medium）登录/注册限流为全局桶，存在“单点耗尽”风险

- 文件：`apps/gateway/user/internal/middleware/rate_limit.go`
- 位置：全局 limiter 定义与判定（`apps/gateway/user/internal/middleware/rate_limit.go:12`、`apps/gateway/user/internal/middleware/rate_limit.go:29`）
- 现象：
  - `registerLimiter/loginLimiter` 为进程级共享桶；
  - 任意高频来源可耗尽令牌，影响全部用户请求。
- 风险：
  - 低成本 DoS（尤其是登录入口）；
  - 无法按 IP/账号维度做精细治理。
- 建议修复：
  - 改为“按 IP 或账号键控的限流器”；
  - 增加过期回收，避免键无限增长。
- 状态：Verified（2026-02-09，已改为按来源键控限流并增加过期清理；相关测试已覆盖）

#### Finding-G-02（Low）`decodeJSON` 未拒绝尾随 JSON 数据

- 文件：
  - `apps/gateway/admin/internal/handler/handler.go`
  - `apps/gateway/user/internal/handler/handler.go`
- 位置：
  - `apps/gateway/admin/internal/handler/handler.go:345`
  - `apps/gateway/user/internal/handler/handler.go:193`
- 现象：
  - 当前仅调用一次 `dec.Decode(out)`；
  - 对 `{"a":1}{"b":2}` 这类尾随内容不会显式拦截。
- 风险：
  - 请求语义可被构造为“前段有效，尾段被忽略”，增加审计与排障复杂度。
- 建议修复：
  - 首次 `Decode` 后再执行一次 `Decode(&struct{}{})` 并要求返回 `io.EOF`。
- 状态：Verified（2026-02-09，`decodeJSON` 已新增尾随数据校验并补充单测）

#### Finding-G-03（Low）Bearer 方案前缀校验大小写敏感，存在兼容性问题

- 文件：
  - `apps/gateway/user/internal/middleware/auth.go`
  - `apps/gateway/admin/internal/middleware/auth.go`
- 位置：
  - `apps/gateway/user/internal/middleware/auth.go:93`
  - `apps/gateway/admin/internal/middleware/auth.go:121`
- 现象：
  - 仅接受精确 `"Bearer "` 前缀；
  - 对 `bearer <token>` 等大小写变体直接拒绝。
- 风险：
  - 与部分客户端/代理实现兼容性较差，增加无效 401。
- 建议修复：
  - 采用大小写不敏感匹配（例如先取首段 scheme 并 `EqualFold`）。
- 状态：Verified（2026-02-09，Bearer 解析已改为 `EqualFold` 大小写不敏感匹配并补充单测）

### 8.3 正向结论

- 管理端路由鉴权链路完整：`AuthRequired` + `RequireDomain/RequireAnyDomain`。
- 用户端公开与鉴权路由边界清晰，抢购接口强制登录。
- 网关统一使用 `context.WithTimeout` 调 RPC，错误回包通过 `grpcerr.FromStatus` 归一化。

## 9. Phase 4 结果（业务 RPC，已完成）

### 9.1 子阶段：user rpc（已完成）

- 已检查目录：`apps/user/rpc/*`
- 已执行：
  - `go test ./apps/user/rpc/... -short`
  - `go vet ./apps/user/rpc/...`
- 结果：通过。

#### Finding-RPC-U-01（Medium）`data_scope=self` 在用户管理鉴权中使用“admin_id==user_id”判定，存在语义偏差

- 文件：`apps/user/rpc/internal/server/authz.go`
- 位置：`authorizeTargetUser`（`apps/user/rpc/internal/server/authz.go:54`）
- 现象：
  - 管理员 token 命中 `data_scope=self` 时，通过 `adminID == targetUserID` 判定是否放行。
  - 管理员 ID 与用户 ID 属于不同实体空间，该判定不具备业务语义。
- 风险：
  - 当 ID 数值偶然碰撞时，可能放行对非预期用户数据的管理操作；
  - “self” 语义被实现为“同数值 ID”，权限模型可解释性差。
- 建议修复：
  - 用户管理接口对管理员统一要求 `data_scope=all`；
  - 或引入显式管理员-用户映射关系后再支持精细 self 范围。
- 状态：Verified（2026-02-09，`data_scope=self` 在用户管理接口已收敛为禁止访问）

### 9.2 子阶段：product rpc（已完成）

- 已检查目录：`apps/product/rpc/*`
- 已执行：
  - `go test ./apps/product/rpc/... -short`
  - `go vet ./apps/product/rpc/...`
- 结果：通过。

#### 本阶段结论

- 管理侧接口统一执行 `product_management` 领域校验；公开接口未引入多余鉴权。
- 商品参数校验（名称、图片、描述、价格、库存、状态）覆盖完整，错误码映射符合约定。
- 库存预扣/回补链路具备 MySQL 事务与幂等记录保护，重放场景在仓储层有测试覆盖。
- 本阶段未新增阻塞级或中高优先级问题。

### 9.3 子阶段：order rpc（已完成）

- 已检查目录：`apps/order/rpc/*`
- 已执行：
  - `go test ./apps/order/rpc/... -short`
  - `go vet ./apps/order/rpc/...`
- 结果：通过。

#### Finding-RPC-O-01（High）超时任务在状态冲突时仍发送秒杀状态事件，可能造成窗口限购计数重复

- 文件：
  - `apps/order/rpc/internal/logic/timeout_job_logic.go`
  - `apps/seckill/rpc/internal/logic/purchase_cache.go`
- 位置：
  - `apps/order/rpc/internal/logic/timeout_job_logic.go:163`
  - `apps/order/rpc/internal/logic/timeout_job_logic.go:166`
  - `apps/order/rpc/internal/logic/timeout_job_logic.go:189`
  - `apps/order/rpc/internal/logic/timeout_job_logic.go:192`
  - `apps/seckill/rpc/internal/logic/purchase_cache.go:270`
- 现象：
  - `handleRefundCompletion` 与 `handleAutoReceive` 在 `ErrOrderStateConflict` 情况下没有 `continue`，仍会读取订单并发送秒杀状态事件；
  - 秒杀侧窗口限购样本 `recordCompletedWindow` 的 member 包含 `occurredAt.UnixNano()`，不同时间的重复事件会被计为新增样本。
- 风险：
  - 并发竞态场景（如用户手动确认与系统自动收货竞争）下，窗口限购计数可能被重复累计，导致误判“已达限购”；
  - 事件噪声增加，回溯分析复杂度上升。
- 建议修复：
  - 对 `ErrOrderStateConflict` 显式 `continue`，仅在本次状态推进成功后发布事件；
  - 或在秒杀侧将窗口样本 member 固定为 `order_id:idx`（去掉时间戳），确保天然幂等。
- 状态：Verified（2026-02-09，状态冲突分支已 `continue`，仅成功推进后发送秒杀状态事件）

### 9.4 子阶段：seckill rpc（已完成）

- 已检查目录：`apps/seckill/rpc/*`
- 已执行：
  - `go test ./apps/seckill/rpc/... -short`
  - `go vet ./apps/seckill/rpc/...`
- 结果：通过。

#### Finding-RPC-S-01（Medium）UV 聚合口径错误，当前实现会按 PV 次数累计而非去重访客

- 文件：`apps/seckill/rpc/internal/repository/mysql_seckill_repository.go`
- 位置：
  - `apps/seckill/rpc/internal/repository/mysql_seckill_repository.go:733`
  - `apps/seckill/rpc/internal/repository/mysql_seckill_repository.go:905`
- 现象：
  - `RecordTraffic` 聚合时，`eventIncrements` 在每次 `pv` 且存在身份时都令 `uv = 1`；
  - 聚合表 `uv` 实际按事件次数叠加，不是“去重访客”。
- 风险：
  - 看板 UV 指标系统性偏大；
  - 转化率（如 `purchase_success / uv`）失真，影响运营决策。
- 建议修复：
  - 增加分钟级 UV 去重键（如 `activity:item:minute:user|client`）并按首次出现计数；
  - 或将字段重命名为“identified_pv”，避免语义误导。
- 状态：Verified（2026-02-09，新增分钟级访客 UV marker 去重键，UV 仅首次访客计数）

#### Finding-RPC-S-02（Medium）活动发布未校验“至少一个启用商品”，可发布空可售活动

- 文件：`apps/seckill/rpc/internal/logic/activity_logic.go`
- 位置：
  - `apps/seckill/rpc/internal/logic/activity_logic.go:238`
  - `apps/seckill/rpc/internal/logic/activity_logic.go:254`
- 现象：
  - 发布前仅校验 `len(items) > 0`；
  - 预占库存循环会跳过禁用商品，若全部禁用仍可发布成功。
- 风险：
  - 产生“已发布但不可购买”的活动；
  - 对运营配置和前端展示形成误导，增加排障成本。
- 建议修复：
  - 发布前统计启用商品数量，要求 `enabled_count >= 1`；
  - 对启用商品同时校验 `reserved_stock_total > 0` 与价格有效。
- 状态：Verified（2026-02-09，发布前已强校验“至少一个启用商品且预占库存>0”）

### 9.5 子阶段：admin rpc（已完成）

- 已检查目录：`apps/admin/rpc/*`
- 已执行：
  - `go test ./apps/admin/rpc/... -short`
  - `go vet ./apps/admin/rpc/...`
- 结果：通过（当前仅配置层有测试，核心逻辑无单测覆盖）。

#### Finding-RPC-A-01（High）`data_scope=self` 仍可读取管理员与审计全量数据

- 文件：
  - `apps/admin/rpc/internal/server/authz.go`
  - `apps/admin/rpc/internal/server/adminrpcserver.go`
- 位置：
  - `apps/admin/rpc/internal/server/authz.go:46`
  - `apps/admin/rpc/internal/server/adminrpcserver.go:189`
  - `apps/admin/rpc/internal/server/adminrpcserver.go:202`
  - `apps/admin/rpc/internal/server/adminrpcserver.go:287`
  - `apps/admin/rpc/internal/server/adminrpcserver.go:300`
  - `apps/admin/rpc/internal/server/adminrpcserver.go:331`
- 现象：
  - 读接口（管理员列表/详情、角色列表/详情、审计日志）统一走 `authorizeAdminManagementDomain`；
  - 该鉴权函数仅校验 `admin_management` 域，不校验 `data_scope`。
- 风险：
  - `data_scope=self` 的账号可读取非本人管理数据与审计日志，突破“仅本人可见”预期；
  - 与一期权限模型（`all/self`）不一致，形成越权面。
- 建议修复：
  - 管理员管理域下的读接口同样要求 `data_scope=all`，或显式增加 `self` 过滤策略；
  - 对 `ListAdminAuditLogs` 至少限制 `self` 仅可查询 `admin_id=自己`。
- 状态：Verified（2026-02-09，管理读接口统一改为 `authorizeAdminManagementAll`）

#### Finding-RPC-A-02（Medium）超级管理员 Bootstrap 未复用账号与密码策略校验

- 文件：`apps/admin/rpc/internal/svc/servicecontext.go`
- 位置：
  - `apps/admin/rpc/internal/svc/servicecontext.go:144`
  - `apps/admin/rpc/internal/svc/servicecontext.go:153`
  - `apps/admin/rpc/internal/svc/servicecontext.go:160`
- 现象：
  - Bootstrap 直接使用环境变量写库并加密密码；
  - 未复用 `normalizeUsername` 与 `validatePasswordStrength` 规则。
- 风险：
  - 可能写入不符合登录规则的用户名（大小写/格式），造成初始化账号不可用；
  - 可能绕过密码强度基线，弱化默认安全配置。
- 建议修复：
  - 在 bootstrap 流程中复用同一套账号/密码校验函数；
  - 对不合规配置在启动阶段明确失败并输出可观测错误。
- 状态：Verified（2026-02-09，bootstrap 新增用户名/密码/显示名校验并补充单测）

## 10. Phase 5 结果（全局回归与文档复核）

### 10.1 执行记录

- 已执行：
  - `go test ./... -short`
  - `go vet ./...`
- 结果：通过。

### 10.2 文档口径复核

- `Finding-RPC-A-01` 已按“最小权限”方案完成实现收敛（管理读接口要求 `data_scope=all`）。
- 建议后续同步更新 `docs/architecture/admin-module-design.md` 对应描述，避免文档与实现再次偏移。

## 11. 问题汇总

- High：3 条（Verified 3）
- Medium：6 条（Verified 6）
- Low：3 条（Verified 3）
- 合计：12 条（Verified 12，Open 0）
