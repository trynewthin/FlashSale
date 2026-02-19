import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Zap, ChevronRight } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { SeckillItemCard } from "@/components/seckill/seckill-item-card"
import { SeckillCountdown } from "@/components/seckill/seckill-countdown"
import { useSeckillActivitiesQuery } from "@/hooks/user/use-seckill-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { type ActivityPublic } from "@/api/modules/seckill"

// ─── 工具 ─────────────────────────────────────────────────────────────────────

function getStatus(startUnix: number, endUnix: number) {
  const now = Math.floor(Date.now() / 1000)
  if (now < startUnix) return { active: false }
  if (now > endUnix) return { active: false }
  return { active: true }
}

// ─── 活动区块（与 home 一致，但显示全部商品） ──────────────────────────────────

function ActivitySection({ activity }: { activity: ActivityPublic }) {
  const navigate = useNavigate()
  const { active } = getStatus(activity.start_at_unix, activity.end_at_unix)
  const items = activity.items ?? []
  const MAX_SHOW = 6
  const visibleItems = items.slice(0, MAX_SHOW)
  const hasMore = items.length > MAX_SHOW

  return (
    <section className="rounded-xl border border-border/60 p-4 space-y-3">
      {/* 倒计时行 */}
      <div className="flex items-center justify-center">
        <SeckillCountdown endUnix={activity.end_at_unix} />
      </div>

      {/* 商品网格 */}
      {items.length === 0 ? (
        <p className="text-sm text-muted-foreground py-4 text-center">暂无商品</p>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
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

      {/* 查看全部 / 商品数量 */}
      {hasMore ? (
        <div className="flex justify-center pt-1">
          <button
            onClick={() => navigate(`/seckill/${activity.activity_id}`)}
            className="flex items-center gap-1 text-sm font-semibold text-muted-foreground hover:text-foreground transition-colors"
          >
            查看全部 {items.length} 件商品
            <ChevronRight className="size-4" />
          </button>
        </div>
      ) : items.length > 0 ? (
        <p className="text-center text-xs text-muted-foreground pt-1">
          共 {items.length} 件商品
        </p>
      ) : null}
    </section>
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

  const now = Math.floor(Date.now() / 1000)
  const sortedActivities = [...items].sort((a, b) => {
    const aActive = a.start_at_unix <= now && a.end_at_unix > now
    const bActive = b.start_at_unix <= now && b.end_at_unix > now
    if (aActive !== bActive) return aActive ? -1 : 1
    return a.start_at_unix - b.start_at_unix
  })

  return (
    <div className="space-y-6">
      {/* 标题行 */}
      <h1 className="text-2xl font-bold text-center">全部秒杀活动</h1>

      {/* 加载骨架 */}
      {listQuery.isLoading && (
        <div className="space-y-6">
          {[0, 1].map((i) => (
            <div key={i} className="rounded-xl border border-border/60 p-4 space-y-3">
              <Skeleton className="h-10 w-64 mx-auto" />
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                {[...Array(6)].map((_, j) => (
                  <div key={j} className="space-y-2">
                    <Skeleton className="aspect-4/3 rounded-xl" />
                    <Skeleton className="h-3 w-full" />
                    <Skeleton className="h-4 w-16" />
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 错误 */}
      {listQuery.isError && (
        <div className="text-center py-12">
          <p className="text-sm text-destructive">{toUserMessage(listQuery.error)}</p>
        </div>
      )}

      {/* 活动列表 */}
      {sortedActivities.map((activity) => (
        <ActivitySection key={activity.activity_id} activity={activity} />
      ))}

      {/* 空状态 */}
      {!listQuery.isLoading && items.length === 0 && (
        <div className="py-16 text-center space-y-3">
          <Zap className="size-10 text-muted-foreground/20 mx-auto" />
          <p className="text-sm text-muted-foreground">暂无秒杀活动</p>
          <Button variant="outline" size="sm" onClick={() => navigate("/products")}>
            浏览全部商品
          </Button>
        </div>
      )}

      {/* 分页 */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</Button>
          <span className="text-sm text-muted-foreground">{page} / {totalPages}</span>
          <Button size="sm" variant="outline" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</Button>
        </div>
      )}
    </div>
  )
}
