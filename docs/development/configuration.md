# 配置说明

更新时间：2026-02-09

## 1. 配置来源

1. 基础配置文件：`configs/local/dev.yaml`
2. 本地环境变量文件：`configs/local/dev.env`（由 `scripts/dev/common.ps1` 自动加载）
3. 服务配置文件：
  - `apps/admin/rpc/etc/admin.yaml`
  - `apps/seckill/rpc/etc/seckill.yaml`
  - `apps/order/rpc/etc/order.yaml`
  - `apps/product/rpc/etc/product.yaml`
  - `apps/user/rpc/etc/user.yaml`
  - `apps/gateway/user/etc/user-gateway.yaml`
  - `apps/gateway/admin/etc/admin-gateway.yaml`

## 2. 基础配置结构

`configs/local/dev.yaml` 包含：

- `log`
- `mysql`
- `redis`
- `kafka`
- `jwt.user` / `jwt.admin`
- `otel`
- `prometheus`

加载入口：`pkg/base/config/config.go`。

## 3. 脚本层环境变量

开发脚本通过 `scripts/dev/common.ps1` 加载 `dev.env`，并做以下兜底：

- `FLASHSALE_MYSQL_HOST` 默认 `localhost`
- 若未设置 `FLASHSALE_MYSQL_PORT` 且已设置 `FLASH_MYSQL_PORT`，自动回填
- 若未设置 `FLASHSALE_REDIS_ADDR` 且已设置 `FLASH_REDIS_PORT`，自动回填
- 若未设置 `FLASHSALE_KAFKA_BROKERS` 且已设置 `FLASH_KAFKA_PORT`，自动回填

## 4. 服务监听地址覆盖

- `FLASHSALE_USER_RPC_LISTEN_ON` -> user RPC 监听地址
- `FLASHSALE_PRODUCT_RPC_LISTEN_ON` -> product RPC 监听地址
- `FLASHSALE_ORDER_RPC_LISTEN_ON` -> order RPC 监听地址
- `FLASHSALE_SECKILL_RPC_LISTEN_ON` -> seckill RPC 监听地址
- `FLASHSALE_ADMIN_RPC_LISTEN_ON` -> admin RPC 监听地址
- `FLASHSALE_USER_GATEWAY_LISTEN_ON` -> user gateway 监听地址
- `FLASHSALE_ADMIN_GATEWAY_LISTEN_ON` -> admin gateway 监听地址

对应代码：

- `apps/admin/rpc/internal/config/config.go`
- `apps/seckill/rpc/internal/config/config.go`
- `apps/order/rpc/internal/config/config.go`
- `apps/product/rpc/internal/config/config.go`
- `apps/user/rpc/internal/config/config.go`
- `apps/gateway/user/internal/config/config.go`
- `apps/gateway/admin/internal/config/config.go`

## 5. 管理员 bootstrap 环境变量

管理员 RPC 启动时支持首个超级管理员幂等初始化：

- `ADMIN_BOOTSTRAP_ENABLED`
- `ADMIN_BOOTSTRAP_USERNAME`
- `ADMIN_BOOTSTRAP_PASSWORD`
- `ADMIN_BOOTSTRAP_DISPLAY_NAME`

对应代码：`apps/admin/rpc/internal/svc/servicecontext.go`。
