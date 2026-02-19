import { useParams, useNavigate } from "react-router-dom"
import { ArrowLeft } from "lucide-react"

import { Skeleton } from "@/components/ui/skeleton"
import { SeckillItemCard } from "@/components/seckill"
import { SeckillCountdown } from "@/components/seckill/seckill-countdown"
import { useSeckillActivityDetailQuery } from "@/hooks/user/use-seckill-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

function getActivityStatus(startUnix: number, endUnix: number) {
  const now = Math.floor(Date.now() / 1000)
  if (now < startUnix) return { active: false }
  if (now > endUnix) return { active: false }
  return { active: true }
}

export function SeckillDetailPage() {
  const { activityId } = useParams<{ activityId: string }>()
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()

  const detailQuery = useSeckillActivityDetailQuery(activityId)
  const activity = detailQuery.data?.activity

  if (detailQuery.isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-32" />
        <div className="rounded-xl border border-border/60 p-4 space-y-3">
          <Skeleton className="h-10 w-64 mx-auto" />
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {[...Array(6)].map((_, i) => (
              <div key={i} className="space-y-2">
                <Skeleton className="aspect-4/3 rounded-xl" />
                <Skeleton className="h-3 w-full" />
                <Skeleton className="h-4 w-16" />
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  if (detailQuery.isError) {
    return (
      <div className="text-center py-12 space-y-3">
        <p className="text-sm text-destructive">{toUserMessage(detailQuery.error)}</p>
        <button
          onClick={() => navigate(-1)}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          ← 返回
        </button>
      </div>
    )
  }

  if (!activity) return null

  const { active } = getActivityStatus(activity.start_at_unix, activity.end_at_unix)
  const items = activity.items ?? []

  return (
    <div className="space-y-6">
      {/* 返回按钮 */}
      <button
        onClick={() => navigate(-1)}
        className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="size-4" />
        返回
      </button>

      {/* 活动区块 —— 与主页一致 */}
      <section className="rounded-xl border border-border/60 p-4 space-y-3">
        {/* 倒计时行 */}
        <div className="flex items-center justify-center">
          <SeckillCountdown endUnix={activity.end_at_unix} />
        </div>

        {/* 商品网格 */}
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground py-8 text-center">暂无秒杀商品</p>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {items.map((item) => (
              <SeckillItemCard
                key={item.item_id}
                activityId={activityId!}
                item={item}
                active={active}
              />
            ))}
          </div>
        )}

        {/* 底部商品数量 */}
        {items.length > 0 && (
          <p className="text-center text-xs text-muted-foreground pt-1">
            共 {items.length} 件商品
          </p>
        )}
      </section>
    </div>
  )
}
