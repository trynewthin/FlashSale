# 仓库结构与代码组织

## 1. 顶层目录

| 目录 | 作用 |
| --- | --- |
| `apps/` | 业务服务与网关实现 |
| `frontend/` | 用户端、管理端、运维端前端 |
| `ops/` | 运维后端、CLI、前端嵌入物和执行器 |
| `deploy/` | Compose、Dockerfile、Nginx、迁移和静态资源目录 |
| `configs/` | 环境配置与运行配置 |
| `pkg/` | 公共基础库与通用能力 |
| `docs/` | 项目文档 |

## 2. 服务代码组织

### `apps/`

- `apps/user/rpc`
- `apps/product/rpc`
- `apps/order/rpc`
- `apps/seckill/rpc`
- `apps/admin/rpc`
- `apps/gateway/user`
- `apps/gateway/admin`
- `apps/media-store`

这种组织方式把业务域与接入层分开，便于单独演进服务配置、Proto 和逻辑。

## 3. 前端代码组织

- `frontend/user`：用户站点。
- `frontend/admin`：后台管理站点。
- `frontend/ops`：运维面板站点。

三端都采用 React + TypeScript + Vite，但运维端额外承担拓扑、容器和任务可视化。

## 4. 运维与命令入口

- `ops/cmd`：当前 CLI 主入口。
- `ops/backend`：`ops-control` 后端实现。
- `ops/web`：运维前端嵌入目录。
- `ops.sh`：面向日常运维的 shell 入口。

## 5. 配置与部署入口

- `configs/deploy.env`：部署环境变量的主要入口。
- `configs/dev.yaml`：开发运行相关配置。
- `deploy/compose/docker-compose.app.yml`：应用总体 Compose 入口。
- `deploy/compose/app/infra.yml`：基础设施层。
- `deploy/compose/app/backend.yml`：业务后端层。
- `deploy/compose/app/proxy.yml`：代理、CDN 和运维控制层。

## 6. 文档维护原则

新的文档不再为每个模块维护一篇长文，而是优先维护：

- 能说明整体结构的页。
- 能说明主链路的页。
- 能直接服务部署和排障的页。
