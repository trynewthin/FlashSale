import { useMemo } from "react"
import {
    Activity,
    AlertTriangle,
    CheckCircle2,
    Gauge,
    Package,
    Timer,
    Zap,
    Lightbulb,
    ShieldAlert,
} from "lucide-react"

import type { JobDetail } from "@/api/types"
import { Badge } from "@/components/ui/badge"
import {
    analyzePerfDiagnostics,
    type PerfDiagnosis,
    type PerfDiagnosisLevel,
} from "@/features/perf-test/perf-diagnosis"
import { computeAvg, computeMax } from "@/features/perf-test/perf-metric-cards"
import type { RealtimeSample } from "@/features/realtime/types"
import { cn } from "@/lib/utils"

// ─── KPI 状态项 ───

interface KpiItem {
    key: string
    label: string
    getValue: (samples: RealtimeSample[]) => number
    unit: string
}

const KPI_ITEMS: KpiItem[] = [
    { key: "avg_qps",     label: "平均 QPS",   getValue: (s) => computeAvg(s, "qps"),               unit: " req/s" },
    { key: "max_qps",     label: "峰值 QPS",   getValue: (s) => computeMax(s, "qps"),               unit: " req/s" },
    { key: "avg_p95",     label: "平均 P95",   getValue: (s) => computeAvg(s, "p95LatencyMs"),      unit: " ms"    },
    { key: "avg_success", label: "成功率",     getValue: (s) => computeAvg(s, "successRate"),       unit: "%"      },
    { key: "avg_sysErr",  label: "系统异常率", getValue: (s) => computeAvg(s, "systemErrorRate"),   unit: "%"      },
    { key: "avg_stock",   label: "库存扣减率", getValue: (s) => computeAvg(s, "stockDeductionRate"), unit: "%"     },
]

function formatKpiValue(value: number, unit: string): string {
    if (value === 0) return "—"
    if (unit.trim() === "%") return `${value.toFixed(1)}%`
    if (unit.includes("ms")) return `${value.toFixed(0)} ms`
    return `${value.toFixed(1)}${unit}`
}

// ─── 诊断级别样式 ───

function levelBg(level: PerfDiagnosisLevel): string {
    switch (level) {
        case "critical": return "border-red-300/50 bg-red-50/80 dark:border-red-700/40 dark:bg-red-950/40"
        case "high":     return "border-amber-300/50 bg-amber-50/80 dark:border-amber-700/40 dark:bg-amber-950/40"
        case "medium":   return "border-sky-300/50 bg-sky-50/80 dark:border-sky-700/40 dark:bg-sky-950/40"
        default:         return "border-emerald-300/50 bg-emerald-50/80 dark:border-emerald-700/40 dark:bg-emerald-950/40"
    }
}

function levelText(level: PerfDiagnosisLevel): string {
    switch (level) {
        case "critical": return "text-red-700 dark:text-red-300"
        case "high":     return "text-amber-700 dark:text-amber-300"
        case "medium":   return "text-sky-700 dark:text-sky-300"
        default:         return "text-emerald-700 dark:text-emerald-300"
    }
}

function levelBadgeVariant(level: PerfDiagnosisLevel) {
    switch (level) {
        case "critical": return "destructive" as const
        case "high":     return "secondary" as const
        default:         return "outline" as const
    }
}

function levelLabel(level: PerfDiagnosisLevel): string {
    switch (level) {
        case "critical": return "优先处理"
        case "high":     return "高优先级"
        case "medium":   return "关注"
        default:         return "正常"
    }
}

const LEVEL_ICONS: Record<PerfDiagnosisLevel, React.ElementType> = {
    critical: ShieldAlert,
    high:     AlertTriangle,
    medium:   Activity,
    info:     CheckCircle2,
}

// ─── 合并卡片 ───

interface PerfSummaryCardProps {
    samples: RealtimeSample[]
    job: JobDetail | null
    className?: string
}

