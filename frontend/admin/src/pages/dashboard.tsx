import {
  Users, Package, ShoppingCart, Zap,
  Clock, ClipboardCheck, Truck, XCircle,
  RefreshCw, TrendingUp, AlertCircle,
} from "lucide-react"
import {
  PieChart, Pie, Cell,
  Tooltip, ResponsiveContainer, Legend,
} from "recharts"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { useAdminPermission } from "@/hooks/auth/use-admin-permission"
import { useAdminSessionState } from "@/hooks/auth/use-admin-session"
import { useDashboardStatsQuery } from "@/hooks/biz/use-dashboard-hooks"

// ── KPI卡片 ──────────────────────────────────────────────
interface KpiCardProps {
  title: string
  value: number | string
  icon: React.ElementType
  iconColor: string
  iconBg: string
  sub?: React.ReactNode
  loading?: boolean
}

function KpiCard({ title, value, icon: Icon, iconColor, iconBg, sub, loading }: KpiCardProps) {
  return (
    <Card className="relative overflow-hidden">
      <CardContent className="pt-5 pb-4 px-5">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{title}</p>
            <p className="text-3xl font-bold tabular-nums tracking-tight">
              {loading ? <span className="text-muted-foreground/40">—</span> : value}
            </p>
            {sub && <div className="text-xs text-muted-foreground">{sub}</div>}
          </div>
          <div className={`flex size-10 items-center justify-center rounded-xl ${iconBg}`}>
            <Icon className={`size-5 ${iconColor}`} />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

// ── 订单状态条目 ──────────────────────────────────────────
interface OrderStatusRowProps {
  label: string
  value: number
  total: number
  color: string
  icon: React.ElementType
}

function OrderStatusRow({ label, value, total, color, icon: Icon }: OrderStatusRowProps) {
  const pct = total > 0 ? Math.round((value / total) * 100) : 0
  return (
    <div className="flex items-center gap-3">
      <div className={`flex size-7 shrink-0 items-center justify-center rounded-md ${color}`}>
        <Icon className="size-3.5 text-white" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between text-xs mb-1">
          <span className="text-muted-foreground">{label}</span>
          <span className="tabular-nums font-medium">{value.toLocaleString()}</span>
        </div>
        <div className="h-1.5 rounded-full bg-muted overflow-hidden">
          <div
            className={`h-full rounded-full transition-all duration-700 ${color}`}
            style={{ width: `${pct}%` }}
          />
        </div>
      </div>
      <span className="w-8 text-right text-xs text-muted-foreground">{pct}%</span>
    </div>
  )
}

// ── 饼图 Tooltip ──────────────────────────────────────────
function PieTooltip({ active, payload }: { active?: boolean; payload?: Array<{ name: string; value: number }> }) {
  if (!active || !payload?.length) return null
  return (
    <div className="rounded-lg border bg-background px-3 py-1.5 text-xs shadow-lg">
      <p className="font-medium">{payload[0].name}</p>
      <p className="text-muted-foreground">{payload[0].value} 个</p>
    </div>
  )
}

// ── 主页面 ────────────────────────────────────────────────
const DOMAIN_LABELS: Record<string, string> = {
  admin_management: "管理员管理",
  user_management: "用户管理",
  product_management: "商品管理",
  order_management: "订单管理",
  seckill_management: "秒杀管理",
}

export function DashboardPage() {
  const { profile } = useAdminSessionState()
  const { domains, dataScope } = useAdminPermission()
  const statsQuery = useDashboardStatsQuery()
  const stats = statsQuery.data
  const loading = statsQuery.isLoading

  const orderTotal = stats?.orders.total ?? 0
  const orderStatuses = [
    { label: "待支付", value: stats?.orders.pending_payment ?? 0, color: "bg-amber-400", icon: Clock },
    { label: "待审核", value: stats?.orders.pending_review ?? 0, color: "bg-orange-500", icon: ClipboardCheck },
    { label: "待发货", value: stats?.orders.pending_ship ?? 0, color: "bg-blue-500", icon: Package },
    { label: "已发货", value: stats?.orders.shipped ?? 0, color: "bg-indigo-500", icon: Truck },
    { label: "已关闭", value: stats?.orders.closed ?? 0, color: "bg-slate-400", icon: XCircle },
  ]

  const PIE_COLORS = ["#6366f1", "#f59e0b", "#94a3b8"]
  const seckillPieData = stats ? [
    { name: "进行中", value: stats.seckill.active },
    { name: "草稿", value: stats.seckill.upcoming },
    { name: "其他", value: Math.max(0, stats.seckill.total - stats.seckill.active - stats.seckill.upcoming) },
  ].filter(d => d.value > 0) : []

  const greeting = (() => {
    const h = new Date().getHours()
    if (h < 6) return "夜深了"
    if (h < 12) return "早上好"
    if (h < 14) return "中午好"
    if (h < 18) return "下午好"
    return "晚上好"
  })()

  const displayName = profile?.display_name || profile?.username || "管理员"

  return (
    <div className="space-y-6 p-6">

      {/* ── 顶部：欢迎 + 刷新 ── */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">
            {greeting}，{displayName} 👋
          </h1>
          <p className="text-sm text-muted-foreground mt-0.5">这是你的运营工作台概览</p>
        </div>
        <Button
          variant="outline" size="sm"
          onClick={() => statsQuery.refetch()}
          disabled={statsQuery.isFetching}
        >
          <RefreshCw className={`mr-1.5 size-3.5 ${statsQuery.isFetching ? "animate-spin" : ""}`} />
          刷新
        </Button>
      </div>

      {/* ── 错误提示 ── */}
      {statsQuery.isError && (
        <div className="flex items-center gap-2 rounded-lg border border-destructive/40 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          <AlertCircle className="size-4 shrink-0" />
          统计数据加载失败，请检查后端服务状态。
        </div>
      )}

      {/* ── KPI 卡片 ── */}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard
          title="用户总数" value={stats?.users.total ?? 0} loading={loading}
          icon={Users} iconColor="text-violet-600" iconBg="bg-violet-100 dark:bg-violet-950"
        />
        <KpiCard
          title="商品总数" value={stats?.products.total ?? 0} loading={loading}
          icon={Package} iconColor="text-blue-600" iconBg="bg-blue-100 dark:bg-blue-950"
          sub={stats ? `在售 ${stats.products.on_sale}` : undefined}
        />
        <KpiCard
          title="订单总数" value={stats?.orders.total ?? 0} loading={loading}
          icon={ShoppingCart} iconColor="text-emerald-600" iconBg="bg-emerald-100 dark:bg-emerald-950"
          sub={stats ? `待处理 ${(stats.orders.pending_payment + stats.orders.pending_review + stats.orders.pending_ship).toLocaleString()}` : undefined}
        />
        <KpiCard
          title="秒杀活动" value={stats?.seckill.total ?? 0} loading={loading}
          icon={Zap} iconColor="text-amber-600" iconBg="bg-amber-100 dark:bg-amber-950"
          sub={stats ? (
            <span className="flex items-center gap-2">
              <span className="flex items-center gap-1"><TrendingUp className="size-3 text-emerald-500" />进行中 {stats.seckill.active}</span>
              <span>草稿 {stats.seckill.upcoming}</span>
            </span>
          ) : undefined}
        />
      </div>

      {/* ── 中部：订单状态 + 秒杀分布 ── */}
      <div className="grid gap-4 lg:grid-cols-3">

        {/* 订单状态分布 */}
        <Card className="lg:col-span-2">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-semibold">订单状态分布</CardTitle>
            <CardDescription className="text-xs">
              共 <span className="font-medium text-foreground">{orderTotal.toLocaleString()}</span> 笔订单
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 px-5 pb-5">
            {loading
              ? Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="h-8 rounded-md bg-muted animate-pulse" />
              ))
              : orderStatuses.map((s) => (
                <OrderStatusRow key={s.label} {...s} total={orderTotal} />
              ))
            }
          </CardContent>
        </Card>

        {/* 秒杀活动饼图 */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-semibold">秒杀活动状态</CardTitle>
            <CardDescription className="text-xs">
              共 <span className="font-medium text-foreground">{stats?.seckill.total ?? 0}</span> 个活动
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col items-center justify-center pb-4">
            {loading || seckillPieData.length === 0 ? (
              <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                {loading ? "加载中…" : "暂无数据"}
              </div>
            ) : (
              <ResponsiveContainer width="100%" height={180}>
                <PieChart>
                  <Pie
                    data={seckillPieData}
                    cx="50%" cy="50%"
                    innerRadius={48} outerRadius={72}
                    paddingAngle={3}
                    dataKey="value"
                  >
                    {seckillPieData.map((_, idx) => (
                      <Cell key={idx} fill={PIE_COLORS[idx % PIE_COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip content={<PieTooltip />} />
                  <Legend
                    iconType="circle" iconSize={8}
                    formatter={(v) => <span className="text-xs text-muted-foreground">{v}</span>}
                  />
                </PieChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>
      </div>

      {/* ── 底部：权限信息 ── */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-semibold">当前账号权限</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap items-center gap-6 px-5 pb-5">
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">账号</p>
            <p className="text-sm font-medium">{profile?.display_name || profile?.username || "-"}</p>
          </div>
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">数据范围</p>
            <Badge variant={dataScope === "all" ? "default" : "secondary"} className="text-xs">
              {dataScope === "all" ? "全部数据" : "仅本人"}
            </Badge>
          </div>
          <div className="space-y-1.5">
            <p className="text-xs text-muted-foreground">权限域</p>
            <div className="flex flex-wrap gap-1.5">
              {domains.length === 0
                ? <span className="text-xs text-muted-foreground">无</span>
                : domains.map((d) => (
                  <Badge key={d} variant="outline" className="text-xs">
                    {DOMAIN_LABELS[d] ?? d}
                  </Badge>
                ))
              }
            </div>
          </div>
        </CardContent>
      </Card>

    </div>
  )
}
