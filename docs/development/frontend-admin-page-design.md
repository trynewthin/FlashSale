# 管理端页面设计文档（基于 Hooks）

更新时间：2026-02-12  
适用模块：`frontend-ops`  
事实源：`docs/development/frontend-admin-api.md`、`docs/development/frontend-admin-hooks-design.md`
组件基线：`docs/development/frontend-shadcn-components.md`

## 1. 目标

1. 把管理端 Hook 能力落为页面架构，明确页面职责、权限关系、组件结构、Hook 绑定。
2. 保证“页面可见性”和“操作可执行性”与后端 `domain + data_scope` 约束一致。

## 2. 页面关系图（管理端）

```text
[管理登录页] -> [运营工作台]
      |            |-> [管理员列表页] -> [管理员详情抽屉]
      |            |-> [角色管理页] -> [角色详情抽屉]
      |            |-> [审计日志页]
      |            |-> [用户管理页]
      |            |-> [商品管理页]
      |            |-> [订单管理页]
      |            -> [秒杀活动列表页] -> [秒杀活动详情页]
```

## 3. 页面设计明细

## 3.1 管理登录页

1. 页面作用
- 完成管理员认证，初始化会话与权限信息。

2. 与其他页面关系
- 登录成功进入 `运营工作台`。
- 刷新失败或退出登录回到本页。

3. 业务组件
- `管理登录卡片 AdminLoginCard`
- `会话状态条 SessionStateBar`

4. 组件构成
- `AdminLoginCard`：用户名输入框、密码输入框、登录按钮、错误提示区。
- `SessionStateBar`：access/refresh 状态、过期提示、重新登录入口。

5. 组件与 Hooks 关联
- 登录提交 -> `useAdminSession`（login）
- 自动续期 -> `useAdminSession`（refresh）
- 401/403 处理 -> `useAdminApiGuard`

6. shadcn 组件构成
- `AdminLoginCard`：`Card` + `CardHeader` + `CardContent` + `Form` + `FormField` + `Input` + `Button` + `Alert`
- `SessionStateBar`：`Alert` + `Badge` + `Button`

## 3.2 运营工作台

1. 页面作用
- 管理端入口聚合页，承载菜单导航、权限态展示、联通性探针。

2. 与其他页面关系
- 根据菜单进入各业务页；退出后回 `管理登录页`。

3. 业务组件
- `侧边导航 SiderNav`
- `权限概览 PermissionSummaryCard`
- `环境探针 OpsHealthCard`

4. 组件构成
- `SiderNav`：模块菜单（用户/商品/订单/秒杀/管理员/角色/审计）。
- `PermissionSummaryCard`：domains 标签、`data_scope` 标签、只读告警。
- `OpsHealthCard`：探针状态、最近探测时间、重试按钮。

5. 组件与 Hooks 关联
- 权限判定 -> `useAdminPermission`
- 环境探针 -> `useOpsPingQuery`
- 退出登录 -> `useAdminSession`（logout）

6. shadcn 组件构成
- `SiderNav`：`Sidebar`（或 `NavigationMenu` 封装）+ `Button`
- `PermissionSummaryCard`：`Card` + `Badge`
- `OpsHealthCard`：`Card` + `Badge` + `Button`

## 3.3 管理员列表页

1. 页面作用
- 管理管理员账号生命周期（创建、编辑、禁用、删除、重置密码、绑定角色）。

2. 与其他页面关系
- 行点击打开 `管理员详情抽屉`。
- 跳转 `角色管理页` 完成角色准备后再绑定。

3. 业务组件
- `管理员筛选栏 AdminFilterBar`
- `管理员表格 AdminTable`
- `创建/编辑管理员弹窗 AdminFormModal`
- `绑定角色弹窗 BindRolesModal`
- `重置密码弹窗 ResetPasswordModal`

4. 组件构成
- `AdminFilterBar`：关键字、状态、查询按钮、重置按钮。
- `AdminTable`：账号、昵称、状态、`data_scope`、角色、操作列。
- `AdminFormModal`：用户名（创建时）、昵称、`data_scope`、状态、保存按钮。
- `BindRolesModal`：角色多选框、确认按钮。

5. 组件与 Hooks 关联
- 列表 -> `useAdminListQuery`
- 详情 -> `useAdminDetailQuery`
- 创建 -> `useCreateAdminMutation`
- 更新 -> `useUpdateAdminMutation`
- 状态变更 -> `useSetAdminStatusMutation`
- 重置密码 -> `useResetAdminPasswordMutation`
- 删除 -> `useDeleteAdminMutation`
- 绑定角色 -> `useBindAdminRolesMutation`

6. shadcn 组件构成
- `AdminFilterBar`：`Input` + `Select` + `Button`
- `AdminTable`：`Table` + `Badge` + `DropdownMenu`
- `AdminFormModal`：`Dialog` + `Form` + `Input` + `Select` + `Button`
- `BindRolesModal`：`Dialog` + `Checkbox` + `ScrollArea` + `Button`
- `ResetPasswordModal`：`Dialog` + `Form` + `Input` + `Button`

