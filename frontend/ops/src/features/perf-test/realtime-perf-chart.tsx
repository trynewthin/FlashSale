import { useMemo } from "react"
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts"

import {
    ChartContainer,
    ChartLegend,
    ChartLegendContent,
    ChartTooltip,
    ChartTooltipContent,
    type ChartConfig,
} from "@/components/ui/chart"
import type { RealtimeSample } from "@/features/realtime/types"
import { cn } from "@/lib/utils"

interface MetricDef {
    key: string
    label: string
    color: string
    axis: "percent" | "count"
    unit: string
    dashed?: boolean
}

// 压测分析图的指标定义：客户端视角 + Prometheus 服务端叠加对比
const PERF_METRICS: MetricDef[] = [
    // 客户端视角（来自 seckillload 进度 JSON）
    { key: "qps", label: "客户端 QPS", color: "var(--chart-3)", axis: "count", unit: " req/s" },
    { key: "p95LatencyMs", label: "客户端 P95", color: "var(--chart-4)", axis: "count", unit: " ms" },
    { key: "successRate", label: "抢购成功率", color: "var(--chart-2)", axis: "percent", unit: "%" },
    { key: "rejectRate", label: "业务拒绝率", color: "var(--chart-5)", axis: "percent", unit: "%" },
    { key: "systemErrorRate", label: "系统异常率", color: "var(--chart-1)", axis: "percent", unit: "%" },
    { key: "networkErrorRate", label: "网络错误率", color: "var(--chart-10)", axis: "percent", unit: "%" },
    { key: "stockDeductionRate", label: "库存扣减率", color: "var(--chart-6)", axis: "percent", unit: "%" },
    // 服务端视角（来自 Prometheus，虚线 + 透明度降低）
    { key: "promQps", label: "服务端 QPS", color: "var(--chart-9)", axis: "count", unit: " req/s", dashed: true },
    { key: "promP99LatencyMs", label: "服务端 P99", color: "var(--chart-7)", axis: "count", unit: " ms", dashed: true },
    { key: "promErrorRate", label: "服务端异常率", color: "var(--chart-8)", axis: "percent", unit: "%", dashed: true },
]

function readMetricValue(sample: RealtimeSample, key: string): number {
    const raw = sample[key as keyof RealtimeSample]
    if (typeof raw === "number" && Number.isFinite(raw)) {
        return raw
    }
    return 0
}

function hasAnyTestData(samples: RealtimeSample[]): boolean {
    return samples.some(
        (s) =>
            (typeof s.qps === "number" && s.qps > 0) ||
            (typeof s.successRate === "number" && s.successRate > 0) ||
            (typeof s.p95LatencyMs === "number" && s.p95LatencyMs > 0)
    )
}

interface RealtimePerfChartProps {
    samples: RealtimeSample[]
    /** compact 模式：无 Card 包裹，纯图表填满父容器 */
    compact?: boolean
    className?: string
}

// RealtimePerfChart — 压测分析图，客户端实线 vs 服务端虚线双视角对比
export function RealtimePerfChart({ samples, compact, className }: RealtimePerfChartProps) {
    const hasData = useMemo(() => hasAnyTestData(samples), [samples])

    const countMetricKeys = useMemo(
        () => PERF_METRICS.filter((m) => m.axis === "count").map((m) => m.key),
        []
    )

    const maxCountY = useMemo(() => {
        if (countMetricKeys.length === 0) return 1
        const values = samples.flatMap((sample) =>
            countMetricKeys.map((key) => readMetricValue(sample, key))
        )
        const maxValue = Math.max(1, ...values)
        return Math.ceil(maxValue * 1.2)
    }, [samples, countMetricKeys])

    const chartConfig: ChartConfig = useMemo(() => {
        const config: ChartConfig = {}
        for (const metric of PERF_METRICS) {
            config[metric.key] = {
                label: metric.label,
                color: metric.color,
            }
        }
        return config
    }, [])

    // 无数据占位
    if (!hasData) {
        const emptyContent = (
            <div className="flex h-full items-center justify-center rounded-lg border border-dashed bg-muted/20 text-sm text-muted-foreground">
                启动压测后自动展示客户端 vs 服务端双视角对比图
            </div>
        )
        if (compact) {
            return <div className={cn("h-full", className)}>{emptyContent}</div>
        }
        return emptyContent
    }

    const chart = (
        <ChartContainer config={chartConfig} className={cn("w-full", compact ? "min-h-[200px] h-full" : "aspect-auto min-h-[200px] h-[360px]")}>
            <AreaChart data={samples}>
                <defs>
                    {PERF_METRICS.filter((m) => !m.dashed).map((metric) => (
                        <linearGradient key={metric.key} id={`fill-perf-${metric.key}`} x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor={`var(--color-${metric.key})`} stopOpacity={0.8} />
                            <stop offset="95%" stopColor={`var(--color-${metric.key})`} stopOpacity={0.1} />
                        </linearGradient>
                    ))}
                </defs>
                <CartesianGrid vertical={false} />
                <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={24} />

                <YAxis
                    yAxisId="percent"
                    orientation="left"
                    domain={[0, 100]}
                    tickLine={false}
                    axisLine={false}
                    width={40}
                    tickFormatter={(v) => `${v}%`}
                />

                <YAxis
                    yAxisId="count"
                    orientation="right"
                    domain={[0, maxCountY]}
                    tickLine={false}
                    axisLine={false}
                    width={40}
                />

                <ChartTooltip
                    cursor={false}
                    content={
                        <ChartTooltipContent
                            labelFormatter={(value) => String(value)}
                            indicator="dot"
                            formatter={(value, _name, item) => {
                                const metric = PERF_METRICS.find((m) => m.key === item.dataKey)
                                if (!metric) return null
                                const formatted = (() => {
                                    if (typeof value !== "number" || Number.isNaN(value)) return String(value ?? "-")
                                    if (metric.unit.trim() === "%") return `${value.toFixed(2)}%`
                                    if (metric.unit.trim().length > 0) return `${value.toFixed(2)}${metric.unit}`
                                    return value.toFixed(0)
                                })()
                                return (
                                    <div className="flex w-full items-center gap-2">
                                        <span
                                            className="inline-block size-2.5 shrink-0 rounded-[2px]"
                                            style={{ backgroundColor: `var(--color-${metric.key})` }}
                                        />
                                        <span className="flex-1 text-muted-foreground">
                                            {metric.label}
                                            {metric.dashed ? " ⌇" : ""}
                                        </span>
                                        <span className="font-mono font-medium tabular-nums text-foreground">{formatted}</span>
                                    </div>
                                )
                            }}
                        />
                    }
                />

                {PERF_METRICS.map((metric) => (
                    <Area
                        key={metric.key}
                        yAxisId={metric.axis}
                        type="monotone"
                        dataKey={metric.key}
                        name={metric.label}
                        stroke={`var(--color-${metric.key})`}
                        fill={metric.dashed ? "transparent" : `url(#fill-perf-${metric.key})`}
                        strokeWidth={2}
                        strokeDasharray={metric.dashed ? "6 3" : undefined}
                        fillOpacity={metric.dashed ? 0 : 0.4}
                        dot={false}
                        isAnimationActive={false}
                        connectNulls
                    />
                ))}

                <ChartLegend content={<ChartLegendContent />} />
            </AreaChart>
        </ChartContainer>
    )

    if (compact) {
        return <div className={cn("h-full", className)}>{chart}</div>
    }

    return chart
}
