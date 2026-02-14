# Dev Scripts Layout

`scripts/dev` 按职责分层，避免所有脚本平铺在同一目录。

## 目录结构

```text
scripts/dev/
  common.ps1
  env/
    up.ps1
    down.ps1
    migrate-up.ps1
    migrate-down.ps1
    smoke.ps1
    start-docker-env.ps1
    clear-data.ps1
    seed-overwrite.ps1
  runtime/
    start-backend.ps1
    stop-backend.ps1
    start-frontend.ps1
    stop-frontend.ps1
  seckill/
    restart-seckill-runtime.ps1
    seckill-prepare.ps1
    seckill-perf.ps1
    seckill-issues.ps1
    seckill-test-suite.ps1
    seckill-capacity-search.ps1
    seckill-k6-open.ps1
    seckill-k6-ladder.ps1
```

## 常用命令

### 环境与数据

```powershell
# 启动 Docker + 迁移 + smoke
./scripts/dev/env/start-docker-env.ps1

# 覆写填充数据（危险操作）
./scripts/dev/env/seed-overwrite.ps1 -Force

# 清理数据（默认保留管理员账号/角色）
./scripts/dev/env/clear-data.ps1 -Force
```

### 服务启停

```powershell
# 启动后端全服务
./scripts/dev/runtime/start-backend.ps1

# 停止后端全服务
./scripts/dev/runtime/stop-backend.ps1

# 启动前端
./scripts/dev/runtime/start-frontend.ps1

# 停止前端
./scripts/dev/runtime/stop-frontend.ps1
```

### 秒杀压测

```powershell
# 分层套件
./scripts/dev/seckill/seckill-test-suite.ps1 -Prepare

# k6 开环
./scripts/dev/seckill/seckill-k6-open.ps1 -Prepare
```
