import { useMemo } from "react"
import {
    ArrowDown,
    ArrowRight,
    ArrowUp,
    Gauge,
    Timer,
    Zap,
    CheckCircle2,
    AlertTriangle,
    Package,
} from "lucide-react"

import type { RealtimeSample } from "@/features/realtime/types"
import { computeAvg, computeMax } from "@/features/realtime/perf-metric-cards"
import { cn } from "@/lib/utils"

// 单个 KPI 定义
interface KpiItem {
    key: string
    label: string
    icon: React.ElementType
    color: string
    getValue: (samples: RealtimeSample[]) => number
    unit: string
    format?: (v: number) => string
    /** 趋势方向：up=越高越好, down=越低越好 */
    trendDirection: "up" | "down"
}

const KPI_ITEMS: KpiItem[] = [
    {
        key: "avg_qps",
        label: "平均 QPS",
        icon: Zap,
        color: "hsl(210, 95%, 55%)",
        getValue: (s) => computeAvg(s, "qps"),
        unit: " req/s",
        trendDirection: "up",
    },
    {
        key: "max_qps",
        label: "峰值 QPS",
        icon: Gauge,
        color: "hsl(260, 70%, 60%)",
        getValue: (s) => computeMax(s, "qps"),
        unit: " req/s",
        trendDirection: "up",
    },
    {
        key: "avg_p95",
        label: "平均 P95",
        icon: Timer,
        color: "hsl(30, 90%, 55%)",
        getValue: (s) => computeAvg(s, "p95LatencyMs"),
        unit: " ms",
        trendDirection: "down",
    },
    {
        key: "avg_success",
        label: "平均成功率",
        icon: CheckCircle2,
        color: "hsl(142, 70%, 45%)",
        getValue: (s) => computeAvg(s, "successRate"),
        unit: "%",
        trendDirection: "up",
    },
    {
        key: "avg_sysError",
        label: "系统异常率",
        icon: AlertTriangle,
        color: "hsl(0, 80%, 55%)",
        getValue: (s) => computeAvg(s, "systemErrorRate"),
        unit: "%",
        trendDirection: "down",
    },
    {
        key: "avg_stock",
        label: "平均扣减率",
        icon: Package,
        color: "hsl(190, 80%, 45%)",
        getValue: (s) => computeAvg(s, "stockDeductionRate"),
        unit: "%",
        trendDirection: "up",
    },
]

// 趋势计算：对比前半段和后半段的差值
function computeTrend(
    samples: RealtimeSample[],
    _metricKey: string,
    samplesFn: (s: RealtimeSample[]) => number
): "up" | "down" | "flat" {
    if (samples.length < 4) return "flat"
    const mid = Math.floor(samples.length / 2)
    const firstHalf = samplesFn(samples.slice(0, mid))
    const secondHalf = samplesFn(samples.slice(mid))
    const diff = secondHalf - firstHalf
    const threshold = Math.abs(firstHalf) * 0.05 || 0.1
    if (diff > threshold) return "up"
    if (diff < -threshold) return "down"
    return "flat"
}

interface PerfSummaryCardProps {
    samples: RealtimeSample[]
    className?: string
}

export function PerfSummaryCard({ samples, className }: PerfSummaryCardProps) {
    const kpis = useMemo(
        () =>
            KPI_ITEMS.map((item) => {
                const value = item.getValue(samples)
                const trend = computeTrend(samples, item.key, item.getValue)
                return { ...item, value, trend }
            }),
        [samples]
    )

    return (
        <div
            className={cn(
                "flex flex-col overflow-hidden rounded-xl border bg-card shadow-sm",
                className
            )}
        >
            {/* 标题 */}
            <div className="border-b px-3 py-1.5">
                <div className="text-xs font-semibold text-foreground">测试摘要</div>
                <div className="text-[10px] text-muted-foreground">
                    基于 {samples.length} 个采样点
                </div>
            </div>

            {/* KPI 网格 — 3 列紧凑布局 */}
            <div className="grid min-h-0 flex-1 grid-cols-3 gap-px overflow-hidden bg-border">
                {kpis.map((kpi) => (
                    <KpiCell key={kpi.key} kpi={kpi} />
                ))}
            </div>
        </div>
    )
}

// ─── 单个 KPI 格子 ───

interface KpiCellProps {
    kpi: KpiItem & { value: number; trend: "up" | "down" | "flat" }
}

function KpiCell({ kpi }: KpiCellProps) {
    const Icon = kpi.icon
    const formatted = formatKpiValue(kpi.value, kpi.unit)

    // 趋势颜色
    const trendIsGood =
        (kpi.trendDirection === "up" && kpi.trend === "up") ||
        (kpi.trendDirection === "down" && kpi.trend === "down")
    const trendIsBad =
        (kpi.trendDirection === "up" && kpi.trend === "down") ||
        (kpi.trendDirection === "down" && kpi.trend === "up")

    const TrendIcon =
        kpi.trend === "up" ? ArrowUp : kpi.trend === "down" ? ArrowDown : ArrowRight

    return (
        <div className="flex items-center gap-2 bg-card px-2.5 py-1.5">
            <div
                className="flex size-6 shrink-0 items-center justify-center rounded-md"
                style={{ backgroundColor: kpi.color + "18" }}
            >
                <Icon className="size-3" style={{ color: kpi.color }} />
            </div>
            <div className="min-w-0 flex-1">
                <div className="text-[9px] leading-tight text-muted-foreground">{kpi.label}</div>
                <div className="flex items-baseline gap-1">
                    <span className="text-sm font-bold tabular-nums leading-tight text-foreground">
                        {formatted}
                    </span>
                    {kpi.trend !== "flat" && (
                        <TrendIcon
                            className={cn(
                                "size-2.5",
                                trendIsGood && "text-emerald-500",
                                trendIsBad && "text-red-400",
                                !trendIsGood && !trendIsBad && "text-muted-foreground"
                            )}
                        />
                    )}
                </div>
            </div>
        </div>
    )
}

function formatKpiValue(value: number, unit: string): string {
    if (value === 0) return "—"
    if (unit.trim() === "%") return `${value.toFixed(1)}%`
    if (unit.includes("ms")) return `${value.toFixed(0)} ms`
    return `${value.toFixed(1)}${unit}`
}
