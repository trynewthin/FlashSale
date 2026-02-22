import { useMemo, useState, useCallback } from "react"
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts"

import { Button } from "@/components/ui/button"
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { RealtimeSample } from "@/features/realtime/types"
import { cn } from "@/lib/utils"

// ─── 指标定义 ───

type MetricAxis = "percent" | "count"
type ChartMetricKey =
  | "portRate"
  | "httpRate"
  | "replicaRate"
  | "promQps"
  | "promP99LatencyMs"
  | "promErrorRate"

type MetricGroupKey = "server_metrics" | "infra_health" | "all"

interface MetricDef {
  key: ChartMetricKey
  label: string
  color: string
  axis: MetricAxis
  unit: string
}

interface MetricGroupDef {
  key: MetricGroupKey
  label: string
  description: string
  metrics: ChartMetricKey[]
}

const METRICS: MetricDef[] = [
  { key: "portRate", label: "端口可用率", color: "var(--chart-1)", axis: "percent", unit: "%" },
  { key: "httpRate", label: "HTTP 健康率", color: "var(--chart-2)", axis: "percent", unit: "%" },
  { key: "replicaRate", label: "副本运行率", color: "var(--chart-8)", axis: "percent", unit: "%" },
  { key: "promQps", label: "服务端 RPC QPS", color: "var(--chart-3)", axis: "count", unit: " req/s" },
  { key: "promP99LatencyMs", label: "服务端 RPC P99", color: "var(--chart-7)", axis: "count", unit: " ms" },
  { key: "promErrorRate", label: "服务端错误率", color: "var(--chart-5)", axis: "percent", unit: "%" },
]

const GROUPS: MetricGroupDef[] = [
  {
    key: "server_metrics",
    label: "服务端指标",
    description: "Prometheus 采集：RPC QPS / P99 延迟 / 错误率。",
    metrics: ["promQps", "promP99LatencyMs", "promErrorRate"],
  },
  {
    key: "infra_health",
    label: "系统健康",
    description: "关注端口与健康检查可用性。",
    metrics: ["portRate", "httpRate", "replicaRate"],
  },
  {
    key: "all",
    label: "全部",
    description: "展示所有维度，用于综合研判。",
    metrics: METRICS.map((m) => m.key),
  },
]

const DEFAULT_GROUP_KEY: MetricGroupKey = "server_metrics"

function metricByKey(key: ChartMetricKey): MetricDef {
  return METRICS.find((m) => m.key === key) ?? METRICS[0]
}

function groupByKey(key: MetricGroupKey): MetricGroupDef {
  return GROUPS.find((g) => g.key === key) ?? GROUPS[0]
}

function readMetricValue(sample: RealtimeSample, key: ChartMetricKey): number {
  const raw = sample[key as keyof RealtimeSample]
  if (typeof raw === "number" && Number.isFinite(raw)) {
    return raw
  }
  return 0
}

// ─── 自定义 Hook：统一管理指标分组与可见性 ───

