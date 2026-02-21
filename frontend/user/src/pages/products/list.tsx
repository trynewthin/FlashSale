import { useState, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { motion } from "motion/react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { EmptyState } from "@/components/ui/empty-state"
import { Search, X, Package, ChevronLeft, ChevronRight, ShoppingBag } from "lucide-react"

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

  // 错误处理
  useEffect(() => {
    if (listQuery.isError) {
      toast.error(toUserMessage(listQuery.error))
    }
  }, [listQuery.isError, listQuery.error, toUserMessage])

  const items = listQuery.data?.list ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  const doSearch = () => { setSearchKeyword(keyword); setPage(1) }
  const clearSearch = () => { setKeyword(""); setSearchKeyword(""); setPage(1) }

  return (
    <div className="space-y-6 pb-12">
      {/* 沉浸式页头与搜索栏 */}
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="relative overflow-hidden rounded-[2rem] bg-card border border-border/50 shadow-xl shadow-black/5 p-6 sm:p-8"
      >
        <div className="absolute top-0 right-0 w-64 h-64 bg-linear-to-bl from-blue-500/10 to-transparent rounded-full blur-3xl pointer-events-none -mr-20 -mt-20" />

        <div className="relative z-10 flex flex-col md:flex-row md:items-end justify-between gap-6">
          <div className="space-y-2">
            <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-blue-50 dark:bg-blue-500/10 border border-blue-100 dark:border-blue-500/20 text-blue-600 dark:text-blue-400 text-xs font-bold tracking-wider uppercase mb-2">
              <ShoppingBag className="size-3.5" />
              All Products
            </div>
            <h1 className="text-3xl sm:text-4xl font-extrabold tracking-tight">全部商品</h1>
            <p className="text-sm sm:text-base text-muted-foreground">
              {total > 0 ? `发现 ${total} 件精选好物` : "汇聚全网硬通货"}
            </p>
          </div>

          {/* 现代化搜索框 */}
          <div className="w-full md:max-w-md flex flex-col gap-2">
            <div className="relative flex-1 group">
              <Search className="absolute left-4 top-1/2 size-4.5 -translate-y-1/2 text-muted-foreground group-focus-within:text-foreground transition-colors pointer-events-none" />
              <Input
                placeholder="想找点什么..."
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                className="pl-11 pr-24 h-12 rounded-full bg-muted/30 border-2 border-transparent focus-visible:border-foreground/20 focus-visible:ring-0 focus-visible:bg-background transition-all text-base shadow-inner"
                onKeyDown={(e) => { if (e.key === "Enter") doSearch() }}
              />
              {keyword && (
                <button onClick={clearSearch} className="absolute right-[88px] top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground h-8 w-8 flex items-center justify-center rounded-full hover:bg-muted transition-colors">
                  <X className="size-4" />
                </button>
              )}
              <Button
                size="sm"
                className="absolute right-1.5 top-1.5 bottom-1.5 h-auto rounded-full px-5 font-semibold shadow-md active:scale-95 transition-transform"
                onClick={doSearch}
              >
                搜索
              </Button>
            </div>

            {/* 搜索状态标签 */}
            {searchKeyword && (
              <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: "auto" }} className="flex items-center gap-2 text-xs">
                <span className="text-muted-foreground ml-2">正在搜索:</span>
                <span className="inline-flex items-center gap-1 rounded-full bg-primary text-primary-foreground px-2.5 py-0.5 font-medium shadow-sm">
                  {searchKeyword}
                  <button onClick={clearSearch} className="hover:bg-primary-foreground/20 rounded-full p-0.5"><X className="size-3" /></button>
                </span>
              </motion.div>
            )}
          </div>
        </div>
      </motion.div>

      {/* 骨架屏加载 */}
      {listQuery.isLoading && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="space-y-3 p-3 rounded-2xl border border-border/50 bg-card">
              <Skeleton className="aspect-square rounded-xl w-full" />
              <div className="space-y-2 pt-2">
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-4 w-1/3" />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 商品网格 */}
      {!listQuery.isLoading && items.length > 0 && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-4">
          {items.map((p, i) => (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: i * 0.05, duration: 0.4, ease: "easeOut" }}
              key={String(p.product_id)}
              onClick={() => navigate(`/products/${p.product_id}`)}
              className="group cursor-pointer flex flex-col rounded-2xl overflow-hidden border border-border/60 bg-card transition-all duration-300 hover:shadow-xl hover:shadow-black/5 hover:-translate-y-1 hover:border-border"
            >
              {/* 图片区域 */}
              <div className="relative aspect-square bg-muted/30 overflow-hidden p-2">
                <div className="relative size-full rounded-xl overflow-hidden bg-white dark:bg-zinc-900 border border-border/50">
                  {p.main_image ? (
                    <img
                      src={cdnUrl(p.main_image)}
                      alt={p.name}
                      className="absolute inset-0 size-full object-cover transition-transform duration-500 group-hover:scale-110"
                      loading="lazy"
                    />
                  ) : (
                    <div className="absolute inset-0 flex items-center justify-center">
                      <Package className="size-10 text-muted-foreground/30" />
                    </div>
                  )}
                  {/* 渐变暗角 */}
                  <div className="absolute inset-0 bg-linear-to-t from-black/20 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />

                  {/* 缺货遮罩 */}
                  {!p.in_stock && (
                    <div className="absolute inset-0 flex items-center justify-center bg-background/60 backdrop-blur-[2px]">
                      <span className="rounded-full bg-foreground text-background px-4 py-1.5 text-xs font-bold tracking-widest uppercase shadow-xl">已售罄</span>
                    </div>
                  )}
                </div>
              </div>

              {/* 信息区域 */}
              <div className="p-4 flex flex-col flex-1 gap-2">
                <p className="text-sm font-semibold leading-relaxed line-clamp-2 text-foreground group-hover:text-primary transition-colors flex-1">
                  {p.name}
                </p>
                <div className="flex items-end justify-between mt-auto">
                  <span className="text-lg font-black tracking-tight text-red-500 drop-shadow-sm">{formatCent(p.price_cent)}</span>
                  {p.in_stock ? (
                    <span className="text-[10px] font-bold tracking-wider uppercase text-emerald-600 bg-emerald-100 dark:bg-emerald-500/20 px-2 py-1 rounded-full">有货</span>
                  ) : (
                    <span className="text-[10px] font-bold tracking-wider uppercase text-muted-foreground bg-muted px-2 py-1 rounded-full">空仓</span>
                  )}
                </div>
              </div>
            </motion.div>
          ))}
        </div>
      )}

      {/* 空状态 */}
      {!listQuery.isLoading && items.length === 0 && (
        <div className="py-12">
          <EmptyState
            icon={Package}
            title={searchKeyword ? "未能找到该商品" : "商品库空空如也"}
            description={searchKeyword ? `已为您搜索 "${searchKeyword}"，好像没有上架喔` : "等待运营人员上架第一件爆品吧"}
            actionLabel={searchKeyword ? "清除搜索词" : "刷新看看"}
            onAction={searchKeyword ? clearSearch : () => listQuery.refetch()}
          />
        </div>
      )}

      {/* 现代化分页 */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-4 pt-8">
          <Button
            size="icon"
            variant="outline"
            className="size-10 rounded-full border-2 hover:bg-foreground hover:text-background transition-all shadow-sm"
            disabled={page <= 1}
            onClick={() => setPage(page - 1)}
          >
            <ChevronLeft className="size-5" />
          </Button>

          <div className="flex bg-muted/50 rounded-full px-4 h-10 items-center border border-border/50 shadow-inner">
            <span className="text-sm font-semibold text-foreground tabular-nums">
              {page} <span className="text-muted-foreground px-1 font-normal">/</span> {totalPages}
            </span>
          </div>

          <Button
            size="icon"
            variant="outline"
            className="size-10 rounded-full border-2 hover:bg-foreground hover:text-background transition-all shadow-sm"
            disabled={page >= totalPages}
            onClick={() => setPage(page + 1)}
          >
            <ChevronRight className="size-5" />
          </Button>
        </div>
      )}
    </div>
  )
}
