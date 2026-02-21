import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Zap } from "lucide-react"

import Counter from "@/components/Counter"
import { useSeckillPurchaseMutation } from "@/hooks/user/use-seckill-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { cdnUrl } from "@/lib/cdn"
import { formatCent } from "@/lib/format"
import { toast } from "sonner"

function discountText(origin: number, seckill: number) {
  const pct = Math.round((1 - seckill / origin) * 100)
  return `${pct}% off`
}

export function SeckillItemCard({
  activityId,
  item,
  active,
}: {
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

  const canBuy = active && item.in_stock && !purchaseMutation.isPending
  const priceYuan = item.seckill_price_cent / 100

  // mount 时从 0 滚到真实价格，触发 Counter 动画
  const [displayPrice, setDisplayPrice] = useState(0)
  useEffect(() => {
    const t = setTimeout(() => setDisplayPrice(priceYuan), 50)
    return () => clearTimeout(t)
  }, [priceYuan])

  const handlePurchase = (e: React.MouseEvent) => {
    e.stopPropagation()
    purchaseMutation.mutate(
      { activity_item_id: item.item_id, quantity: 1 },
      {
        onSuccess: (data) => navigate(`/orders/${data.order_id}`),
        onError: (err) => toast.error(toUserMessage(err))
      }
    )
  }

  return (
    <div className="group flex flex-col bg-card rounded-2xl overflow-hidden border border-border/40 shadow hover:shadow-lg hover:-translate-y-1 transition-all duration-200 cursor-pointer">
      {/* 图片区 */}
      <div className="relative aspect-4/3 bg-muted overflow-hidden">
        {item.snapshot_main_image ? (
          <img
            src={cdnUrl(item.snapshot_main_image)}
            alt={item.snapshot_name}
            className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center text-xs text-muted-foreground">
            暂无图片
          </div>
        )}

        {/* 折扣标签 */}
        {item.origin_price_cent > item.seckill_price_cent && (
          <div className="absolute top-2 left-2 bg-red-500 text-white text-[10px] font-bold px-1.5 py-0.5 rounded">
            {discountText(item.origin_price_cent, item.seckill_price_cent)}
          </div>
        )}

        {/* 售罄遮罩 */}
        {!item.in_stock && (
          <div className="absolute inset-0 bg-black/40 flex items-center justify-center">
            <span className="text-white text-sm font-medium tracking-wide">已售罄</span>
          </div>
        )}
      </div>

      {/* 信息区 */}
      <div className="flex flex-col gap-1 p-2">
        {/* 商品名 */}
        <p className="text-xs font-bold leading-snug line-clamp-1">
          {item.snapshot_name}
        </p>

        {/* 价格区 */}
        <div className="space-y-0.5">
          {/* 秒杀价 + 原价同行 */}
          <div className="flex items-end gap-2 flex-wrap">
            <div className="flex items-center gap-1">
              <span className="text-xs font-semibold text-red-500">仅需</span>
              <span className="text-red-500 font-bold text-lg leading-none">¥</span>
              <Counter
                value={displayPrice}
                places={[
                  ...(priceYuan >= 1000 ? [1000] : []),
                  ...(priceYuan >= 100 ? [100] : []),
                  ...(priceYuan >= 10 ? [10] : []),
                  1, ".", 10, 1,
                ] as (number | ".")[]}
                fontSize={16}
                padding={3}
                gap={1}
                horizontalPadding={0}
                textColor="rgb(239 68 68)"
                fontWeight={700}
                gradientFrom="transparent"
                gradientTo="transparent"
              />
            </div>
            <span className="text-[11px] text-muted-foreground line-through">
              {formatCent(item.origin_price_cent)}
            </span>
          </div>
        </div>

        {/* 抢购按钮 */}
        <button
          onClick={handlePurchase}
          disabled={!canBuy}
          className={[
            "w-full flex items-center justify-center gap-1 h-6 rounded-md text-[11px] font-semibold transition-colors mt-0.5",
            canBuy
              ? "bg-red-500 hover:bg-red-600 active:bg-red-700 text-white"
              : "bg-muted text-muted-foreground cursor-not-allowed",
          ].join(" ")}
        >
          {!active ? (
            "未开放"
          ) : !item.in_stock ? (
            "已售罄"
          ) : purchaseMutation.isPending ? (
            "抢购中…"
          ) : (
            <>
              <Zap className="size-3" />
              立即抢购
            </>
          )}
        </button>
      </div>
    </div>
  )
}
