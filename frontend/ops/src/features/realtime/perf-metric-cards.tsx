/* eslint-disable react-refresh/only-export-components */

import { useMemo } from "react"
import { Info } from "lucide-react"
import {
    Area,
    AreaChart,
    Bar,
    BarChart,
    CartesianGrid,
    Cell,
    Line,
    LineChart,
    Pie,
    PieChart,
    RadialBar,
    RadialBarChart,
    XAxis,
    YAxis,
} from "recharts"

import {
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
    type ChartConfig,
} from "@/components/ui/chart"
import {
    Popover,
    PopoverContent,
    PopoverTrigger,
} from "@/components/ui/popover"
import type { RealtimeSample } from "@/features/realtime/types"
import { cn } from "@/lib/utils"

// ─── 图表类型定义 ───

export type PerfChartKind = "area" | "bar" | "line" | "pie" | "radial" | "donut"

export interface PerfMetricDef {
    key: string
    label: string
    color: string
    unit: string
    chartKind: PerfChartKind
    /** 用于描述该指标的一句话摘要 */
    description: string
    /** 指标含义与分析方法 */
    analysis: string
    /** 是否来自服务端（虚线风格） */
    serverSide?: boolean
}

// ─── 全量指标注册 ───

export const PERF_METRICS: PerfMetricDef[] = [
    {
        key: "qps",
        label: "客户端 QPS",
        color: "hsl(210, 95%, 55%)",
        unit: " req/s",
        chartKind: "area",
        description: "每秒完成请求数",
        analysis: "反映系统实际吞吐能力。曲线稳定说明无瓶颈；若中后段明显下降，可能是连接池耗尽、MySQL 锁等待或 Go GC 导致。与服务端 QPS 对比可判断请求是否在网关层被丢弃。",
    },
    {
        key: "promQps",
        label: "服务端 QPS",
        color: "hsl(260, 70%, 60%)",
        unit: " req/s",
        chartKind: "area",
        description: "Prometheus 采集 RPC QPS",
        analysis: "Prometheus 侧采集的 gRPC 请求速率。若远低于客户端 QPS，说明请求在网关或负载均衡层被限流/拒绝。两者差值越大，网关层丢包越严重。",
        serverSide: true,
    },
    {
        key: "p95LatencyMs",
        label: "客户端 P95 延迟",
        color: "hsl(30, 90%, 55%)",
        unit: " ms",
        chartKind: "area",
        description: "95% 分位请求延迟",
        analysis: "95% 的请求在此时间内完成。理想值 <500ms。若持续 >1s 说明有慢查询或锁竞争；若出现尖峰可能是 GC 暂停或网络抖动。关注是否随 QPS 增长而线性上升（资源饱和）。",
    },
    {
        key: "promP99LatencyMs",
        label: "服务端 P99 延迟",
        color: "hsl(340, 70%, 55%)",
        unit: " ms",
        chartKind: "area",
        description: "Prometheus 采集 P99 延迟",
        analysis: "服务端视角的 99 分位延迟，包含 MySQL 事务、Redis 操作的全链路耗时。若远高于客户端 P95，说明最慢 5% 请求受锁竞争或事务超时影响严重，需检查 innodb_lock_wait_timeout。",
        serverSide: true,
    },
    {
        key: "successRate",
        label: "抢购成功率",
        color: "hsl(142, 70%, 45%)",
        unit: "%",
        chartKind: "area",
        description: "成功下单占总请求比例",
        analysis: "秒杀场景天然很低属于正常：库存 100 并发 1000 时约 10%。关注开始阶段的值是否合理——若一开始就是 0% 需检查活动配置。与库存扣减率的差值反映 Redis→DB 的转化损耗。",
    },
    {
        key: "rejectRate",
        label: "业务拒绝率",
        color: "hsl(45, 85%, 50%)",
        unit: "%",
        chartKind: "area",
        description: "正常竞争拒绝（库存不足/限购等）",
        analysis: "库存不足、限购超限、活动未开始等正常业务拒绝的占比。这些不是错误而是秒杀竞争的必然结果。高拒绝率（>80%）在抢购中后段属正常。若从一开始就 100% 需检查活动状态。",
    },
    {
        key: "systemErrorRate",
        label: "系统异常率",
        color: "hsl(0, 80%, 55%)",
        unit: "%",
        chartKind: "area",
        description: "非预期系统错误（真正的问题）",
        analysis: "排除已知业务拒绝后的异常比例。理想值 0%。若 >1% 说明存在未处理的错误码或内部 panic；若持续增长可能是 DB 连接池耗尽或 Redis 超时。这是唯一需要「报警」的错误指标。",
    },
    {
        key: "networkErrorRate",
        label: "网络错误率",
        color: "hsl(25, 85%, 50%)",
        unit: "%",
        chartKind: "area",
        description: "连接/超时类错误率",
        analysis: "连接被拒、超时、DNS 解析失败等非业务错误。理想值 0%。若 >5% 说明网关过载或服务不可用；持续增长则可能是连接池耗尽或容器重启。需检查 Docker 健康状态。",
    },
    {
        key: "stockDeductionRate",
        label: "库存扣减率",
        color: "hsl(190, 80%, 45%)",
        unit: "%",
        chartKind: "area",
        description: "成功扣减库存占比",
        analysis: "成功执行 MySQL 库存 UPDATE 的比例。反映 Redis 预扣到 DB 落库的转化率。若显著低于成功率，说明 Redis 预扣成功但 DB 层竞争严重（行锁超时）。理想状态两者应接近一致。",
    },
    {
        key: "promErrorRate",
        label: "服务端异常率",
        color: "hsl(15, 75%, 50%)",
        unit: "%",
        chartKind: "area",
        description: "Prometheus gRPC 非 OK 状态码占比",
        analysis: "服务端 gRPC 返回非 OK 状态码的比例。注意：业务级拒绝（如库存不足）通常返回 OK 状态码 + 业务错误码，所以此指标应很低。若 >5% 说明有真正的 RPC 层错误。",
        serverSide: true,
    },
    {
        key: "purchaseTaskQueueDepth",
        label: "异步队列深度",
        color: "hsl(200, 80%, 50%)",
        unit: "",
        chartKind: "line",
        description: "异步购买任务队列当前排队数",
        analysis: "异步购买快路径的待处理任务数。正常运行时应接近 0。若持续增长说明 Worker 消费不过来（order-rpc 慢或 Worker 数不够），最终会触发 queue full → 回退同步路径 → 延迟飙高。",
        serverSide: true,
    },
    {
        key: "purchaseTaskQueueCap",
        label: "异步队列容量",
        color: "hsl(200, 60%, 70%)",
        unit: "",
        chartKind: "line",
        description: "异步购买任务队列总容量",
        analysis: "队列的最大容量（配置值 AsyncPurchaseQueueSize）。对比 depth 可判断饱和程度：depth/cap > 80% 时应考虑扩容 Worker 或增大队列。",
        serverSide: true,
    },
    {
        key: "purchaseTaskDropped",
        label: "任务丢弃数",
        color: "hsl(0, 85%, 60%)",
        unit: "",
        chartKind: "bar",
        description: "过去 1 分钟异步任务被丢弃次数",
        analysis: "队列满时新任务被丢弃的次数。>0 表示异步快路径已失效，请求回退到同步 MySQL 路径——这是高延迟的直接原因。紧急程度最高，需立即扩容 Worker 或排查 order-rpc 瓶颈。",
        serverSide: true,
    },
]