export function PerfSummaryCard({ samples, job, className }: PerfSummaryCardProps) {
    const kpis = useMemo(
        () => KPI_ITEMS.map((item) => ({
            ...item,
            formatted: formatKpiValue(item.getValue(samples), item.unit),
        })),
        [samples]
    )

    const report = useMemo(() => analyzePerfDiagnostics(samples, job), [samples, job])
    const { primary, secondary } = report

    const DiagIcon = LEVEL_ICONS[primary.level]

    return (
        <div
            className={cn(
                "col-span-2 row-span-2 flex overflow-hidden rounded-xl border bg-card shadow-sm",
                className
            )}
        >
            {/* ── 左侧：KPI 状态网格 ── */}
            <div
                className="h-full shrink-0 p-2"
                style={{
                    display: "grid",
                    gridTemplateRows: "repeat(6, 1fr)",
                    gridAutoFlow: "column",
                    gridAutoColumns: "auto",
                    gap: "5px",
                }}
            >
                {kpis.map((kpi) => (
                    <div
                        key={kpi.key}
                        className="flex min-w-[9rem] items-center justify-between gap-4 rounded-lg border bg-muted/20 px-3 py-1.5"
                    >
                        <span className="text-[11px] whitespace-nowrap text-muted-foreground">
                            {kpi.label}
                        </span>
                        <span className="text-sm font-semibold tabular-nums text-foreground">
                            {kpi.formatted}
                        </span>
                    </div>
                ))}
            </div>

            {/* ── 右侧：分析面板（溢出时滚动，滚动条在卡片右边缘）── */}
            <div className="flex min-w-0 flex-1 flex-col gap-2 overflow-y-auto p-3">

                {/* 主诊断卡 */}
                <div className={cn("rounded-xl border p-3", levelBg(primary.level))}>
                    <div className="flex items-center gap-2">
                        <DiagIcon className={cn("size-4 shrink-0", levelText(primary.level))} />
                        <Badge variant={levelBadgeVariant(primary.level)} className="text-[10px]">
                            {levelLabel(primary.level)}
                        </Badge>
                        <span className={cn("truncate text-xs font-semibold", levelText(primary.level))}>
                            {primary.title}
                        </span>
                        <span className="ml-auto shrink-0 text-[10px] text-muted-foreground">
                            置信度 {Math.round(primary.confidence * 100)}%
                        </span>
                    </div>
                    <p className="mt-2 text-[11px] leading-[1.6] text-foreground/75">
                        {primary.summary}
                    </p>
                </div>

                {/* 证据指标卡 */}
                <div className="rounded-xl border bg-muted/20 p-3">
                    <div className="mb-2 flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
                        <Gauge className="size-3" />
                        证据指标
                    </div>
                    <div className="flex flex-wrap gap-1.5">
                        {primary.evidence.map((e) => (
                            <Badge key={e} variant="outline" className="rounded-md px-2 py-0.5 text-[11px] font-mono">
                                {e}
                            </Badge>
                        ))}
                    </div>
                </div>

                {/* 次要信号 */}
                {secondary.length > 0 && (
                    <div className="space-y-1.5">
                        {secondary.map((s) => (
                            <SecondarySignal key={s.kind} diagnosis={s} />
                        ))}
                    </div>
                )}

                {/* 建议行动 */}
                {primary.suggestions.length > 0 && (
                    <div className="rounded-xl border bg-muted/20 p-3">
                        <div className="mb-2 flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
                            <Lightbulb className="size-3" />
                            建议行动
                        </div>
                        <div className="space-y-1.5">
                            {primary.suggestions.map((s, i) => (
                                <div key={i} className="flex gap-2 text-[11px] leading-[1.6] text-foreground/80">
                                    <span className="mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full bg-muted text-[9px] font-bold text-muted-foreground">
                                        {i + 1}
                                    </span>
                                    <span>{s}</span>
                                </div>
                            ))}
                        </div>
                    </div>
                )}
            </div>
        </div>
    )
}

// ─── 次要信号条 ───

function SecondarySignal({ diagnosis }: { diagnosis: PerfDiagnosis }) {
    const Icon = LEVEL_ICONS[diagnosis.level]
    return (
        <div className={cn("flex items-start gap-2 rounded-xl border p-2.5", levelBg(diagnosis.level))}>
            <Icon className={cn("mt-0.5 size-3.5 shrink-0", levelText(diagnosis.level))} />
            <div className="min-w-0">
                <div className="flex items-center gap-1.5">
                    <Badge variant={levelBadgeVariant(diagnosis.level)} className="text-[9px]">
                        {levelLabel(diagnosis.level)}
                    </Badge>
                    <span className={cn("text-[11px] font-semibold truncate", levelText(diagnosis.level))}>
                        {diagnosis.title}
                    </span>
                </div>
                <p className="mt-1 text-[10px] leading-[1.6] text-muted-foreground">{diagnosis.summary}</p>
            </div>
        </div>
    )
}
