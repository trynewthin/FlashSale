import { useState } from "react"
import { useParams, useNavigate } from "react-router-dom"
import { ArrowLeft, Package, CreditCard, ClipboardCheck, Truck, CheckCircle2, XCircle, Info } from "lucide-react"

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"

import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Stepper, type StepperStep } from "@/components/ui/stepper"
import { useOrderDetailQuery } from "@/hooks/user/use-order-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import {
  ORDER_STATUS,
  ORDER_STATUS_LABELS,
  PAYMENT_STATUS_LABELS,
  REVIEW_STATUS_LABELS,
  SHIPPING_STATUS_LABELS,
  canCancelOrder,
  canConfirmReceipt,
  canPayOrder,
} from "@/features/order/status"

import { PayDialog, CancelDialog, ConfirmReceiptDialog } from "@/components/order"

import { formatCent, formatUnix } from "@/lib/format"
import { cdnUrl } from "@/lib/cdn"

// ─── 步骤映射 ─────────────────────────────────────────────────────────────────

const ORDER_STEPS: StepperStep[] = [
  { label: "下单" },
  { label: "支付" },
  { label: "审核" },
  { label: "发货" },
  { label: "完成" },
]

const CLOSED_STEPS: StepperStep[] = [
  { label: "下单" },
  { label: "已关闭" },
]

function getStepIndex(orderStatus: number): { steps: StepperStep[]; current: number } {
  switch (orderStatus) {
    case ORDER_STATUS.pending_pay:
      return { steps: ORDER_STEPS, current: 1 }
    case ORDER_STATUS.pending_review:
      return { steps: ORDER_STEPS, current: 2 }
    case ORDER_STATUS.pending_ship:
      return { steps: ORDER_STEPS, current: 3 }
    case ORDER_STATUS.shipped:
      return { steps: ORDER_STEPS, current: 4 }
    case ORDER_STATUS.closed:
      return { steps: CLOSED_STEPS, current: 1 }
    default:
      return { steps: ORDER_STEPS, current: 0 }
  }
}

