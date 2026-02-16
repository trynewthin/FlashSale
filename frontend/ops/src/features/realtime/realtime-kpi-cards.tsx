import { Activity, Cpu, Gauge, Server } from "lucide-react"

import type { PerfTestMetricPoint } from "@/features/realtime/perf-report-parser"
import type { RealtimeSample } from "@/features/realtime/types"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

interface RealtimeKPICardsProps {
  latest: RealtimeSample | null
  previous: RealtimeSample | null
  latestTestMetric?: PerfTestMetricPoint | null
  previousTestMetric?: PerfTestMetricPoint | null
}

function deltaText(current: number, previous?: number, unit = ""): string {
  if (previous === undefined) {
    return "-"
  }
  const delta = current - previous
  const sign = delta > 0 ? "+" : ""
  return `${sign}${delta.toFixed(2)}${unit}`
}

// RealtimeKPICards 展示实时核心指标与短周期变化。
export function RealtimeKPICards({ latest, previous, latestTestMetric, previousTestMetric }: RealtimeKPICardsProps) {
  const prev = previous ?? undefined

  const infraCards = [
    {
      key: "port",
      title: "端口可用率",
      icon: Activity,
      value: latest ? `${latest.portRate.toFixed(2)}%` : "-",
      delta: latest ? deltaText(latest.portRate, prev?.portRate, "%") : "-",
    },
    {
      key: "http",
      title: "HTTP 健康率",
      icon: Gauge,
      value: latest ? `${latest.httpRate.toFixed(2)}%` : "-",
      delta: latest ? deltaText(latest.httpRate, prev?.httpRate, "%") : "-",
    },
    {
      key: "containers",
      title: "运行容器数",
      icon: Server,
      value: latest ? `${latest.runningContainers}/${latest.totalContainers}` : "-",
      delta: latest ? deltaText(latest.runningContainers, prev?.runningContainers) : "-",
    },
    {
      key: "replicas",
      title: "副本运行率",
      icon: Cpu,
      value: latest ? `${latest.replicaRate.toFixed(2)}%` : "-",
      delta: latest ? deltaText(latest.replicaRate, prev?.replicaRate, "%") : "-",
    },
  ] as const
  const testCards = latestTestMetric
    ? [
        {
          key: "test-qps",
          title: "测试 QPS",
          value: latestTestMetric.qps.toFixed(2),
          delta: deltaText(latestTestMetric.qps, previousTestMetric?.qps),
        },
        {
          key: "test-success",
          title: "测试成功率",
          value: `${latestTestMetric.successRate.toFixed(2)}%`,
          delta: deltaText(latestTestMetric.successRate, previousTestMetric?.successRate, "%"),
        },
        {
          key: "test-error",
          title: "测试错误率",
          value: `${latestTestMetric.errorRate.toFixed(2)}%`,
          delta: deltaText(latestTestMetric.errorRate, previousTestMetric?.errorRate, "%"),
        },
        {
          key: "test-p95",
          title: "测试 P95",
          value: `${latestTestMetric.p95LatencyMs.toFixed(2)}ms`,
          delta: deltaText(latestTestMetric.p95LatencyMs, previousTestMetric?.p95LatencyMs, "ms"),
        },
        {
          key: "test-stock",
          title: "库存扣减成功率",
          value: `${latestTestMetric.stockDeductionRate.toFixed(2)}%`,
          delta: deltaText(latestTestMetric.stockDeductionRate, previousTestMetric?.stockDeductionRate, "%"),
        },
      ]
    : []
  const cards = [...infraCards, ...testCards]

  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
      {cards.map((item) => {
        return (
          <Card key={item.key}>
            <CardHeader className="pb-2">
              <CardTitle className="flex items-center justify-between text-sm font-medium">
                {item.title}
                {"icon" in item && item.icon ? <item.icon className="size-4 text-muted-foreground" /> : null}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-1">
              <div className="text-2xl font-semibold tabular-nums">{item.value}</div>
              <Badge variant="outline" className="tabular-nums">
                较上一点 {item.delta}
              </Badge>
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}