// ─── 工具函数 ───

export function readMetricValue(sample: RealtimeSample, key: string): number {
    const raw = sample[key as keyof RealtimeSample]
    if (typeof raw === "number" && Number.isFinite(raw)) return raw
    return 0
}

export function hasAnyPerfData(samples: RealtimeSample[]): boolean {
    return samples.some(
        (s) =>
            (typeof s.qps === "number" && s.qps > 0) ||
            (typeof s.successRate === "number" && s.successRate > 0) ||
            (typeof s.p95LatencyMs === "number" && s.p95LatencyMs > 0)
    )
}

export function computeAvg(samples: RealtimeSample[], key: string): number {
    if (samples.length === 0) return 0
    const values = samples.map((s) => readMetricValue(s, key)).filter((v) => v > 0)
    if (values.length === 0) return 0
    return values.reduce((a, b) => a + b, 0) / values.length
}

export function computeMax(samples: RealtimeSample[], key: string): number {
    if (samples.length === 0) return 0
    return Math.max(0, ...samples.map((s) => readMetricValue(s, key)))
}

export function computeLatest(samples: RealtimeSample[], key: string): number {
    if (samples.length === 0) return 0
    return readMetricValue(samples[samples.length - 1], key)
}

/** Y 轴标签格式化：紧凑显示 */
function formatAxisValue(value: number, unit: string): string {
    if (unit.trim() === "%") return `${value.toFixed(0)}`
    if (value >= 10_000) return `${(value / 1000).toFixed(0)}k`
    if (value >= 1000) return `${(value / 1000).toFixed(1)}k`
    if (value >= 100) return `${value.toFixed(0)}`
    if (value >= 1) return `${value.toFixed(1)}`
    return `${value.toFixed(2)}`
}

