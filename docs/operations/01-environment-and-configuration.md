# 环境与配置

## 1. 运行基础依赖

系统当前围绕本地或单机 Docker Compose 运行，核心依赖包括：

- Docker / Docker Compose
- Go 1.25
- Bun

## 2. 主要配置入口

### `configs/deploy.env`

部署相关环境变量的主要入口，负责：

- 宿主机端口映射
- 数据库、Grafana、media-store 等凭据
- 服务对外端口
- 应用运行时连接参数
- 运维控制台访问密钥

### `configs/dev.yaml`

开发运行所需的配置文件，用于本地 smoke 和服务配置补充。

## 3. 宿主机端口概览

| 组件 | 默认宿主机端口 |
| --- | --- |
| MySQL | `13306` |
| Redis | `16379` |
| Kafka | `39092` |
| Nginx | `18000` |
| ops-control | `9100` |
| CDN | `19000` |
| media-store | `19001` |
| Jaeger UI | `16686` |
| Prometheus | `9090` |
| Grafana | `3000` |

## 4. 配置原则

- 文档以当前仓库存在的入口为准，不复用已失效命令。
- 优先调整 `configs/deploy.env`，不要把端口和凭据散落到多个脚本里。
- 如果本机端口被占用，先改 env 文件，再执行重新部署。

## 5. 需要特别注意的点

- `FLASH_MEDIA_STORE_SECRET` 必须显式配置，避免文件管理链路失控。
- `FLASHSALE_OPS_ACCESS_KEY` 控制运维控制台访问，应视为敏感配置。
- etcd 宿主机端口要避开系统保留区间和本机已有监听。
