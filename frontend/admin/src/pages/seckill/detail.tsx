import { useState } from "react"
import { useParams, useNavigate } from "react-router-dom"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { ArrowLeft, Plus } from "lucide-react"

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

const ITEM_STATUS: Record<number, string> = { 1: "启用", 2: "禁用" }
const LIMIT_MODE: Record<number, string> = { 0: "不限", 1: "按窗口", 2: "按活动" }
function formatCent(cent: number) { return `¥${(cent / 100).toFixed(2)}` }
function formatUnix(unix: number) { if (!unix) return "-"; return new Date(unix * 1000).toLocaleString("zh-CN") }

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

  if (detailQuery.isLoading) {
    return <div className="flex items-center justify-center p-12 text-muted-foreground">加载中…</div>
  }

  if (detailQuery.isError) {
    return <div className="p-6"><Alert variant="destructive"><AlertDescription>{toUserMessage(detailQuery.error)}</AlertDescription></Alert></div>
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
    if (!activityId) {
      return
    }
    createItemMutation.mutate(itemForm, { onSuccess: () => setItemFormOpen(false) })
  }

  const orders = ordersQuery.data?.list ?? []
  const ordersTotal = ordersQuery.data?.total ?? 0
  const ordersTotalPages = Math.ceil(ordersTotal / 20)

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => navigate("/seckill")}><ArrowLeft className="size-4" /></Button>
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{activity.title}</h1>
          <p className="text-sm text-muted-foreground">{activity.description}</p>
        </div>
        <Badge className="ml-auto" variant={activity.status === 2 ? "default" : "secondary"}>
          {activity.status === 1 ? "草稿" : activity.status === 2 ? "已发布" : activity.status === 3 ? "已下线" : "未知"}
        </Badge>
      </div>

      <div className="grid grid-cols-2 gap-4 text-sm">
        <div><span className="text-muted-foreground">开始时间：</span>{formatUnix(activity.start_at_unix)}</div>
        <div><span className="text-muted-foreground">结束时间：</span>{formatUnix(activity.end_at_unix)}</div>
      </div>

      <Separator />

      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-medium">活动商品</h2>
          <Button size="sm" onClick={openCreateItem}><Plus className="mr-1 size-4" />添加商品</Button>
        </div>
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>商品</TableHead>
                <TableHead>秒杀价</TableHead>
                <TableHead>预占库存</TableHead>
                <TableHead>可售</TableHead>
                <TableHead>已售</TableHead>
                <TableHead>限购</TableHead>
                <TableHead>状态</TableHead>
                <TableHead className="w-16">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(!activity.items || activity.items.length === 0) && (
                <TableRow><TableCell colSpan={8} className="text-center text-muted-foreground">暂无商品</TableCell></TableRow>
              )}
              {activity.items?.map((item) => (
                <TableRow key={item.item_id}>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      {item.snapshot_main_image && <img src={cdnUrl(item.snapshot_main_image)} alt="" className="size-8 rounded object-cover" />}
                      <span className="text-sm">{item.snapshot_name}</span>
                    </div>
                  </TableCell>
                  <TableCell>{formatCent(item.seckill_price_cent)}</TableCell>
                  <TableCell>{item.reserved_stock_total}</TableCell>
                  <TableCell>{item.available_stock}</TableCell>
                  <TableCell>{item.sold_stock}</TableCell>
                  <TableCell className="text-xs">{LIMIT_MODE[item.user_limit_mode] ?? item.user_limit_mode} / {item.user_limit_qty}件</TableCell>
                  <TableCell><Badge variant={item.status === 1 ? "default" : "secondary"}>{ITEM_STATUS[item.status] ?? item.status}</Badge></TableCell>
                  <TableCell>
                    <ItemRowActions activityId={activityId!} item={item}
                      onEdit={() => openEditItem(item)} onDelete={() => setDeleteItem(item)} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>

      <Separator />

      <div className="space-y-3">
        <h2 className="text-lg font-medium">活动订单</h2>
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>订单号</TableHead>
                <TableHead>用户</TableHead>
                <TableHead>数量</TableHead>
                <TableHead>订单状态</TableHead>
                <TableHead>支付</TableHead>
                <TableHead>关闭原因</TableHead>
                <TableHead>时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {orders.length === 0 && (
                <TableRow><TableCell colSpan={7} className="text-center text-muted-foreground">{ordersQuery.isLoading ? "加载中…" : "暂无订单"}</TableCell></TableRow>
              )}
              {orders.map((o) => {
                const paymentStatus = Number(o.payment_status ?? 0)
                return (
                  <TableRow key={o.link_id}>
                    <TableCell className="font-mono text-xs">{o.order_no}</TableCell>
                    <TableCell>{o.user_id}</TableCell>
                    <TableCell>{o.quantity}</TableCell>
                    <TableCell><Badge variant="outline">{ORDER_STATUS_LABELS[o.order_status] ?? o.order_status}</Badge></TableCell>
                    <TableCell><Badge variant="secondary">{PAYMENT_STATUS_LABELS[paymentStatus] ?? paymentStatus}</Badge></TableCell>
                    <TableCell className="text-xs text-muted-foreground">{o.close_reason || "-"}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{formatUnix(o.created_at_unix)}</TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">共 {ordersTotal} 条</span>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" disabled={orderPage <= 1} onClick={() => setOrderPage(orderPage - 1)}>上一页</Button>
            <span className="flex items-center px-2">{orderPage} / {ordersTotalPages || 1}</span>
            <Button size="sm" variant="outline" disabled={orderPage >= ordersTotalPages} onClick={() => setOrderPage(orderPage + 1)}>下一页</Button>
          </div>
        </div>
      </div>

      <Dialog open={itemFormOpen} onOpenChange={setItemFormOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader><DialogTitle>{editItem ? "编辑活动商品" : "添加活动商品"}</DialogTitle></DialogHeader>
          {editItem && activityId ? (
            <ItemEditForm activityId={activityId} item={editItem} formData={itemForm} setFormData={setItemForm} onClose={() => setItemFormOpen(false)} />
          ) : (
            <form onSubmit={handleCreateItem} className="space-y-4">
              <ItemFormFields formData={itemForm} setFormData={setItemForm} isCreate />
              {createItemMutation.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(createItemMutation.error)}</AlertDescription></Alert>}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setItemFormOpen(false)}>取消</Button>
                <Button type="submit" disabled={createItemMutation.isPending}>{createItemMutation.isPending ? "保存中…" : "保存"}</Button>
              </div>
            </form>
          )}
        </DialogContent>
      </Dialog>

      {deleteItem && activityId && <DeleteItemDialog activityId={activityId} item={deleteItem} onClose={() => setDeleteItem(null)} />}
    </div>
  )
}
