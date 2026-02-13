import { useParams, useNavigate } from "react-router-dom"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { ArrowLeft, ShoppingCart } from "lucide-react"

import { useProductDetailQuery } from "@/hooks/user/use-product-hooks"
import { useCreateOrderMutation } from "@/hooks/user/use-order-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

function formatCent(cent: number) { return `¥${(cent / 100).toFixed(2)}` }

export function ProductDetailPage() {
  const { productId } = useParams<{ productId: string }>()
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()

  const detailQuery = useProductDetailQuery(productId)
  const createOrderMutation = useCreateOrderMutation()
  const product = detailQuery.data?.product

  if (detailQuery.isLoading) {
    return <div className="flex items-center justify-center py-12 text-muted-foreground">加载中…</div>
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
    <div className="space-y-4">
      <Button variant="ghost" size="sm" onClick={() => navigate("/products")}>
        <ArrowLeft className="mr-1 size-4" />返回商品列表
      </Button>

      <Card>
        {product.main_image && (
          <div className="aspect-video w-full bg-muted">
            <img src={product.main_image} alt={product.name} className="size-full object-contain" />
          </div>
        )}
        <CardHeader>
          <div className="flex items-start justify-between">
            <CardTitle className="text-lg">{product.name}</CardTitle>
            <span className={`text-xs font-medium ${product.in_stock ? "text-green-600" : "text-destructive"}`}>
              {product.in_stock ? "有货" : "缺货"}
            </span>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-baseline gap-2">
            <span className="text-2xl font-bold text-primary">{formatCent(product.price_cent)}</span>
            <span className={`text-sm ${product.in_stock ? "text-green-600" : "text-muted-foreground"}`}>
              {product.in_stock ? "有货" : "缺货"}
            </span>
          </div>

          <Separator />

          <div>
            <p className="mb-1 text-sm font-medium">商品描述</p>
            <p className="text-sm text-muted-foreground whitespace-pre-wrap">{product.description || "暂无描述"}</p>
          </div>

          {createOrderMutation.isError && (
            <Alert variant="destructive"><AlertDescription>{toUserMessage(createOrderMutation.error)}</AlertDescription></Alert>
          )}

          <Button className="w-full" disabled={!product.in_stock || createOrderMutation.isPending}
            onClick={handleBuy}>
            <ShoppingCart className="mr-2 size-4" />
            {createOrderMutation.isPending ? "下单中…" : !product.in_stock ? "已售罄" : "立即购买"}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
