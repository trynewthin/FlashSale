# FlashSale 脚本使用指南

> 本文档基于 `scripts/` 目录下的所有脚本源码编写，涵盖集群管理、数据填充、冒烟测试及实用技巧。

---

## 目录

- [1. 架构概览](#1-架构概览)
- [2. 集群脚本（scripts/clusters/）](#2-集群脚本scriptsclusters)
  - [2.1 _common.sh — 公共 Helper](#21-_commonsh--公共-helper)
  - [2.2 infra.sh — 集群A：数据基础设施](#22-infrash--集群a数据基础设施)
  - [2.3 backend.sh — 集群B：后端业务服务](#23-backendsh--集群b后端业务服务)
  - [2.4 proxy.sh — 集群C：反向代理 / CDN / 文件管理](#24-proxysh--集群c反向代理--cdn--文件管理)
  - [2.5 ops.sh — 集群D：Ops 控制面](#25-opssh--集群dops-控制面)
  - [2.6 observability.sh — 集群E：可观测性（可选）](#26-observabilitysh--集群e可观测性可选)
- [3. 总调度脚本](#3-总调度脚本)
  - [3.1 rebuild.sh — 构建与部署](#31-rebuildsh--构建与部署)
  - [3.2 reset-env.sh — 完全重置](#32-reset-envsh--完全重置)
  - [3.3 smoke-test.sh — 冒烟测试](#33-smoke-testsh--冒烟测试)
- [4. 数据管理命令（fs data）](#4-数据管理命令fs-data)
  - [4.1 data clear](#41-data-clear)
  - [4.2 data seed-overwrite](#42-data-seed-overwrite)
  - [4.3 data seed-products](#43-data-seed-products)
- [5. 实用技巧与最佳实践](#5-实用技巧与最佳实践)
- [6. 常见场景速查](#6-常见场景速查)

---

## 1. 架构概览

FlashSale 的 Docker 服务被划分为 **5 个集群**，按依赖关系分层启动：

```
[A] infra        MySQL / Redis / Kafka / etcd
     ↓
[B] backend      migrate + rpc×5 + gateway×2              ← 需要 infra healthy
     ↓
[C] proxy        nginx + cdn + media-store                ← 需要 backend gateway healthy
     ↓
[D] ops          ops-control (embed 前端 + docker.sock)   ← 需要 backend 镜像
[E] observability  jaeger / prometheus / grafana          ← 可选
```

Docker Compose 配置使用 `include` 指令拆分为 4 个子文件：

```
deploy/compose/
├── docker-compose.app.yml    ← 主入口（include 4 个子文件）
├── app/
│   ├── infra.yml             ← MySQL, Redis, Kafka, etcd
│   ├── backend.yml           ← migrate, 5 RPC, 2 gateway
│   ├── proxy.yml             ← Nginx, CDN, media-store, ops-control
│   └── observability.yml     ← Jaeger, Prometheus, Grafana
```

全量构建时，`rebuild.sh` 自动按此依赖关系编排：infra → backend∥proxy 并行构建 → backend up → proxy up → ops → observability(可选)。

> **构建可见性**：`backend.Dockerfile` 采用多 stage 架构，每个 Go 二进制（共 9 个）拥有独立的 BuildKit stage，编译进度逐个实时可见，且只有变更代码对应的二进制会重新编译。

---

## 2. 集群脚本（scripts/clusters/）

每个集群对应 `scripts/clusters/` 下的一个 `.sh` 文件，统一支持 `build`、`up`、`all` 子命令。

### 2.1 _common.sh — 公共 Helper

**不要直接执行此文件**，它被各集群脚本 `source` 引入。

提供的公共函数：

| 函数 | 用途 |
|------|------|
| `dc <args>` | 封装 `docker compose -f <compose文件>`，自动输出命令日志 |
| `dc_build [--no-cache] services...` | 构建镜像 |
| `dc_up [--force-recreate] services...` | 后台启动服务 |
| `wait_healthy <timeout_sec> containers...` | 轮询等待容器 healthy，超时返回 1 |
| `run_parallel "cmd1" "cmd2" ...` | 并行执行多个命令，任一失败则整体失败 |
| `step/ok/warn/fail/info` | 带颜色的日志输出 |

### 2.2 infra.sh — 集群A：数据基础设施

管理 MySQL、Redis、Kafka、etcd 四个基础服务。

```bash
# 启动（已运行则跳过）
bash scripts/clusters/infra.sh

# 停止（保留数据）
bash scripts/clusters/infra.sh down

# ⚠️ 停止并清除所有数据卷（需要手动确认输入 'yes'）
bash scripts/clusters/infra.sh down-volumes
```

**健康检查**：等待所有 4 个容器 healthy，超时 120 秒。

### 2.3 backend.sh — 集群B：后端业务服务

包含：migrate(job) + user-rpc / product-rpc / order-rpc / seckill-rpc / admin-rpc + user-gateway / admin-gateway。

```bash
# 完整构建 + 启动
bash scripts/clusters/backend.sh

# 只构建镜像（不启动，用于并行编排）
bash scripts/clusters/backend.sh build

# 只重启（镜像已存在）
bash scripts/clusters/backend.sh up

# 无缓存构建
bash scripts/clusters/backend.sh build --no-cache
```

**构建架构**：`backend.Dockerfile` 使用多 stage 设计，9 个 Go 二进制各自拥有独立的 BuildKit stage（`build-user-rpc`、`build-seckill-rpc` 等），编译进度逐个可见。仅修改了某个服务的代码时，其他服务直接走 Docker 层缓存。

**健康检查**：等待两个 gateway 容器 healthy，超时 120 秒。

### 2.4 proxy.sh — 集群C：反向代理 / CDN / 文件管理

包含：nginx（反向代理，依赖 gateway healthy）+ cdn（静态文件服务）+ media-store（文件管理）。cdn 和 media-store 共享 `deploy/cdn/assets` 卷。

> ⚠️ nginx depends_on user-gateway + admin-gateway，因此 proxy up 必须在 backend up 完成之后。

```bash
# 完整构建 + 启动
bash scripts/clusters/proxy.sh

# 只构建 media-store 镜像（nginx + cdn 无需构建）
bash scripts/clusters/proxy.sh build

# 只重启
bash scripts/clusters/proxy.sh up
```

### 2.5 ops.sh — 集群D：Ops 控制面

Ops 控制面 embed 前端 SPA，挂载 docker.sock 用于容器管理。**依赖 `flashsale-backend:local` 镜像**，因此必须在 backend build 完成后才能构建。

```bash
# 完整构建（编译前端 + 构建镜像 + 启动）
bash scripts/clusters/ops.sh

# 只构建镜像
bash scripts/clusters/ops.sh build

# 只重启
bash scripts/clusters/ops.sh up

# ⭐ 热更新（最快 ~15s，详见技巧章节）
bash scripts/clusters/ops.sh hot
```

#### `hot` 命令详解

这是最实用的开发加速命令：

1. 编译 ops 前端（`bun run build`）→ 同步到 `cmd/fs/internal/ops/web/`
2. 交叉编译 Go 二进制（`GOOS=linux GOARCH=amd64`）
3. `docker cp` 替换容器内的 `/app/bin/fs`
4. `docker restart` 重启容器
5. 等待 healthy（30s 超时）

**跳过了 Docker 镜像重建**，所以极快。

### 2.6 observability.sh — 集群E：可观测性（可选）

可选的监控组件：Jaeger（链路追踪）、Prometheus（指标）、Grafana（面板）。

```bash
# 启动
bash scripts/clusters/observability.sh

# 停止
bash scripts/clusters/observability.sh down
```

启动后的访问地址：
| 服务 | 地址 | 默认账号 |
|------|------|---------|
| Grafana | http://localhost:3000 | admin / admin |
| Prometheus | http://localhost:9090 | — |
| Jaeger | http://localhost:16686 | — |

---

## 3. 总调度脚本

### 3.1 rebuild.sh — 构建与部署

总调度脚本，按依赖关系编排所有集群的构建和启动。

```bash
# 全量重建所有集群
bash scripts/rebuild.sh

# 只重建后端
bash scripts/rebuild.sh --scope backend

# 只重建代理/CDN
bash scripts/rebuild.sh --scope proxy

# 只重建 ops
bash scripts/rebuild.sh --scope ops

# ⭐ ops 热更新（最快方式）
bash scripts/rebuild.sh --scope ops --hot

# 无缓存全量重建
bash scripts/rebuild.sh --no-cache

# 全量重建 + 启动可观测性
bash scripts/rebuild.sh --with-obs

# 全量重建 + 冒烟测试（需先 seed 数据）
bash scripts/rebuild.sh --with-test
```

> 注：`--scope cdn` 仍可使用（自动映射为 `proxy`），但建议使用新名称。

**全量构建的 4 个阶段**：

| 阶段 | 内容 | 并行性 |
|------|------|-------|
| Phase 1/4 | Infra up | 串行 |
| Phase 2/4 | Backend build ∥ Proxy build | **并行**（每个镜像内部 9 个 stage 也并行） |
| Phase 3/4 | Backend up → Proxy up | **串行**（proxy 依赖 gateway healthy） |
| Phase 4/4 | Ops build + up | 串行（依赖 backend 镜像） |

> `--low-mem` 模式下 Phase 2 改为串行，且 BuildKit 限制为单 stage 串行编译（`--max-parallelism 1` + `-p 2`）。

### 3.2 reset-env.sh — 完全重置

⚠️ **危险操作**：删除所有容器和数据卷，从零开始重建。

```bash
# 完全重置（5 秒倒计时，可 Ctrl+C 取消）
bash scripts/reset-env.sh

# 重置 + 可观测性
bash scripts/reset-env.sh --with-obs

# 重置 + 冒烟测试
bash scripts/reset-env.sh --with-test
```

执行流程：
1. 停止所有容器 + 删除数据卷
2. 调用 `rebuild.sh` 全量重建
3. 可选运行冒烟测试

### 3.3 smoke-test.sh — 冒烟测试

自动化测试 6 个维度的 API 可用性。

```bash
# 默认参数运行
bash scripts/smoke-test.sh

# 自定义 base URL
bash scripts/smoke-test.sh --base http://10.0.0.5

# 跳过某些 section
bash scripts/smoke-test.sh --skip ops,cdn

# 自定义端口
bash scripts/smoke-test.sh --nginx-port 18000
```

**测试 Section**：

| # | Section | 测试脚本 | 内容 |
|---|---------|---------|------|
| 1 | Infrastructure Health | test-infra.sh | 基础设施连通性 |
| 2 | Ops Control APIs | test-ops.sh | Ops 控制面 API |
| 3+4 | Business + User APIs | test-user.sh | 用户注册/登录/下单 |
| Auth | Auth Guard | test-auth.sh | 鉴权拦截测试 |
| 5 | Admin Management | test-admin.sh | 管理员用户管理 |
| 6 | CDN + Media Store | test-cdn.sh | 文件上传/下载 |

默认端口配置：
- Nginx Gateway: `18000`
- Ops Control: `9100`
- CDN: `19000`
- Media Store: `19001`

---

## 4. 数据管理命令（fs data）

`fs` 是项目的 CLI 工具，`data` 子命令管理演示数据。在 ops 容器中或本地执行。

### 4.1 data clear

清空所有业务数据表（TRUNCATE），刷新 Redis。

```bash
fs data clear --force

# 选项
--clear-admin          # 同时清空管理员账号和角色
--keep-infra-tables    # 保留 outbox/idempotency 表
--skip-redis-flush     # 跳过 Redis FLUSHDB
--env-file <path>      # 环境变量文件（使用 '-' 跳过加载）
```

清理的数据库和表：
- `flash_order`: orders, order_events
- `flash_seckill`: seckill_activities, seckill_activity_items, seckill_stock_ledger 等
- `flash_product`: products
- `flash_user`: users
- `flash_admin`: admin_audit_logs, admin_refresh_tokens（默认不清管理员账号）

### 4.2 data seed-overwrite

覆写式填充：清空数据 → 创建管理员 → 注册用户 → 创建 3 个商品（相对路径图片） → 创建秒杀活动 → 创建示例订单。

```bash
fs data seed-overwrite --force

# 常用选项
--seckill-duration-minutes 120    # 秒杀持续时间（默认 120 分钟）
--seckill-price-cent 9900         # 秒杀价格（分）
--seckill-reserved-stock 5000     # 预留库存
--admin-base-url http://...       # 管理网关地址
--user-base-url http://...        # 用户网关地址
```

输出文件位于 `log/data/`：
- `seed-overwrite.result.json` — 完整结果
- `admin.token.txt` / `user.token.txt` — Token
- `perf.product_id.txt` / `perf.activity_id.txt` / `perf.item_id.txt` — 性能测试用 ID

### 4.3 data seed-products

创建大量商品并自动建立秒杀活动，**不会清空已有数据**（新增式）。

```bash
fs data seed-products --force

# 常用选项
--count 30                     # 商品数量（默认 30）
--seckill-duration 60          # 秒杀活动时长，分钟（默认 60）
--admin-base-url http://...    # 管理网关地址
--nginx-base-url http://...    # Nginx 地址（图片上传）
```

执行流程：
1. 管理员登录
2. 逐个创建商品（带 SVG 占位图片上传）
3. 创建 1 个秒杀活动（标题："限时秒杀 MM-DD HH:mm"）
4. 把所有商品加入活动（秒杀价 = 原价 × 30%~70%，预留 10% 库存）
5. 发布活动
6. 写结果到 `log/data/seed-products.result.json`

---

## 5. 实用技巧与最佳实践

### 🚀 技巧 1：ops 热更新（最常用）

改了 Go 代码或 ops 前端后，不需要重建整个 Docker 镜像：

```bash
bash scripts/clusters/ops.sh hot
```

**原理**：本地交叉编译 → `docker cp` 替换二进制 → 重启容器。约 15 秒完成。

等价于：
```bash
bash scripts/rebuild.sh --scope ops --hot
```

### 🚀 技巧 2：只改后端代码时，用 scope 精准重建

```bash
bash scripts/rebuild.sh --scope backend
```

不会重建 Proxy/Ops，省时 50%+。

### 🚀 技巧 3：管道截取关键输出

脚本输出很多日志，只关心最终结果时可用管道截取：

```bash
# 只看最后 8 行（成功/失败摘要）
bash scripts/clusters/ops.sh hot 2>&1 | tail -8 | cat

# 冒烟测试只看结果汇总
bash scripts/smoke-test.sh 2>&1 | tail -5
```

### 🚀 技巧 4：seed 数据后立即看效果

```bash
# 清空 + 填充 30 个商品 + 秒杀活动
# 在 ops 控制面执行：
fs data clear --force
fs data seed-products --force --env-file=-

# 或在本地（需要 Go 环境）：
go run ./cmd/fs data clear --force
go run ./cmd/fs data seed-products --force
```

### 🚀 技巧 5：前端开发无需任何脚本

```bash
cd frontend/user && bun dev    # HMR 自动生效
cd frontend/ops && bun dev     # Ops 前端同理
```

改前端代码时浏览器自动刷新，完全不需要运行任何 cluster 脚本。

### 🚀 技巧 6：可选启动可观测性

```bash
# 单独启动
bash scripts/clusters/observability.sh

# 或全量重建时附带
bash scripts/rebuild.sh --with-obs
```

### 🚀 技巧 7：跳过特定冒烟测试

```bash
# 跳过 CDN 和 Ops 的测试
bash scripts/smoke-test.sh --skip ops,cdn
```

---

## 6. 常见场景速查

| 场景 | 命令 | 耗时预估 |
|------|------|---------|
| 首次搭建环境 | `bash scripts/rebuild.sh` | 5-10 分钟 |
| 完全重置（数据全清） | `bash scripts/reset-env.sh` | 5-10 分钟 |
| 只改了 Go 后端代码 | `bash scripts/rebuild.sh --scope backend` | 2-3 分钟 |
| 只改了 Go 代码（ops 相关） | `bash scripts/clusters/ops.sh hot` | **~15 秒** |
| 只改了 ops 前端 | `bash scripts/clusters/ops.sh hot` | **~15 秒** |
| 只改了 user 前端 | `bun dev` (HMR) | **即时** |
| 填充演示数据（30 商品 + 秒杀） | `fs data seed-products --force` | ~30 秒 |
| 清空所有业务数据 | `fs data clear --force` | ~2 秒 |
| 跑冒烟测试 | `bash scripts/smoke-test.sh` | ~15 秒 |
| 启动监控面板 | `bash scripts/clusters/observability.sh` | ~30 秒 |
| 基础设施停机（保留数据） | `bash scripts/clusters/infra.sh down` | ~5 秒 |
| 基础设施停机（清除数据） | `bash scripts/clusters/infra.sh down-volumes` | ~5 秒 |

---

> 📝 最后更新：2026-02-22  
> 📁 源文件目录：`scripts/`、`scripts/clusters/`、`scripts/tests/`、`cmd/fs/data_cmd.go`  
> 📁 Compose 配置：`deploy/compose/docker-compose.app.yml` → `deploy/compose/app/*.yml`  
> 📁 Dockerfile：`deploy/docker/backend.Dockerfile`（多 stage 架构）
