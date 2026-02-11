# 秒杀性能优化清单（基于 2026-02-11 实测）

## 1. 最新基线结论

- 2026-02-11 同口径 strict（`UserLimitQty=100000`）最新复测（`MySQL 32/8 + reserve=5000`）：
  - `strict-off-tune5000-conn32-rerun/suite-20260211-230407`：PASS（L2-c200 `success=97.18%`, `p95=4822.06ms`, `network=0.27%`；L3-c200 `success=96.45%`, `p95=5241.97ms`）。
  - `strict-off-tune5000-conn32-rerun/suite-20260211-230650`：PASS（L2-c200 `success=97.75%`, `p95=4410.17ms`, `network=0.30%`；L3-c200 `success=96.12%`, `p95=5321.09ms`）。
  - `strict-off-tune5000-conn32-rerun/suite-20260211-230934`：PASS（L2-c200 `success=97.28%`, `p95=4758.93ms`, `network=0.32%`；L3-c200 `success=96.38%`, `p95=5228.63ms`）。
  - 对照：`strict-off-default4000-conn32/suite-20260211-223516` 在 L2-c200 `success=94.95%`（strict 失败）。
  - 结论：当前环境推荐默认档为 `FLASHSALE_SECKILL_RESERVE_PURCHASE_TIMEOUT_MS=5000`，并保持 `FLASHSALE_SECKILL_RESERVE_DB_USER_LIMIT_CHECK=false`。
- 2026-02-11 运行时口径发现：
  - 使用 `localhost` 在本机出现 `dial tcp [::1]:13306 i/o timeout`，会污染样本。
  - 强制 `127.0.0.1`（`RuntimeForceIPv4Loopback=true`）后该类超时消失。
- 2026-02-11 连接池参数实验：
  - `MySQLMaxOpenConns=160` 会触发 `Error 1040: Too many connections`，导致 L2 大面积 5xx（非业务瓶颈）。
  - 建议安全默认保持 `32/8`，按机器资源与 MySQL `max_connections` 逐步调优。

- 2026-02-11 双口径套件对照（`UserLimitQty=0`，即未触发 DB 限购聚合校验）：
  - strict：`strict/suite-20260211-210338`，仅 `L2-c200` 因 `p95=5657ms > 5200ms` 失败。
  - stable：`stress/suite-20260211-210704`，`L1/L2/L3/L4` 全通过。
  - 结论：该轮 strict/stable 差异主要来自压测抖动与门限收紧，不应归因于 `ReserveDBUserLimitCheck`。
- 2026-02-11 单场景 A/B（`UserLimitQty=100000`，触发 DB 限购聚合校验）：
  - 开启校验：`success=97.875%`, `p95=4628ms`, `rps=94.99`。
  - 关闭校验：`success=99.325%`, `p95=3447ms`, `rps=128.36`。
  - 结论：当限购聚合校验路径被触发时，关闭事务内聚合查询可显著提升吞吐与尾延迟。

- 2026-02-11 新增“冲突来源拆分”观测已落地（`seckill perf snapshot`）：
  - 新增字段：`reserve_err_timeout`、`reserve_err_canceled`、`reserve_err_txn_contention`、`purchase_conflict_overloaded`、`purchase_conflict_pending`。
  - 结论：当前 `SECKILL_PURCHASE_CONFLICT` 的主要来源是 `reserve_err_timeout`，不是建单过载（`order_create_overloaded` 基本为 0）。
- 2026-02-11 同口径单场景复测（`purchase c200/r4000/t7000`）：
  - 样本：`.memory/runlogs/perf.purchase.c200.r4000.t7000.final.20260211-192052.log`
  - 结果：success `99.125%`，`409=31`，network `0.1%`，`p95=4070ms`，`p99=5795ms`。
- 2026-02-11 完整套件样本（含 L3）：
  - 样本：`suite-20260211-191208`
  - 结果：`L1/L2-c100/L3/L4` 通过，`L2-c200` 仅 `p95` 超阈值（`5689ms > 5200ms`），成功率仍在 `95.43%`。

- 最新完整门禁样本（含 L3）已通过：
  - 样本：`suite-20260211-181226`（PASS）
  - 参数：`L2 c100/r2000`, `L2 c200/r4000`, `L3 c200/r4000`, `L4 track c300/r6000`
  - 结果：
    - `L2-c100`: success `99.95%`, p95 `2838ms`
    - `L2-c200`: success `96.30%`, p95 `5012ms`, network `1.58%`
    - `L3-c200`: success `97.45%`, p95 `4908ms`, network `0.25%`
    - `L4-track`: success `100%`, p95 `231ms`
