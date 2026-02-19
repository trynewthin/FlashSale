import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Search, X } from "lucide-react"

import { type OrderView } from "@/api/modules/order"
import { useAdminOrderListQuery } from "@/hooks/biz/use-order-mgmt-hooks"
import { useAdminPermission } from "@/hooks/auth/use-admin-permission"
import { useApiError } from "@/hooks/common/use-api-error"
import { formatCent, formatUnix } from "@/lib/format"
import {
  ORDER_STATUS_LABELS,
  PAYMENT_STATUS_LABELS,
  REVIEW_FILTER_OPTIONS,
  REVIEW_STATUS,
  REVIEW_STATUS_LABELS,
  SHIPPING_STATUS_LABELS,
} from "@/features/order/status"

import { ReviewOrderDialog, ShipOrderDialog, OrderRowActions } from "@/components/order"



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
    <div className="space-y-4 p-6">
      <h1 className="text-2xl font-semibold tracking-tight">订单管理</h1>

      <div className="flex items-end gap-3">
        <div className="space-y-1">
          <Label className="text-xs">订单号</Label>
          <Input placeholder="order_no" value={orderNo} onChange={(e) => setOrderNo(e.target.value)} className="w-48"
            onKeyDown={(e) => e.key === "Enter" && handleSearch()} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">订单状态</Label>
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v ?? "")}>
            <SelectTrigger className="w-28"><SelectValue placeholder="全部" /></SelectTrigger>
            <SelectContent>
              {Object.entries(ORDER_STATUS_LABELS).map(([k, v]) => <SelectItem key={k} value={k}>{v}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label className="text-xs">审核状态</Label>
          <Select value={reviewFilter} onValueChange={(v) => setReviewFilter(v ?? "")}>
            <SelectTrigger className="w-28"><SelectValue placeholder="全部" /></SelectTrigger>
            <SelectContent>
              {Object.entries(REVIEW_FILTER_OPTIONS).map(([k, v]) => <SelectItem key={k} value={k}>{v}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
        <Button size="sm" onClick={handleSearch}><Search className="mr-1 size-4" />查询</Button>
        <Button size="sm" variant="outline" onClick={handleReset}><X className="mr-1 size-4" />重置</Button>
      </div>

      {listQuery.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(listQuery.error)}</AlertDescription></Alert>}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>订单号</TableHead>
              <TableHead>用户</TableHead>
              <TableHead>金额</TableHead>
              <TableHead>订单状态</TableHead>
              <TableHead>支付</TableHead>
              <TableHead>审核</TableHead>
              <TableHead>发货</TableHead>
              <TableHead>时间</TableHead>
              <TableHead className="w-16">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 && (
              <TableRow><TableCell colSpan={9} className="text-center text-muted-foreground">{listQuery.isLoading ? "加载中…" : "暂无数据"}</TableCell></TableRow>
            )}
            {items.map((order) => {
              // 后端 proto3 在 0 值字段时会省略，前端按默认值兜底显示。
              const paymentStatus = Number(order.payment_status ?? 0)
              const reviewStatus = Number(order.review_status ?? 0)
              const shippingStatus = Number(order.shipping_status ?? 0)
              return (
                <TableRow key={order.order_id}>
                  <TableCell className="font-mono text-xs">{order.order_no}</TableCell>
                  <TableCell>{order.user_id}</TableCell>
                  <TableCell>{formatCent(order.total_amount_cent)}</TableCell>
                  <TableCell><Badge variant="outline">{ORDER_STATUS_LABELS[order.order_status] ?? order.order_status}</Badge></TableCell>
                  <TableCell><Badge variant="secondary">{PAYMENT_STATUS_LABELS[paymentStatus] ?? paymentStatus}</Badge></TableCell>
                  <TableCell><Badge variant={reviewStatus === REVIEW_STATUS.passed ? "default" : "secondary"}>{REVIEW_STATUS_LABELS[reviewStatus] ?? reviewStatus}</Badge></TableCell>
                  <TableCell><Badge variant="secondary">{SHIPPING_STATUS_LABELS[shippingStatus] ?? shippingStatus}</Badge></TableCell>
                  <TableCell className="text-sm text-muted-foreground">{formatUnix(order.created_at_unix)}</TableCell>
                  <TableCell>
                    <OrderRowActions order={order} canReview={canReview} canShip={canShip}
                      onReview={() => setReviewTarget(order)} onShip={() => setShipTarget(order)} />
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>

      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">共 {total} 条</span>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</Button>
          <span className="flex items-center px-2">{page} / {totalPages || 1}</span>
          <Button size="sm" variant="outline" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</Button>
        </div>
      </div>

      {reviewTarget && <ReviewOrderDialog order={reviewTarget} onClose={() => setReviewTarget(null)} />}
      {shipTarget && <ShipOrderDialog order={shipTarget} onClose={() => setShipTarget(null)} />}
    </div>
  )
}

