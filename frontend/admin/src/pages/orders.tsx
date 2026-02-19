import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Search, X, AlertCircle } from "lucide-react"

import { type OrderView } from "@/api/modules/order"
import { useAdminOrderListQuery } from "@/hooks/biz/use-order-mgmt-hooks"
import { useAdminPermission } from "@/hooks/auth/use-admin-permission"
import { useApiError } from "@/hooks/common/use-api-error"
import { formatCent, formatUnix } from "@/lib/format"
import {
  ORDER_STATUS_LABELS,
  PAYMENT_STATUS_LABELS,
  REVIEW_FILTER_OPTIONS,
  REVIEW_STATUS_LABELS,
  SHIPPING_STATUS_LABELS,
} from "@/features/order/status"

import { ReviewOrderDialog, ShipOrderDialog, OrderRowActions } from "@/components/order"

// ── 彩色状态徽标 ──────────────────────────────────────────
function StatusPill({ label, color }: { label: string; color: string }) {
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${color}`}>
      {label}
    </span>
  )
}

const ORDER_STATUS_COLORS: Record<number, string> = {
  1: "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-400",   // 待支付
  2: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-400", // 待审核
  3: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-400",       // 待发货
  4: "bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-400", // 已发货
  5: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400", // 已完成
  6: "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400",    // 已关闭
}

const PAYMENT_COLORS: Record<number, string> = {
  0: "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400",
  1: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400",
  2: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400",
}

const REVIEW_COLORS: Record<number, string> = {
  0: "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400",
  1: "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-400",
  2: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400",
  3: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400",
}

const SHIPPING_COLORS: Record<number, string> = {
  0: "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400",
  1: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-400",
  2: "bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-400",
  3: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400",
}

export function OrderManagementPage() {
  const { toUserMessage } = useApiError()
  const { can } = useAdminPermission()
  const [page, setPage] = useState(1)
  const [orderNo, setOrderNo] = useState("")
  const [searchOrderNo, setSearchOrderNo] = useState("")
  const [statusFilter, setStatusFilter] = useState("")
  const [reviewFilter, setReviewFilter] = useState("")

  const listQuery = useAdminOrderListQuery({
    page, page_size: 20,
    order_no: searchOrderNo || undefined,
    order_status: statusFilter ? Number(statusFilter) : undefined,
    review_status: reviewFilter ? Number(reviewFilter) : undefined,
  })

  const [reviewTarget, setReviewTarget] = useState<OrderView | null>(null)
  const [shipTarget, setShipTarget] = useState<OrderView | null>(null)

  const canReview = can("order_management") || can("order_review_management")
  const canShip = can("order_management")

  const items = listQuery.data?.list ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / 20)

  const handleSearch = () => { setSearchOrderNo(orderNo); setPage(1) }
  const handleReset = () => { setOrderNo(""); setSearchOrderNo(""); setStatusFilter(""); setReviewFilter(""); setPage(1) }

  return (
    <div className="space-y-5 p-6">

      {/* 顶部 */}
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">订单管理</h1>
        <p className="text-sm text-muted-foreground mt-0.5">共 {total} 笔订单</p>
      </div>

      {/* 筛选栏 */}
      <div className="flex flex-wrap items-end gap-3 rounded-lg border bg-card px-4 py-3">
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">订单号</Label>
          <Input placeholder="order_no" value={orderNo} onChange={(e) => setOrderNo(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSearch()} className="h-8 w-48 text-sm font-mono" />
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">订单状态</Label>
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v ?? "")}
            items={Object.entries(ORDER_STATUS_LABELS).map(([k, v]) => ({ value: k, label: v }))}>
            <SelectTrigger className="h-8 w-28 text-sm"><SelectValue placeholder="全部" /></SelectTrigger>
            <SelectContent>
              {Object.entries(ORDER_STATUS_LABELS).map(([k, v]) => <SelectItem key={k} value={k}>{v}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">审核状态</Label>
          <Select value={reviewFilter} onValueChange={(v) => setReviewFilter(v ?? "")}
            items={Object.entries(REVIEW_FILTER_OPTIONS).map(([k, v]) => ({ value: k, label: v }))}>
            <SelectTrigger className="h-8 w-28 text-sm"><SelectValue placeholder="全部" /></SelectTrigger>
            <SelectContent>
              {Object.entries(REVIEW_FILTER_OPTIONS).map(([k, v]) => <SelectItem key={k} value={k}>{v}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
        <div className="flex gap-2">
          <Button size="sm" className="h-8" onClick={handleSearch}>
            <Search className="mr-1 size-3.5" />查询
          </Button>
          <Button size="sm" variant="outline" className="h-8" onClick={handleReset}>
            <X className="mr-1 size-3.5" />重置
          </Button>
        </div>
      </div>

      {/* 错误 */}
      {listQuery.isError && (
        <div className="flex items-center gap-2 rounded-lg border border-destructive/40 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          <AlertCircle className="size-4 shrink-0" />{toUserMessage(listQuery.error)}
        </div>
      )}

      {/* 表格 */}
      <div className="rounded-lg border overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-muted/40 hover:bg-muted/40">
              <TableHead className="pl-4">订单号</TableHead>
              <TableHead>用户</TableHead>
              <TableHead>金额</TableHead>
              <TableHead>订单状态</TableHead>
              <TableHead>支付</TableHead>
              <TableHead>审核</TableHead>
              <TableHead>发货</TableHead>
              <TableHead>下单时间</TableHead>
              <TableHead className="w-14 pr-4">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={9} className="py-12 text-center text-sm text-muted-foreground">
                  {listQuery.isLoading
                    ? <span className="flex items-center justify-center gap-2"><span className="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />加载中…</span>
                    : "暂无订单"}
                </TableCell>
              </TableRow>
            )}
            {items.map((order) => {
              const paymentStatus = Number(order.payment_status ?? 0)
              const reviewStatus = Number(order.review_status ?? 0)
              const shippingStatus = Number(order.shipping_status ?? 0)
              const orderStatus = Number(order.order_status ?? 0)
              return (
                <TableRow key={order.order_id}>
                  <TableCell className="pl-4">
                    <code className="text-xs font-mono text-muted-foreground">{order.order_no}</code>
                  </TableCell>
                  <TableCell className="text-xs font-mono text-muted-foreground">{order.user_id}</TableCell>
                  <TableCell className="text-sm font-semibold">{formatCent(order.total_amount_cent)}</TableCell>
                  <TableCell>
                    <StatusPill label={ORDER_STATUS_LABELS[orderStatus] ?? String(orderStatus)} color={ORDER_STATUS_COLORS[orderStatus] ?? "bg-muted text-muted-foreground"} />
                  </TableCell>
                  <TableCell>
                    <StatusPill label={PAYMENT_STATUS_LABELS[paymentStatus] ?? String(paymentStatus)} color={PAYMENT_COLORS[paymentStatus] ?? "bg-muted text-muted-foreground"} />
                  </TableCell>
                  <TableCell>
                    <StatusPill label={REVIEW_STATUS_LABELS[reviewStatus] ?? String(reviewStatus)} color={REVIEW_COLORS[reviewStatus] ?? "bg-muted text-muted-foreground"} />
                  </TableCell>
                  <TableCell>
                    <StatusPill label={SHIPPING_STATUS_LABELS[shippingStatus] ?? String(shippingStatus)} color={SHIPPING_COLORS[shippingStatus] ?? "bg-muted text-muted-foreground"} />
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground whitespace-nowrap">{formatUnix(order.created_at_unix)}</TableCell>
                  <TableCell className="pr-4">
                    <OrderRowActions order={order} canReview={canReview} canShip={canShip}
                      onReview={() => setReviewTarget(order)} onShip={() => setShipTarget(order)} />
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>

      {/* 分页 */}
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">第 {page} 页 · 共 {total} 条</span>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</Button>
          <span className="flex items-center px-2 text-muted-foreground">{page} / {totalPages || 1}</span>
          <Button size="sm" variant="outline" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</Button>
        </div>
      </div>

      {reviewTarget && <ReviewOrderDialog order={reviewTarget} onClose={() => setReviewTarget(null)} />}
      {shipTarget && <ShipOrderDialog order={shipTarget} onClose={() => setShipTarget(null)} />}
    </div>
  )
}
