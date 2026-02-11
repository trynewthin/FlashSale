# 2026-02-11 秒杀开环压测阶梯报告

## 1. 执行目标

- 使用 k6 `constant-arrival-rate` 验证固定到达率下的链路稳定性。
- 对比购买链路与埋点链路的容量边界，补齐闭环并发模型之外的观测。

## 2. 测试方法

1. 使用 `scripts/dev/seckill-k6-ladder.ps1` 执行阶梯测试。
2. 每轮自动创建新活动（`seckill-prepare.ps1`），避免历史库存污染。
3. 结果判定口径：
- `effective_rate`：非基础设施失败占比（非 network/5xx/DB_ERROR/SYS_INTERNAL）。
- `network_rate`：网络错误占比。
- `p95`：HTTP 请求延迟 95 分位。

## 3. 购买链路（open model）

- 报告路径：`.memory/runlogs/k6-ladder-20260211-134505/ladder-report.md`
- 参数：
  - `duration=10s`
  - `rates=80,120,160 req/s`
  - threshold: `effective>=0.95`, `network<=0.03`, `p95<=5000ms`

结果：

| Rate(req/s) | EffectiveRate | NetworkRate | P95(ms) | 结论 |
|---:|---:|---:|---:|---|
| 80 | 0.8606 | 0 | 3225.37 | FAIL |
| 120 | 0.5442 | 0 | 3258.07 | FAIL |
| 160 | 0.6827 | 0 | 1988.14 | FAIL |

结论：

- 在开环固定到达率下，购买链路在 `80 req/s` 已出现明显基础设施失败。
- 失败特征不是网络错误，而是服务侧业务失败（`DB_ERROR/SYS_INTERNAL`）占比上升。
- 这与闭环模型中“可通过门禁”并不矛盾：闭环会因响应变慢而自动降发，开环不会。

## 4. 埋点链路（open model）

- 报告路径：`.memory/runlogs/k6-ladder-20260211-134610/ladder-report.md`
- 参数：
  - `duration=10s`
  - `rates=500,1000,1500 req/s`
  - threshold: `effective>=0.99`, `network<=0.01`, `p95<=500ms`

结果：

| Rate(req/s) | EffectiveRate | NetworkRate | P95(ms) | 结论 |
|---:|---:|---:|---:|---|
| 500 | 1.0 | 0 | 2.24 | PASS |
| 1000 | 1.0 | 0 | 3.34 | PASS |
| 1500 | 1.0 | 0 | 25.84 | PASS |

结论：

- 埋点链路在当前参数下具有明显更高容量余量。
- 当前核心瓶颈集中在购买建单链路，而非埋点链路。

## 5. 与闭环门禁的联合结论

- 闭环门禁（样本：`.memory/runlogs/suite-20260211-132417/suite-report.md`）可以在 `L3 c140` 通过。
- 开环阶梯显示购买链路在固定 `80 req/s` 已有基础设施失败。
- 阶段验收建议同时保留两类门禁：
  1. 闭环：保证“功能与稳定回归”；
  2. 开环：保证“固定流量目标下可持续服务”。

## 6. 后续优化建议（按优先级）

1. 优先优化 `seckill -> order CreateOrderFromSeckill` 路径（并发舱壁、超时、慢查询）。
2. 将购买链路 open-model 验收阈值纳入 CI 夜间任务（建议先从 `50/60/80 req/s` 阶梯开始）。
3. 对 `DB_ERROR/SYS_INTERNAL` 增加细分指标，区分连接池耗尽、事务冲突、超时取消来源。