// ─── 页面 ─────────────────────────────────────────────────────────────────────

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
    return (
      <div className="space-y-6">
        <Skeleton className="h-6 w-24" />
        <div className="rounded-xl border border-border/60 p-6 space-y-4">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-20 w-full" />
          <Skeleton className="h-32 w-full" />
        </div>
      </div>
    )
  }

  if (detailQuery.isError) {
    return (
      <div className="text-center py-12 space-y-3">
        <p className="text-sm text-destructive">{toUserMessage(detailQuery.error)}</p>
        <button
          onClick={() => navigate(-1)}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          ← 返回
        </button>
      </div>
    )
  }

  if (!order) return null

  const orderStatus = Number(order.order_status ?? 0)
  const paymentStatus = Number(order.payment_status ?? 0)
  const reviewStatus = Number(order.review_status ?? 0)
  const shippingStatus = Number(order.shipping_status ?? 0)
  const isClosed = orderStatus === ORDER_STATUS.closed

  const canPay = canPayOrder(order)
  const canCancel = canCancelOrder(order)
  const canReceipt = canConfirmReceipt(order)

  const { steps, current } = getStepIndex(orderStatus)

  return (
    <div className="space-y-6">
      {/* 返回 */}
      <button
        onClick={() => navigate("/orders")}
        className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="size-4" />
        返回订单列表
      </button>

      <div className="rounded-xl border border-border/60 overflow-hidden">
        {/* 商品信息 + 右上角详情按钮 */}
        <div className="relative p-6">
          {/* 右上角 Popover */}
          <Popover>
            <PopoverTrigger className="absolute top-4 right-4 flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors rounded-md px-2 py-1 hover:bg-muted/50">
              <Info className="size-3.5" />
              详情
            </PopoverTrigger>
            <PopoverContent align="end" className="w-80 space-y-4">
              <h3 className="text-sm font-semibold">订单信息</h3>
              <div className="grid grid-cols-2 gap-x-4 gap-y-2.5">
                <InfoRow icon={<CreditCard className="size-3" />} label="支付状态" value={PAYMENT_STATUS_LABELS[paymentStatus] ?? "-"} />
                <InfoRow icon={<ClipboardCheck className="size-3" />} label="审核状态" value={REVIEW_STATUS_LABELS[reviewStatus] ?? "-"} />
                <InfoRow icon={<Truck className="size-3" />} label="发货状态" value={SHIPPING_STATUS_LABELS[shippingStatus] ?? "-"} />
                <InfoRow icon={<Package className="size-3" />} label="来源" value={order.order_source === 2 ? "秒杀" : "普通"} />
                <div className="col-span-2"><InfoRow label="创建时间" value={formatUnix(order.created_at_unix)} /></div>
                {order.tracking_no && <div className="col-span-2"><InfoRow icon={<Truck className="size-3" />} label="物流单号" value={order.tracking_no} /></div>}
                {order.close_reason && <div className="col-span-2"><InfoRow label="关闭原因" value={order.close_reason} /></div>}
              </div>
              {(order.receiver_name || order.receiver_phone || order.receiver_address) && (
                <>
                  <div className="h-px bg-border/40" />
                  <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">收件信息</h3>
                  <div className="grid grid-cols-2 gap-x-4 gap-y-2.5">
                    {order.receiver_name && <InfoRow label="收件人" value={order.receiver_name} />}
                    {order.receiver_phone && <InfoRow label="电话" value={order.receiver_phone} />}
                    {order.receiver_address && <div className="col-span-2"><InfoRow label="地址" value={order.receiver_address} /></div>}
                  </div>
                </>
              )}
              <div className="pt-1">
                <p className="text-[10px] text-muted-foreground/50 font-mono">{order.order_no}</p>
              </div>
            </PopoverContent>
          </Popover>

          <div className="flex items-center gap-4 pr-16">
            {order.main_image && (
              <img
                src={cdnUrl(order.main_image)}
                alt=""
                className="size-20 rounded-lg object-cover border border-border/40"
              />
            )}
            <div className="flex-1 min-w-0 space-y-1">
              <p className="font-medium truncate">{order.product_name}</p>
              <div className="flex items-baseline gap-2">
                <span className="text-lg font-bold text-primary">
                  {formatCent(order.total_amount_cent)}
                </span>
                <span className="text-xs text-muted-foreground">
                  {formatCent(order.unit_price_cent)} × {order.quantity}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* 分隔 */}
        <div className="h-px bg-border/60" />

        {/* 状态 + 步骤条 + 按钮 */}
        <div className="p-6 space-y-5">
          <div className="flex items-center gap-2">
            {isClosed ? (
              <XCircle className="size-5 text-destructive" />
            ) : orderStatus === ORDER_STATUS.shipped ? (
              <CheckCircle2 className="size-5 text-emerald-500" />
            ) : (
              <Package className="size-5 text-primary" />
            )}
            <h1 className="text-lg font-bold">
              {ORDER_STATUS_LABELS[orderStatus] ?? "未知状态"}
            </h1>
          </div>

          <Stepper steps={steps} currentStep={current} />

          {(canPay || canCancel || canReceipt) && (
            <div className="flex gap-2 pt-1">
              {canPay && (
                <Button onClick={() => setPayOpen(true)}>
                  <CreditCard className="size-4 mr-1.5" />
                  立即支付
                </Button>
              )}
              {canReceipt && (
                <Button size="sm" onClick={() => setConfirmReceiptOpen(true)}>
                  <ClipboardCheck className="size-3.5 mr-1.5" />
                  确认收货
                </Button>
              )}
              {canCancel && (
                <Button size="sm" variant="ghost" className="ml-auto" onClick={() => setCancelOpen(true)}>
                  取消订单
                </Button>
              )}
            </div>
          )}
        </div>
      </div>

      {/* 弹窗 */}
      {payOpen && orderId && <PayDialog orderId={orderId} onClose={() => setPayOpen(false)} />}
      {cancelOpen && orderId && <CancelDialog orderId={orderId} onClose={() => setCancelOpen(false)} />}
      {confirmReceiptOpen && orderId && <ConfirmReceiptDialog orderId={orderId} onClose={() => setConfirmReceiptOpen(false)} />}
    </div>
  )
}

// ─── 信息行 ───────────────────────────────────────────────────────────────────

function InfoRow({ label, value, icon }: { label: string; value: string; icon?: React.ReactNode }) {
  return (
    <div className="flex items-start gap-2">
      {icon && <span className="text-muted-foreground/60 mt-0.5">{icon}</span>}
      <div className="min-w-0">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-sm break-all">{value || "-"}</p>
      </div>
    </div>
  )
}
