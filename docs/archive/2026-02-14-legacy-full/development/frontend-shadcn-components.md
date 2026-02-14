# 前端 shadcn 组件集合清单

更新时间：2026-02-12  
适用模块：`frontend-user`、`frontend-ops`  
来源：`docs/development/frontend-user-page-design.md`、`docs/development/frontend-admin-page-design.md`

## 1. 目的

1. 统一声明当前页面设计已使用的 shadcn 组件集合。
2. 为组件实现、代码评审与设计对齐提供单一事实源。

## 2. 组件集合（已使用）

## 2.1 基础输入与表单

- `Form`
- `FormField`
- `Input`
- `Textarea`
- `Select`
- `Checkbox`
- `RadioGroup`

## 2.2 按钮与操作

- `Button`
- `DropdownMenu`

## 2.3 信息展示

- `Card`
- `CardHeader`
- `CardContent`
- `Badge`
- `Separator`
- `Table`
- `TableHeader`
- `TableBody`
- `ScrollArea`

## 2.4 反馈与状态

- `Alert`
- `AlertDialog`

## 2.5 弹层与容器

- `Dialog`
- `DialogHeader`
- `DialogContent`
- `Sheet`
- `Popover`

## 2.6 导航与布局

- `NavigationMenu`（或基于其封装的侧边导航）
- `Tabs`

## 2.7 日期与图表

- `Calendar`
- `ChartContainer`

## 3. 约束

1. 页面新增业务组件时，若引入新的 shadcn 组件，必须同步更新本清单。
2. 若仅使用自定义业务组件，需在业务组件文档中注明其底层由哪些 shadcn 组件组成。
3. `Pagination` 当前以业务封装为主，底层一般由 `Button + 文本信息` 组合实现；若后续引入统一 `Pagination` 组件，也需回填本清单。

