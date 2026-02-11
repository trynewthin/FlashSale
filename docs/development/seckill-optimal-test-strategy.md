# 秒杀压测最优方案（实战版）

更新时间：2026-02-11

## 1. 目标

- 降低每轮压测准备成本，避免无效重跑。
- 同时覆盖回归门禁（稳定性）与洪峰模拟（真实流量形态）。
- 输出可比的结构化结果，便于阶段对账与回归比较。

## 2. 采用的双模型策略

### 2.1 闭环并发模型（现有主力）

- 工具：`cmd/perf/seckillload` + `scripts/dev/seckill-test-suite.ps1`
- 用途：回归门禁、功能正确性、对账与幂等验证。
- 特点：每个并发请求等待响应后再发下一个请求，适合发现链路稳定性问题。

### 2.2 开环到达率模型（已落地）

- 工具：k6 `constant-arrival-rate`（脚本：`scripts/perf/k6/seckill_open_model.js`，入口：`scripts/dev/seckill-k6-open.ps1`）
- 用途：模拟秒杀洪峰、评估固定 RPS 下系统是否掉队。
- 依据：
  - k6 对 open/closed model 区分明确：<https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/>
  - 常量到达率执行器说明：<https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/constant-arrival-rate/>
  - arrival-rate 的 VU 分配机制：<https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/arrival-rate-vu-allocation/>

## 3. 已落地的效率优化

1. 压测工具二进制复用
- `seckill-perf.ps1` 支持 `-RunnerMode binary -BuildBinary`
- `seckill-test-suite.ps1` 支持 `-LoadRunnerMode binary`
- 避免每阶段 `go run` 重新编译。

2. 自动容量搜索
- 新增：`scripts/dev/seckill-capacity-search.ps1`
- 自动输出推荐稳定并发。

3. 门禁并发可配置
- `seckill-test-suite.ps1` 新增：
  - `L2PurchaseConcurrency/L2PurchaseRequests`
  - `L3PurchaseConcurrency/L3PurchaseRequests`

4. 阶段稳定化探针
- `EnableStageStabilize=true`（默认）
- 阶段间先跑小流量探针，再进入下一阶段，减少串扰误判。

## 4. 标准执行顺序

1. 容量搜索（2~4 分钟）
```powershell
./scripts/dev/seckill-capacity-search.ps1 -Prepare -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt -LoadRunnerMode binary
```

2. 门禁验收（按容量结果回填并发）
```powershell
./scripts/dev/seckill-test-suite.ps1 -Prepare -EnableL3 -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt -LoadRunnerMode binary -L2PurchaseConcurrency 120 -L2PurchaseRequests 2400 -L3PurchaseConcurrency 140 -L3PurchaseRequests 2800
```

2.1 strict 同口径推荐模板（触发限购路径）
```powershell
$env:FLASHSALE_SECKILL_RESERVE_DB_USER_LIMIT_CHECK = "false"
./scripts/dev/seckill-test-suite.ps1 `
  -GateProfile strict -EnableL3:$true -Prepare `
  -UserLimitQty 100000 `
  -RestartRuntimeBeforeRun:$true `
  -RuntimeForceIPv4Loopback:$true `
  -RuntimeMySQLMaxOpenConns 32 -RuntimeMySQLMaxIdleConns 8 `
  -RuntimeReservePurchaseTimeoutMs 5000
```

3. 问题复现场景（仅失败时）
```powershell
./scripts/dev/seckill-issues.ps1 -ActivityId <activity_id> -ItemId <item_id> -TokenFile .memory/runlogs/user.tokens.txt
```

4. 开环到达率压测（补充固定 RPS 视角）
```powershell
./scripts/dev/seckill-k6-open.ps1 -Prepare -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt -PurchaseRate 120 -TrackRate 0 -DurationSeconds 30
```

5. 开环阶梯压测（自动找 RPS 拐点）
```powershell
./scripts/dev/seckill-k6-ladder.ps1 -Mode purchase -Rates "80,120,160" -DurationSeconds 10 -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt
./scripts/dev/seckill-k6-ladder.ps1 -Mode track -Rates "500,1000,1500" -DurationSeconds 10 -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt
```

## 5. 本机最新实测结论（2026-02-11）

- 容量搜索：`capacity-20260211-131525`
  - 推荐稳定并发：`120`
- 门禁样本：`suite-20260211-132417`
  - 全门禁 PASS
  - L3(`c140`)：success `100%`、network `0%`、p95 `2792ms`、p99 `3086ms`
- 同口径 strict（`UserLimitQty=100000`）最新样本（启用 L3）：
  - `strict-off-tune5000-conn32-rerun/suite-20260211-230407`：PASS
    - L2-c200：success `97.18%`、p95 `4822.06ms`、network `0.27%`
    - L3-c200：success `96.45%`、p95 `5241.97ms`、network `0.08%`
  - `strict-off-tune5000-conn32-rerun/suite-20260211-230650`：PASS
    - L2-c200：success `97.75%`、p95 `4410.17ms`、network `0.30%`
    - L3-c200：success `96.12%`、p95 `5321.09ms`、network `0.10%`
  - `strict-off-tune5000-conn32-rerun/suite-20260211-230934`：PASS
    - L2-c200：success `97.28%`、p95 `4758.93ms`、network `0.32%`
    - L3-c200：success `96.38%`、p95 `5228.63ms`、network `0.05%`
  - 对照：`strict-off-default4000-conn32/suite-20260211-223516` 在 L2-c200 `success=94.95%`（strict 失败）。
  - 结论：在当前机器与链路口径下，`MySQL 32/8 + ReservePurchaseTimeoutMs=5000` 是更稳的 strict 默认档。
- 本地网络口径结论：
  - 使用 `localhost` 时出现 `dial tcp [::1]:13306 i/o timeout` 会污染压测样本。
  - 压测前建议固定 `RuntimeForceIPv4Loopback=true`，强制走 `127.0.0.1`。

## 6. 实操注意事项

1. 压测前确保管理员网关与核心 RPC 全部健康。
2. 使用 `seckill-prepare.ps1` 保证活动新鲜且库存充足。
3. 压测中如出现大量 `SYS_INTERNAL/DB_ERROR`，优先判定为下游过载，不要先怀疑“同 IP 限流”。
4. 限流开关建议通过环境变量控制，压测时按需放开强度：
- `FLASHSALE_RATE_LIMIT_ENABLED`
- `FLASHSALE_SECKILL_PURCHASE_RATE_LIMIT_ENABLED`
- `FLASHSALE_SECKILL_TRACK_RATE_LIMIT_ENABLED`
5. 压测前优先使用脚本内置运行时重启参数，避免“连接池/本地解析口径”干扰结果：
- `RestartRuntimeBeforeRun`
- `RuntimeForceIPv4Loopback`
- `RuntimeMySQLMaxOpenConns/RuntimeMySQLMaxIdleConns`
- `RuntimeReservePurchaseTimeoutMs`

## 7. Web 参考

- k6 Scenarios Concepts（Open vs Closed）
  - <https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/>
- k6 Constant Arrival Rate
  - <https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/constant-arrival-rate/>
- k6 Arrival-rate VU Allocation
  - <https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/arrival-rate-vu-allocation/>
- Vegeta（恒定速率压测 + 避免 coordinated omission）
  - <https://github.com/tsenart/vegeta>
- wrk2（恒吞吐与 coordinated omission 说明）
  - <https://github.com/giltene/wrk2>
