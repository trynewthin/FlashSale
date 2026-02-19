import { useState } from "react"
import { useNavigate } from "react-router-dom"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Search } from "lucide-react"

import { useProductsQuery } from "@/hooks/user/use-product-hooks"
import { cdnUrl } from "@/lib/cdn"
import { useApiError } from "@/hooks/common/use-api-error"

import { formatCent } from "@/lib/format"

export function ProductListPage() {
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState("")
  const [searchKeyword, setSearchKeyword] = useState("")

  const listQuery = useProductsQuery({
    page, page_size: 12,
    keyword: searchKeyword || undefined,
  })

  const items = listQuery.data?.list ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / 12)

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索商品…"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            className="pl-9"
            onKeyDown={(e) => { if (e.key === "Enter") { setSearchKeyword(keyword); setPage(1) } }}
          />
        </div>
        <Button size="sm" onClick={() => { setSearchKeyword(keyword); setPage(1) }}>搜索</Button>
      </div>

      {listQuery.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(listQuery.error)}</AlertDescription></Alert>}

      {listQuery.isLoading && <p className="text-center text-muted-foreground py-12">加载中…</p>}

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4">
        {items.map((p) => (
          <Card key={p.product_id} className="cursor-pointer overflow-hidden transition-shadow hover:shadow-md"
            onClick={() => navigate(`/products/${p.product_id}`)}>
            {p.main_image && (
              <div className="aspect-square bg-muted">
                <img src={cdnUrl(p.main_image)} alt={p.name} className="size-full object-cover" />
              </div>
            )}
            {!p.main_image && <div className="flex aspect-square items-center justify-center bg-muted text-muted-foreground text-xs">暂无图片</div>}
            <CardContent className="p-3 space-y-1">
              <p className="text-sm font-medium line-clamp-2">{p.name}</p>
              <div className="flex items-center justify-between">
                <span className="text-base font-semibold text-primary">{formatCent(p.price_cent)}</span>
                <span className={`text-xs ${p.in_stock ? "text-green-600" : "text-muted-foreground"}`}>
                  {p.in_stock ? "有货" : "缺货"}
                </span>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {items.length === 0 && !listQuery.isLoading && (
        <p className="text-center text-muted-foreground py-12">暂无商品</p>
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
