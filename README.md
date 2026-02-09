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
./scripts/dev/up.ps1
```

2. 执行数据库迁移：

```powershell
./scripts/dev/migrate-up.ps1
```

3. 执行 smoke 检查：

```powershell
./scripts/dev/smoke.ps1
```

4. 停止基础环境：

```powershell
./scripts/dev/down.ps1
```

启用可观测性组件：

```powershell
./scripts/dev/up.ps1 -Observability
```

## 端口与连接配置

开发脚本统一读取：`configs/local/dev.env`

- `FLASH_*`：Docker 对外端口
- `FLASHSALE_*`：应用连接覆盖参数
- `FLASH_MYSQL_ROOT_PASSWORD` / `FLASH_MYSQL_APP_PASSWORD`：数据库容器与应用账号密码
- `FLASH_GRAFANA_ADMIN_USER` / `FLASH_GRAFANA_ADMIN_PASSWORD`：Grafana 管理员凭据

如果本机端口被占用，只需修改 `configs/local/dev.env`，再执行脚本即可。

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
- 系统总览：`docs/architecture/system-overview.md`
- 网关运行态：`docs/architecture/gateway-runtime.md`
- 用户 RPC 运行态：`docs/architecture/user-rpc-runtime.md`
- 商品 RPC 运行态：`docs/architecture/product-rpc-runtime.md`
- 订单 RPC 运行态：`docs/architecture/order-rpc-runtime.md`
- 管理员模块设计：`docs/architecture/admin-module-design.md`