## 3.4 角色管理页

1. 页面作用
- 管理角色定义与领域权限集合（domains）。

2. 与其他页面关系
- 与 `管理员列表页`形成角色供给关系。
- 可进入 `审计日志页`查看角色变更记录。

3. 业务组件
- `角色筛选栏 RoleFilterBar`
- `角色表格 RoleTable`
- `角色表单弹窗 RoleFormModal`
- `领域配置弹窗 RoleDomainsModal`

4. 组件构成
- `RoleTable`：角色编码、角色名称、状态、领域数、操作列。
- `RoleDomainsModal`：domain 多选、保存按钮、变更提示文案。

5. 组件与 Hooks 关联
- 列表 -> `useRoleListQuery`
- 详情 -> `useRoleDetailQuery`
- 创建 -> `useCreateRoleMutation`
- 更新 -> `useUpdateRoleMutation`
- 删除 -> `useDeleteRoleMutation`
- 域配置 -> `useSetRoleDomainsMutation`

6. shadcn 组件构成
- `RoleFilterBar`：`Input` + `Select` + `Button`
- `RoleTable`：`Table` + `Badge` + `DropdownMenu`
- `RoleFormModal`：`Dialog` + `Form` + `Input` + `Select` + `Button`
- `RoleDomainsModal`：`Dialog` + `Checkbox` + `ScrollArea` + `Button`

## 3.5 审计日志页

1. 页面作用
- 查询管理员关键操作审计，满足追溯与复盘。

2. 与其他页面关系
- 可回跳 `管理员列表页` 或 `角色管理页`定位目标对象。

3. 业务组件
- `审计筛选栏 AuditFilterBar`
- `审计表格 AuditLogTable`
- `审计详情抽屉 AuditDetailDrawer`

4. 组件构成
- `AuditFilterBar`：操作者、动作、结果、时间范围、查询按钮。
- `AuditLogTable`：操作者、action、target、result、request_id、时间。
- `AuditDetailDrawer`：`detail_json` 展开、上下文信息。

5. 组件与 Hooks 关联
- 审计列表 -> `useAuditLogsQuery`

6. shadcn 组件构成
- `AuditFilterBar`：`Input` + `Select` + `Popover` + `Calendar` + `Button`
- `AuditLogTable`：`Table` + `Badge`
- `AuditDetailDrawer`：`Sheet` + `ScrollArea` + `Separator`

## 3.6 用户管理页

1. 页面作用
- 对用户进行查询、改昵称、删除。

2. 与其他页面关系
- 可联动 `订单管理页`排查用户订单问题（通过 user_id）。

3. 业务组件
- `用户查询卡 UserLookupCard`
- `用户信息卡 ManagedUserCard`
- `昵称编辑弹窗 UserNicknameModal`
- `删除确认弹窗 DeleteConfirmModal`

4. 组件构成
- `UserLookupCard`：用户 ID 输入框、查询按钮。
- `ManagedUserCard`：用户基础信息、改昵称按钮、删除按钮。

5. 组件与 Hooks 关联
- 用户查询 -> `useManagedUserProfileQuery`
- 改昵称 -> `useManagedUserNicknameMutation`
- 删除用户 -> `useManagedUserDeleteMutation`

6. shadcn 组件构成
- `UserLookupCard`：`Card` + `Input` + `Button`
- `ManagedUserCard`：`Card` + `Badge` + `Button`
- `UserNicknameModal`：`Dialog` + `Form` + `Input` + `Button`
- `DeleteConfirmModal`：`AlertDialog` + `Button`

## 3.7 商品管理页

1. 页面作用
- 管理商品创建、更新、删除、查看与列表检索。

2. 与其他页面关系
- 与 `秒杀活动详情页`有数据关联（活动商品来源于商品池）。

3. 业务组件
- `商品筛选栏 ProductFilterBar`
- `商品表格 ProductTable`
- `商品编辑抽屉 ProductUpsertDrawer`

4. 组件构成
- `ProductTable`：SKU、名称、价格、库存、状态、操作列。
- `ProductUpsertDrawer`：名称、主图、描述、价格、库存、状态、保存按钮。

5. 组件与 Hooks 关联
- 列表 -> `useAdminProductListQuery`
- 详情 -> `useAdminProductDetailQuery`
- 创建 -> `useCreateProductMutation`
- 更新 -> `useUpdateProductMutation`
- 删除 -> `useDeleteProductMutation`

6. shadcn 组件构成
- `ProductFilterBar`：`Input` + `Select` + `Button`
- `ProductTable`：`Table` + `Badge` + `DropdownMenu`
- `ProductUpsertDrawer`：`Sheet` + `Form` + `Input` + `Textarea` + `Select` + `Button`

## 3.8 订单管理页

1. 页面作用
- 处理订单查询、审核、发货三类核心操作。

