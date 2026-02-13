import { useNavigate } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Zap } from "lucide-react"

import { useSeckillPurchaseMutation } from "@/hooks/user/use-seckill-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

function formatCent(cent: number) { return `¥${(cent / 100).toFixed(2)}` }

export function SeckillItemCard({ activityId, item, active }: {
  activityId: string
  item: {
    item_id: string | number
    product_id: string | number
    sku_code: string
    snapshot_name: string
    snapshot_main_image: string
    origin_price_cent: number
    seckill_price_cent: number
    in_stock: boolean
    max_qty_per_order: number
  }
  active: boolean
}) {
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()
  const purchaseMutation = useSeckillPurchaseMutation(activityId)

  const handlePurchase = () => {
    purchaseMutation.mutate(
      { activity_item_id: item.item_id, quantity: 1 },
      { onSuccess: (data) => navigate(`/orders/${data.order_id}`) }
    )
  }

  const canBuy = active && item.in_stock

  return (
    <Card>
      <CardContent className="flex items-center gap-4 p-4">
        {item.snapshot_main_image && (
          <img src={item.snapshot_main_image} alt={item.snapshot_name} className="size-20 rounded object-cover" />
        )}
        {!item.snapshot_main_image && (
          <div className="flex size-20 items-center justify-center rounded bg-muted text-xs text-muted-foreground">无图</div>
        )}
        <div className="flex-1 space-y-1">
          <p className="font-medium">{item.snapshot_name}</p>
          <p className="text-xs text-muted-foreground">SKU: {item.sku_code}</p>
          <div className="flex items-baseline gap-2">
            <span className="text-lg font-bold text-destructive">{formatCent(item.seckill_price_cent)}</span>
            <span className="text-xs text-muted-foreground line-through">{formatCent(item.origin_price_cent)}</span>
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span>{item.in_stock ? "有货" : "缺货"}</span>
            <span>限购 {item.max_qty_per_order} 件/单</span>
          </div>
        </div>
        <div className="flex flex-col items-end gap-2">
          <Button size="sm" disabled={!canBuy || purchaseMutation.isPending} onClick={handlePurchase}>
            <Zap className="mr-1 size-3" />
            {purchaseMutation.isPending ? "抢购中…" : !active ? "未开放" : !item.in_stock ? "已抢光" : "立即抢购"}
          </Button>
          {purchaseMutation.isError && (
            <p className="text-xs text-destructive max-w-32 text-right">{toUserMessage(purchaseMutation.error)}</p>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
