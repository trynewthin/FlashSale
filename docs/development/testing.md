# 测试说明

更新时间：2026-02-09

## 1. 全量测试

```powershell
go test ./...
```

## 2. 用户仓储集成测试

文件：`apps/user/rpc/internal/repository/mysql_user_repository_integration_test.go`

该测试依赖真实 MySQL，执行前需设置：

- `FLASHSALE_TEST_MYSQL_DSN`

当 DSN 未设置或 `users` 表不存在时，测试会 `Skip`。

## 3. 现有测试分布

- 基础能力层：`pkg/base/*_test.go`
- 网关配置与鉴权：`apps/gateway/*/internal/*_test.go`
- 用户 RPC 逻辑与鉴权：`apps/user/rpc/internal/*_test.go`
- 商品 RPC 逻辑、鉴权与仓储：`apps/product/rpc/internal/*_test.go`
- 订单 RPC 逻辑、鉴权与仓储：`apps/order/rpc/internal/*_test.go`
- 秒杀 RPC 逻辑与链路：`apps/seckill/rpc/internal/logic/*_test.go`
- 管理员网关鉴权与处理：`apps/gateway/admin/internal/*_test.go`
- 管理员 RPC 配置：`apps/admin/rpc/internal/config/*_test.go`

## 4. 订单仓储集成测试

文件：`apps/order/rpc/internal/repository/mysql_order_repository_integration_test.go`

依赖真实 MySQL，执行前需设置：

- `FLASHSALE_TEST_MYSQL_DSN`

当 DSN 未设置或 `orders/order_events` 表不存在时，测试会 `Skip`。