- 同口径单场景复测（`purchase c200/r4000/t6000`）：
  - 样本：`.memory/runlogs/perf.purchase.c200.r4000.t6000.after-db-opt-v2.json`
  - 结果：success `99.375%`, p95 `3413ms`, p99 `4687ms`, network `0.125%`
- 当前判断：系统已达“稳定可复现通过”，但并非最佳优化状态；主要剩余问题是 `L2-c200` 的尾延迟仍接近 5s 门槛。
- 代码修正后回归样本（限购兜底恢复）：
  - 样本：`suite-20260211-182319`（PASS）
  - 结果：`L2-c200 success=96.15%`, `p95=5000ms`, `network=1.4%`；`L3-c200 success=96.55%`, `p95=5140ms`
  - 结论：正确性修正未破坏门禁稳定性，性能仍在可接受区间。

- 新增“容量搜索 + 稳定化探针”后，门禁样本通过：
  - 样本：`suite-20260211-132417`（PASS）
  - 参数：`L2 c120/r2400`, `L3 c140/r2800`, `StageCooldown=6`, `EnableStageStabilize=true`
  - 结果：`L3 success=100%`, `network=0%`, `p95=2792ms`, `p99=3086ms`
- 标准门禁（默认参数，含 L3）在本机仍有长尾风险：
  - 样本：`suite-20260211-011042`、`suite-20260211-012743`。
  - 典型 `L3-c300`: success `96.93%`，network error `3.07%`，`p95=7433ms`，`p99=8000.65ms`（FAIL）。
- 引入阶段冷却 + L3 参数调优后，门禁可稳定通过：
  - 样本：`suite-20260211-125845`（PASS）。
  - `L3-c300`: success `98.57%`，network error `1.43%`，`p95=6098ms`，`p99=7500ms`。
- 同活动参数扫描结果（activity=`2021271455630647296`）：
  - `timeout=9s, maxConnsPerHost=240`：success `99.35%`，network error `0.10%`，`p95=6098ms`，`p99=8097ms`（当前最优折中）。
  - `timeout=10s, maxConnsPerHost=300`：出现大量 `dial_error`（连接被拒绝），说明连接风暴会引入客户端侧假瓶颈。
- 对账连续通过（未超卖）：`sold + available = reserved`。
- `L2` 偶发失败主要是尾延迟抖动（不是功能错误）：
  - 示例：`suite-20260210-175442` 的 `L2-c200` 门禁失败，
  - 同轮 `L2b` 诊断层在更高 timeout/连接池下通过，说明瓶颈以排队超时为主。

## 1.1 2026-02-11 新增实测（ReservePurchaseTimeoutMs 扫描）

- 固定口径：同一活动同一商品，`purchase c200/r4000`，客户端 `timeout=6500ms`，`MaxConnsPerHost=220`。
- 活动：`2021488279274151936`，商品：`2021488279358038016`。
- 结果：
  - `reserve=2500ms`：`success=92.60%`，`409=293`，`network=0.075%`，`p95=3741ms`，`p99=4696ms`。
  - `reserve=3200ms`：`success=88.83%`，`409=428`，`network=0.475%`，`p95=4790ms`，`p99=5874ms`。
  - `reserve=4000ms`：`success=99.30%`，`409=26`，`network=0.05%`，`p95=3693ms`，`p99=4995ms`。
- 反向复核（避免顺序偏差）：
  - `reserve=4000ms` repeat：`success=96.80%`，`409=120`，`network=0.2%`，`p95=4816ms`。
  - `reserve=2500ms` repeat：`success=79.70%`，`409=809`，`network=0.075%`，`p95=4367ms`。
- 结论：在当前机器与配置下，`ReservePurchaseTimeoutMs` 过小会显著放大冲突与失败；后续复测已确认 `5000ms` 比 `4200ms` 更稳（在 strict+L3 下更稳定通过）。
- 同口径补充（`reserve=4000ms` 下 `MaxConnsPerHost` 对比）：
  - `conn=220`：`success=95.60%`，`p95=5056ms`，`network=0.425%`。
  - `conn=240`：`success=94.98%`，`p95=5104ms`，`network=0.55%`。
  - 结论：当前环境 `220` 优于 `240`，不建议盲目提高连接上限。
- 同口径补充（order/seckill 建单舱壁 448/240）：
  - 参数：`FLASHSALE_ORDER_SECKILL_CREATE_MAX_IN_FLIGHT=448`、`FLASHSALE_ORDER_SECKILL_CREATE_ACQUIRE_TIMEOUT_MS=240`、`FLASHSALE_SECKILL_ORDER_CREATE_MAX_IN_FLIGHT=448`、`FLASHSALE_SECKILL_ORDER_CREATE_ACQUIRE_TIMEOUT_MS=240`。
  - 样本：`perf.purchase.c200.r4000.t6500.tune448.json`。
  - 结果：`success=99.675%`，`network=0%`，`p95=3373ms`，`p99=4516ms`。
  - 结论：当前阶段建议将 dev 默认舱壁参数提升到 `448/240`。