// ─── 单指标图表卡片 ───

interface PerfMetricChartCardProps {
    metric: PerfMetricDef
    samples: RealtimeSample[]
    className?: string
}

export function PerfMetricChartCard({ metric, samples, className }: PerfMetricChartCardProps) {
    const latestValue = computeLatest(samples, metric.key)
    const avgValue = computeAvg(samples, metric.key)

    const formattedLatest = formatValue(latestValue, metric.unit)
    const formattedAvg = formatValue(avgValue, metric.unit)

    const config: ChartConfig = useMemo(
        () => ({
            [metric.key]: { label: metric.label, color: metric.color },
        }),
        [metric.key, metric.label, metric.color]
    )

    return (
        <div
            className={cn(
                "group flex flex-col overflow-hidden rounded-xl border bg-card shadow-sm transition-shadow hover:shadow-md",
                className
            )}
        >
            {/* 头部：指标名 + 最新值 */}
            <div className="flex items-center justify-between gap-2 px-3 pb-1 pt-2.5">
                <div className="min-w-0">
                    <div className="flex items-center gap-1.5">
                        <span
                            className="inline-block size-2 shrink-0 rounded-full"
                            style={{ backgroundColor: metric.color }}
                        />
                        <span className="truncate text-xs font-semibold text-foreground">
                            {metric.label}
                        </span>
                        {metric.serverSide && (
                            <span className="rounded-sm bg-muted px-1 py-0.5 text-[9px] text-muted-foreground">
                                服务端
                            </span>
                        )}
                        <MetricInfoPopover metric={metric} />
                    </div>
                    <div className="mt-0.5 text-[10px] text-muted-foreground">{metric.description}</div>
                </div>
                <div className="shrink-0 text-right">
                    <div className="text-base font-bold tabular-nums text-foreground">{formattedLatest}</div>
                    <div className="text-[10px] text-muted-foreground">
                        avg {formattedAvg}
                    </div>
                </div>
            </div>

            {/* 图表区域：最大化占比 */}
            <div className="min-h-0 flex-1 overflow-hidden pb-1">
                <MetricChart metric={metric} samples={samples} config={config} />
            </div>
        </div>
    )
}

// ─── 根据 chartKind 渲染不同图表 ───

function MetricChart({
    metric,
    samples,
    config,
}: {
    metric: PerfMetricDef
    samples: RealtimeSample[]
    config: ChartConfig
}) {
    switch (metric.chartKind) {
        case "area":
            return <AreaMetricChart metric={metric} samples={samples} config={config} />
        case "line":
            return <LineMetricChart metric={metric} samples={samples} config={config} />
        case "bar":
            return <BarMetricChart metric={metric} samples={samples} config={config} />
        case "radial":
            return <RadialMetricChart metric={metric} samples={samples} config={config} />
        case "pie":
        case "donut":
            return <DonutMetricChart metric={metric} samples={samples} config={config} />
        default:
            return <AreaMetricChart metric={metric} samples={samples} config={config} />
    }
}

