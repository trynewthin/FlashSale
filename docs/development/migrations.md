# 数据库迁移说明

更新时间：2026-02-09

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
./scripts/dev/env/migrate-up.ps1
```

回滚：

```powershell
./scripts/dev/env/migrate-down.ps1 -Steps 1
```

全部回滚：

```powershell
./scripts/dev/env/migrate-down.ps1 -All
```

## 4. 命名规范

采用 `golang-migrate` 兼容命名：

- `000001__infra_tables.up.sql`
- `000001__infra_tables.down.sql`

## 5. 当前业务相关迁移

用户模块额外迁移：

- `deploy/migrations/user/000002__user_accounts.up.sql`
- `deploy/migrations/user/000002__user_accounts.down.sql`

商品模块额外迁移：

- `deploy/migrations/product/000002__products.up.sql`
- `deploy/migrations/product/000002__products.down.sql`

订单模块额外迁移：

- `deploy/migrations/order/000002__orders.up.sql`
- `deploy/migrations/order/000002__orders.down.sql`
- `deploy/migrations/order/000003__orders_seckill_fields.up.sql`
- `deploy/migrations/order/000003__orders_seckill_fields.down.sql`

秒杀模块额外迁移：

- `deploy/migrations/seckill/000002__seckill_core.up.sql`
- `deploy/migrations/seckill/000002__seckill_core.down.sql`

管理员模块额外迁移：

- `deploy/migrations/admin/000002__admin_core.up.sql`
- `deploy/migrations/admin/000002__admin_core.down.sql`

