# FlashSale

FlashSale 是一个面向秒杀场景的电商系统项目，覆盖用户侧购物、后台管理、秒杀活动、运维控制和本地部署编排。它不是单一接口的压测样例，而是一套从业务链路到运行体系都相对完整的工程实现。

> 系统总览图预留：`docs/assets/images/README-system-overview.png`
>
> 演示 GIF 预留：`docs/assets/gifs/README-seckill-demo.gif`

## 这是什么

这个项目聚焦在“秒杀系统如何真正落成一套可运行的应用”。

除了抢购链路本身，仓库还包含：

- 用户端、管理端、运维端三套前端
- 用户、商品、订单、秒杀、管理员五个业务服务
- 网关、文件管理、CDN、运维控制台等外围能力
- Docker Compose、Nginx 和基础可观测性组件

## 核心特点

- 业务拆分清晰：用户、商品、订单、秒杀、管理员分别由独立 RPC 服务承载，通过用户网关和管理网关对外提供 API。
- 秒杀链路完整：下单流程结合 Kafka 异步建单、幂等控制、补偿处理和死信队列，覆盖高竞争场景下常见的问题。
- 三端闭环：用户端负责交易体验，管理端负责业务运营，运维端负责状态查看、任务和维护操作。
- 运维能力内建：项目自带 `ops-control`，可以结合容器状态、任务执行和运行信息做日常维护。
- 素材链路独立：`media-store` 负责上传与管理，`cdn` 负责对外只读访问，适合商品图和活动资源场景。
- 部署结构分层：基础设施、业务后端、代理与可观测性按 Compose 子文件拆分，便于扩展和排查问题。

## 系统组成

| 层次 | 组成 |
| --- | --- |
| 业务服务 | `user-rpc`、`product-rpc`、`order-rpc`、`seckill-rpc`、`admin-rpc` |
| API 入口 | `user-gateway`、`admin-gateway` |
| 辅助能力 | `media-store`、`cdn`、`ops-control` |
| 前端应用 | `frontend/user`、`frontend/admin`、`frontend/ops` |
| 基础设施 | MySQL、Redis、Kafka、etcd、Nginx、Prometheus、Grafana、Jaeger |

## 当前覆盖范围

当前版本已经完成基础阶段所需的主要骨架和链路，包括：

- 用户、商品、订单、秒杀、管理员相关的核心业务能力
- 图片管理、CDN 访问和项目专用运维控制台
- 限流、异步消费、补偿、服务发现和基础可观测性
- 面向本地或单机环境的 Docker 部署能力

项目介绍以当前仓库已经实现的能力为准。

## 技术栈

- 后端：Go 1.25、go-zero、gRPC / Proto、MySQL、Redis、Kafka、etcd
- 前端：React 19、TypeScript、Vite、Tailwind CSS 4、TanStack Query、Zustand
- 部署与观测：Docker Compose、Nginx、Prometheus、Grafana、Jaeger

## 文档导航

- 文档总览：[docs/README.md](docs/README.md)
- 系统需求定位：[docs/positioning/README.md](docs/positioning/README.md)
- 系统架构：[docs/architecture/README.md](docs/architecture/README.md)
- 部署与维护：[docs/operations/README.md](docs/operations/README.md)
- 图片与 GIF 占位：[docs/assets/README.md](docs/assets/README.md)
