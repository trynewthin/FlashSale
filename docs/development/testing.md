# 测试说明

更新时间：2026-02-07

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
