# 数据库迁移说明

更新时间：2026-02-07

## 1. 数据库初始化

初始化 SQL：`deploy/mysql/init/001_init.sql`

默认创建 5 个库：

- `flash_user`
- `flash_admin`
- `flash_product`
- `flash_order`
- `flash_seckill`

## 2. 迁移目录

迁移按库拆分：

- `deploy/migrations/user`
- `deploy/migrations/admin`
- `deploy/migrations/product`
- `deploy/migrations/order`
- `deploy/migrations/seckill`

## 3. 执行脚本

向前迁移：

```powershell
./scripts/dev/migrate-up.ps1
```

回滚：

```powershell
./scripts/dev/migrate-down.ps1 -Steps 1
```

全部回滚：

```powershell
./scripts/dev/migrate-down.ps1 -All
```

## 4. 命名规范

采用 `golang-migrate` 兼容命名：

- `000001__infra_tables.up.sql`
- `000001__infra_tables.down.sql`

## 5. 当前业务相关迁移

用户模块额外迁移：

- `deploy/migrations/user/000002__user_accounts.up.sql`
- `deploy/migrations/user/000002__user_accounts.down.sql`
