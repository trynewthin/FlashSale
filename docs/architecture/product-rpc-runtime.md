# 商品 RPC 运行态（As-Is）

更新时间：2026-02-07

## 服务入口

- 启动文件：`apps/product/rpc/product.go`
- 协议文件：`apps/product/rpc/product.proto`
- 默认监听：`0.0.0.0:8084`

## RPC 方法

- 管理端：
  - `CreateProduct`
  - `UpdateProduct`
  - `DeleteProduct`
  - `GetProductAdmin`
  - `ListProductsAdmin`
- 公开端：
  - `GetProductPublic`
  - `ListProductsPublic`

## 模型与规则

- 单 SKU 模型，价格字段 `price_cent`（分，`int64`）。
- `status` 二态：`0=off_shelf`、`1=on_shelf`。
- 软删除：`deleted_at`。
- `sku_code` 服务端生成，格式 `SPU{snowflake_id}`，唯一。
- 分页默认：`page=1`、`page_size=20`、`max=100`。

## 可见性

- 用户公开接口仅返回：未删除 + 上架商品。
- 用户侧不返回库存数量，仅返回 `in_stock` 布尔值。

## 数据库迁移

- `deploy/migrations/product/000002__products.up.sql`
- `deploy/migrations/product/000002__products.down.sql`