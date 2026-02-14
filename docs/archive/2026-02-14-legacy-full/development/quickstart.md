# 本地开发快速开始

更新时间：2026-02-09

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
go run ./cmd/fs env up
```

启用可观测性组件：

```powershell
go run ./cmd/fs env up --observability
```

## 4. 执行数据库迁移

```powershell
go run ./cmd/fs env migrate-up
```

该脚本会对 `flash_user`、`flash_admin`、`flash_product`、`flash_order`、`flash_seckill` 依次执行迁移。

## 5. 运行连通性检查

```powershell
go run ./cmd/fs env smoke
```

## 6. 启动应用服务

```powershell
go run ./cmd/fs runtime start-backend
```

## 7. 停止环境

```powershell
go run ./cmd/fs env down
```

删除数据卷：

```powershell
go run ./cmd/fs env down --remove-volumes
```
