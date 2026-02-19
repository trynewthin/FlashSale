import { useState } from "react"
import { useNavigate } from "react-router-dom"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/select"
import { Label } from "@/components/ui/label"

import { useOrderListQuery } from "@/hooks/user/use-order-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { ORDER_STATUS_LABELS } from "@/features/order/status"

import { formatCent, formatUnix } from "@/lib/format"
import { cdnUrl } from "@/lib/cdn"

export function OrderListPage() {
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState("")

  const listQuery = useOrderListQuery({
    page, page_size: 10,
    order_status: statusFilter ? Number(statusFilter) : undefined,
  })

  const items = listQuery.data?.list ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / 10)

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">我的订单</h1>
        <div className="flex items-center gap-2">
          <Label className="text-xs">状态</Label>
          <Select value={statusFilter || "all"} onValueChange={(v: string | null) => { setStatusFilter(!v || v === "all" ? "" : v); setPage(1) }}>
            <SelectTrigger className="w-24">
              <span className="flex flex-1 text-start">
                {statusFilter ? (ORDER_STATUS_LABELS[Number(statusFilter)] ?? statusFilter) : "全部"}
              </span>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部</SelectItem>
              {Object.entries(ORDER_STATUS_LABELS).map(([k, v]) => <SelectItem key={k} value={k}>{v}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
      </div>

      {listQuery.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(listQuery.error)}</AlertDescription></Alert>}
      {listQuery.isLoading && <p className="text-center text-muted-foreground py-12">加载中…</p>}

      <div className="space-y-3">
        {items.map((order) => (
          <div
            key={order.order_id}
            className="rounded-xl border border-border/60 p-4 cursor-pointer transition-shadow hover:shadow-md"
            onClick={() => navigate(`/orders/${order.order_id}`)}
          >
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-3">
                {order.main_image && <img src={cdnUrl(order.main_image)} alt="" className="size-14 rounded-lg object-cover" />}
                <div className="space-y-0.5">
                  <p className="text-sm font-medium line-clamp-1">{order.product_name}</p>
                  <p className="text-xs text-muted-foreground">{formatUnix(order.created_at_unix)}</p>
                </div>
              </div>
              <div className="text-right space-y-0.5">
                <Badge variant="outline">{ORDER_STATUS_LABELS[order.order_status] ?? order.order_status}</Badge>
                <p className="text-sm font-semibold text-primary">{formatCent(order.total_amount_cent)}</p>
                <p className="text-xs text-muted-foreground">×{order.quantity}</p>
              </div>
            </div>
          </div>
        ))}
      </div>

      {items.length === 0 && !listQuery.isLoading && (
        <p className="text-center text-muted-foreground py-12">暂无订单</p>
      )}

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-4">
          <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</Button>
          <span className="text-sm text-muted-foreground">{page} / {totalPages}</span>
          <Button size="sm" variant="outline" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</Button>
        </div>
      )}
    </div>
  )
}
