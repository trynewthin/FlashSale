# 网关接口与后端适配对齐审查报告

更新时间：2026-02-09  
审查范围：`apps/gateway/user`、`apps/gateway/admin` 与 `apps/*/rpc`

## 1. 审查目标

基于当前实现，验证以下两点：

1. 网关用户/管理员接口是否完整覆盖角色需求。
2. 网关接口调用的 RPC 方法、鉴权语义、参数语义是否与后端实现对齐。

## 2. 审查方法

1. 路由对账：`handler.go` 路由注册与对应 handler。
2. 调用对账：`RPCCli.Method` 与各 `*.proto` 中 `rpc Method` 对齐。
3. 鉴权对账：网关中间件要求与 RPC 服务端 `authz` 二次校验一致性。
4. 运行校验：`go test`（gateway + rpc server）验证编译与基础行为。

自动比对结果：

- 网关调用方法数：57
- proto 暴露方法数：62
- “网关调用但后端不存在”数量：0

结论：无方法级断链。

## 3. 角色视角对齐结果

### 3.1 用户

网关接口（`user-gateway`）：

- 账号：注册/登录/资料/改昵称/注销
- 商品：公开列表/详情
- 订单：创建、支付确认、取消、收货、详情、列表
- 秒杀：活动列表/详情、抢购、埋点

后端对齐：

- 对接 `user rpc`、`product rpc`、`order rpc`、`seckill rpc`，方法均存在。
- 用户态接口在 RPC 侧均做 `user_id == token.subject` 校验。

结论：对齐。

### 3.2 用户管理员（`user_management`）

网关接口（`admin-gateway`）：

- 用户详情、改昵称、删除用户

后端对齐：

- 调用 `user rpc` 对应方法存在。
- `user rpc` 服务端支持 admin token 且要求 `user_management` 域。

结论：对齐。

### 3.3 商品管理员（`product_management`）

网关接口：

- 商品 CRUD + 列表

后端对齐：

- 对接 `product rpc` 管理侧方法完整。
- `product rpc` 服务端要求 admin token 且 `product_management` 域。

结论：对齐。

### 3.4 订单管理员（`order_management`）

网关接口：

- 审核、发货、详情、列表

后端对齐：

- 对接 `order rpc` 管理侧方法完整。
- `order rpc` 服务端会从 token 覆盖 `admin_id`，防止前端伪造。
- 订单超时任务、库存回补链路可工作。

结论：对齐。

### 3.5 审核管理员（业务角色）

当前接口现状：

- 审核接口已存在（`ReviewOrderAdmin`）。
- 已支持独立审核域 `order_review_management`。

后端对齐结论：

- 功能可用，且已支持“审核岗位独立权限域”。

建议：

- 若要进一步细化履约岗位，可新增 `order_fulfillment_management` 并把发货从 `order_management` 继续拆分。

### 3.6 秒杀活动管理员（`seckill_management`）

网关接口：

- 活动 CRUD
- 活动商品新增/更新/删除
- 发布/下线
- 流量看板
- 活动订单追溯

后端对齐：

- 对接 `seckill rpc` 管理侧方法完整。
- `seckill rpc` 服务端要求 admin token 且 `seckill_management` 域。
- 发布/下线已接入活动库存预占与释放。

结论：对齐。

### 3.7 管理员管理（`admin_management`）

网关接口：

- 登录/刷新/退出
- 个人资料/改密
- 管理员与角色管理
- 审计日志查询

后端对齐：

- 对接 `admin rpc` 方法完整。
- 写操作：`admin_management + data_scope=all`
- 读操作：`admin_management`（不强制 `all`）
- refresh 轮换与 logout token 归属校验已实现。

结论：对齐。

## 4. 关键一致性检查

### 4.1 鉴权链路一致性

- 网关：`AuthRequired + RequireDomain`
- RPC：再次校验 token 与 domain（关键服务均有 server authz）
- 敏感字段（如 `admin_id`）在服务端覆盖，避免伪造

结论：一致。

### 4.2 跨服务调用权限一致性

- `order rpc`、`seckill rpc` 调 `product rpc` 库存接口时，使用内部签发的 admin token（`product_management`）。

结论：一致。

### 4.3 配置与客户端初始化一致性

- user/admin gateway 均已初始化对应 RPC client。
- yaml 中目标地址已覆盖 user/product/order/seckill/admin。

结论：一致。

## 5. 发现的问题与建议

### 5.1 已修复：审核岗位独立权限域（2026-02-09）

- 修复点：
  - 新增域：`order_review_management`。
  - 网关路由：
    - 审核接口支持 `order_management` 或 `order_review_management`。
    - 发货接口保持 `order_management`。
  - `order rpc` 服务端：
    - 审核/查询接口支持 `order_management` 或 `order_review_management`。
    - 发货接口仅 `order_management`。
- 效果：可实现“只审不发”角色配置，同时保持历史 `order_management` 角色兼容。

### 5.2 已修复：匿名埋点 `user_id` 伪造风险（2026-02-09）

- 修复点：
  - 网关埋点接口不接收请求体 `user_id`，仅按服务端规则填充。
  - 若请求携带合法用户 token，则由服务端解析并写入 `user_id`。
  - 若无 token 或 token 非法，则按匿名事件处理（`user_id=0`，依赖 `client_id`）。
  - `seckill rpc` 在 `TrackEvent` 中对 `user_id>0` 进行 token 二次校验，防止伪造。
- 效果：前端无法伪造任意 `user_id` 埋点数据。

## 6. 最终结论

1. 网关用户/管理员接口与后端 RPC 实现整体对齐，无调用断链。  
2. 审核管理员已支持独立权限域，角色隔离可落地。  
3. 当前系统已具备继续做前端联调与更细粒度权限拆分（如履约域）的基础。
