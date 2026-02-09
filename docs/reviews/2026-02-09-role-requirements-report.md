# 角色视角需求与功能点综合报告

更新时间：2026-02-09  
适用范围：FlashSale 当前代码基线（`dev` 分支）

## 1. 报告目标

从以下角色视角梳理业务需求与功能点，形成后续产品设计、权限拆分与排期依据：

- 用户
- 管理员（用户管理员、商品管理员、订单管理员、审核管理员、秒杀活动管理员）

本报告同时给出“当前已实现能力”与“建议补齐项”。

## 2. 角色与权限域映射

当前管理端权限以 `domain` 维度控制（见 `apps/gateway/admin/internal/authz/authorizer.go`）：

- `user_management`
- `product_management`
- `order_management`
- `seckill_management`
- `admin_management`
- `operations`

建议角色映射如下：

| 角色 | 推荐 domain | 当前状态 |
|---|---|---|
| 用户管理员 | `user_management` | 已支持 |
| 商品管理员 | `product_management` | 已支持 |
| 订单管理员 | `order_management` | 已支持 |
| 审核管理员 | 建议独立 `order_review_management`（或阶段性复用 `order_management`） | 当前未独立 |
| 秒杀活动管理员 | `seckill_management` | 已支持 |
| 平台管理员（管理员管理） | `admin_management` | 已支持 |

## 3. 用户视角

### 3.1 核心业务目标

- 快速完成注册/登录。
- 可浏览商品与秒杀活动。
- 可发起下单、支付确认、取消、收货。
- 可查询订单状态进度。

### 3.2 功能点清单

1. 账号与资料
- 注册：`POST /api/v1/user/register`
- 登录：`POST /api/v1/user/login`
- 查看资料：`GET /api/v1/user/profile`
- 修改昵称：`PATCH /api/v1/user/nickname`
- 注销（软删）：`DELETE /api/v1/user`

2. 商品浏览
- 商品列表：`GET /api/v1/products`
- 商品详情：`GET /api/v1/products/{product_id}`

3. 订单交易
- 创建订单：`POST /api/v1/orders`
- 支付并确认收货信息：`POST /api/v1/orders/{order_id}/pay-confirm`
- 取消订单：`POST /api/v1/orders/{order_id}/cancel`
- 确认收货：`POST /api/v1/orders/{order_id}/confirm-receipt`
- 订单详情/列表：`GET /api/v1/orders/{order_id}`、`GET /api/v1/orders`

4. 秒杀参与
- 秒杀活动列表/详情：`GET /api/v1/seckill/activities`、`GET /api/v1/seckill/activities/{activity_id}`
- 秒杀下单：`POST /api/v1/seckill/activities/{activity_id}/purchase`
- 行为上报：`POST /api/v1/seckill/activities/{activity_id}/track`

### 3.3 用户侧关键需求（建议验收口径）

- 可用性：核心接口 P95 延迟可控，失败码语义稳定。
- 一致性：下单-扣库存-关单回补链路可追溯。
- 安全性：JWT 鉴权 + 用户仅能访问本人订单/资料。
- 可解释性：订单状态需对用户可读（待支付/待审核/待发货/已发货/已关闭）。

## 4. 用户管理员视角

### 4.1 核心目标

- 对用户账号进行查看、纠错与管控，处理用户侧基础运营问题。

### 4.2 功能点

- 查询用户：`GET /api/v1/admin/users/{user_id}`
- 修改昵称：`PATCH /api/v1/admin/users/{user_id}/nickname`
- 删除用户：`DELETE /api/v1/admin/users/{user_id}`

### 4.3 权限与审计要求

- 必须具备 `user_management`。
- 所有变更动作应可审计（操作者、对象、时间、结果）。
- 对“删除用户”建议补充二次确认与操作理由字段（后续增强）。

## 5. 商品管理员视角

### 5.1 核心目标

- 管理可售商品池，维护商品信息、库存与上下架状态。

### 5.2 功能点

- 新建商品：`POST /api/v1/admin/products`
- 更新商品：`PATCH /api/v1/admin/products/{product_id}`
- 删除商品：`DELETE /api/v1/admin/products/{product_id}`
- 商品详情/列表：`GET /api/v1/admin/products/{product_id}`、`GET /api/v1/admin/products`

### 5.3 关键需求

- 数据模型为单 SKU，价格单位为分，软删除。
- 用户侧仅可见“上架且未删除”商品。
- 库存需与订单/秒杀链路协同，支持幂等预扣与释放。

## 6. 订单管理员视角

### 6.1 核心目标

- 管理订单履约：审核、发货、状态追踪与异常处理。

### 6.2 功能点

- 审核订单：`POST /api/v1/admin/orders/{order_id}/review`
- 发货：`POST /api/v1/admin/orders/{order_id}/ship`
- 查询订单：`GET /api/v1/admin/orders/{order_id}`
- 订单列表：`GET /api/v1/admin/orders`

