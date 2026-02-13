import { useState } from "react"
import { useParams, useNavigate } from "react-router-dom"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { ArrowLeft } from "lucide-react"

import { useOrderDetailQuery } from "@/hooks/user/use-order-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

import { PayDialog, CancelDialog, ConfirmReceiptDialog } from "@/components/order"

const ORDER_STATUS: Record<number, string> = { 1: "待支付", 2: "已支付", 3: "已审核", 4: "已发货", 5: "已收货", 6: "已关闭" }
const PAYMENT_STATUS: Record<number, string> = { 0: "未支付", 1: "已支付", 2: "已退款" }
const REVIEW_STATUS: Record<number, string> = { 0: "待审核", 1: "通过", 2: "拒绝" }
const SHIPPING_STATUS: Record<number, string> = { 0: "未发货", 1: "已发货", 2: "已收货" }

function formatCent(cent: number) { return `¥${(cent / 100).toFixed(2)}` }
function formatUnix(unix: number) { if (!unix) return "-"; return new Date(unix * 1000).toLocaleString("zh-CN") }

export function OrderDetailPage() {
  const { orderId } = useParams<{ orderId: string }>()
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()

  const detailQuery = useOrderDetailQuery(orderId)
  const order = detailQuery.data?.order

  const [payOpen, setPayOpen] = useState(false)
  const [cancelOpen, setCancelOpen] = useState(false)
  const [confirmReceiptOpen, setConfirmReceiptOpen] = useState(false)

  if (detailQuery.isLoading) {
    return <div className="flex items-center justify-center py-12 text-muted-foreground">加载中…</div>
  }

  if (detailQuery.isError) {
    return <Alert variant="destructive"><AlertDescription>{toUserMessage(detailQuery.error)}</AlertDescription></Alert>
  }

  if (!order) return null

  const canPay = order.order_status === 1
  const canCancel = order.order_status === 1
  const canConfirmReceipt = order.order_status === 4

  return (
    <div className="space-y-4">
      <Button variant="ghost" size="sm" onClick={() => navigate("/orders")}>
        <ArrowLeft className="mr-1 size-4" />返回订单列表
      </Button>

      <Card>
        <CardHeader>
          <div className="flex items-start justify-between">
            <CardTitle className="text-lg">订单详情</CardTitle>
            <Badge variant="outline">{ORDER_STATUS[order.order_status] ?? order.order_status}</Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-4">
            {order.main_image && <img src={order.main_image} alt="" className="size-20 rounded object-cover" />}
            <div className="space-y-1">
              <p className="font-medium">{order.product_name}</p>
              <p className="text-sm text-muted-foreground">SKU: {order.sku_code}</p>
              <p className="text-sm">单价 {formatCent(order.unit_price_cent)} × {order.quantity}</p>
              <p className="text-base font-semibold text-primary">{formatCent(order.total_amount_cent)}</p>
            </div>
          </div>

          <Separator />

          <div className="grid grid-cols-2 gap-3 text-sm">
            <InfoRow label="订单号" value={order.order_no} />
            <InfoRow label="来源" value={order.order_source === 2 ? "秒杀" : "普通"} />
            <InfoRow label="支付状态" value={PAYMENT_STATUS[order.payment_status] ?? "-"} />
            <InfoRow label="审核状态" value={REVIEW_STATUS[order.review_status] ?? "-"} />
            <InfoRow label="发货状态" value={SHIPPING_STATUS[order.shipping_status] ?? "-"} />
            <InfoRow label="创建时间" value={formatUnix(order.created_at_unix)} />
            {order.tracking_no && <InfoRow label="物流单号" value={order.tracking_no} />}
            {order.close_reason && <InfoRow label="关闭原因" value={order.close_reason} />}
            {order.receiver_name && <InfoRow label="收件人" value={order.receiver_name} />}
            {order.receiver_phone && <InfoRow label="收件电话" value={order.receiver_phone} />}
            {order.receiver_address && <InfoRow label="收件地址" value={order.receiver_address} />}
          </div>

          <Separator />

          <div className="flex gap-2">
            {canPay && <Button size="sm" onClick={() => setPayOpen(true)}>支付 & 填写信息</Button>}
            {canCancel && <Button size="sm" variant="outline" onClick={() => setCancelOpen(true)}>取消订单</Button>}
            {canConfirmReceipt && <Button size="sm" onClick={() => setConfirmReceiptOpen(true)}>确认收货</Button>}
          </div>
        </CardContent>
      </Card>

      {payOpen && orderId && <PayDialog orderId={orderId} onClose={() => setPayOpen(false)} />}
      {cancelOpen && orderId && <CancelDialog orderId={orderId} onClose={() => setCancelOpen(false)} />}
      {confirmReceiptOpen && orderId && <ConfirmReceiptDialog orderId={orderId} onClose={() => setConfirmReceiptOpen(false)} />}
    </div>
  )
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-sm break-all">{value || "-"}</p>
    </div>
  )
}