// ─── Area Chart ───

function AreaMetricChart({
    metric,
    samples,
    config,
}: {
    metric: PerfMetricDef
    samples: RealtimeSample[]
    config: ChartConfig
}) {
    const maxY = useMemo(() => {
        const values = samples.map((s) => readMetricValue(s, metric.key))
        return Math.ceil(Math.max(1, ...values) * 1.15)
    }, [samples, metric.key])

    return (
        <ChartContainer config={config} className="h-full w-full">
            <AreaChart data={samples} margin={{ top: 4, right: 8, bottom: 16, left: 0 }}>
                <defs>
                    <linearGradient id={`fill-${metric.key}`} x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor={metric.color} stopOpacity={0.65} />
                        <stop offset="95%" stopColor={metric.color} stopOpacity={0.05} />
                    </linearGradient>
                </defs>
                <CartesianGrid vertical={false} strokeDasharray="3 3" strokeOpacity={0.15} />
                <XAxis
                    dataKey="label"
                    tickLine={false}
                    axisLine={false}
                    fontSize={9}
                    tick={{ fill: "var(--color-muted-foreground)" }}
                    interval="preserveStartEnd"
                    minTickGap={40}
                />
                <YAxis
                    domain={[0, maxY]}
                    tickLine={false}
                    axisLine={false}
                    fontSize={9}
                    width={28}
                    tick={{ fill: "var(--color-muted-foreground)" }}
                    tickFormatter={(v: number) => formatAxisValue(v, metric.unit)}
                />
                <ChartTooltip
                    cursor={false}
                    content={<ChartTooltipContent hideLabel indicator="line" />}
                />
                <Area
                    type="monotone"
                    dataKey={metric.key}
                    stroke={metric.color}
                    fill={`url(#fill-${metric.key})`}
                    strokeWidth={2}
                    dot={false}
                    isAnimationActive={false}
                    connectNulls
                />
            </AreaChart>
        </ChartContainer>
    )
}

// ─── Line Chart ───

function LineMetricChart({
    metric,
    samples,
    config,
}: {
    metric: PerfMetricDef
    samples: RealtimeSample[]
    config: ChartConfig
}) {
    const maxY = useMemo(() => {
        const values = samples.map((s) => readMetricValue(s, metric.key))
        return Math.ceil(Math.max(1, ...values) * 1.15)
    }, [samples, metric.key])

    return (
        <ChartContainer config={config} className="w-full">
            <LineChart data={samples} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" strokeOpacity={0.15} />
                <XAxis dataKey="label" hide />
                <YAxis domain={[0, maxY]} hide />
                <ChartTooltip
                    cursor={false}
                    content={<ChartTooltipContent hideLabel indicator="dot" />}
                />
                <Line
                    type="monotone"
                    dataKey={metric.key}
                    stroke={metric.color}
                    strokeWidth={2.5}
                    dot={false}
                    isAnimationActive={false}
                    connectNulls
                    strokeDasharray={metric.serverSide ? "6 3" : undefined}
                />
            </LineChart>
        </ChartContainer>
    )
}

// ─── Bar Chart ───

function BarMetricChart({
    metric,
    samples,
    config,
}: {
    metric: PerfMetricDef
    samples: RealtimeSample[]
    config: ChartConfig
}) {
    // 只取最近 20 个点避免柱子过窄
    const recent = useMemo(() => samples.slice(-20), [samples])
    const maxY = useMemo(() => {
        const values = recent.map((s) => readMetricValue(s, metric.key))
        const m = Math.max(0.1, ...values)
        return Math.ceil(m * 1.2)
    }, [recent, metric.key])

    return (
        <ChartContainer config={config} className="w-full">
            <BarChart data={recent} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" strokeOpacity={0.15} />
                <XAxis dataKey="label" hide />
                <YAxis domain={[0, maxY]} hide />
                <ChartTooltip
                    cursor={false}
                    content={<ChartTooltipContent hideLabel indicator="line" />}
                />
                <Bar
                    dataKey={metric.key}
                    fill={metric.color}
                    radius={[3, 3, 0, 0]}
                    isAnimationActive={false}
                    fillOpacity={0.8}
                />
            </BarChart>
        </ChartContainer>
    )
}