2. 与其他页面关系
- 点击订单进入 `订单详情抽屉`。
- 可跳转 `用户管理页`查看下单用户信息。

3. 业务组件
- `订单筛选栏 AdminOrderFilterBar`
- `订单表格 AdminOrderTable`
- `审核弹窗 ReviewOrderModal`
- `发货弹窗 ShipOrderModal`

4. 组件构成
- `AdminOrderTable`：订单号、用户、金额、订单状态、支付状态、审核状态、发货状态、操作列。
- `ReviewOrderModal`：审核结果单选、备注输入框、确认按钮。
- `ShipOrderModal`：物流单号输入框、确认按钮。

5. 组件与 Hooks 关联
- 列表 -> `useAdminOrderListQuery`
- 详情 -> `useAdminOrderDetailQuery`
- 审核 -> `useReviewOrderMutation`
- 发货 -> `useShipOrderMutation`

6. shadcn 组件构成
- `AdminOrderFilterBar`：`Input` + `Select` + `Button`
- `AdminOrderTable`：`Table` + `Badge` + `DropdownMenu`
- `ReviewOrderModal`：`Dialog` + `RadioGroup` + `Textarea` + `Button`
- `ShipOrderModal`：`Dialog` + `Form` + `Input` + `Button`

## 3.9 秒杀活动列表页

1. 页面作用
- 管理秒杀活动生命周期（创建、更新、删除、发布、下线）。

2. 与其他页面关系
- 行点击进入 `秒杀活动详情页`配置活动商品、查看流量与订单。

3. 业务组件
- `活动筛选栏 SeckillActivityFilterBar`
- `活动表格 SeckillActivityTable`
- `活动编辑弹窗 SeckillActivityFormModal`

4. 组件构成
- `SeckillActivityTable`：标题、时间窗口、状态、操作列。
- `SeckillActivityFormModal`：标题、描述、样式配置 JSON、开始时间、结束时间、保存按钮。

5. 组件与 Hooks 关联
- 列表 -> `useSeckillActivityListQuery`
- 创建 -> `useCreateSeckillActivityMutation`
- 更新 -> `useUpdateSeckillActivityMutation`
- 删除 -> `useDeleteSeckillActivityMutation`
- 发布 -> `usePublishSeckillActivityMutation`
- 下线 -> `useOfflineSeckillActivityMutation`

6. shadcn 组件构成
- `SeckillActivityFilterBar`：`Input` + `Select` + `Button`
- `SeckillActivityTable`：`Table` + `Badge` + `DropdownMenu`
- `SeckillActivityFormModal`：`Dialog` + `Form` + `Input` + `Textarea` + `Popover` + `Calendar` + `Button`

## 3.10 秒杀活动详情页

1. 页面作用
- 配置活动商品并查看活动流量与订单追溯数据。

2. 与其他页面关系
- 返回 `秒杀活动列表页`。
- 与 `订单管理页`互跳查看订单详情。

3. 业务组件
- `活动商品表格 ActivityItemTable`
- `活动商品编辑弹窗 ActivityItemFormModal`
- `流量看板 TrafficPanel`
- `活动订单表格 ActivityOrderTable`

4. 组件构成
- `ActivityItemTable`：商品快照、秒杀价、预占库存、可售库存、限购规则、操作列。
- `ActivityItemFormModal`：`product_id`、秒杀价、预占库存、限购模式、窗口秒数、限购件数、单次上限、状态。
- `TrafficPanel`：PV、UV、点击、抢购尝试、成功/失败、支付成功、订单关闭。
- `ActivityOrderTable`：订单号、用户、数量、订单状态、支付状态、关闭原因、时间。

5. 组件与 Hooks 关联
- 活动详情 -> `useSeckillActivityDetailQuery`
- 新增商品 -> `useCreateSeckillItemMutation`
- 更新商品 -> `useUpdateSeckillItemMutation`
- 移除商品 -> `useRemoveSeckillItemMutation`
- 流量数据 -> `useSeckillTrafficQuery`
- 活动订单 -> `useSeckillOrdersQuery`

6. shadcn 组件构成
- `ActivityItemTable`：`Table` + `Badge` + `DropdownMenu`
- `ActivityItemFormModal`：`Dialog` + `Form` + `Input` + `Select` + `Button`
- `TrafficPanel`：`Card` + `Tabs` + `ChartContainer` + `Badge`
- `ActivityOrderTable`：`Table` + `Badge` + `Button`

## 4. 页面级实现约束

1. 所有页面按钮显隐由 `useAdminPermission` 统一控制，禁止在组件内硬编码权限字符串。
2. `admin_management` 页面进入前应先判断 `data_scope=all`，否则直接展示无权限态。
3. 敏感写操作组件必须二次确认并显示不可逆提示。
4. 所有列表页统一使用 `page/page_size`，并与 QueryKey 维度保持一致。
5. 会话策略统一 `401 -> refresh(单飞) -> 重放一次`，失败后登出并清缓存。
