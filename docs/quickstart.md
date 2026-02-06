# 开发说明

## 目录结构

- `deploy/compose`：Docker Compose 编排
- `deploy/mysql/init`：数据库初始化 SQL
- `deploy/migrations/*`：按库拆分的迁移脚本
- `pkg/base/*`：公共基础能力
- `cmd/smoke/*`：连通性检查程序
- `scripts/dev/*`：开发脚本

## 迁移命名规范

采用 `golang-migrate` 兼容命名：

- `000001__infra_tables.up.sql`
- `000001__infra_tables.down.sql`

## 公共能力接入点

后续业务服务可直接复用：

- `config.Load`
- `logx.New`
- `mysqlx.Open` / `redisx.New`
- `kafkax.NewProducer` / `kafkax.NewConsumer`
- `authx.Init` / `authx.Issue` / `authx.Parse`
- `idempotency.Guard`

## 运行时配置

所有开发脚本会自动加载：`configs/local/dev.env`

修改该文件后，重新执行以下命令即可生效：

- `scripts/dev/up.ps1`
- `scripts/dev/migrate-up.ps1`
- `scripts/dev/smoke.ps1`
