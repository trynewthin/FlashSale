# 常见问题与排障

## 1. 前端登录接口 404

现象：

- 浏览器请求 `http://127.0.0.1:5174/api/v1/admin/auth/login` 返回 404。

排查：

1. 确认 `admin-gateway` 已启动（`http://127.0.0.1:8083/healthz`）。
2. 确认前端 Vite 代理配置生效（`/api` 代理到 `8083` 或 `8082`）。
3. 重启前端开发服务。

## 2. Docker 内服务互通失败

现象：

- RPC 连接报错 `connection refused 127.0.0.1:*`。

排查：

1. 容器间连接不能依赖宿主机 `127.0.0.1`。
2. 检查是否已使用环境变量覆盖 RPC Target。
3. 在 compose 网络内应使用服务名或 host gateway。

## 3. 秒杀压测网络错误高

现象：

- `network_errors` 明显偏高。

排查：

1. 先降开环 `-rate`，确认系统在稳定区间。
2. 调整 HTTP 连接池参数：`max-idle-conns`、`max-idle-conns-per-host`、`max-conns-per-host`。
3. 检查后端是否触发限流或超时（查看 `log/services`）。

## 4. 管理端无权限（403）

现象：

- 已登录但接口返回 403。

排查：

1. 检查管理员 `domains` 是否包含目标域（如 `product_management`）。
2. 检查 `data_scope` 是否满足要求（部分管理写接口要求 `all`）。
3. 权限变更后需要重新登录或刷新 token 生效。

## 5. 迁移失败

现象：

- `migrate-up` 报连接或版本错误。

排查：

1. `go run ./cmd/fs env up` 确保 MySQL 已就绪。
2. 检查 `configs/local/dev.env` 的 MySQL 用户与密码。
3. 如脏版本，先评估是否需要 `migrate-down --all` 后重建。

## 6. 快速健康检查命令

```powershell
# 基础连通性
go run ./cmd/fs env smoke

# 网关健康
curl http://127.0.0.1:8082/healthz
curl http://127.0.0.1:8083/healthz

# 任务列表
go run ./cmd/fs ops jobs --limit 20 --key-env FLASHSALE_OPS_ACCESS_KEY
```
