# 系统总览（As-Is）

更新时间：2026-02-08

## 已落地模块

- `apps/user/rpc`：用户 RPC
- `apps/product/rpc`：商品 RPC（单 SKU）
- `apps/order/rpc`：订单 RPC（完整虚拟交易流程）
- `apps/gateway/user`：用户网关（用户接口 + 商品公开浏览）
- `apps/gateway/admin`：管理员网关（用户管理 + 商品管理 + 订单管理）
- `pkg/base/*`：公共基础能力

## 运行拓扑

1. 用户/管理员请求进入网关 HTTP。
2. 网关完成鉴权与权限控制（管理端需要领域角色）。
3. 网关通过 zrpc/gRPC 调用下游 `user rpc`、`product rpc`、`order rpc`。
4. order rpc 调用 product rpc 完成库存预扣与回补。
5. user rpc 访问 `flash_user`，product rpc 访问 `flash_product`，order rpc 访问 `flash_order`。

## 未落地模块

- `seckill`
