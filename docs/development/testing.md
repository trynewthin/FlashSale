# 测试说明

更新时间：2026-02-11

## 1. 全量测试

```powershell
go test ./...
```

## 2. 用户仓储集成测试

文件：`apps/user/rpc/internal/repository/mysql_user_repository_integration_test.go`

该测试依赖真实 MySQL，执行前需设置：

- `FLASHSALE_TEST_MYSQL_DSN`

当 DSN 未设置或 `users` 表不存在时，测试会 `Skip`。

## 3. 现有测试分布

- 基础能力层：`pkg/base/*_test.go`
- 网关配置与鉴权：`apps/gateway/*/internal/*_test.go`
- 用户 RPC 逻辑与鉴权：`apps/user/rpc/internal/*_test.go`
- 商品 RPC 逻辑、鉴权与仓储：`apps/product/rpc/internal/*_test.go`
- 订单 RPC 逻辑、鉴权与仓储：`apps/order/rpc/internal/*_test.go`
- 秒杀 RPC 逻辑与链路：`apps/seckill/rpc/internal/logic/*_test.go`
- 管理员网关鉴权与处理：`apps/gateway/admin/internal/*_test.go`
- 管理员 RPC 配置：`apps/admin/rpc/internal/config/*_test.go`

## 4. 订单仓储集成测试

文件：`apps/order/rpc/internal/repository/mysql_order_repository_integration_test.go`

依赖真实 MySQL，执行前需设置：

- `FLASHSALE_TEST_MYSQL_DSN`

当 DSN 未设置或 `orders/order_events` 表不存在时，测试会 `Skip`。

## 5. 秒杀压测与问题场景脚本

仓库已提供本地可执行的秒杀压测脚本：

- `scripts/dev/seckill/seckill-perf.ps1`：单场景压测（购买/幂等/埋点）
- `scripts/dev/seckill/seckill-prepare.ps1`：自动创建并发布压测活动（避免活动过期、库存脏数据）
- `scripts/dev/seckill/seckill-issues.ps1`：典型秒杀问题场景组合测试
- `scripts/dev/seckill/seckill-test-suite.ps1`：分层门禁压测（预检->幂等->容量分档->埋点->对账->Markdown报告）
- `scripts/dev/seckill/seckill-capacity-search.ps1`：快速容量搜索（自动扫描并发阶梯，输出稳定并发上限）
- `scripts/dev/seckill/seckill-k6-open.ps1`：k6 开环压测（到达率模型，支持 Docker 执行）
- `scripts/dev/seckill/seckill-k6-ladder.ps1`：k6 开环阶梯压测（自动跑多档 RPS 并输出拐点报告）
- `scripts/dev/seckill/restart-seckill-runtime.ps1`：压测前重启核心服务（自动加载 `configs/local/dev.env`，避免环境变量缺失）

示例：

