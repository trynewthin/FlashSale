import { useMemo } from "react"
import { CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import type { RealtimeSample } from "@/features/realtime/types"

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
    { key: "qps", label: "客户端 QPS", color: "#7c3aed", axis: "count", unit: " req/s" },
    { key: "p95LatencyMs", label: "客户端 P95", color: "#db2777", axis: "count", unit: " ms" },
    { key: "successRate", label: "客户端成功率", color: "#059669", axis: "percent", unit: "%" },
    { key: "errorRate", label: "客户端错误率", color: "#ef4444", axis: "percent", unit: "%" },
    { key: "networkErrorRate", label: "网络错误率", color: "#f59e0b", axis: "percent", unit: "%" },
    { key: "stockDeductionRate", label: "库存扣减率", color: "#14b8a6", axis: "percent", unit: "%" },
    // 服务端视角（来自 Prometheus，虚线）
    { key: "promQps", label: "服务端 QPS", color: "#a78bfa", axis: "count", unit: " req/s", dashed: true },
    { key: "promP99LatencyMs", label: "服务端 P99", color: "#f472b6", axis: "count", unit: " ms", dashed: true },
    { key: "promErrorRate", label: "服务端错误率", color: "#fca5a5", axis: "percent", unit: "%", dashed: true },
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
}

// RealtimePerfChart — 压测分析图，客户端实线 vs 服务端虚线双视角对比
export function RealtimePerfChart({ samples }: RealtimePerfChartProps) {
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

    if (!hasData) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle className="text-sm">压测分析</CardTitle>
                    <CardDescription>启动压测后自动展示客户端 vs 服务端双视角对比图。</CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="flex h-[200px] items-center justify-center rounded-lg border border-dashed bg-muted/20 text-sm text-muted-foreground">
                        等待压测数据…
                    </div>
                </CardContent>
            </Card>
        )
    }

    return (
        <Card>
            <CardHeader className="space-y-1">
                <CardTitle className="text-sm">压测分析 — 双视角对比</CardTitle>
                <CardDescription>
                    实线 = 客户端（seckillload 观测） · 虚线 = 服务端（Prometheus 采集）
                </CardDescription>
            </CardHeader>

            <CardContent>
                <div className="h-[360px] w-full">
                    <ResponsiveContainer width="100%" height="100%">
                        <LineChart data={samples}>
                            <CartesianGrid vertical={false} strokeDasharray="3 3" />
                            <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={24} />

                            <YAxis
                                yAxisId="percent"
                                orientation="left"
                                domain={[0, 100]}
                                tickLine={false}
                                axisLine={false}
                                width={40}
                                tickFormatter={(value) => `${value}%`}
                            />

                            <YAxis
                                yAxisId="count"
                                orientation="right"
                                domain={[0, maxCountY]}
                                tickLine={false}
                                axisLine={false}
                                width={40}
                            />

                            <Tooltip
                                formatter={(value, name, item) => {
                                    const rawDataKey = (item as { dataKey?: unknown } | undefined)?.dataKey
                                    const metric = typeof rawDataKey === "string"
                                        ? PERF_METRICS.find((m) => m.key === rawDataKey)
                                        : PERF_METRICS.find((m) => m.label === name)
                                    const def = metric ?? PERF_METRICS[0]
                                    if (typeof value !== "number" || Number.isNaN(value)) {
                                        return [String(value ?? "-"), def.label]
                                    }
                                    if (def.unit.trim() === "%") {
                                        return [`${value.toFixed(2)}%`, def.label]
                                    }
                                    if (def.unit.trim().length > 0) {
                                        return [`${value.toFixed(2)}${def.unit}`, def.label]
                                    }
                                    return [value.toFixed(0), def.label]
                                }}
                            />

                            <Legend />

                            {PERF_METRICS.map((metric) => (
                                <Line
                                    key={metric.key}
                                    yAxisId={metric.axis}
                                    type="monotone"
                                    dataKey={metric.key}
                                    name={metric.label}
                                    stroke={metric.color}
                                    strokeWidth={2}
                                    strokeDasharray={metric.dashed ? "6 3" : undefined}
                                    dot={false}
                                    isAnimationActive={false}
                                    connectNulls
                                />
                            ))}
                        </LineChart>
                    </ResponsiveContainer>
                </div>
            </CardContent>
        </Card>
    )
}
