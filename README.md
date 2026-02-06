# FlashSale 基础设施底座

本仓库用于承载基于 `go-zero/goctl` 微服务体系的基础层能力，不包含具体业务模块实现。

## 覆盖范围

- 本地 Docker 环境：MySQL、Redis、Kafka
- 可选可观测性环境：Jaeger、Prometheus、Grafana
- 公共 Go 基础包：`pkg/base/*`
- 分库迁移脚本：`deploy/migrations/*`
- 基础连通性自检：`cmd/smoke/*`

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

并将结果登记到 `docs/image-digests.md`。

## 架构文档

- 代码架构说明：`docs/code-architecture.md`
