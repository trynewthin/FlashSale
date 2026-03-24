# 部署与日常维护

## 1. Compose 分层

应用总入口是：

- `deploy/compose/docker-compose.app.yml`

它进一步拆成四层：

- `app/infra.yml`：MySQL、Redis、Kafka、etcd
- `app/backend.yml`：迁移、RPC 服务、网关
- `app/proxy.yml`：Nginx、CDN、media-store、ops-control
- `app/observability.yml`：Prometheus、Grafana、Jaeger

## 2. 核心服务说明

### 业务面

- `user-rpc`
- `product-rpc`
- `order-rpc`
- `seckill-rpc`
- `admin-rpc`
- `user-gateway`
- `admin-gateway`

### 运维与资源面

- `ops-control`
- `cdn`
- `media-store`
- `nginx`

## 3. 维护入口

### CLI

- `go run ./ops/cmd env ...`
- `go run ./ops/cmd runtime ...`
- `go run ./ops/cmd ops ...`

### 运维控制台

`ops-control` 负责承接项目专用的状态查看、任务运行、性能相关操作和容器信息聚合。

## 4. 素材与文件维护

- 上传、删除、列目录走 `media-store`
- 对外访问走 `cdn`
- 两者通过共享目录协作

这能避免把对外静态访问和后台写入动作混到同一服务里。

## 5. 日常维护建议

- 任何端口冲突先回到 `configs/deploy.env` 处理。
- 先确认依赖容器健康，再判断业务服务本身是否异常。
- 运维问题优先区分是“容器未启动”“依赖未满足”还是“服务已启动但健康检查失败”。
