import { useState, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { motion } from "motion/react"
import { toast } from "sonner"
import { Search, PackageOpen, ChevronLeft, ChevronRight, X, Clock, ShoppingCart } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { EmptyState } from "@/components/ui/empty-state"
import { Skeleton } from "@/components/ui/skeleton"

import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/select"

import { useOrderListQuery } from "@/hooks/user/use-order-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { ORDER_STATUS_LABELS } from "@/features/order/status"

import { formatCent, formatUnix } from "@/lib/format"
import { cdnUrl } from "@/lib/cdn"

export function OrderListPage() {
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState("")
  const [orderNoInput, setOrderNoInput] = useState("")
  const [orderNoSearch, setOrderNoSearch] = useState("")

  const handleSearch = () => {
    setOrderNoSearch(orderNoInput.trim())
    setPage(1)
  }

  const clearSearch = () => {
    setOrderNoInput("")
    setOrderNoSearch("")
    setPage(1)
  }

  const listQuery = useOrderListQuery({
    page, page_size: 10,
    order_status: statusFilter ? Number(statusFilter) : undefined,
    order_no: orderNoSearch || undefined,
  })

  // 错误处理
  useEffect(() => {
    if (listQuery.isError) {
      toast.error(toUserMessage(listQuery.error))
    }
  }, [listQuery.isError, listQuery.error, toUserMessage])

  const items = listQuery.data?.list ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / 10)

  // 翻页滚动到顶部
  useEffect(() => {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }, [page])

  const getStatusColor = (status: number) => {
    // 1-待支付 2-已支付 3-已取消 4-已退款
    switch (status) {
      case 1: return "bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400 border-amber-200 dark:border-amber-500/30"
      case 2: return "bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-400 border-emerald-200 dark:border-emerald-500/30"
      case 3: return "bg-muted text-muted-foreground border-border"
      case 4: return "bg-blue-100 text-blue-700 dark:bg-blue-500/20 dark:text-blue-400 border-blue-200 dark:border-blue-500/30"
      default: return "bg-muted text-muted-foreground border-border"
    }
  }

  return (
    <div className="space-y-6 pb-12">
      {/* 沉浸式页头与搜索栏 */}
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="relative overflow-hidden rounded-[2rem] bg-card border border-border/50 shadow-xl shadow-black/5 p-6 sm:p-8"
      >
        <div className="absolute top-0 right-0 w-64 h-64 bg-linear-to-bl from-emerald-500/10 to-transparent rounded-full blur-3xl pointer-events-none -mr-20 -mt-20" />

        <div className="relative z-10 flex flex-col md:flex-row md:items-end justify-between gap-6">
          <div className="space-y-2">
            <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-100 dark:border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs font-bold tracking-wider uppercase mb-2">
              <ShoppingCart className="size-3.5" />
              My Orders
            </div>
            <h1 className="text-3xl sm:text-4xl font-extrabold tracking-tight">我的订单</h1>
            <p className="text-sm sm:text-base text-muted-foreground">
              {total > 0 ? `共找到 ${total} 笔交易记录` : "随时追踪你的所有订单状态"}
            </p>
          </div>

          {/* 现代化搜索与过滤 */}
          <div className="w-full md:max-w-xl flex flex-col gap-3">
            <div className="flex flex-col sm:flex-row sm:items-center gap-3">
              <div className="relative flex-1 group">
                <Search className="absolute left-4 top-1/2 size-4.5 -translate-y-1/2 text-muted-foreground group-focus-within:text-foreground transition-colors pointer-events-none" />
                <Input
                  placeholder="输入完整的订单号..."
                  value={orderNoInput}
                  onChange={(e) => setOrderNoInput(e.target.value)}
                  className="pl-11 pr-24 h-12 rounded-full bg-muted/30 border-2 border-transparent focus-visible:border-foreground/20 focus-visible:ring-0 focus-visible:bg-background transition-all text-sm shadow-inner"
                  onKeyDown={(e) => { if (e.key === "Enter") handleSearch() }}
                />
                {orderNoInput && (
                  <button onClick={clearSearch} className="absolute right-[88px] top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground h-8 w-8 flex items-center justify-center rounded-full hover:bg-muted transition-colors">
                    <X className="size-4" />
                  </button>
                )}
                <Button
                  size="sm"
                  className="absolute right-1.5 top-1.5 bottom-1.5 h-auto rounded-full px-5 font-semibold shadow-md active:scale-95 transition-transform"
                  onClick={handleSearch}
                >
                  搜索
                </Button>
              </div>

              <div className="shrink-0">
                <Select value={statusFilter || "all"} onValueChange={(v: string | null) => { setStatusFilter(!v || v === "all" ? "" : v); setPage(1) }}>
                  <SelectTrigger className="h-12 w-full sm:w-36 rounded-full border-2 border-transparent bg-muted/30 focus:ring-0 focus:border-foreground/20 shadow-inner px-4 font-medium transition-all">
                    <span className="flex flex-1 text-center items-center justify-center gap-2">
                      {statusFilter ? (ORDER_STATUS_LABELS[Number(statusFilter)] ?? statusFilter) : "全部状态"}
                    </span>
                  </SelectTrigger>
                  <SelectContent className="rounded-2xl shadow-xl">
                    <SelectItem value="all" className="rounded-xl my-0.5 cursor-pointer">全部状态</SelectItem>
                    {Object.entries(ORDER_STATUS_LABELS).map(([k, v]) => (
                      <SelectItem key={k} value={k} className="rounded-xl my-0.5 cursor-pointer">{v}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            {orderNoSearch && (
              <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: "auto" }} className="flex items-center gap-2 text-xs">
                <span className="text-muted-foreground ml-2">正在搜索订单:</span>
                <span className="inline-flex items-center gap-1 rounded-full bg-primary text-primary-foreground px-2.5 py-0.5 font-medium shadow-sm">
                  {orderNoSearch}
                  <button onClick={clearSearch} className="hover:bg-primary-foreground/20 rounded-full p-0.5"><X className="size-3" /></button>
                </span>
              </motion.div>
            )}
          </div>
        </div>
      </motion.div>

      {/* 骨架屏加载 */}
      {listQuery.isLoading && (
        <div className="space-y-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex flex-col sm:flex-row items-center gap-4 p-5 rounded-2xl border border-border/50 bg-card">
              <Skeleton className="size-20 rounded-xl shrink-0" />
              <div className="flex-1 space-y-2 w-full">
                <Skeleton className="h-5 w-3/4 max-w-[300px]" />
                <Skeleton className="h-4 w-1/2 max-w-[200px]" />
              </div>
              <div className="w-full sm:w-auto flex sm:flex-col items-center sm:items-end justify-between gap-2 shrink-0">
                <Skeleton className="h-6 w-16 rounded-full" />
                <Skeleton className="h-5 w-24" />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 订单列表 */}
      {!listQuery.isLoading && items.length > 0 && (
        <div className="space-y-4">
          {items.map((order, i) => (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: i * 0.05, duration: 0.4, ease: "easeOut" }}
              key={order.order_id}
              onClick={() => navigate(`/orders/${order.order_id}`)}
              className="group cursor-pointer rounded-2xl border border-border/60 bg-card transition-all duration-300 hover:shadow-xl hover:shadow-black/5 hover:-translate-y-1 hover:border-border overflow-hidden relative"
            >
              {/* 装饰性背景微光 (特定状态如待支付) */}
              {order.order_status === 1 && (
                <div className="absolute top-0 right-0 w-32 h-32 bg-amber-500/5 rounded-full blur-2xl pointer-events-none -mr-10 -mt-10" />
              )}
              {order.order_status === 2 && (
                <div className="absolute top-0 right-0 w-32 h-32 bg-emerald-500/5 rounded-full blur-2xl pointer-events-none -mr-10 -mt-10" />
              )}

              <div className="p-5 flex flex-col sm:flex-row gap-5 relative z-10">
                {/* 图片与基本信息 */}
                <div className="flex items-start gap-4 flex-1">
                  <div className="relative shrink-0 size-20 sm:size-24 rounded-xl bg-muted overflow-hidden border border-border/50">
                    {order.main_image ? (
                      <img
                        src={cdnUrl(order.main_image)}
                        alt={order.product_name}
                        className="size-full object-cover transition-transform duration-500 group-hover:scale-110"
                      />
                    ) : (
                      <div className="size-full flex items-center justify-center">
                        <PackageOpen className="size-8 text-muted-foreground/30" />
                      </div>
                    )}
                  </div>
                  <div className="flex flex-col h-full pt-1 pb-1">
                    <p className="text-sm sm:text-base font-bold leading-snug line-clamp-2 text-foreground group-hover:text-primary transition-colors">
                      {order.product_name}
                    </p>
                    <div className="mt-auto space-y-1">
                      <div className="flex items-center gap-1.5 text-xs text-muted-foreground font-mono bg-muted/50 w-fit px-2 py-0.5 rounded-md">
                        <span className="font-semibold text-[10px] tracking-widest uppercase opacity-70">ID</span>
                        {order.order_id}
                      </div>
                      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                        <Clock className="size-3 opacity-60" />
                        {formatUnix(order.created_at_unix)}
                      </div>
                    </div>
                  </div>
                </div>

                {/* 状态与金额 */}
                <div className="flex sm:flex-col items-center sm:items-end justify-between sm:justify-center border-t sm:border-t-0 sm:border-l border-border/30 pt-4 sm:pt-0 sm:pl-6 shrink-0 min-w-[120px]">
                  <Badge variant="outline" className={`mb-auto font-bold tracking-widest text-[10px] px-2.5 py-1 ${getStatusColor(order.order_status)}`}>
                    {ORDER_STATUS_LABELS[order.order_status] ?? order.order_status}
                  </Badge>
                  <div className="text-right mt-1 sm:mt-0">
                    <p className="text-xl font-black tracking-tight text-foreground group-hover:text-primary transition-colors">
                      {formatCent(order.total_amount_cent)}
                    </p>
                    <p className="text-xs font-semibold text-muted-foreground relative -top-0.5">
                      共 {order.quantity} 件商品
                    </p>
                  </div>
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
            icon={Search}
            title={orderNoSearch ? "未找到指定订单" : (statusFilter ? "该状态下暂无订单" : "您还没有任何订单")}
            description={orderNoSearch ? `没有找到订单号为 "${orderNoSearch}" 的记录` : (statusFilter ? `目前没有处在这个状态的订单。` : "快去商场看看，挑选中意的商品吧！")}
            actionLabel={orderNoSearch || statusFilter ? "清除所有筛选条件" : "去首页逛逛"}
            onAction={orderNoSearch || statusFilter ? clearSearch : () => navigate("/")}
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
