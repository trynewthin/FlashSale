import { useEffect, useMemo, useState } from "react"
import { CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import type { RealtimeSample } from "@/features/realtime/types"
import { cn } from "@/lib/utils"

type MetricAxis = "percent" | "count"
type ChartMetricKey =
  | "portRate"
  | "httpRate"
  | "replicaRate"
  | "runningContainers"
  | "totalContainers"
  | "runningReplicas"
  | "totalReplicas"
  | "promQps"
  | "promP99LatencyMs"
  | "promErrorRate"

type MetricGroupKey = "server_metrics" | "infra_health" | "capacity" | "all"

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
  { key: "portRate", label: "端口可用率", color: "#0ea5e9", axis: "percent", unit: "%" },
  { key: "httpRate", label: "HTTP 健康率", color: "#10b981", axis: "percent", unit: "%" },
  { key: "replicaRate", label: "副本运行率", color: "#f97316", axis: "percent", unit: "%" },
  { key: "runningContainers", label: "运行容器数", color: "#6366f1", axis: "count", unit: "" },
  { key: "totalContainers", label: "总容器数", color: "#94a3b8", axis: "count", unit: "" },
  { key: "runningReplicas", label: "运行副本数", color: "#16a34a", axis: "count", unit: "" },
  { key: "totalReplicas", label: "总副本数", color: "#cbd5e1", axis: "count", unit: "" },
  { key: "promQps", label: "服务端 RPC QPS", color: "#7c3aed", axis: "count", unit: " req/s" },
  { key: "promP99LatencyMs", label: "服务端 RPC P99", color: "#db2777", axis: "count", unit: " ms" },
  { key: "promErrorRate", label: "服务端错误率", color: "#ef4444", axis: "percent", unit: "%" },
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
    key: "capacity",
    label: "容量状态",
    description: "观察容器与副本容量变化。",
    metrics: ["runningContainers", "totalContainers", "runningReplicas", "totalReplicas"],
  },
  {
    key: "all",
    label: "全部",
    description: "展示所有维度，用于综合研判。",
    metrics: METRICS.map((metric) => metric.key),
  },
]

const DEFAULT_GROUP_KEY: MetricGroupKey = "server_metrics"

interface RealtimeUnifiedChartProps {
  samples: RealtimeSample[]
}

function metricByKey(key: ChartMetricKey): MetricDef {
  return METRICS.find((metric) => metric.key === key) ?? METRICS[0]
}

function metricByLabel(label: string): MetricDef | undefined {
  return METRICS.find((metric) => metric.label === label)
}

function groupByKey(key: MetricGroupKey): MetricGroupDef {
  return GROUPS.find((group) => group.key === key) ?? GROUPS[0]
}

function readMetricValue(sample: RealtimeSample, key: ChartMetricKey): number {
  const raw = sample[key as keyof RealtimeSample]
  if (typeof raw === "number" && Number.isFinite(raw)) {
    return raw
  }
  return 0
}

