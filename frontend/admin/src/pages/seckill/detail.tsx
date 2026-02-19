import { useState } from "react"
import { useParams, useNavigate } from "react-router-dom"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { ArrowLeft, Plus, Zap, Clock, Package, ShoppingCart, TrendingDown, AlertCircle } from "lucide-react"

import { type ActivityItemAdmin } from "@/api/modules/seckill"
import {
  useSeckillActivityDetailQuery,
  useCreateSeckillItemMutation,
  useSeckillOrdersQuery,
} from "@/hooks/biz/use-seckill-mgmt-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

import {
  ItemFormFields,
  ItemEditForm, DeleteItemDialog, ItemRowActions,
} from "@/components/seckill"
import { ORDER_STATUS_LABELS, PAYMENT_STATUS_LABELS } from "@/features/order/status"
import { cdnUrl } from "@/lib/cdn"
import { formatCent, formatUnix } from "@/lib/format"

// ── 常量 ──────────────────────────────────────────────────
const ITEM_STATUS: Record<number, { label: string; color: string }> = {
  0: { label: "停用", color: "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400" },
  1: { label: "启用", color: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400" },
}
const LIMIT_MODE: Record<number, string> = { 0: "不限", 1: "按窗口", 2: "按活动" }

const ACTIVITY_STATUS: Record<number, { label: string; color: string }> = {
  0: { label: "草稿", color: "bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400" },
  1: { label: "已发布", color: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400" },
  2: { label: "已下线", color: "bg-slate-100 text-slate-400 dark:bg-slate-800 dark:text-slate-500" },
}

const ORDER_STATUS_COLORS: Record<number, string> = {
  1: "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-400",
  2: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-400",
  3: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-400",
  4: "bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-400",
  5: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400",
  6: "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400",
}

const PAYMENT_COLORS: Record<number, string> = {
  0: "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400",
  1: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400",
  2: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400",
}

function StatusPill({ label, color }: { label: string; color: string }) {
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${color}`}>
      {label}
    </span>
  )
}

// ── 主页面 ────────────────────────────────────────────────
export function SeckillActivityDetailPage() {
  const { activityId } = useParams<{ activityId: string }>()
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()

  const detailQuery = useSeckillActivityDetailQuery(activityId)
  const activity = detailQuery.data?.activity

  const [orderPage, setOrderPage] = useState(1)
  const ordersQuery = useSeckillOrdersQuery(activityId, { page: orderPage, page_size: 20 })

  const [itemFormOpen, setItemFormOpen] = useState(false)
  const [editItem, setEditItem] = useState<ActivityItemAdmin | null>(null)
  const [deleteItem, setDeleteItem] = useState<ActivityItemAdmin | null>(null)
  const [itemForm, setItemForm] = useState({
    product_id: "" as string, seckill_price_cent: 0, reserved_stock_total: 0,
    user_limit_mode: 0, user_limit_window_sec: 0, user_limit_qty: 0, max_qty_per_order: 1, status: 1,
  })

  const createItemMutation = useCreateSeckillItemMutation(activityId ?? "")

  // ── 加载 / 错误 ──────────────────────────────────────────
  if (detailQuery.isLoading) {
    return (
      <div className="flex items-center justify-center p-16 text-muted-foreground">
        <span className="flex items-center gap-2">
          <span className="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          加载中…
        </span>
      </div>
    )
  }

  if (detailQuery.isError) {
    return (
      <div className="p-6">
        <div className="flex items-center gap-2 rounded-lg border border-destructive/40 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          <AlertCircle className="size-4 shrink-0" />
          {toUserMessage(detailQuery.error)}
        </div>
      </div>
    )
  }

  if (!activity) return null

  const openCreateItem = () => {
    setEditItem(null)
    setItemForm({ product_id: "", seckill_price_cent: 0, reserved_stock_total: 0, user_limit_mode: 0, user_limit_window_sec: 0, user_limit_qty: 0, max_qty_per_order: 1, status: 1 })
    setItemFormOpen(true)
  }

  const openEditItem = (item: ActivityItemAdmin) => {
    setEditItem(item)
    setItemForm({
      product_id: item.product_id, seckill_price_cent: item.seckill_price_cent,
      reserved_stock_total: item.reserved_stock_total, user_limit_mode: item.user_limit_mode,
      user_limit_window_sec: item.user_limit_window_sec, user_limit_qty: item.user_limit_qty,
      max_qty_per_order: item.max_qty_per_order, status: item.status,
    })
    setItemFormOpen(true)
  }

  const handleCreateItem = (e: React.FormEvent) => {
    e.preventDefault()
    if (!activityId) return
    createItemMutation.mutate(itemForm, { onSuccess: () => setItemFormOpen(false) })
  }

  const orders = ordersQuery.data?.list ?? []
  const ordersTotal = ordersQuery.data?.total ?? 0
  const ordersTotalPages = Math.ceil(ordersTotal / 20)

  const actStatus = Number(activity.status ?? 0)
  const actStatusInfo = ACTIVITY_STATUS[actStatus]

  // 统计
  const totalItems = activity.items?.length ?? 0
  const totalStock = activity.items?.reduce((s, i) => s + (i.reserved_stock_total ?? 0), 0) ?? 0
  const totalSold = activity.items?.reduce((s, i) => s + (i.sold_stock ?? 0), 0) ?? 0
  const totalAvail = activity.items?.reduce((s, i) => s + (i.available_stock ?? 0), 0) ?? 0

  return (
    <div className="space-y-6 p-6">

      {/* ── 顶部 Hero 区 ── */}
      <div className="rounded-xl border bg-card p-6">
        <div className="flex items-start gap-4">
          {/* 返回 */}
          <Button variant="ghost" size="icon" className="mt-0.5 shrink-0" onClick={() => navigate("/seckill")}>
            <ArrowLeft className="size-4" />
          </Button>

          {/* 图标 */}
          <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-amber-100 dark:bg-amber-950">
            <Zap className="size-5 text-amber-600 dark:text-amber-400" />
          </span>

          {/* 标题区 */}
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2.5 flex-wrap">
              <h1 className="text-xl font-semibold leading-tight">{activity.title}</h1>
              {actStatusInfo && (
                <StatusPill label={actStatusInfo.label} color={actStatusInfo.color} />
              )}
            </div>
            {activity.description && (
              <p className="mt-1 text-sm text-muted-foreground">{activity.description}</p>
            )}
            <div className="mt-3 flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
              <span className="flex items-center gap-1.5">
                <Clock className="size-3.5" />
                开始：{formatUnix(activity.start_at_unix)}
              </span>
              <span className="flex items-center gap-1.5">
                <Clock className="size-3.5" />
                结束：{formatUnix(activity.end_at_unix)}
              </span>
            </div>
          </div>
        </div>

        {/* KPI 小卡片 */}
        <div className="mt-5 grid grid-cols-4 gap-3 border-t pt-5">
          {[
            { icon: <Package className="size-4 text-blue-500" />, label: "商品数", value: totalItems },
            { icon: <TrendingDown className="size-4 text-amber-500" />, label: "预占库存", value: totalStock },
            { icon: <ShoppingCart className="size-4 text-emerald-500" />, label: "已售", value: totalSold },
            { icon: <Package className="size-4 text-indigo-500" />, label: "剩余可售", value: totalAvail },
          ].map(({ icon, label, value }) => (
            <div key={label} className="flex items-center gap-3 rounded-lg bg-muted/40 px-4 py-3">
              {icon}
              <div>
                <p className="text-xs text-muted-foreground">{label}</p>
                <p className="text-lg font-semibold">{value}</p>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* ── 活动商品 ── */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">活动商品</h2>
          <Button size="sm" onClick={openCreateItem}>
            <Plus className="mr-1.5 size-3.5" />添加商品
          </Button>
        </div>

        <div className="rounded-lg border overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/40 hover:bg-muted/40">
                <TableHead className="pl-4">商品</TableHead>
                <TableHead>秒杀价</TableHead>
                <TableHead>预占库存</TableHead>
                <TableHead>可售 / 已售</TableHead>
                <TableHead>限购</TableHead>
                <TableHead>状态</TableHead>
                <TableHead className="w-14 pr-4">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(!activity.items || activity.items.length === 0) && (
                <TableRow>
                  <TableCell colSpan={7} className="py-10 text-center text-sm text-muted-foreground">
                    暂无商品，点击右上角添加
                  </TableCell>
                </TableRow>
              )}
              {activity.items?.map((item) => {
                const itemStatusInfo = ITEM_STATUS[Number(item.status ?? 0)]
                return (
                  <TableRow key={item.item_id}>
                    <TableCell className="pl-4">
                      <div className="flex items-center gap-2.5">
                        {item.snapshot_main_image
                          ? <img src={cdnUrl(item.snapshot_main_image)} alt="" className="size-9 rounded-md object-cover border" />
                          : <span className="flex size-9 items-center justify-center rounded-md bg-muted"><Package className="size-4 text-muted-foreground" /></span>
                        }
                        <span className="text-sm font-medium">{item.snapshot_name}</span>
                      </div>
                    </TableCell>
                    <TableCell className="font-semibold text-rose-600 dark:text-rose-400">
                      {formatCent(item.seckill_price_cent)}
                    </TableCell>
                    <TableCell className="text-sm">{item.reserved_stock_total}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1 text-sm">
                        <span className="text-emerald-600 dark:text-emerald-400">{item.available_stock}</span>
                        <span className="text-muted-foreground">/</span>
                        <span>{item.sold_stock}</span>
                      </div>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {LIMIT_MODE[item.user_limit_mode] ?? item.user_limit_mode}
                      {item.user_limit_qty > 0 && ` · ${item.user_limit_qty}件`}
                    </TableCell>
                    <TableCell>
                      {itemStatusInfo && (
                        <StatusPill label={itemStatusInfo.label} color={itemStatusInfo.color} />
                      )}
                    </TableCell>
                    <TableCell className="pr-4">
                      <ItemRowActions activityId={activityId!} item={item}
                        onEdit={() => openEditItem(item)} onDelete={() => setDeleteItem(item)} />
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      </div>

      {/* ── 活动订单 ── */}
      <div className="space-y-3">
        <h2 className="text-base font-semibold">活动订单</h2>

        <div className="rounded-lg border overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/40 hover:bg-muted/40">
                <TableHead className="pl-4">订单号</TableHead>
                <TableHead>用户</TableHead>
                <TableHead>数量</TableHead>
                <TableHead>订单状态</TableHead>
                <TableHead>支付</TableHead>
                <TableHead>关闭原因</TableHead>
                <TableHead className="pr-4">时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {orders.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} className="py-10 text-center text-sm text-muted-foreground">
                    {ordersQuery.isLoading
                      ? <span className="flex items-center justify-center gap-2"><span className="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />加载中…</span>
                      : "暂无订单"}
                  </TableCell>
                </TableRow>
              )}
              {orders.map((o) => {
                const paymentStatus = Number(o.payment_status ?? 0)
                const orderStatus = Number(o.order_status ?? 0)
                return (
                  <TableRow key={o.link_id}>
                    <TableCell className="pl-4">
                      <code className="text-xs font-mono text-muted-foreground">{o.order_no}</code>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">{o.user_id}</TableCell>
                    <TableCell className="text-sm">{o.quantity}</TableCell>
                    <TableCell>
                      <StatusPill
                        label={ORDER_STATUS_LABELS[orderStatus] ?? String(orderStatus)}
                        color={ORDER_STATUS_COLORS[orderStatus] ?? "bg-muted text-muted-foreground"}
                      />
                    </TableCell>
                    <TableCell>
                      <StatusPill
                        label={PAYMENT_STATUS_LABELS[paymentStatus] ?? String(paymentStatus)}
                        color={PAYMENT_COLORS[paymentStatus] ?? "bg-muted text-muted-foreground"}
                      />
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{o.close_reason || "—"}</TableCell>
                    <TableCell className="pr-4 text-xs text-muted-foreground whitespace-nowrap">{formatUnix(o.created_at_unix)}</TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>

        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">第 {orderPage} 页 · 共 {ordersTotal} 条</span>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" disabled={orderPage <= 1} onClick={() => setOrderPage(orderPage - 1)}>上一页</Button>
            <span className="flex items-center px-2 text-muted-foreground">{orderPage} / {ordersTotalPages || 1}</span>
            <Button size="sm" variant="outline" disabled={orderPage >= ordersTotalPages} onClick={() => setOrderPage(orderPage + 1)}>下一页</Button>
          </div>
        </div>
      </div>

      {/* ── 弹窗 ── */}
      <Dialog open={itemFormOpen} onOpenChange={setItemFormOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editItem ? "编辑活动商品" : "添加活动商品"}</DialogTitle>
          </DialogHeader>
          {editItem && activityId ? (
            <ItemEditForm activityId={activityId} item={editItem} formData={itemForm} setFormData={setItemForm} onClose={() => setItemFormOpen(false)} />
          ) : (
            <form onSubmit={handleCreateItem} className="space-y-4">
              <ItemFormFields formData={itemForm} setFormData={setItemForm} isCreate />
              {createItemMutation.isError && (
                <Alert variant="destructive"><AlertDescription>{toUserMessage(createItemMutation.error)}</AlertDescription></Alert>
              )}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setItemFormOpen(false)}>取消</Button>
                <Button type="submit" disabled={createItemMutation.isPending}>
                  {createItemMutation.isPending ? "保存中…" : "保存"}
                </Button>
              </div>
            </form>
          )}
        </DialogContent>
      </Dialog>

      {deleteItem && activityId && (
        <DeleteItemDialog activityId={activityId} item={deleteItem} onClose={() => setDeleteItem(null)} />
      )}
    </div>
  )
}