// ─── Radial (Gauge) Chart ───

function RadialMetricChart({
    metric,
    samples,
    config,
}: {
    metric: PerfMetricDef
    samples: RealtimeSample[]
    config: ChartConfig
}) {
    const latest = computeLatest(samples, metric.key)
    const data = useMemo(
        () => [{ name: metric.label, value: Math.min(latest, 100), fill: metric.color }],
        [metric.label, latest, metric.color]
    )

    return (
        <ChartContainer config={config} className="w-full">
            <RadialBarChart
                data={data}
                startAngle={180}
                endAngle={0}
                innerRadius="65%"
                outerRadius="100%"
                barSize={14}
                margin={{ top: 0, right: 0, bottom: -20, left: 0 }}
            >
                <RadialBar
                    dataKey="value"
                    background={{ fill: "var(--muted)" }}
                    cornerRadius={8}
                    isAnimationActive={false}
                />
                <text
                    x="50%"
                    y="48%"
                    textAnchor="middle"
                    dominantBaseline="middle"
                    className="fill-foreground text-lg font-bold"
                >
                    {latest.toFixed(1)}%
                </text>
            </RadialBarChart>
        </ChartContainer>
    )
}

// ─── Donut (Pie) Chart ───

function DonutMetricChart({
    metric,
    samples,
    config,
}: {
    metric: PerfMetricDef
    samples: RealtimeSample[]
    config: ChartConfig
}) {
    const latest = computeLatest(samples, metric.key)
    const remainder = Math.max(0, 100 - latest)
    const data = useMemo(
        () => [
            { name: metric.label, value: latest },
            { name: "余量", value: remainder },
        ],
        [metric.label, latest, remainder]
    )

    return (
        <ChartContainer config={config} className="w-full">
            <PieChart margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
                <Pie
                    data={data}
                    cx="50%"
                    cy="50%"
                    innerRadius="55%"
                    outerRadius="85%"
                    paddingAngle={2}
                    dataKey="value"
                    isAnimationActive={false}
                    strokeWidth={0}
                >
                    <Cell fill={metric.color} />
                    <Cell fill="var(--muted)" />
                </Pie>
                <text
                    x="50%"
                    y="50%"
                    textAnchor="middle"
                    dominantBaseline="middle"
                    className="fill-foreground text-base font-bold"
                >
                    {latest.toFixed(1)}%
                </text>
            </PieChart>
        </ChartContainer>
    )
}

// ─── 指标分析 Popover ───

function MetricInfoPopover({ metric }: { metric: PerfMetricDef }) {
    return (
        <Popover>
            <PopoverTrigger
                render={
                    <button
                        type="button"
                        className="inline-flex size-4 shrink-0 items-center justify-center rounded-full text-muted-foreground/50 transition-colors hover:bg-muted hover:text-foreground"
                    >
                        <Info className="size-3" />
                    </button>
                }
            />
            <PopoverContent
                side="top"
                align="start"
                className="max-w-[280px] space-y-1.5 p-3 text-xs"
            >
                <div className="flex items-center gap-1.5 font-semibold text-foreground">
                    <span
                        className="inline-block size-2 rounded-full"
                        style={{ backgroundColor: metric.color }}
                    />
                    {metric.label}
                </div>
                <p className="leading-relaxed text-muted-foreground">
                    {metric.analysis}
                </p>
            </PopoverContent>
        </Popover>
    )
}

// ─── 格式化 ───

function formatValue(value: number, unit: string): string {
    if (unit.trim() === "%") return `${value.toFixed(1)}%`
    if (unit.trim().length > 0) return `${value.toFixed(1)}${unit}`
    return value.toFixed(0)
}