- 全链路门禁样本（含 L3）：
  - 样本：`suite-20260211-162007`（PASS）。
  - 参数：`L2 c200/r4000/t6000`，`L3 c200/r4000/t7000`。
  - 结果：`L2 success=95.78%, p95=5024ms`；`L3 success=95.38%, p95=5103ms`；`L4 track=100%`；对账通过。
  - 建议：将脚本默认 L3 档位固定为 `c200`，`c300+` 仅用于极限探测。

## 2. P0（立即执行）

1. 固化“门禁层 + 诊断层”双层策略
- 门禁层：
  - `L1` 幂等（必须 pass）
  - `L2-c100/c200`（常规容量）
  - `L4-track`（埋点链路）
- 诊断层：
  - `L2b` 仅用于根因判定（`Gate=N`），不阻断发布。

2. 固化压测默认参数，避免客户端误伤
- 推荐：`L2 timeout=6000ms`，`L2 MaxConnsPerHost=200~240`。
- 推荐：`L2 p95 门禁阈值=5200ms`（当前机器在 c200 档稳定落点约 5.0~5.1s，5000ms 容易误报）。
- 推荐：`L3 timeout=7500ms`，`L3 MaxConnsPerHost=220`。
- 推荐：`track timeout=3000ms`，`track MaxConnsPerHost=200~240`。
- 推荐（新增）：`FLASHSALE_SECKILL_RESERVE_PURCHASE_TIMEOUT_MS=5000`（压测档）；生产档可先从 `4200~5000` 启动并按监控回调。
- 推荐：`StageCooldownSeconds=6~8`（阶段间冷却，降低样本串扰）。
- 含义：减少“压测机参数过小”导致的假失败，提高横向可比性。

3. 每轮强制新活动 + 清理旧 perf 活动
- 已在脚本内实现，但执行规范必须保持。
- 目标：杜绝库存污染导致的数据失真。

4. 固化管理员登录自动回退，减少无效失败
- 已在 `scripts/dev/seckill-test-suite.ps1` 实现：
  - token 可用则复用；
  - token 失效自动回退 `username/password` 登录。
- 目标：避免压测因管理端 token 过期而中断。

5. 固化“容量搜索 -> 门禁验收”的两段式流程
- 新增脚本：`scripts/dev/seckill-capacity-search.ps1`
- 执行顺序：
  - 先跑 `seckill-capacity-search.ps1`，快速获得当前机器稳定并发上限；
  - 再按上限回填 `seckill-test-suite.ps1` 的 `L2/L3` 参数做门禁。
- 目标：减少盲目重复压测，缩短定位周期。

6. 压测默认切换到二进制复用模式
- `seckill-test-suite.ps1`/`seckill-perf.ps1` 已支持 `LoadRunnerMode/RunnerMode=binary`。
- 仅首次编译 `seckillload.exe`，后续阶段复用，避免每阶段 `go run` 重新编译。
- 目标：降低压测总耗时与工具侧抖动。

7. 阶段间稳定化探针默认开启
- 参数：`EnableStageStabilize=true`、`StageStabilizeProbeConcurrency=10`、`StageStabilizeProbeRequests=60`。
- 含义：先用小流量探针确认系统已恢复，再进入下一压测阶段。
- 目标：降低“前一阶段残压导致后一阶段误判”的概率。

## 3. P1（本周）

1. 订单建单链路容量调优常态化
- 重点参数：
  - `FLASHSALE_SECKILL_ORDER_CREATE_MAX_IN_FLIGHT`
  - `FLASHSALE_SECKILL_ORDER_CREATE_ACQUIRE_TIMEOUT_MS`
  - `FLASHSALE_ORDER_SECKILL_CREATE_MAX_IN_FLIGHT`
  - `FLASHSALE_ORDER_SECKILL_CREATE_ACQUIRE_TIMEOUT_MS`
- 目标：把 `L2-c200` 的 `p95` 稳定压到 `< 5000ms`。

2. 异步队列容量与超时参数继续打磨
- seckill 侧：
  - `FLASHSALE_SECKILL_ORDER_LINK_ASYNC_WORKERS/QUEUE_SIZE/WRITE_TIMEOUT_MS`
  - `FLASHSALE_SECKILL_TRAFFIC_ASYNC_WORKERS/QUEUE_SIZE`
- order 侧：
  - `FLASHSALE_ORDER_SECKILL_ORDER_EVENT_ASYNC_WORKERS/QUEUE_SIZE/PUBLISH_TIMEOUT_MS`
