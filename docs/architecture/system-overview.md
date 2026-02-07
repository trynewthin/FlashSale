# FlashSale 系统总览（As-Is）

更新时间：2026-02-07

## 1. 当前代码范围

当前仓库不是纯底座，已包含以下可运行模块：

- `apps/user/rpc`：用户模块 RPC 服务
- `apps/gateway/user`：用户侧 HTTP 网关
- `apps/gateway/admin`：管理员侧 HTTP 网关
- `pkg/base/*`：公共基础能力包
- `cmd/smoke/*`：MySQL/Redis/Kafka 连通性自检

尚未落地独立服务代码：`product`、`order`、`seckill`。

## 2. 运行拓扑

1. 客户端调用网关 HTTP 接口（user/admin）。
2. 网关完成鉴权、限流（用户侧）与权限校验（管理员侧）。
3. 网关通过 zrpc/gRPC 调用 `apps/user/rpc`。
4. 用户 RPC 访问 MySQL（`flash_user`）。
5. Redis/Kafka 目前主要用于基础能力和 smoke 验证。

## 3. 默认端口

- 用户 RPC：`0.0.0.0:8081`（`apps/user/rpc/etc/user.yaml`）
- 用户网关：`0.0.0.0:8082`（`apps/gateway/user/etc/user-gateway.yaml`）
- 管理网关：`0.0.0.0:8083`（`apps/gateway/admin/etc/admin-gateway.yaml`）

## 4. 目录映射

- 应用层：`apps/`
- 基础能力：`pkg/base/`
- 运行脚本：`scripts/dev/`
- 部署与迁移：`deploy/`
- 本地配置：`configs/local/`

## 5. 文档口径

本目录文档描述的是“当前已实现状态（as-is）”。未来规划请单独写为“target/proposal”，避免与现状混写。
