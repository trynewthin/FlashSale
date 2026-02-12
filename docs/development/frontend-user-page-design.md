# 用户端页面设计文档（基于 Hooks）

更新时间：2026-02-12  
适用模块：`frontend-user`  
事实源：`docs/development/frontend-user-api.md`、`docs/development/frontend-user-hooks-design.md`
组件基线：`docs/development/frontend-shadcn-components.md`

## 1. 目标

1. 把用户端 Hook 能力转成页面级设计，保证页面职责、页面关系、组件结构、Hook 绑定一致。
2. 输出可直接交给前端实现的页面与组件清单，避免“接口有了但页面行为不明确”。

## 2. 页面关系图（用户端）

```text
[登录/注册页] -> [商品列表页] -> [商品详情页] -> [创建订单动作]
       |               |                 |
       |               -> [秒杀活动列表页] -> [秒杀活动详情页] -> [秒杀下单动作]
       |                                                     |
       -> [个人中心页] -> [订单列表页] -> [订单详情页] <- [支付确认/取消/确认收货]
```

## 3. 页面设计明细

## 3.1 登录/注册页

1. 页面作用
- 提供账号注册、登录入口，建立用户会话。

2. 与其他页面关系
- 登录成功跳转 `商品列表页` 或登录前目标页。
- 注册成功可直接建立会话并进入 `商品列表页`。

3. 业务组件
- `登录卡片 LoginCard`
- `注册卡片 RegisterCard`

4. 组件构成
- `LoginCard`：手机号输入框、密码输入框、登录按钮、去注册切换按钮、错误提示区。
- `RegisterCard`：手机号输入框、昵称输入框、密码输入框、确认密码输入框、注册按钮、去登录切换按钮、错误提示区。

5. 组件与 Hooks 关联
- `LoginCard` -> `useLoginMutation`
- `RegisterCard` -> `useRegisterMutation`
- 页面初始化会话校验 -> `useUserProfileQuery`（可选，已有 token 时拉取）

6. shadcn 组件构成
- `LoginCard`：`Card` + `CardHeader` + `CardContent` + `Form` + `FormField` + `Input` + `Button` + `Alert`
- `RegisterCard`：`Card` + `CardHeader` + `CardContent` + `Form` + `FormField` + `Input` + `Button` + `Alert`

## 3.2 商品列表页

1. 页面作用
- 展示可浏览商品，支持分页与关键词查询。

2. 与其他页面关系
- 点击商品卡片进入 `商品详情页`。
- 顶部入口可跳转 `秒杀活动列表页`、`订单列表页`。

3. 业务组件
- `商品搜索栏 ProductSearchBar`
- `商品列表 ProductGrid`
- `商品卡片 ProductCard`
- `分页器 PaginationBar`

4. 组件构成
- `ProductSearchBar`：关键词输入框、搜索按钮、重置按钮。
- `ProductCard`：商品主图、商品名、价格、库存态（`in_stock`）、查看详情按钮。
- `PaginationBar`：页码、每页条数选择、上一页/下一页按钮。

5. 组件与 Hooks 关联
- 页面数据 -> `useProductsQuery`
- 卡片跳转参数 -> `product_id`

6. shadcn 组件构成
- `ProductSearchBar`：`Input` + `Button`
- `ProductGrid`：布局容器 + `Card` 列表
- `ProductCard`：`Card` + `Badge` + `Button`
- `PaginationBar`：`Button` + 分页信息文本（或接入 `Pagination` 组件封装）

## 3.3 商品详情页

1. 页面作用
- 展示单商品详情并发起普通下单。

2. 与其他页面关系
- 下单成功后可跳转 `订单详情页`。
- 返回入口为 `商品列表页`。

3. 业务组件
- `商品信息面板 ProductInfoPanel`
- `下单卡片 CreateOrderCard`

4. 组件构成
- `ProductInfoPanel`：主图、商品名、描述、价格、在库状态。
- `CreateOrderCard`：下单来源选择（普通/秒杀来源只读策略）、下单按钮、错误提示。

5. 组件与 Hooks 关联
- 商品详情 -> `useProductDetailQuery`
- 创建订单 -> `useCreateOrderMutation`

6. shadcn 组件构成
- `ProductInfoPanel`：`Card` + `Badge` + `Separator`
- `CreateOrderCard`：`Card` + `RadioGroup` + `Button` + `Alert`

## 3.4 订单列表页

1. 页面作用
- 展示用户全部订单并支持按状态筛选。

2. 与其他页面关系
- 点击订单进入 `订单详情页`。
- 可回跳 `商品列表页` 或 `秒杀活动列表页`继续购物。

