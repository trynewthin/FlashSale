# 从拉取到跑通（后端 + 前端 + 测试）

## 1. 前置环境

- Go：建议 `1.22+`
- Docker Desktop（含 compose）
- Bun（前端）
- Git

## 2. 拉取与进入项目

```powershell
git clone <your-repo-url> FlashSale
cd FlashSale
```

## 3. 准备本地环境变量

```powershell
# 首次可复制模板
Copy-Item configs/local/dev.env.example configs/local/dev.env
```

> 非生产环境可直接使用默认值；如端口冲突，修改 `configs/local/dev.env` 的 `FLASH_*` 端口。

## 4. 启动 Docker 基础环境

```powershell
go run ./cmd/fs env up
```

可选（带可观测组件）：

```powershell
go run ./cmd/fs env up --observability
```

## 5. 执行迁移与连通性检查

```powershell
go run ./cmd/fs env migrate-up
go run ./cmd/fs env smoke
```

## 6. 启动后端服务

```powershell
go run ./cmd/fs runtime start-backend
```

服务端口默认：

- user-rpc: `8081`
- user-gateway: `8082`
- admin-gateway: `8083`
- product-rpc: `8084`
- order-rpc: `8085`
- seckill-rpc: `8086`
- admin-rpc: `8087`

## 7. 初始化演示数据（覆写）

```powershell
go run ./cmd/fs data seed-overwrite --force
```

该命令会：

- 清理业务数据
- 重建超级管理员、种子用户、商品、活动、订单
- 输出结果到 `log/data/seed-overwrite.result.json`

## 8. 启动两个前端

```powershell
# 安装依赖（首次）
go run ./cmd/fs runtime start-frontend --install-deps

# 非首次可直接启动
go run ./cmd/fs runtime start-frontend
```

访问地址：

- 用户端：`http://127.0.0.1:5173`
- 管理端：`http://127.0.0.1:5174`

## 9. 运行测试

### 9.1 后端

```powershell
go test ./... -short
go vet ./...
```

### 9.2 前端

```powershell
cd frontend/user
bun run lint
bun run test:run
bun run build

cd ../admin
bun run lint
bun run test:run
bun run build
```

## 10. 秒杀压测（示例）

```powershell
# 闭环购买压测
go run ./cmd/fs perf purchase-stress -concurrency 200 -requests 4000 -timeout 7s -output json

# 幂等压测
go run ./cmd/fs perf idempotency -concurrency 50 -requests 200 -expect-max-success 1 -output json

# 开环购买压测
go run ./cmd/fs perf purchase-open -rate 120 -open-duration 30s -concurrency 300 -output json
```

## 11. 停止服务

```powershell
go run ./cmd/fs runtime stop-frontend
go run ./cmd/fs runtime stop-backend
go run ./cmd/fs env down
```

如需删除容器卷：

```powershell
go run ./cmd/fs env down --remove-volumes
```
