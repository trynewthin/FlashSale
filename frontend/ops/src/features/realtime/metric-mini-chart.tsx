import { useMemo } from "react"
import { Area, AreaChart, ResponsiveContainer, YAxis } from "recharts"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart"
import type { RealtimeSample } from "@/features/realtime/types"
import type { MetricDef } from "@/features/realtime/realtime-unified-chart"

interface MetricMiniChartProps {
  metric: MetricDef
  samples: RealtimeSample[]
}

function readMetricValue(sample: RealtimeSample, key: string): number {
  const raw = sample[key as keyof RealtimeSample]
  return typeof raw === "number" && Number.isFinite(raw) ? raw : 0
}

export function MetricMiniChart({ metric, samples }: MetricMiniChartProps) {
  const config = useMemo(() => {
    return {
      [metric.key]: {
        label: metric.label,
        color: metric.color,
      },
    } satisfies ChartConfig
  }, [metric])

  const latestValue = useMemo(() => {
    if (samples.length === 0) return 0
    return readMetricValue(samples[samples.length - 1], metric.key)
  }, [samples, metric.key])

  const formattedValue = useMemo(() => {
    if (metric.key === "promP99LatencyMs" && latestValue === 0 && samples.length > 0 && samples[samples.length - 1].promP99LatencyMs == null) {
      return "样本不足"
    }
    if (metric.unit.trim() === "%") return `${latestValue.toFixed(2)}%`
    if (metric.unit.trim().length > 0) return `${latestValue.toFixed(2)}${metric.unit}`
    return latestValue.toFixed(0)
  }, [latestValue, metric, samples])

  return (
    <Card className="flex flex-col overflow-hidden shadow-sm border border-border/40 hover:border-border/80 transition-colors h-full">
      <CardHeader className="flex flex-row items-center justify-between pb-0 space-y-0 px-4">
        <CardTitle className="text-sm font-medium text-muted-foreground">{metric.label}</CardTitle>
        <div className="text-sm font-bold tabular-nums tracking-tight text-right">{formattedValue}</div>
      </CardHeader>
      <CardContent className="p-0 flex-1 min-h-[60px]">
        <ChartContainer config={config} className="h-full w-full aspect-auto">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={samples} margin={{ top: 5, right: 0, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id={`fill-mini-${metric.key}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={`var(--color-${metric.key})`} stopOpacity={0.4} />
                  <stop offset="95%" stopColor={`var(--color-${metric.key})`} stopOpacity={0.0} />
                </linearGradient>
              </defs>
              <YAxis 
                hide 
                domain={metric.axis === "percent" ? [0, 100] : ["auto", "auto"]} 
              />
              <ChartTooltip
                cursor={false}
                content={<ChartTooltipContent indicator="line" hideLabel />}
              />
              <Area
                type="monotone"
                dataKey={metric.key}
                stroke={`var(--color-${metric.key})`}
                fill={`url(#fill-mini-${metric.key})`}
                strokeWidth={2}
                dot={false}
                isAnimationActive={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </ChartContainer>
      </CardContent>
    </Card>
  )
}
