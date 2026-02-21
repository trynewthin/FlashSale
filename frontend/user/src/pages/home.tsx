import { useNavigate } from "react-router-dom"
import { Zap, Package, ShoppingCart, ChevronRight } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { SeckillItemCard } from "@/components/seckill/seckill-item-card"
import { SeckillCountdown } from "@/components/seckill/seckill-countdown"
import { useSeckillActivitiesQuery } from "@/hooks/user/use-seckill-hooks"
import { useProductsQuery } from "@/hooks/user/use-product-hooks"
import { EmptyState } from "@/components/ui/empty-state"
import { type ActivityPublic } from "@/api/modules/seckill"
import { motion } from "motion/react"
import { formatCent } from "@/lib/format"
import { cdnUrl } from "@/lib/cdn"

// ─── Hero ──────────────────────────────────────────────────────────────────────

function HeroSection() {
    const navigate = useNavigate()
    const { data } = useProductsQuery({ page_size: 3 })
    const products = data?.list ?? []

    return (
        <section className="relative w-full pt-10 pb-8 sm:pt-16 sm:pb-12 mb-8 border-b border-border/30">
            {/* 动态广域背景光晕 — w-screen 配合 overflow-x-hidden 在根 div 防止水平滚动条 */}
            <div className="absolute inset-0 left-1/2 -translate-x-1/2 w-screen pointer-events-none">
                <div className="absolute top-0 right-[5%] -translate-y-1/3 w-[700px] h-[700px] bg-red-500/15 dark:bg-red-400/8 blur-[130px] rounded-full" />
                <div className="absolute bottom-0 left-[5%] translate-y-1/3 w-[600px] h-[600px] bg-orange-400/12 dark:bg-orange-400/6 blur-[130px] rounded-full" />
            </div>

            <div className="relative z-10 grid grid-cols-1 md:grid-cols-2 gap-10 items-center max-w-5xl mx-auto px-4 sm:px-6">
                {/* 左侧：文字信息信息 */}
                <motion.div
                    initial={{ opacity: 0, x: -30, filter: "blur(10px)" }}
                    animate={{ opacity: 1, x: 0, filter: "blur(0px)" }}
                    transition={{ duration: 0.7, ease: "easeOut" }}
                    className="space-y-6 text-center md:text-left"
                >
                    <div className="mx-auto md:mx-0 inline-flex items-center gap-2 px-3 py-1 rounded-full bg-red-50 dark:bg-red-500/10 border border-red-100 dark:border-red-500/20 text-red-600 dark:text-red-400 text-[11px] font-bold tracking-widest uppercase mb-1">
                        <span className="relative flex h-1.5 w-1.5">
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75"></span>
                            <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-red-500"></span>
                        </span>
                        Next-Gen Flash Sale
                    </div>

                    <h1 className="text-4xl sm:text-5xl lg:text-5xl xl:text-6xl font-black tracking-tight text-foreground leading-[1.15]">
                        极致低价 <br />
                        <span className="text-transparent bg-clip-text bg-linear-to-r from-red-500 via-orange-500 to-amber-500">
                            手快有手慢无
                        </span>
                    </h1>

                    <p className="text-base sm:text-lg text-muted-foreground mx-auto md:mx-0 leading-relaxed max-w-[320px] sm:max-w-md">
                        摒弃套路，回归真实大促。我们只上架经受得住考验的硬货，为您提供颠覆认知的绝美极价。
                    </p>

                    <div className="flex flex-row items-center justify-center md:justify-start gap-4 pt-4">
                        <Button
                            size="lg"
                            className="rounded-full px-8 h-12 text-base shadow-xl shadow-red-500/20 hover:shadow-red-500/40 hover:-translate-y-0.5 transition-all font-bold"
                            onClick={() => navigate("/seckill")}
                        >
                            <Zap className="size-5 mr-1.5" />
                            参与秒杀
                        </Button>
                        <Button
                            variant="outline"
                            size="lg"
                            className="rounded-full px-8 h-12 text-base font-semibold border-2 hover:bg-accent/50 hover:border-border"
                            onClick={() => navigate("/products")}
                        >
                            随便逛逛
                        </Button>
                    </div>
                </motion.div>

                {/* 右侧：创意商品陈列矩阵 (Awwwards 风格画廊) */}
                <div className="hidden md:flex relative h-[420px] w-full items-center justify-center -ml-6 lg:ml-0">
                    {/* 柔和的底座发光 */}
                    <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-72 h-72 bg-background/50 backdrop-blur-3xl rounded-full border border-border/50 shadow-2xl pointer-events-none" />

                    {products.length > 0 ? (
                        products.slice(0, 3).map((item, index) => {
                            // 卡片散开角度与位移
                            const rot = [-12, 0, 15]
                            const xOff = [-80, 0, 80]
                            const yOff = [30, -20, 50]
                            const zIndex = [10, 20, 15] // 中间那张最凸显
                            const scales = [0.85, 1, 0.9]

                            return (
                                <motion.div
                                    key={item.product_id}
                                    initial={{ opacity: 0, y: 120, rotate: 0 }}
                                    animate={{
                                        opacity: 1,
                                        x: xOff[index],
                                        y: yOff[index],
                                        rotate: rot[index],
                                        scale: scales[index]
                                    }}
                                    transition={{
                                        duration: 0.8,
                                        delay: 0.15 * index,
                                        ease: [0.22, 1, 0.36, 1]
                                    }}
                                    style={{ zIndex: zIndex[index] }}
                                    onClick={() => navigate(`/products/${item.product_id}`)}
                                    whileHover={{
                                        scale: scales[index] + 0.05,
                                        y: yOff[index] - 10,
                                        transition: { duration: 0.3 }
                                    }}
                                    className="absolute cursor-pointer w-[200px] aspect-4/5 rounded-[1.5rem] bg-card overflow-hidden shadow-2xl border border-border flex flex-col group"
                                >
                                    {/* 图片 */}
                                    <div className="relative flex-1 bg-muted">
                                        {item.main_image ? (
                                            <img src={cdnUrl(item.main_image)} alt={item.name} className="absolute inset-0 size-full object-cover transition-transform duration-700 group-hover:scale-110" />
                                        ) : (
                                            <div className="absolute inset-0 flex items-center justify-center">
                                                <Package className="size-8 text-muted-foreground/30" />
                                            </div>
                                        )}
                                        {/* 渐变遮罩压暗底部 */}
                                        <div className="absolute inset-0 bg-linear-to-t from-black/80 via-black/10 to-transparent pointer-events-none" />

                                        {/* 文字在图内底部浮动 */}
                                        <div className="absolute bottom-0 left-0 w-full p-4 flex flex-col gap-0.5 pointer-events-none">
                                            <span className="text-white text-sm font-bold truncate tracking-tight">{item.name}</span>
                                            <span className="text-red-400 font-extrabold text-base">{formatCent(item.price_cent)}</span>
                                        </div>
                                    </div>
                                </motion.div>
                            )
                        })
                    ) : (
                        // 预留空态
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            transition={{ delay: 0.2 }}
                            className="absolute inset-0 flex items-center justify-center z-20 pointer-events-none"
                        >
                            <Skeleton className="w-[200px] aspect-4/5 rounded-[1.5rem] shadow-xl" />
                        </motion.div>
                    )}
                </div>
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
    {
        title: "限时秒杀",
        desc: "极致低价天天抢",
        icon: Zap,
        path: "/seckill",
        glow: "from-red-500/20 to-transparent",
        iconColor: "text-red-500 bg-red-100 dark:bg-red-500/20"
    },
    {
        title: "精选商品",
        desc: "发现全网硬通货",
        icon: Package,
        path: "/products",
        glow: "from-blue-500/20 to-transparent",
        iconColor: "text-blue-500 bg-blue-100 dark:bg-blue-500/20"
    },
    {
        title: "我的订单",
        desc: "追踪履约进度",
        icon: ShoppingCart,
        path: "/orders",
        glow: "from-emerald-500/20 to-transparent",
        iconColor: "text-emerald-500 bg-emerald-100 dark:bg-emerald-500/20"
    },
]

function QuickEntries() {
    const navigate = useNavigate()

    return (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 pb-4">
            {entries.map((e, index) => (
                <motion.button
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.2 + index * 0.1, ease: "easeOut" }}
                    key={e.path}
                    onClick={() => navigate(e.path)}
                    className="group relative flex items-center justify-between p-5 rounded-3xl bg-card border border-border/60 hover:border-border overflow-hidden transition-all hover:shadow-xl hover:-translate-y-1 text-left w-full"
                >
                    {/* 背景发光渐变 (仅在此卡片内部) */}
                    <div className={`absolute top-0 right-0 w-32 h-32 bg-linear-to-bl ${e.glow} rounded-full blur-2xl -mr-10 -mt-10 pointer-events-none transition-transform duration-500 group-hover:scale-125`} />

                    <div className="flex items-center gap-4 relative z-10">
                        <div className={`size-12 rounded-2xl flex items-center justify-center ${e.iconColor} shadow-inner transition-transform duration-300 group-hover:scale-110 group-hover:-rotate-6`}>
                            <e.icon className="size-6" />
                        </div>
                        <div>
                            <h3 className="font-bold text-base text-foreground mb-0.5">{e.title}</h3>
                            <p className="text-xs text-muted-foreground font-medium">{e.desc}</p>
                        </div>
                    </div>

                    <ChevronRight className="relative z-10 size-4 text-muted-foreground opacity-30 group-hover:opacity-100 group-hover:translate-x-1 transition-all duration-300" />
                </motion.button>
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
        <div className="space-y-2">
            <HeroSection />

            <div className="max-w-5xl mx-auto space-y-6">
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
                    <div className="py-8">
                        <EmptyState
                            icon={Zap}
                            title="暂无进行中的活动"
                            description="当前时段暂时没有秒杀活动，去逛逛其他好物吧"
                            actionLabel="浏览商品"
                            onAction={() => navigate("/products")}
                        />
                    </div>
                )}
            </div>
        </div>
    )
}
