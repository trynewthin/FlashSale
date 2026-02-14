# Ops 控制台使用指南

## 1. 定位

Ops 控制台是独立于业务后台的“运维控制面”，主要用于：

- 启停环境与服务
- 数据清理与覆写填充
- 触发压测任务
- 实时查看任务日志（SSE 流式）

## 2. 启动方式

### 2.1 启动可视化运维容器（可选）

```powershell
go run ./cmd/fs env ops-up
```

默认端口：

- dozzle：`18081`
- portainer：`19000`

### 2.2 启动 ops-control 服务

```powershell
$env:FLASHSALE_OPS_ACCESS_KEY="replace_me"
go run ./cmd/fs runtime start-ops-control --addr 0.0.0.0:18080 --auth-key-env FLASHSALE_OPS_ACCESS_KEY
```

访问：`http://127.0.0.1:18080`

## 3. CLI 调用 ops-control

```powershell
# 列任务
go run ./cmd/fs ops tasks --key-env FLASHSALE_OPS_ACCESS_KEY

# 触发任务
go run ./cmd/fs ops run --task env.start --key-env FLASHSALE_OPS_ACCESS_KEY

# 看任务列表
go run ./cmd/fs ops jobs --limit 20 --key-env FLASHSALE_OPS_ACCESS_KEY

# 看某任务日志
go run ./cmd/fs ops logs --job <job_id> --key-env FLASHSALE_OPS_ACCESS_KEY
```

## 4. 白名单任务（当前版本）

- 环境：`env.start`、`env.stop`、`env.migrate_up`、`env.migrate_down`
- 数据：`data.seed_overwrite`、`data.clear`
- 运行时：`runtime.start_backend`、`runtime.stop_backend`、`runtime.start_frontend`、`runtime.stop_frontend`、`runtime.start_ops_control`、`runtime.stop_ops_control`
- 压测：`perf.purchase_stress`、`perf.idempotency`、`perf.track_stress`、`perf.purchase_open`、`perf.track_open`

## 5. 日志与运行记录

- 服务日志：`.memory/runlogs/services/*.log`
- 前端日志：`.memory/runlogs/frontends/*.log`
- 任务日志：`.memory/runlogs/ops-jobs/<yyyyMMddHH>/<job_id>.log`
- 种子数据结果：`.memory/runlogs/seed-overwrite.result.json`

## 6. 远程访问建议（非生产演示环境）

- 仅暴露必要端口（如 `18080`）。
- 使用环境变量密钥控制访问，不将密钥写入仓库。
- 若挂 Nginx，对外统一走 HTTPS 与反向代理。
