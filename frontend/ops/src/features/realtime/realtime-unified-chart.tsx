/* eslint-disable react-refresh/only-export-components */

import { useMemo } from "react"
import { Area, AreaChart, CartesianGrid, XAxis, YAxis, ResponsiveContainer } from "recharts"

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

export type MetricAxis = "percent" | "count"
export type ChartMetricKey =
  | "portRate"
  | "httpRate"
  | "replicaRate"
  | "promQps"
  | "promP99LatencyMs"
  | "promErrorRate"
  | "purchaseKafkaPublishRate"
  | "purchaseKafkaPublishFailed"
  | "orderStateConsumeRate"

export type MetricGroupKey = "server_metrics" | "infra_health" | "kafka_pipeline" | "all"

export interface MetricDef {
  key: ChartMetricKey
  label: string
  color: string
  axis: MetricAxis
  unit: string
}

export interface MetricGroupDef {
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
  { key: "purchaseKafkaPublishRate", label: "Kafka 发布速率", color: "hsl(200 80% 50%)", axis: "count", unit: " req/s" },
  { key: "orderStateConsumeRate", label: "订单状态消费速率", color: "hsl(200 60% 70%)", axis: "count", unit: " req/s" },
  { key: "purchaseKafkaPublishFailed", label: "Kafka 发布失败数", color: "hsl(0 85% 60%)", axis: "count", unit: "" },
]

const GROUPS: MetricGroupDef[] = [
  {
    key: "server_metrics",
    label: "服务端指标",
    description: "Prometheus 采集的 RPC QPS、P99 延迟和错误率。",
    metrics: ["promQps", "promP99LatencyMs", "promErrorRate"],
  },
  {
    key: "infra_health",
    label: "系统健康",
    description: "端口、HTTP 和副本健康视图。",
    metrics: ["portRate", "httpRate", "replicaRate"],
  },
  {
    key: "kafka_pipeline",
    label: "Kafka 链路",
    description: "秒杀建单发布与订单状态回流的 Kafka 指标。",
    metrics: ["purchaseKafkaPublishRate", "orderStateConsumeRate", "purchaseKafkaPublishFailed"],
  },
  {
    key: "all",
    label: "全部",
    description: "展示全部实时监控维度。",
    metrics: METRICS.map((metric) => metric.key),
  },
]


export function metricByKey(key: ChartMetricKey): MetricDef {
  return METRICS.find((metric) => metric.key === key) ?? METRICS[0]
}

export function groupByKey(key: MetricGroupKey): MetricGroupDef {
  return GROUPS.find((group) => group.key === key) ?? GROUPS[0]
}

function readMetricValue(sample: RealtimeSample, key: ChartMetricKey): number {
  const raw = sample[key as keyof RealtimeSample]
  return typeof raw === "number" && Number.isFinite(raw) ? raw : 0
}



interface RealtimeUnifiedChartProps {
  samples: RealtimeSample[]
  visibleKeys: ChartMetricKey[]
  mode?: "live" | "replay"
  className?: string
}

export function RealtimeUnifiedChart({ samples, visibleKeys, mode = "live", className }: RealtimeUnifiedChartProps) {
  const visibleMetrics = useMemo(() => {
    const defs = visibleKeys.map((key) => metricByKey(key))
    return defs.length > 0 ? defs : [METRICS[0]]
  }, [visibleKeys])

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
    return Math.ceil(Math.max(1, ...values) * 1.2)
  }, [countMetricKeys, samples])

  const chartConfig: ChartConfig = useMemo(() => {
    const config: ChartConfig = {}
    for (const metric of visibleMetrics) {
      config[metric.key] = { label: metric.label, color: metric.color }
    }
    return config
  }, [visibleMetrics])

  return (
    <ChartContainer config={chartConfig} className={cn("min-h-[200px] w-full !aspect-auto", className)} data-mode={mode}>
      <ResponsiveContainer width="100%" height="100%">
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

          <ChartTooltip
            cursor={false}
            content={
              <ChartTooltipContent
                labelFormatter={(value) => String(value)}
                indicator="dot"
                formatter={(value, _name, item) => {
                  const metric = METRICS.find((candidate) => candidate.key === item.dataKey)
                  if (!metric) {
                    return null
                  }
                  const formatted = (() => {
                    if (metric.key === "promP99LatencyMs" && value == null) {
                      return "样本不足"
                    }
                    if (typeof value !== "number" || Number.isNaN(value)) {
                      return String(value ?? "-")
                    }
                    if (metric.unit.trim() === "%") {
                      return `${value.toFixed(2)}%`
                    }
                    if (metric.unit.trim().length > 0) {
                      return `${value.toFixed(2)}${metric.unit}`
                    }
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

          <ChartLegend verticalAlign="top" align="left" content={<ChartLegendContent className="-ml-1 justify-start" />} wrapperStyle={{ paddingBottom: 16 }} />
        </AreaChart>
      </ResponsiveContainer>
    </ChartContainer>
  )
}
