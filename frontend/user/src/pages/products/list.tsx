import { useState } from "react"
import { useNavigate } from "react-router-dom"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Search, X, Package, ChevronLeft, ChevronRight } from "lucide-react"

import { useProductsQuery } from "@/hooks/user/use-product-hooks"
import { cdnUrl } from "@/lib/cdn"
import { useApiError } from "@/hooks/common/use-api-error"
import { formatCent } from "@/lib/format"

const PAGE_SIZE = 12

export function ProductListPage() {
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState("")
  const [searchKeyword, setSearchKeyword] = useState("")

  const listQuery = useProductsQuery({
    page, page_size: PAGE_SIZE,
    keyword: searchKeyword || undefined,
  })

  const items = listQuery.data?.list ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  const doSearch = () => { setSearchKeyword(keyword); setPage(1) }
  const clearSearch = () => { setKeyword(""); setSearchKeyword(""); setPage(1) }

  return (
    <div className="space-y-5">
      {/* 页头 */}
      <div className="space-y-1">
        <h1 className="text-xl font-bold tracking-tight">全部商品</h1>
        <p className="text-sm text-muted-foreground">
          {total > 0 ? `共 ${total} 件商品` : "发现好物，从这里开始"}
        </p>
      </div>

      {/* 搜索栏 */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground pointer-events-none" />
          <Input
            placeholder="搜索商品名称…"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            className="pl-9 pr-9 h-10 rounded-full bg-muted/50 border-transparent focus:bg-background focus:border-border transition-colors"
            onKeyDown={(e) => { if (e.key === "Enter") doSearch() }}
          />
          {keyword && (
            <button onClick={clearSearch} className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground">
              <X className="size-4" />
            </button>
          )}
        </div>
        <Button size="sm" className="rounded-full px-5 h-10" onClick={doSearch}>搜索</Button>
      </div>

      {/* 搜索标签 */}
      {searchKeyword && (
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>搜索：</span>
          <span className="inline-flex items-center gap-1 rounded-full bg-primary/10 text-primary px-2.5 py-0.5 font-medium">
            {searchKeyword}
            <button onClick={clearSearch}><X className="size-3" /></button>
          </span>
        </div>
      )}

      {/* 错误提示 */}
      {listQuery.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(listQuery.error)}</AlertDescription></Alert>}

      {/* 骨架屏加载 */}
      {listQuery.isLoading && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="space-y-2">
              <Skeleton className="aspect-square rounded-xl" />
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-1/3" />
            </div>
          ))}
        </div>
      )}

      {/* 商品网格 */}
      {!listQuery.isLoading && items.length > 0 && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
          {items.map((p) => (
            <button
              key={String(p.product_id)}
              onClick={() => navigate(`/products/${p.product_id}`)}
              className="group text-left rounded-xl overflow-hidden border bg-card transition-all duration-200 hover:shadow-lg hover:-translate-y-0.5 active:scale-[0.98]"
            >
              {/* 图片区域 */}
              <div className="relative aspect-square bg-muted overflow-hidden">
                {p.main_image ? (
                  <img
                    src={cdnUrl(p.main_image)}
                    alt={p.name}
                    className="size-full object-cover transition-transform duration-300 group-hover:scale-105"
                    loading="lazy"
                  />
                ) : (
                  <div className="flex size-full items-center justify-center">
                    <Package className="size-10 text-muted-foreground/30" />
                  </div>
                )}
                {/* 缺货遮罩 */}
                {!p.in_stock && (
                  <div className="absolute inset-0 flex items-center justify-center bg-black/40 backdrop-blur-[2px]">
                    <span className="rounded-full bg-black/60 px-3 py-1 text-xs font-medium text-white">已售罄</span>
                  </div>
                )}
              </div>

              {/* 信息区域 */}
              <div className="p-3 space-y-1.5">
                <p className="text-sm font-medium leading-snug line-clamp-2 group-hover:text-primary transition-colors">
                  {p.name}
                </p>
                <div className="flex items-center justify-between">
                  <span className="text-base font-bold text-red-500">{formatCent(p.price_cent)}</span>
                  {p.in_stock ? (
                    <span className="text-[10px] font-medium text-emerald-600 bg-emerald-50 dark:bg-emerald-950/40 px-1.5 py-0.5 rounded">有货</span>
                  ) : (
                    <span className="text-[10px] font-medium text-muted-foreground bg-muted px-1.5 py-0.5 rounded">售罄</span>
                  )}
                </div>
              </div>
            </button>
          ))}
        </div>
      )}

      {/* 空状态 */}
      {!listQuery.isLoading && items.length === 0 && (
        <div className="py-16 text-center space-y-3">
          <Package className="size-12 text-muted-foreground/20 mx-auto" />
          <p className="text-sm text-muted-foreground">
            {searchKeyword ? `未找到"${searchKeyword}"相关商品` : "暂无商品"}
          </p>
          {searchKeyword && (
            <Button variant="outline" size="sm" className="rounded-full" onClick={clearSearch}>
              清除搜索
            </Button>
          )}
        </div>
      )}

      {/* 分页 */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-3 pt-2">
          <Button
            size="icon"
            variant="outline"
            className="size-8 rounded-full"
            disabled={page <= 1}
            onClick={() => setPage(page - 1)}
          >
            <ChevronLeft className="size-4" />
          </Button>
          <span className="text-sm text-muted-foreground tabular-nums">
            {page} / {totalPages}
          </span>
          <Button
            size="icon"
            variant="outline"
            className="size-8 rounded-full"
            disabled={page >= totalPages}
            onClick={() => setPage(page + 1)}
          >
            <ChevronRight className="size-4" />
          </Button>
        </div>
      )}
    </div>
  )
}
