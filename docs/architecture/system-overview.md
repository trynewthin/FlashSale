# 系统总览（As-Is）

更新时间：2026-02-09

## 已落地模块

- `apps/user/rpc`：用户 RPC
- `apps/product/rpc`：商品 RPC（单 SKU）
- `apps/order/rpc`：订单 RPC（完整虚拟交易流程）
- `apps/seckill/rpc`：秒杀 RPC（活动、抢购、活动订单追溯、流量聚合）
- `apps/admin/rpc`：管理员 RPC（认证、RBAC、管理员/角色管理、审计）
- `apps/gateway/user`：用户网关（用户 + 商品 + 订单 + 秒杀公开/用户接口）
- `apps/gateway/admin`：管理员网关（用户/商品/订单/秒杀/管理员管理）
- `pkg/base/*`：公共基础能力

## 运行拓扑

1. 用户/管理员请求进入网关 HTTP。
2. 网关完成鉴权与权限控制（管理端需要领域角色）。
3. 网关通过 zrpc/gRPC 调用下游 `user rpc`、`product rpc`、`order rpc`、`seckill rpc`、`admin rpc`。
4. `order rpc` 调用 `product rpc` 完成库存预扣与回补；`seckill rpc` 与 `order rpc/product rpc` 协同完成活动库存与下单链路。
5. 各服务分库访问：`flash_user`、`flash_product`、`flash_order`、`flash_seckill`、`flash_admin`。

## 权限模型现状

- 用户端：`user token` + 目标用户校验。
- 管理端：`admin token` + `domains` 领域校验。
- 管理员模块：在领域校验基础上引入 `data_scope`（`all/self`）与会话续期能力。
