import { useParams, useNavigate } from "react-router-dom"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import {
  ArrowLeft, ShoppingCart, Package, CheckCircle, XCircle, Shield, Truck,
} from "lucide-react"

import { useProductDetailQuery } from "@/hooks/user/use-product-hooks"
import { useCreateOrderMutation } from "@/hooks/user/use-order-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { formatCent } from "@/lib/format"
import { cdnUrl } from "@/lib/cdn"

export function ProductDetailPage() {
  const { productId } = useParams<{ productId: string }>()
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()

  const detailQuery = useProductDetailQuery(productId)
  const createOrderMutation = useCreateOrderMutation()
  const product = detailQuery.data?.product

  // ── 加载骨架 ─────────────────────────────────────────────────
  if (detailQuery.isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-24" />
        <div className="flex flex-col md:flex-row gap-6">
          <Skeleton className="aspect-square w-full md:w-80 rounded-2xl shrink-0" />
          <div className="flex-1 space-y-3">
            <Skeleton className="h-6 w-3/4" />
            <Skeleton className="h-8 w-32" />
            <Skeleton className="h-px w-full" />
            <Skeleton className="h-20 w-full" />
            <Skeleton className="h-12 w-full rounded-xl" />
          </div>
        </div>
      </div>
    )
  }

  if (detailQuery.isError) {
    return <Alert variant="destructive"><AlertDescription>{toUserMessage(detailQuery.error)}</AlertDescription></Alert>
  }

  if (!product) return null

  const handleBuy = () => {
    createOrderMutation.mutate(
      { product_id: product.product_id, order_source: 1 },
      { onSuccess: (data) => navigate(`/orders/${data.order.order_id}`) }
    )
  }

  return (
    <div className="space-y-5">
      {/* 返回导航 */}
      <button
        onClick={() => navigate("/products")}
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="size-4" />
        商品列表
      </button>

      {/* 主体：小屏上下、大屏左右 */}
      <div className="flex flex-col md:flex-row gap-6">

        {/* ── 左侧：图片 + 名称 + 价格 ────────────────────────── */}
        <div className="space-y-4 md:w-80 shrink-0">
          {/* 主图 */}
          <div className="relative aspect-square w-full overflow-hidden rounded-2xl bg-muted">
            {product.main_image ? (
              <img
                src={cdnUrl(product.main_image)}
                alt={product.name}
                className="size-full object-cover"
              />
            ) : (
              <div className="flex size-full items-center justify-center">
                <Package className="size-16 text-muted-foreground/20" />
              </div>
            )}
            {/* 库存角标 */}
            <div className="absolute top-3 right-3">
              {product.in_stock ? (
                <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500 px-2.5 py-1 text-xs font-medium text-white shadow-sm">
                  <CheckCircle className="size-3" />有货
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 rounded-full bg-gray-800/70 backdrop-blur-sm px-2.5 py-1 text-xs font-medium text-white shadow-sm">
                  <XCircle className="size-3" />已售罄
                </span>
              )}
            </div>
          </div>

          {/* 名称 */}
          <h1 className="text-lg font-bold tracking-tight leading-snug">
            {product.name}
          </h1>

          {/* 价格卡片 */}
          <div className="flex items-baseline gap-2 rounded-xl bg-linear-to-r from-red-50 to-orange-50 dark:from-red-950/30 dark:to-orange-950/20 px-4 py-3">
            <span className="text-xs text-red-400 font-medium">价格</span>
            <span className="text-2xl font-extrabold text-red-500 tabular-nums tracking-tight">
              {formatCent(product.price_cent)}
            </span>
          </div>

          {/* SKU */}
          {product.sku_code && (
            <div className="text-xs text-muted-foreground">
              SKU：<code className="font-mono text-foreground/70">{product.sku_code}</code>
            </div>
          )}
        </div>

        {/* ── 右侧：描述 + 保障 + 购买 ───────────────────────── */}
        <div className="flex-1 flex flex-col space-y-5">
          {/* 服务保障 */}
          <div className="flex items-center gap-4 text-xs text-muted-foreground">
            <span className="inline-flex items-center gap-1">
              <Shield className="size-3 text-emerald-500" />正品保障
            </span>
            <span className="inline-flex items-center gap-1">
              <Truck className="size-3 text-blue-500" />极速发货
            </span>
            <span className="inline-flex items-center gap-1">
              <CheckCircle className="size-3 text-amber-500" />七天退换
            </span>
          </div>

          <Separator />

          {/* 商品描述 */}
          <div className="space-y-2 flex-1">
            <p className="text-sm font-semibold">商品详情</p>
            <p className="text-sm text-muted-foreground leading-relaxed whitespace-pre-wrap">
              {product.description || "暂无详细描述"}
            </p>
          </div>

          <Separator />

          {/* 下单错误 */}
          {createOrderMutation.isError && (
            <Alert variant="destructive"><AlertDescription>{toUserMessage(createOrderMutation.error)}</AlertDescription></Alert>
          )}

          {/* 购买区 */}
          <div className="flex items-center gap-4 pt-1">
            <div className="min-w-0">
              <span className="text-lg font-extrabold text-red-500 tabular-nums">{formatCent(product.price_cent)}</span>
            </div>
            <Button
              size="lg"
              className="flex-1 rounded-full text-base font-semibold shadow-lg shadow-primary/20 disabled:shadow-none transition-shadow"
              disabled={!product.in_stock || createOrderMutation.isPending}
              onClick={handleBuy}
            >
              <ShoppingCart className="mr-2 size-4" />
              {createOrderMutation.isPending
                ? "下单中…"
                : !product.in_stock
                  ? "已售罄"
                  : "立即购买"
              }
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