```powershell
# 购买压力
./scripts/dev/seckill/seckill-perf.ps1 -Scenario purchase-stress -ActivityId 1001 -ItemId 2001 -TokenFile .\tokens.txt -Concurrency 200 -Requests 5000

# 同幂等键冲突
./scripts/dev/seckill/seckill-perf.ps1 -Scenario idempotency -ActivityId 1001 -ItemId 2001 -Token "<jwt>" -Concurrency 50 -Requests 200

# 埋点高并发（建议限制每主机并发连接，减少本机连接风暴误差）
./scripts/dev/seckill/seckill-perf.ps1 -Scenario track-stress -ActivityId 1001 -ItemId 2001 -Concurrency 300 -Requests 6000 -MaxConnsPerHost 200

# 典型问题组合（幂等+抢购+埋点）
./scripts/dev/seckill/seckill-issues.ps1 -ActivityId 1001 -ItemId 2001 -TokenFile .\tokens.txt

# 自动准备新活动后再执行组合测试
./scripts/dev/seckill/seckill-issues.ps1 -Prepare -BaseUrl http://127.0.0.1:8082 -AdminBaseUrl http://127.0.0.1:8083 -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .\tokens.txt

# 推荐：一键分层门禁压测（默认跑到 L2，失败即停，自动产出报告）
./scripts/dev/seckill/seckill-test-suite.ps1 -Prepare -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt

# 稳定门禁（推荐日常回归）与严格门禁（性能压测）两档
./scripts/dev/seckill/seckill-test-suite.ps1 -Prepare -GateProfile stable -EnableL3 -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt
./scripts/dev/seckill/seckill-test-suite.ps1 -Prepare -GateProfile strict -EnableL3 -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt

# 如果要评估“限购聚合校验”开关收益，请设置 UserLimitQty>0（否则该校验路径不会触发）
./scripts/dev/seckill/seckill-test-suite.ps1 -Prepare -GateProfile strict -EnableL3 -UserLimitQty 100000 -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt

# 推荐先做容量快速搜索，再跑门禁（减少盲目重跑）
./scripts/dev/seckill/seckill-capacity-search.ps1 -Prepare -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt

# 需要探测高压拐点时再开启 L3
./scripts/dev/seckill/seckill-test-suite.ps1 -Prepare -EnableL3 -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt

# 连续压测诊断（不中断、自动补跑 L2 诊断层）
./scripts/dev/seckill/seckill-test-suite.ps1 -Prepare -EnableL3 -StopOnFirstFailure false -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt

# 按目标机器调参（避免客户端参数过小导致“假失败”）
./scripts/dev/seckill/seckill-test-suite.ps1 -Prepare -EnableL3 -StopOnFirstFailure false -L2PurchaseTimeoutMs 7000 -L2PurchaseMaxConnsPerHost 240 -L3PurchaseTimeoutMs 9000 -L3PurchaseMaxConnsPerHost 300 -TrackTimeoutMs 3500 -TrackMaxConnsPerHost 240 -StageCooldownSeconds 8 -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt

# 自定义门禁阈值（用于不同机器规格的阶段性目标）
./scripts/dev/seckill/seckill-test-suite.ps1 -Prepare -EnableL3 -StopOnFirstFailure false -L3PurchaseTimeoutMs 9000 -L3MaxP95Ms 7600 -L3MaxP99Ms 8200 -L3MinSuccessRate 0.95 -L3MaxNetworkErrorRate 0.05 -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt

# 二进制模式（只编译一次，后续复用，减少 go run 编译开销）
./scripts/dev/seckill/seckill-test-suite.ps1 -Prepare -LoadRunnerMode binary -BuildLoadBinary
./scripts/dev/seckill/seckill-perf.ps1 -Scenario purchase-stress -ActivityId 1001 -ItemId 2001 -TokenFile .\tokens.txt -RunnerMode binary -BuildBinary

# 按容量搜索结果调整门禁并发档位（避免固定 c200/c300 无效重跑）
./scripts/dev/seckill/seckill-test-suite.ps1 -Prepare -EnableL3 -L2PurchaseConcurrency 120 -L2PurchaseRequests 2400 -L3PurchaseConcurrency 160 -L3PurchaseRequests 3200 -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt

# k6 开环压测（固定到达率，默认走 Docker）
./scripts/dev/seckill/seckill-k6-open.ps1 -Prepare -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt -PurchaseRate 120 -TrackRate 0 -DurationSeconds 30

# k6 开环阶梯压测（自动输出推荐 RPS）
./scripts/dev/seckill/seckill-k6-ladder.ps1 -Mode purchase -Rates "80,120,160" -DurationSeconds 10 -AdminUsername admin_root -AdminPassword Admin1234 -TokenFile .memory/runlogs/user.tokens.txt

# 压测前重启核心服务（默认重启 order/seckill，可选加上 user-gateway）
./scripts/dev/seckill/restart-seckill-runtime.ps1
./scripts/dev/seckill/restart-seckill-runtime.ps1 -RestartUserGateway

# 压测前重启并固定本地网络口径（推荐）
# - 强制 IPv4 回环，避免 localhost 命中 IPv6 回环超时
# - 连接池保持安全默认值，避免触发 MySQL "Too many connections"
./scripts/dev/seckill/restart-seckill-runtime.ps1 -ForceIPv4Loopback true -MySQLMaxOpenConns 32 -MySQLMaxIdleConns 8

# strict 同口径（UserLimitQty>0）推荐模板：
# 1) 关闭 DB 限购聚合校验（最终一致口径）
# 2) 将秒杀预扣超时设为 5000ms（当前稳定档）
$env:FLASHSALE_SECKILL_RESERVE_DB_USER_LIMIT_CHECK = "false"
./scripts/dev/seckill/seckill-test-suite.ps1 -GateProfile strict -EnableL3:$true -Prepare -UserLimitQty 100000 -RuntimeReservePurchaseTimeoutMs 5000 -RestartRuntimeBeforeRun:$true
```

对应 Go 实现入口：

- `cmd/perf/seckillload/main.go`

说明：