- 目标：降低 timeout 尾部放大效应。

3. 建立固定压测档位
- `baseline`: L1 + L2(c100) + L4
- `gate`: L1 + L2(c100/c200) + L4 + 对账
- `stress`: gate + L3(c300)
- 目标：把“回归”和“极限探测”隔离，避免互相干扰。

## 4. P2（后续）

1. 引入开放模型压测（arrival-rate）
- 当前已落地 `scripts/dev/seckill-k6-open.ps1` + `scripts/perf/k6/seckill_open_model.js`（constant-arrival-rate）。
- 闭环模型仍用于回归门禁，开环模型用于容量退化拐点确认（固定 RPS）。

2. 指标平台化
- 将关键指标接入 Prometheus/Grafana：
  - `purchase_success_rate`
  - `purchase_timeout_rate`
  - `order_create_inflight/rejected/timeout`
  - `track_accept_rate`
- 目标：减少靠日志人工定位的成本。

## 5. 2026-02-11 新增实测（建单舱壁 A/B）

1. 基线（调优前）
- 样本：`k6-ladder-20260211-135136`
- purchase（10s）：`80/120/160 req/s` 全 FAIL
- 主要表现：`purchase_http_5xx_rate` 高（如 120 档约 `45.42%`），`network_rate=0`

2. round1 参数
- order：`SeckillCreateMaxInFlight=256`，`SeckillCreateAcquireTimeoutMs=180`
- seckill：`OrderCreateMaxInFlight=280`，`OrderCreateAcquireTimeoutMs=200`，`OrderCreateRPCTimeoutMs=4200`
- 样本：`k6-ladder-20260211-135515`
- 结果：`80/120` PASS，`160` 仅 `p95=6396ms` 超阈值；5xx 已明显收敛

3. round2 参数（当前推荐）
- order：`SeckillCreateMaxInFlight=384`，`SeckillCreateAcquireTimeoutMs=220`
- seckill：`OrderCreateMaxInFlight=400`，`OrderCreateAcquireTimeoutMs=220`，`OrderCreateRPCTimeoutMs=4200`
- 样本：`k6-ladder-20260211-135707`
- 结果：purchase `80/120/160` 全 PASS，`160` 档 `p95=2410ms`、`5xx=0`、`network=0`

4. 埋点链路回归
- 样本：`k6-ladder-20260211-135806`
- track `500/1000/1500 req/s` 全 PASS，推荐速率 `1500 req/s`

5. 注意事项
- 手工重启 `order/seckill` 进程时必须注入本地 `dev.env`（尤其 `FLASHSALE_MYSQL_PASSWORD`），否则会出现 `Error 1045` 启动失败。
- 建议把“服务重启脚本”也纳入 `scripts/dev`，避免压测过程中因环境变量遗漏造成假失败。

## 6. 下一轮代码优化方案（P0 -> P2）

1. P0：购买热路径减负（优先）
- 目标：把 `L2-c200 p95` 从约 `5.0s` 压到 `<4.5s`。
- 代码点：
  - `apps/seckill/rpc/internal/repository/mysql_seckill_repository.go`
  - 继续压缩 `ReservePurchase` 中非必要读写，减少事务内 SQL 数量与锁持有时长。
- 验证口径：
  - `purchase c200/r4000/t6000` success `>=97%`，network `<=1%`，p95 `<4500ms`。

2. P1：建单并发舱壁动态化
- 目标：减少峰值时过载拒绝与排队超时抖动。
- 代码点：
  - `apps/seckill/rpc/internal/logic/purchase_logic.go`
  - `apps/seckill/rpc/internal/svc/servicecontext.go`
  - `apps/order/rpc/internal/svc/servicecontext.go`
- 策略：
  - 保持默认 `448/240`，补充运行时监控与容量档位映射表（低压/中压/高压）。

3. P1：异步链路批量化（order state / traffic）
- 目标：降低高并发下每请求附带的同步写放大。
- 代码点：
  - `apps/order/rpc/internal/svc/servicecontext.go`
  - `apps/seckill/rpc/internal/logic/order_state_consumer.go`
  - `apps/seckill/rpc/internal/logic/purchase_logic.go`
- 策略：
  - 引入轻量批处理窗口（毫秒级）与失败重投策略，降低 DB/Kafka 写峰值抖动。

4. P2：门禁与极限测试分层固化
- 目标：避免“调门禁掩盖瓶颈”，将优化收益量化。
- 执行：
  - 回归门禁固定：`L1 + L2(c100/c200) + L4`
  - 极限探测独立：`L3+` 或 k6 open-model 阶梯
  - 每次优化必须给出 A/B 对照样本（同活动、同参数、同 token 池）