3. 业务组件
- `订单筛选条 OrderFilterBar`
- `订单列表 OrderList`
- `订单项卡片 OrderItemCard`
- `分页器 PaginationBar`

4. 组件构成
- `OrderFilterBar`：状态下拉、查询按钮、清空按钮。
- `OrderItemCard`：订单号、金额、订单状态、支付状态、创建时间、查看详情按钮。

5. 组件与 Hooks 关联
- 列表数据 -> `useOrderListQuery`

6. shadcn 组件构成
- `OrderFilterBar`：`Select` + `Button`
- `OrderList`：`Table` + `TableHeader` + `TableBody`
- `OrderItemCard`：`Card` + `Badge` + `Button`
- `PaginationBar`：`Button` + 分页信息文本（或接入 `Pagination` 组件封装）

## 3.5 订单详情页

1. 页面作用
- 展示订单全状态，并承载支付确认、取消、确认收货动作。

2. 与其他页面关系
- 操作成功后刷新本页并回刷 `订单列表页`。

3. 业务组件
- `订单状态时间线 OrderTimeline`
- `订单信息卡 OrderInfoCard`
- `操作面板 OrderActionPanel`

4. 组件构成
- `OrderTimeline`：下单、支付确认、审核、发货、收货、关闭等节点。
- `OrderActionPanel`：支付确认按钮、取消订单按钮、确认收货按钮（按状态显示）。

5. 组件与 Hooks 关联
- 详情查询 -> `useOrderDetailQuery`
- 支付确认 -> `useConfirmPaymentAndInfoMutation`
- 取消订单 -> `useCancelOrderMutation`
- 确认收货 -> `useConfirmReceiptMutation`

6. shadcn 组件构成
- `OrderTimeline`：`Card` + `Separator` + `Badge`（时间线节点用自定义样式拼装）
- `OrderInfoCard`：`Card` + `Badge`
- `OrderActionPanel`：`Card` + `Button` + `AlertDialog`

## 3.6 秒杀活动列表页

1. 页面作用
- 展示可见秒杀活动，支持分页浏览。

2. 与其他页面关系
- 点击活动进入 `秒杀活动详情页`。
- 与 `商品列表页`互为导流页。

3. 业务组件
- `活动列表 ActivityList`
- `活动卡片 ActivityCard`
- `分页器 PaginationBar`

4. 组件构成
- `ActivityCard`：活动标题、时间窗口、状态标签、描述摘要、进入活动按钮。

5. 组件与 Hooks 关联
- 列表数据 -> `useSeckillActivitiesQuery`
- 曝光/点击埋点 -> `useSeckillTrackMutation`

6. shadcn 组件构成
- `ActivityList`：布局容器 + `Card` 列表
- `ActivityCard`：`Card` + `Badge` + `Button`
- `PaginationBar`：`Button` + 分页信息文本（或接入 `Pagination` 组件封装）

## 3.7 秒杀活动详情页

1. 页面作用
- 展示活动商品并执行秒杀购买流程。

2. 与其他页面关系
- 秒杀成功跳转 `订单详情页` 或 `订单列表页`。
- 返回入口为 `秒杀活动列表页`。

3. 业务组件
- `活动头部 ActivityHeader`
- `活动商品列表 ActivityItemList`
- `秒杀商品卡 SeckillItemCard`
- `抢购弹层 PurchaseDialog`

4. 组件构成
- `SeckillItemCard`：商品快照图、名称、原价、秒杀价、库存态、限购信息、数量选择器、立即抢购按钮。
- `PurchaseDialog`：数量确认、幂等提交中状态、成功/失败提示。

5. 组件与 Hooks 关联
- 活动详情 -> `useSeckillActivityDetailQuery`
- 抢购提交 -> `useSeckillPurchaseMutation`
- 点击/尝试/结果埋点 -> `useSeckillTrackMutation`

6. shadcn 组件构成
- `ActivityHeader`：`Card` + `Badge` + `Separator`
- `ActivityItemList`：布局容器 + `Card` 列表
- `SeckillItemCard`：`Card` + `Badge` + `Input`（数量）+ `Button`
- `PurchaseDialog`：`Dialog` + `DialogHeader` + `DialogContent` + `Button` + `Alert`

## 4. 页面级实现约束

1. 所有写操作组件都要有“提交中”禁用态，避免重复触发。
2. 秒杀购买与埋点由 Hook 统一注入 `idempotency_key`，页面层不自造第二套逻辑。
3. 私有数据页面在 QueryKey 中必须包含 `userId`，避免账号切换串缓存。
4. `AUTH_UNAUTHORIZED` 统一走登录失效处理；秒杀冲突码展示“处理中，请稍后到订单页查看”。
