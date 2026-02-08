# 本地开发快速开始

更新时间：2026-02-08

## 1. 前置条件

- Docker / Docker Compose 可用
- Go 版本与 `go.mod` 一致（当前 `go 1.25.0`）

## 2. 初始化本地环境变量

首次使用：

```powershell
Copy-Item configs/local/dev.env.example configs/local/dev.env
```

然后按本机实际情况修改 `configs/local/dev.env` 中端口和密码。

## 3. 启动基础依赖

```powershell
./scripts/dev/up.ps1
```

启用可观测性组件：

```powershell
./scripts/dev/up.ps1 -Observability
```

## 4. 执行数据库迁移

```powershell
./scripts/dev/migrate-up.ps1
```

该脚本会对 `flash_user`、`flash_admin`、`flash_product`、`flash_order`、`flash_seckill` 依次执行迁移。

## 5. 运行连通性检查

```powershell
./scripts/dev/smoke.ps1
```

## 6. 启动应用服务

启动用户 RPC：

```powershell
go run ./apps/user/rpc -f apps/user/rpc/etc/user.yaml
```

启动商品 RPC：

```powershell
go run ./apps/product/rpc -f apps/product/rpc/etc/product.yaml
```

启动订单 RPC：

```powershell
go run ./apps/order/rpc -f apps/order/rpc/etc/order.yaml
```

启动用户网关：

```powershell
go run ./apps/gateway/user -f apps/gateway/user/etc/user-gateway.yaml
```

启动管理员网关：

```powershell
go run ./apps/gateway/admin -f apps/gateway/admin/etc/admin-gateway.yaml
```

## 7. 停止环境

```powershell
./scripts/dev/down.ps1
```

删除数据卷：

```powershell
./scripts/dev/down.ps1 -RemoveVolumes
```
