import { useState, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { motion } from "motion/react"
import { toast } from "sonner"
import { Zap, ChevronRight, ChevronLeft, Timer, ArrowRight, PackageOpen } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { EmptyState } from "@/components/ui/empty-state"
import { SeckillItemCard } from "@/components/seckill/seckill-item-card"
import { SeckillCountdown } from "@/components/seckill/seckill-countdown"

import { useSeckillActivitiesQuery } from "@/hooks/user/use-seckill-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { type ActivityPublic } from "@/api/modules/seckill"

// ─── 工具 ─────────────────────────────────────────────────────────────────────

function getStatus(startUnix: number, endUnix: number) {
  const now = Math.floor(Date.now() / 1000)
  if (now < startUnix) return { active: false, upcoming: true }
  if (now > endUnix) return { active: false, upcoming: false }
  return { active: true, upcoming: false }
}

// ─── 活动区块 ──────────────────────────────────────────────────────────────────

function ActivitySection({ activity, index }: { activity: ActivityPublic, index: number }) {
  const navigate = useNavigate()
  const { active, upcoming } = getStatus(activity.start_at_unix, activity.end_at_unix)
  const items = activity.items ?? []
  const MAX_SHOW = 6
  const visibleItems = items.slice(0, MAX_SHOW)
  const hasMore = items.length > MAX_SHOW

  return (
    <motion.section
      initial={{ opacity: 0, scale: 0.98, y: 20 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      transition={{ delay: index * 0.1, duration: 0.5, ease: "easeOut" }}
      className="relative overflow-hidden rounded-[2rem] bg-card border border-border/50 shadow-xl shadow-black/5 p-5 sm:p-8 group"
    >
      {/* 状态光晕 (根据活动状态变化) */}
      <div className={`absolute top-0 right-0 w-64 h-64 rounded-full blur-3xl pointer-events-none -mr-20 -mt-20 transition-colors duration-1000 ${active ? "bg-red-500/10 dark:bg-red-500/5" : upcoming ? "bg-amber-500/10 dark:bg-amber-500/5" : "bg-muted"
        }`} />

      <div className="relative z-10 space-y-6">
        {/* 倒计时与状态行 */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-muted/30 p-4 sm:px-6 sm:py-4 rounded-2xl border border-border/40">
          <div className="flex items-center gap-3 w-full sm:w-auto">
            <div className={`size-10 rounded-full flex items-center justify-center shrink-0 shadow-inner ${active ? "bg-linear-to-tr from-red-500 to-orange-500 text-white animate-pulse" :
                upcoming ? "bg-amber-100 dark:bg-amber-900/40 text-amber-600 dark:text-amber-500" :
                  "bg-muted-foreground/20 text-muted-foreground"
              }`}>
              {active ? <Zap className="size-5" /> : upcoming ? <Timer className="size-5" /> : <PackageOpen className="size-5" />}
            </div>
            <div className="flex flex-col">
              <span className="font-bold text-base sm:text-lg tracking-tight">
                {active ? "正在秒杀中" : upcoming ? "距离开始还有" : "活动已结束"}
              </span>
              <span className="text-xs text-muted-foreground font-medium">#{activity.activity_id} 场次</span>
            </div>
          </div>

          {/* 倒计时巨幕 */}
          <div className="w-full sm:w-auto flex justify-center sm:justify-end bg-background/50 px-4 py-2 rounded-xl border border-border/50 shadow-inner">
            <SeckillCountdown endUnix={active ? activity.end_at_unix : activity.start_at_unix} fontSize={28} />
          </div>
        </div>

        {/* 商品网格 */}
        {items.length === 0 ? (
          <div className="py-10 text-center flex flex-col items-center">
            <PackageOpen className="size-10 text-muted-foreground/20 mb-3" />
            <p className="text-sm text-muted-foreground font-medium">本场暂无秒杀商品</p>
          </div>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3 sm:gap-4">
            {visibleItems.map((item) => (
              <SeckillItemCard
                key={item.item_id}
                activityId={String(activity.activity_id)}
                item={{
                  ...item,
                  item_id: String(item.item_id),
                  product_id: String(item.product_id),
                }}
                active={active}
              />
            ))}
          </div>
        )}

        {/* 底部操作区 */}
        <div className="flex items-center justify-between pt-2 border-t border-border/30">
          <span className="text-xs font-semibold text-muted-foreground bg-muted/50 px-3 py-1.5 rounded-full">
            共 {items.length} 款商品
          </span>

          {hasMore ? (
            <Button
              variant="ghost"
              className="rounded-full gap-1.5 text-primary hover:text-primary hover:bg-primary/10 group/btn"
              onClick={() => navigate(`/seckill/${activity.activity_id}`)}
            >
              查看全部
              <ArrowRight className="size-4 group-hover/btn:translate-x-1 transition-transform" />
            </Button>
          ) : (
            items.length > 0 && (
              <Button
                variant="ghost"
                className="rounded-full gap-1.5 text-muted-foreground hover:text-foreground group/btn"
                onClick={() => navigate(`/seckill/${activity.activity_id}`)}
              >
                进入专场
                <ChevronRight className="size-4 group-hover/btn:translate-x-1 transition-transform" />
              </Button>
            )
          )}
        </div>
      </div>
    </motion.section>
  )
}

// ─── 秒杀列表页 ───────────────────────────────────────────────────────────────

export function SeckillListPage() {
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()
  const [page, setPage] = useState(1)

  const listQuery = useSeckillActivitiesQuery({ page, page_size: 10 })
  const items = listQuery.data?.list ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / 10)

  // 错误处理
  useEffect(() => {
    if (listQuery.isError) {
      toast.error(toUserMessage(listQuery.error))
    }
  }, [listQuery.isError, listQuery.error, toUserMessage])

  const now = Math.floor(Date.now() / 1000)
  const sortedActivities = [...items].sort((a, b) => {
    const aActive = a.start_at_unix <= now && a.end_at_unix > now
    const bActive = b.start_at_unix <= now && b.end_at_unix > now
    if (aActive !== bActive) return aActive ? -1 : 1
    return a.start_at_unix - b.start_at_unix
  })

  // 翻页滚动到顶部
  useEffect(() => {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }, [page])

  return (
    <div className="space-y-8 pb-12">
      {/* 沉浸式页头 */}
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="relative overflow-hidden rounded-[2rem] bg-card border border-border/50 shadow-xl shadow-black/5 p-6 sm:p-10 flex flex-col items-center justify-center text-center"
      >
        <div className="absolute top-0 inset-x-0 h-64 bg-linear-to-b from-red-500/10 via-orange-500/5 to-transparent pointer-events-none" />
        <div className="absolute top-0 right-1/4 w-64 h-64 bg-red-500/10 blur-[100px] rounded-full pointer-events-none" />

        <div className="relative z-10 space-y-3 max-w-lg mx-auto">
          <div className="inline-flex justify-center items-center gap-1.5 px-4 py-1.5 rounded-full bg-red-50 dark:bg-red-500/10 border border-red-100 dark:border-red-500/20 text-red-600 dark:text-red-400 text-xs font-bold tracking-widest uppercase mb-2 shadow-sm">
            <Zap className="size-4 animate-pulse" />
            Flash Sale
          </div>
          <h1 className="text-3xl sm:text-5xl font-black tracking-tighter bg-clip-text text-transparent bg-linear-to-r from-foreground to-foreground/70">
            限时秒杀专场
          </h1>
          <p className="text-sm sm:text-base text-muted-foreground font-medium">
            每日精选爆款，全网底价，手慢无！抢到就是赚到。
          </p>
        </div>
      </motion.div>

      {/* 加载骨架 */}
      {listQuery.isLoading && (
        <div className="space-y-6">
          {[0, 1].map((i) => (
            <div key={i} className="rounded-[2rem] border border-border/50 bg-card p-5 sm:p-8 space-y-6 shadow-sm">
              <Skeleton className="h-20 w-full rounded-2xl" />
              <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-6 gap-4">
                {[...Array(6)].map((_, j) => (
                  <div key={j} className="space-y-3">
                    <Skeleton className="aspect-square rounded-xl" />
                    <Skeleton className="h-4 w-full" />
                    <Skeleton className="h-5 w-1/2" />
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 活动列表 */}
      {!listQuery.isLoading && items.length > 0 && (
        <div className="space-y-8">
          {sortedActivities.map((activity, index) => (
            <ActivitySection key={activity.activity_id} activity={activity} index={index} />
          ))}
        </div>
      )}

      {/* 空状态 */}
      {!listQuery.isLoading && items.length === 0 && (
        <div className="py-12">
          <EmptyState
            icon={Zap}
            title="今日秒杀已告一段落"
            description="下一波惊天特惠正在精心筹备中，去看看别的商品吧"
            actionLabel="逛逛全部商品"
            onAction={() => navigate("/products")}
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