// RealtimeUnifiedChart — 系统监控图（基础设施健康 + Prometheus 服务端指标）
export function RealtimeUnifiedChart({ samples }: RealtimeUnifiedChartProps) {
  const [activeGroup, setActiveGroup] = useState<MetricGroupKey>(DEFAULT_GROUP_KEY)
  const [visibleKeys, setVisibleKeys] = useState<ChartMetricKey[]>(() => groupByKey(DEFAULT_GROUP_KEY).metrics)

  const activeGroupDef = useMemo(() => groupByKey(activeGroup), [activeGroup])
  const groupMetrics = useMemo(
    () => activeGroupDef.metrics.map((key) => metricByKey(key)),
    [activeGroupDef.metrics]
  )

  useEffect(() => {
    setVisibleKeys(activeGroupDef.metrics)
  }, [activeGroupDef])

  const visibleMetrics = useMemo(() => {
    if (visibleKeys.length === 0) {
      return groupMetrics.slice(0, 1)
    }
    return groupMetrics.filter((metric) => visibleKeys.includes(metric.key))
  }, [groupMetrics, visibleKeys])

  const hasPercentMetric = visibleMetrics.some((metric) => metric.axis === "percent")
  const hasCountMetric = visibleMetrics.some((metric) => metric.axis === "count")

  const countMetricKeys = useMemo(
    () => visibleMetrics.filter((metric) => metric.axis === "count").map((metric) => metric.key),
    [visibleMetrics]
  )

  const maxCountY = useMemo(() => {
    if (countMetricKeys.length === 0) {
      return 1
    }
    const values = samples.flatMap((sample) => countMetricKeys.map((key) => readMetricValue(sample, key)))
    const maxValue = Math.max(1, ...values)
    return Math.ceil(maxValue * 1.2)
  }, [samples, countMetricKeys])

  const toggleMetric = (metricKey: ChartMetricKey) => {
    setVisibleKeys((prev) => {
      if (prev.includes(metricKey)) {
        if (prev.length <= 1) {
          return prev
        }
        return prev.filter((item) => item !== metricKey)
      }
      const nextSet = new Set(prev)
      nextSet.add(metricKey)
      return activeGroupDef.metrics.filter((key) => nextSet.has(key))
    })
  }

  return (
    <Card>
      <CardHeader className="space-y-3">
        <div className="space-y-1">
          <CardTitle>系统监控</CardTitle>
          <CardDescription>{activeGroupDef.description}</CardDescription>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {GROUPS.map((group) => (
            <Button
              key={group.key}
              size="sm"
              variant={activeGroup === group.key ? "secondary" : "outline"}
              className="h-8 px-3 text-xs"
              onClick={() => setActiveGroup(group.key)}
            >
              {group.label}
            </Button>
          ))}
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {groupMetrics.map((metric) => {
            const enabled = visibleKeys.includes(metric.key)
            return (
              <Button
                key={metric.key}
                size="sm"
                variant={enabled ? "secondary" : "outline"}
                className={cn("h-7 px-2 text-xs", !enabled && "text-muted-foreground")}
                onClick={() => toggleMetric(metric.key)}
              >
                <span className="mr-1.5 inline-block h-2 w-2 rounded-full" style={{ backgroundColor: metric.color }} />
                {metric.label}
              </Button>
            )
          })}
        </div>
      </CardHeader>

      <CardContent>
        <div className="h-[320px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={samples}>
              <CartesianGrid vertical={false} strokeDasharray="3 3" />
              <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={24} />

              {hasPercentMetric ? (
                <YAxis
                  yAxisId="percent"
                  orientation="left"
                  domain={[0, 100]}
                  tickLine={false}
                  axisLine={false}
                  width={40}
                  tickFormatter={(value) => `${value}%`}
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

              <Tooltip
                formatter={(value, name, item) => {
                  const rawDataKey = (item as { dataKey?: unknown } | undefined)?.dataKey
                  const metricFromDataKey =
                    typeof rawDataKey === "string" ? METRICS.find((metric) => metric.key === rawDataKey) : undefined
                  const metricFromLabel = typeof name === "string" ? metricByLabel(name) : undefined
                  const metric = metricFromDataKey ?? metricFromLabel ?? METRICS[0]
                  if (typeof value !== "number" || Number.isNaN(value)) {
                    return [String(value ?? "-"), metric.label]
                  }
                  if (metric.unit.trim() === "%") {
                    return [`${value.toFixed(2)}%`, metric.label]
                  }
                  if (metric.unit.trim().length > 0) {
                    return [`${value.toFixed(2)}${metric.unit}`, metric.label]
                  }
                  return [value.toFixed(0), metric.label]
                }}
              />

              <Legend />

              {visibleMetrics.map((metric) => (
                <Line
                  key={metric.key}
                  yAxisId={metric.axis}
                  type="monotone"
                  dataKey={metric.key}
                  name={metric.label}
                  stroke={metric.color}
                  strokeWidth={2}
                  dot={false}
                  isAnimationActive={false}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  )
}