export function useChartMetrics() {
  const [activeGroup, setActiveGroup] = useState<MetricGroupKey>(DEFAULT_GROUP_KEY)
  const [visibleKeys, setVisibleKeys] = useState<ChartMetricKey[]>(() => groupByKey(DEFAULT_GROUP_KEY).metrics)

  const activeGroupDef = useMemo(() => groupByKey(activeGroup), [activeGroup])
  const groupMetrics = useMemo(
    () => activeGroupDef.metrics.map((key) => metricByKey(key)),
    [activeGroupDef.metrics]
  )

  const handleGroupChange = useCallback((key: MetricGroupKey) => {
    setActiveGroup(key)
    setVisibleKeys(groupByKey(key).metrics)
  }, [])

  const toggleMetric = useCallback((metricKey: ChartMetricKey) => {
    setVisibleKeys((prev) => {
      if (prev.includes(metricKey)) {
        if (prev.length <= 1) return prev
        return prev.filter((k) => k !== metricKey)
      }
      const nextSet = new Set(prev)
      nextSet.add(metricKey)
      // 保持在当前分组中的顺序
      const currentGroupMetrics = groupByKey(activeGroup).metrics
      return currentGroupMetrics.filter((k) => nextSet.has(k))
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeGroup])

  return { activeGroup, activeGroupDef, groupMetrics, visibleKeys, handleGroupChange, toggleMetric }
}

// ─── 筛选控制面板（独立组件，用于页面下方） ───

interface MetricFilterPanelProps {
  activeGroup: MetricGroupKey
  groupMetrics: MetricDef[]
  visibleKeys: ChartMetricKey[]
  onGroupChange: (key: MetricGroupKey) => void
  onToggleMetric: (key: ChartMetricKey) => void
}

export function MetricFilterPanel({
  activeGroup,
  groupMetrics,
  visibleKeys,
  onGroupChange,
  onToggleMetric,
}: MetricFilterPanelProps) {
  return (
    <div className="flex h-full flex-col gap-3">
      {/* 上方：可滚动的指标泳道 */}
      <div className="max-h-[88px] overflow-y-auto">
        <div className="flex flex-wrap items-center gap-1.5">
          {groupMetrics.map((metric) => {
            const enabled = visibleKeys.includes(metric.key)
            return (
              <Button
                key={metric.key}
                size="sm"
                variant={enabled ? "secondary" : "outline"}
                className={cn("h-7 px-2.5 text-xs", !enabled && "text-muted-foreground")}
                onClick={() => onToggleMetric(metric.key)}
              >
                <span className="mr-1.5 inline-block size-2 rounded-full" style={{ backgroundColor: metric.color }} />
                {metric.label}
              </Button>
            )
          })}
        </div>
      </div>
      {/* 下方：指标分组选择 */}
      <Select value={activeGroup} onValueChange={(v) => onGroupChange(v as MetricGroupKey)}>
        <SelectTrigger className="h-8 w-full text-xs">
          <SelectValue placeholder="选择分组" />
        </SelectTrigger>
        <SelectContent className="rounded-xl">
          {GROUPS.map((g) => (
            <SelectItem key={g.key} value={g.key} className="rounded-lg text-xs">
              {g.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

// ─── 纯粹的图表组件（无 Card 包裹） ───

interface RealtimeUnifiedChartProps {
  samples: RealtimeSample[]
  visibleKeys: ChartMetricKey[]
  className?: string
}

// RealtimeUnifiedChart — 纯 AreaChart，由父级控制布局。
export function RealtimeUnifiedChart({ samples, visibleKeys, className }: RealtimeUnifiedChartProps) {
  const visibleMetrics = useMemo(() => {
    const defs = visibleKeys.map((key) => metricByKey(key))
    return defs.length > 0 ? defs : [METRICS[0]]
  }, [visibleKeys])

  const hasPercentMetric = visibleMetrics.some((m) => m.axis === "percent")
  const hasCountMetric = visibleMetrics.some((m) => m.axis === "count")

  const countMetricKeys = useMemo(
    () => visibleMetrics.filter((m) => m.axis === "count").map((m) => m.key),
    [visibleMetrics]
  )

  const maxCountY = useMemo(() => {
    if (countMetricKeys.length === 0) return 1
    const values = samples.flatMap((sample) => countMetricKeys.map((key) => readMetricValue(sample, key)))
    const maxValue = Math.max(1, ...values)
    return Math.ceil(maxValue * 1.2)
  }, [samples, countMetricKeys])

  const chartConfig: ChartConfig = useMemo(() => {
    const config: ChartConfig = {}
    for (const metric of visibleMetrics) {
      config[metric.key] = { label: metric.label, color: metric.color }
    }
    return config
  }, [visibleMetrics])

  return (
    <ChartContainer config={chartConfig} className={cn("min-h-[200px] w-full", className)}>
      <AreaChart data={samples}>
        <defs>
          {visibleMetrics.map((metric) => (
            <linearGradient key={metric.key} id={`fill-${metric.key}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={`var(--color-${metric.key})`} stopOpacity={0.8} />
              <stop offset="95%" stopColor={`var(--color-${metric.key})`} stopOpacity={0.1} />
            </linearGradient>
          ))}
        </defs>
        <CartesianGrid vertical={false} />
        <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={24} />

        {hasPercentMetric ? (
          <YAxis
            yAxisId="percent"
            orientation="left"
            domain={[0, 100]}
            tickLine={false}
            axisLine={false}
            width={40}
            tickFormatter={(v) => `${v}%`}
          />
        ) : null}

        {hasCountMetric ? (
          <YAxis
            yAxisId="count"
            orientation="right"
            domain={[0, maxCountY]}
            tickLine={false}
            axisLine={false}
            width={40}
          />
        ) : null}

        <ChartTooltip
          cursor={false}
          content={
            <ChartTooltipContent
              labelFormatter={(value) => String(value)}
              indicator="dot"
              formatter={(value, _name, item) => {
                const metric = METRICS.find((m) => m.key === item.dataKey)
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
                    <span className="flex-1 text-muted-foreground">{metric.label}</span>
                    <span className="font-mono font-medium tabular-nums text-foreground">{formatted}</span>
                  </div>
                )
              }}
            />
          }
        />

        {visibleMetrics.map((metric) => (
          <Area
            key={metric.key}
            yAxisId={metric.axis}
            type="monotone"
            dataKey={metric.key}
            name={metric.label}
            stroke={`var(--color-${metric.key})`}
            fill={`url(#fill-${metric.key})`}
            strokeWidth={2}
            dot={false}
            isAnimationActive={false}
          />
        ))}

        <ChartLegend content={<ChartLegendContent />} />
      </AreaChart>
    </ChartContainer>
  )
}
