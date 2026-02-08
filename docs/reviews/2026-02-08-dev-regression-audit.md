# Dev 分支阶段性回归排查报告（2026-02-08）

## 1. 审查范围与目标
- 分支：`dev`
- 目标：对当前 `dev` 分支进行阶段性综合回归排查，输出可用于合并后持续治理的基线结论。
- 重点：功能回归、权限链路、状态机一致性、库存与幂等、注释与文档基线。

## 2. 执行计划（阶段化）
| 阶段 | 状态 | 内容 |
|---|---|---|
| Phase 1 | 进行中 | 基线快照与自动化检查（test/vet/代码快照） |
| Phase 2 | 待执行 | 网关与鉴权链路审查（路由、domain、subject） |
| Phase 3 | 待执行 | 订单/商品/秒杀一致性审查（库存、幂等、补偿、状态回流） |
| Phase 4 | 待执行 | 注释覆盖与文档对齐审查 |
| Phase 5 | 待执行 | 全局结论、风险分级、后续建议 |

## 3. 阶段记录

### Phase 1 - 基线快照与自动化检查
- 状态：已完成
- 执行结果：
  1. 工作区快照：`git-review-snapshot` 输出显示 `dev` 分支工作区干净（0 changed files）。
  2. 自动化门禁：
     - `go test ./... -short`：通过
     - `go vet ./...`：通过
  3. 最近提交链路：
     - `55007a8 Merge branch 'backend-dev' into dev`
     - `b1deb42 feat(seckill): implement phase1 seckill module with order/product integration`

### Phase 2 - 网关与鉴权链路审查
- 状态：已完成
- 审查范围：
  - `apps/gateway/admin/internal/handler/handler.go`
  - `apps/gateway/admin/internal/middleware/auth.go`
  - `apps/gateway/admin/internal/authz/authorizer.go`
  - `apps/gateway/user/internal/handler/handler.go`
  - `apps/gateway/user/internal/middleware/auth.go`
- 结论：
  1. 管理端路由均通过 `AuthRequired + RequireDomain`，领域隔离完整（`user/product/order/seckill`）。
  2. `seckill_management` 域已接入并用于全部管理员秒杀接口。
  3. 用户端公开/登录态接口边界清晰：活动列表/详情/埋点可匿名，抢购接口要求用户 JWT。
  4. 本阶段未发现阻断问题。

### Phase 3 - 订单/商品/秒杀一致性审查
- 状态：已完成
- 审查范围：
  - `apps/order/rpc/internal/logic/cancel_order_logic.go`
  - `apps/order/rpc/internal/logic/confirm_payment_and_info_logic.go`
  - `apps/order/rpc/internal/logic/create_order_from_seckill_logic.go`
  - `apps/product/rpc/internal/logic/stock_order_logic.go`
  - `apps/seckill/rpc/internal/logic/activity_logic.go`
  - `apps/seckill/rpc/internal/logic/purchase_logic.go`
  - `apps/seckill/rpc/internal/logic/order_state_consumer.go`
- 关键结论：
  1. 已验证：秒杀订单关闭不再回补 product 库存，改由 seckill 侧根据订单状态回流回补活动库存，方向正确。
  2. 已验证：活动发布/下线库存预占与释放流程、幂等台账写入链路完整。
  3. 原高风险问题（Finding-01）已在后续修复中关闭：同幂等键请求可回退 DB 并继续走建单幂等恢复链路。

### Phase 4 - 注释与文档基线审查
- 状态：已完成
- 注释基线（非生成 Go 文件）：
  - 扫描范围：`apps/`、`pkg/`、`cmd/`
  - 统计结果：`TOTAL=137`，`WITH_HEADER=137`，`MISSING=0`
- 文档对齐结论：
  1. `docs/architecture/system-overview.md` 仍标注“`seckill` 未落地”，与代码事实不一致。
  2. `docs/architecture/gateway-runtime.md` 缺少秒杀路由段落，已落后于当前网关实现。
  3. 当前缺少 `docs/architecture/seckill-rpc-runtime.md` 运行态说明文档。

### Phase 5 - 全局结论
- 状态：已完成
- 综合结论：`dev` 分支自动化门禁通过，主链路可运行；原高风险一致性问题已修复，剩余 1 个文档一致性问题建议尽快同步。

## 4. Findings（按严重级别）

### Finding-01（高，已修复）秒杀“建单不确定失败”可能导致库存长期占用且请求无法恢复
- 文件：
  - `apps/seckill/rpc/internal/logic/purchase_logic.go`
  - `apps/seckill/rpc/internal/repository/mysql_seckill_repository.go`
- 现象：
  1. 秒杀先执行库存预扣（DB 台账 key=`idempotency_key:reserve`，唯一约束）。
  2. 若调用 `order.CreateOrderFromSeckill` 出现不确定错误（网络抖动/超时），当前分支返回“订单处理中”且不立即补偿。
  3. 客户端重试同幂等键会命中缓存/DB 幂等冲突，无法重新驱动建单。
- 风险：
  - 若订单实际未创建，则库存可能被长期占用；
  - 用户无确定性恢复路径（同幂等键无法继续）。
- 修复结果（2026-02-08）：
  1. `apps/seckill/rpc/internal/logic/purchase_cache.go`：缓存层命中重复幂等键时改为 `handled=false`，回退 DB 链路，不再直接拦截失败。
  2. `apps/seckill/rpc/internal/logic/purchase_logic.go`：`ReservePurchase` 返回 `ErrIdempotencyConflict` 时继续走 `CreateOrderFromSeckill` 幂等建单恢复，并避免在该分支做即时库存补偿。
  3. 新增回归测试：`apps/seckill/rpc/internal/logic/purchase_logic_test.go`，覆盖“可恢复成功”和“恢复失败返回处理中且不误补偿”。

### Finding-02（中）架构文档与代码状态不一致（秒杀已落地但文档未更新）
- 文件：
  - `docs/architecture/system-overview.md`
  - `docs/architecture/gateway-runtime.md`
- 现象：
  - 文档仍声明 seckill 未落地，且网关运行态缺少秒杀接口说明。
- 建议：
  1. 更新系统总览的“已落地模块”与拓扑说明。
  2. 在网关运行态文档补齐 admin/user 秒杀路由。
  3. 新增 `docs/architecture/seckill-rpc-runtime.md`，覆盖数据模型、库存策略、订单回流、幂等与补偿策略。

## 5. 复现与检查命令
```bash
python C:/Users/anzelin/.codex/skills/git-review-snapshot/scripts/git_review_snapshot.py workspace --repo .
go test ./... -short
go vet ./...
```

## 6. 最终建议
1. 同步补齐秒杀架构文档，避免后续开发与运维认知漂移。
2. 完成文档更新后，再执行一次 `go test ./...` + 重点链路 smoke 回归并归档结果。
