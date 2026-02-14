# FlashSale 基础层与核心模块仓库

本仓库承载基于 `go-zero/goctl` 的基础能力与已落地核心模块实现。

## 当前实现范围

- 本地 Docker 环境：MySQL、Redis、Kafka
- 可选可观测性环境：Jaeger、Prometheus、Grafana
- 公共 Go 基础包：`pkg/base/*`
- 分库迁移脚本：`deploy/migrations/*`
- 基础连通性自检：`cmd/smoke/*`
- 用户 RPC 服务：`apps/user/rpc`
- 商品 RPC 服务：`apps/product/rpc`
- 订单 RPC 服务：`apps/order/rpc`
- 秒杀 RPC 服务：`apps/seckill/rpc`
- 管理员 RPC 服务：`apps/admin/rpc`
- 用户网关：`apps/gateway/user`
- 管理员网关：`apps/gateway/admin`

## 快速开始

1. 启动基础环境：

```powershell
go run ./cmd/fs env up
```

2. 执行数据库迁移：

```powershell
go run ./cmd/fs env migrate-up
```

3. 执行 smoke 检查：

```powershell
go run ./cmd/fs env smoke
```

4. 停止基础环境：

```powershell
go run ./cmd/fs env down
```

启用可观测性组件：

```powershell
go run ./cmd/fs env up --observability
```

## 独立运维控制台（第一版）

运维控制台与业务服务解耦，支持 Web 可视化与 CLI 双入口：

```powershell
# 0) 配置访问密钥（建议写入 configs/local/dev.env）
$env:FLASHSALE_OPS_ACCESS_KEY="replace_me_strong_key"

# 1) 启动独立可视化组件（容器日志与容器管理）
go run ./cmd/fs env ops-up

# 2) 启动 ops-control（任务编排 API + Web 页面）
go run ./cmd/fs ops server --addr 0.0.0.0:18080 --repo-root . --auth-key-env FLASHSALE_OPS_ACCESS_KEY

# 3) 访问
# ops-control: http://127.0.0.1:18080
# dozzle:     http://127.0.0.1:18081
# portainer:  http://127.0.0.1:19000
```

CLI 也可直接调 ops-control：

```powershell
go run ./cmd/fs ops tasks --key-env FLASHSALE_OPS_ACCESS_KEY
go run ./cmd/fs ops run --task env.start --key-env FLASHSALE_OPS_ACCESS_KEY
go run ./cmd/fs ops jobs --limit 20 --key-env FLASHSALE_OPS_ACCESS_KEY
go run ./cmd/fs ops logs --job <job_id> --key-env FLASHSALE_OPS_ACCESS_KEY
```

远程访问建议：
- 仅开放 `18080` 到受信网络。
- 生产建议在 Nginx 后挂 TLS，并加 IP 白名单。
- 密钥只放环境变量，不写入仓库。

## 端口与连接配置

开发 CLI 统一读取：`configs/local/dev.env`

- `FLASH_*`：Docker 对外端口
- `FLASHSALE_*`：应用连接覆盖参数
- `FLASH_MYSQL_ROOT_PASSWORD` / `FLASH_MYSQL_APP_PASSWORD`：数据库容器与应用账号密码
- `FLASH_GRAFANA_ADMIN_USER` / `FLASH_GRAFANA_ADMIN_PASSWORD`：Grafana 管理员凭据

如果本机端口被占用，只需修改 `configs/local/dev.env`，再执行 `fs` 命令即可。

## 组件版本

- MySQL: `mysql:8.4`
- Redis: `redis:7.2-alpine`
- Kafka: `confluentinc/cp-kafka:7.6.1`
- Jaeger: `jaegertracing/all-in-one:1.57`
- Prometheus: `prom/prometheus:v2.53.1`
- Grafana: `grafana/grafana:10.4.5`

## 镜像 Digest 记录

首次拉取镜像后，可执行：

```powershell
docker image inspect --format='{{index .RepoDigests 0}}' mysql:8.4
docker image inspect --format='{{index .RepoDigests 0}}' redis:7.2-alpine
docker image inspect --format='{{index .RepoDigests 0}}' confluentinc/cp-kafka:7.6.1
```

并将结果登记到 `docs/operations/image-digests.md`。

## 架构文档

- 文档索引：`docs/README.md`
- 系统与代码总览：`docs/architecture/01-system-and-code-architecture.md`
- 目录预览：`docs/architecture/02-directory-preview.md`
- 网关模块：`docs/architecture/modules/gateway-module.md`
- 用户模块：`docs/architecture/modules/user-module.md`
- 商品模块：`docs/architecture/modules/product-module.md`
- 订单模块：`docs/architecture/modules/order-module.md`
- 秒杀模块：`docs/architecture/modules/seckill-module.md`
- 管理员模块：`docs/architecture/modules/admin-module.md`