- 购买场景必须提供 `-Token` 或 `-TokenFile`。
- `seckill-prepare.ps1` 支持两种管理员认证：`-AdminUsername/-AdminPassword`（推荐，自动登录拿新 token）或 `-AdminToken/-AdminTokenFile`。
- `seckill-prepare.ps1` 默认会先确保压测商品库存充足（`-EnsureProductStock`，默认目标 `-ProductStockTarget 60000`，并保证不低于 `reserved_stock_total * 2`），减少“准备阶段因库存不足频繁降级”的样本污染。
- 幂等场景断言口径是“唯一订单数 <= 1”，允许同幂等键返回同一成功结果。
- `-ExpectMaxSuccess` 在普通场景校验成功数上限；在幂等场景校验唯一订单数上限（推荐设为 `1`）。
- `-MaxNetworkErrors` 用于限制网络错误容忍度（默认 `999999`，建议通过成功率/网络错误率阈值统一判定，不再因单次网络抖动提前中断）。
- `-MaxIdleConns/-MaxIdleConnsPerHost/-MaxConnsPerHost/-DisableKeepAlive` 用于调节压测客户端连接池，帮助区分“客户端连接风暴”与“服务端瓶颈”。
- 输出包含：吞吐（RPS）、成功数、业务码分布、HTTP 状态分布、延迟分位（P50/P95/P99）。
- `seckill-test-suite.ps1` 产物目录为 `.memory/runlogs/suite-<timestamp>/`，包含每阶段 raw/json 与 `suite-report.md`。
- `seckill-test-suite.ps1` 在 `-Prepare` 时会按库存阶梯自动回退（输入值 -> 12000 -> 8000 -> 5000），减少因为库存不足导致的测试中断。
- `seckill-test-suite.ps1` 新增 `UserLimitQty` 参数，透传到 `seckill-prepare.ps1`；当 `UserLimitQty=0` 时数据库限购聚合校验路径不会触发。
- `seckill-test-suite.ps1` 默认会清理历史 `perf-activity-*` 活动并在测试后下线当前活动，避免长期占用库存污染后续压测口径。
- `seckill-test-suite.ps1` 的 `StopOnFirstFailure/AutoDiagnoseOnL2Failure` 支持 `true/false/1/0`，用于控制“失败即停”与“L2 自动诊断层”。
- `seckill-test-suite.ps1` 在 `-AdminToken/-AdminTokenFile` 失效时，会自动回退到 `-AdminUsername/-AdminPassword` 登录（也可通过环境变量 `FLASHSALE_ADMIN_USERNAME/FLASHSALE_ADMIN_PASSWORD` 注入）。
- `seckill-prepare.ps1`、`seckill-issues.ps1`、`seckill-test-suite.ps1`、`seckill-capacity-search.ps1`、`seckill-k6-open.ps1`、`seckill-k6-ladder.ps1` 默认会自动加载 `configs/local/dev.env`（`-AutoLoadDevEnv=true`），压测前无需手工导出管理员环境变量。
- `seckill-test-suite.ps1` 新增运行时重启参数：
  - `RestartRuntimeBeforeRun`
  - `RuntimeForceIPv4Loopback`
  - `RuntimeMySQLMaxOpenConns/RuntimeMySQLMaxIdleConns`
  - `RuntimeReservePurchaseTimeoutMs`
- `restart-seckill-runtime.ps1` 新增压测注入参数：
  - `ForceIPv4Loopback`
  - `MySQLMaxOpenConns/MySQLMaxIdleConns`
  - `SeckillReservePurchaseTimeoutMs`
- `suite-report.md` 的 `Gate` 列表示是否计入门禁结果；诊断层 `Gate=N`，仅用于根因判断，不阻塞门禁。
- `L2/L3` 的成功率、网络错误率、P95/P99 门禁阈值支持脚本参数覆盖，便于按压测机规格做阶段性目标管理。
- `GateProfile` 支持 `stable/strict` 两档：
  - `stable`：日常回归推荐，降低边界抖动误报；
  - `strict`：优化阶段推荐，收紧尾延迟与成功率门槛。
- `L2PurchaseConcurrency/L2PurchaseRequests/L3PurchaseConcurrency/L3PurchaseRequests` 支持自定义，建议由容量搜索结果驱动。
- `L2-c100` 门禁参数已开放：`L2C100Concurrency/L2C100Requests/L2C100TimeoutMs/L2C100MaxConnsPerHost/L2C100MinSuccessRate/L2C100MaxNetworkErrorRate/L2C100MaxP95Ms/L2C100MaxP99Ms`，避免被硬编码阈值误伤。
- 当前默认 `L3` 已调整为稳定档：`c200/r4000/timeout=7000/maxConns=220`，用于日常回归；需探测极限时再手工升到 `c300+`。
- `StageCooldownSeconds` 用于阶段间冷却，减少 L2 压力残留对 L3 的污染，默认 `6` 秒。
- `EnableStageStabilize` 默认开启，会在阶段间做小流量探针，探针达标后再进入下一阶段，减少“残余压力污染样本”。
- `LoadRunnerMode=binary` 可显著降低多阶段压测耗时，建议在稳定回归中默认使用。
- 建议执行顺序：`seckill-capacity-search.ps1`（快速找拐点）-> `seckill-test-suite.ps1`（门禁验收）-> `seckill-issues.ps1`（问题复现）。
- `seckill-k6-open.ps1` 用于补充开环模型，验证固定 RPS 下的真实退化点；与闭环门禁配合使用。
- 手工重启 `order/seckill/user-gateway` 前建议执行 `restart-seckill-runtime.ps1`，确保 `FLASHSALE_MYSQL_PASSWORD` 等环境变量已注入。