### 6.3 关键需求

- 可按状态筛选（订单状态、审核状态、用户、订单号）。
- 审核拒绝/超时触发退款与库存回补。
- 订单状态流转需强约束，禁止非法跃迁。

## 7. 审核管理员视角

### 7.1 角色定位

- 专注“订单审核”而非“发货执行”的岗位角色。
- 重点处理：待审核订单、审核通过/拒绝、审核原因留痕。

### 7.2 目标功能点（业务期望）

- 审核订单（通过/拒绝）；
- 查看审核队列、审核历史；
- 审核 SLA 看板（待审超时、拒绝率、处理时长）。

### 7.3 当前实现与缺口

- 当前审核接口存在，但权限仍使用 `order_management` 与发货共域。
- 尚未独立 `审核管理员` 专属权限域。

### 7.4 建议落地方案

1. 阶段一（低改造）
- 角色层面约束：审核管理员只分配流程权限，不分配发货岗位操作。

2. 阶段二（推荐）
- 新增 domain：`order_review_management`
- 路由拆分：
  - 审核接口绑定 `order_review_management`
  - 发货接口绑定 `order_fulfillment_management` 或保留 `order_management`

3. 阶段三
- 审核工作台：待审队列、超时预警、拒绝原因模板化。

## 8. 秒杀活动管理员视角

### 8.1 核心目标

- 以活动为单位配置秒杀，控制活动商品、库存、价格、限购、发布/下线。
- 可追踪活动流量与活动订单。

### 8.2 功能点

- 活动管理：创建、更新、删除、详情、列表
- 活动商品管理：新增/更新/移除
- 活动状态：发布、下线
- 经营分析：流量看板、活动订单追溯

对应路由：

- `POST /api/v1/admin/seckill/activities`
- `PATCH /api/v1/admin/seckill/activities/{activity_id}`
- `DELETE /api/v1/admin/seckill/activities/{activity_id}`
- `GET /api/v1/admin/seckill/activities/{activity_id}`
- `GET /api/v1/admin/seckill/activities`
- `POST /api/v1/admin/seckill/activities/{activity_id}/items`
- `PUT /api/v1/admin/seckill/activities/{activity_id}/items/{item_id}`
- `DELETE /api/v1/admin/seckill/activities/{activity_id}/items/{item_id}`
- `POST /api/v1/admin/seckill/activities/{activity_id}/publish`
- `POST /api/v1/admin/seckill/activities/{activity_id}/offline`
- `GET /api/v1/admin/seckill/activities/{activity_id}/traffic`
- `GET /api/v1/admin/seckill/activities/{activity_id}/orders`

### 8.3 关键需求

- 活动发布前预占库存，失败可回滚。
- 抢购链路需防超卖、幂等、防重入、失败补偿。
- 流量指标与订单状态可追溯到活动和活动商品维度。

## 9. 管理员管理（平台管理员）视角

虽不在本次点名的 5 类运营角色中，但当前系统已落地该能力，建议纳入权限治理基座：

- 登录、刷新、退出、个人资料、改密；
- 管理员/角色 CRUD；
- 角色域绑定；
- 审计日志查询；
- `all/self` 数据范围控制。

该能力由 `admin_management` 域承载，是后续精细化岗位拆分的基础。

## 10. 当前能力对齐结论

| 角色 | 需求覆盖度 | 结论 |
|---|---|---|
| 用户 | 高 | 已具备完整基础交易链路（含秒杀参与） |
| 用户管理员 | 高 | 核心管理能力已落地 |
| 商品管理员 | 高 | 商品 CRUD 与上下架能力已落地 |
| 订单管理员 | 高 | 审核/发货/查询能力已落地 |
| 审核管理员 | 中 | 有审核功能，但无独立权限域与工作台 |
| 秒杀活动管理员 | 高 | 活动、商品、发布、流量、订单追溯已落地 |

## 11. 优先级建议（下一步）

### P0（建议立即规划）

- 审核管理员权限独立化（域拆分 + 路由拆分）。
- 审核动作强审计（原因模板、批注、责任人统计）。

### P1（短期）

- 角色维度运营报表：
  - 用户管理员：账号处理量、封禁/删除趋势
  - 商品管理员：上新/下架效率、库存异常告警
  - 订单管理员：发货及时率、超时率
  - 审核管理员：审核时长、拒绝率、超时率
  - 秒杀管理员：活动转化率、超卖/补偿异常率

### P2（中期）

- 岗位工作台化（审核台、发货台、活动台）。
- 更细粒度的数据范围（从 `all/self` 扩展到“部门/业务线/活动范围”）。

---

如需下一步，我可以基于这份报告直接输出“角色-权限-接口-数据表”的实施矩阵（可用于 PRD 与测试用例联动）。
