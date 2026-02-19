import { useNavigate } from "react-router-dom"
import { Zap, Package, ShoppingCart, ChevronRight } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { SeckillItemCard } from "@/components/seckill/seckill-item-card"
import { SeckillCountdown } from "@/components/seckill/seckill-countdown"
import { useSeckillActivitiesQuery } from "@/hooks/user/use-seckill-hooks"
import { type ActivityPublic } from "@/api/modules/seckill"

// ─── Hero ──────────────────────────────────────────────────────────────────────

function HeroSection() {
    const navigate = useNavigate()

    return (
        <section className="text-center space-y-5 pt-8 pb-4">
            <p className="text-xs font-medium text-muted-foreground tracking-widest uppercase">
                Flash Sale
            </p>
            <h1 className="text-3xl sm:text-4xl font-extrabold tracking-tight">
                限时秒杀，<span className="text-red-500">手快有</span>
            </h1>
            <p className="text-sm text-muted-foreground max-w-sm mx-auto leading-relaxed">
                精选好物极致低价，每一场都是与时间的竞速
            </p>
            <div className="flex items-center justify-center gap-2.5 pt-1">
                <Button
                    className="rounded-full px-5"
                    onClick={() => navigate("/seckill")}
                >
                    <Zap className="size-3.5 mr-1" />
                    去抢购
                </Button>
                <Button
                    variant="outline"
                    className="rounded-full px-5"
                    onClick={() => navigate("/products")}
                >
                    逛一逛
                </Button>
            </div>
        </section>
    )
}

// ─── 秒杀预览 ──────────────────────────────────────────────────────────────────

function SeckillPreview({ activity }: { activity: ActivityPublic }) {
    const navigate = useNavigate()
    const now = Math.floor(Date.now() / 1000)
    const active = activity.start_at_unix <= now && now <= activity.end_at_unix
    const items = (activity.items ?? []).slice(0, 4)

    return (
        <section className="space-y-3">
            {/* 标题行 */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <div className="flex items-center gap-1 bg-red-500 text-white text-[11px] font-bold px-2 py-0.5 rounded">
                        <Zap className="size-3" />
                        限时秒杀
                    </div>
                </div>
                <button
                    onClick={() => navigate(`/seckill/${activity.activity_id}`)}
                    className="flex items-center gap-0.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
                >
                    更多
                    <ChevronRight className="size-3" />
                </button>
            </div>

            {/* 倒计时 */}
            <div className="flex justify-center py-3">
                <SeckillCountdown endUnix={activity.end_at_unix} fontSize={36} />
            </div>

            {/* 商品 */}
            {items.length > 0 && (
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                    {items.map((item) => (
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
        </section>
    )
}

// ─── 快捷入口 ──────────────────────────────────────────────────────────────────

const entries = [
    { icon: Zap, label: "秒杀", path: "/seckill", color: "text-red-500 bg-red-50" },
    { icon: Package, label: "商品", path: "/products", color: "text-blue-500 bg-blue-50" },
    { icon: ShoppingCart, label: "订单", path: "/orders", color: "text-emerald-500 bg-emerald-50" },
]

function QuickEntries() {
    const navigate = useNavigate()

    return (
        <div className="flex justify-around py-2">
            {entries.map((e) => (
                <button
                    key={e.path}
                    onClick={() => navigate(e.path)}
                    className="flex flex-col items-center gap-1.5 group"
                >
                    <div className={`size-11 rounded-2xl flex items-center justify-center ${e.color} transition-transform group-hover:scale-105`}>
                        <e.icon className="size-5" />
                    </div>
                    <span className="text-xs font-medium text-muted-foreground group-hover:text-foreground transition-colors">
                        {e.label}
                    </span>
                </button>
            ))}
        </div>
    )
}

// ─── 主页 ──────────────────────────────────────────────────────────────────────

export function HomePage() {
    const navigate = useNavigate()
    const listQuery = useSeckillActivitiesQuery({ page: 1, page_size: 10 })

    const allActivities = listQuery.data?.list ?? []
    const now = Math.floor(Date.now() / 1000)

    const liveActivity = allActivities.find(
        (a) => a.start_at_unix <= now && a.end_at_unix > now
    )
    const upcomingActivity = !liveActivity
        ? allActivities
            .filter((a) => a.start_at_unix > now)
            .sort((a, b) => a.start_at_unix - b.start_at_unix)[0]
        : undefined

    const previewActivity = liveActivity ?? upcomingActivity

    return (
        <div className="space-y-6">
            <HeroSection />

            {/* 快捷入口 */}
            <QuickEntries />

            {/* 分隔线 */}
            <div className="border-t border-border/50" />

            {/* 秒杀预览 */}
            {listQuery.isLoading && (
                <div className="space-y-3">
                    <Skeleton className="h-5 w-24" />
                    <Skeleton className="h-16 rounded-xl" />
                    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                        {[...Array(4)].map((_, i) => (
                            <div key={i} className="space-y-2">
                                <Skeleton className="aspect-4/3 rounded-xl" />
                                <Skeleton className="h-3 w-full" />
                                <Skeleton className="h-4 w-16" />
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {previewActivity && <SeckillPreview activity={previewActivity} />}

            {!listQuery.isLoading && !previewActivity && (
                <div className="py-12 text-center space-y-2">
                    <Zap className="size-8 text-muted-foreground/20 mx-auto" />
                    <p className="text-sm text-muted-foreground">暂无进行中的活动</p>
                    <Button variant="outline" size="sm" onClick={() => navigate("/products")}>
                        浏览商品
                    </Button>
                </div>
            )}
        </div>
    )
}
