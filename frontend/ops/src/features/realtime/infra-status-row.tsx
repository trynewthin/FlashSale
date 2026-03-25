import { Activity, Cpu, Gauge, Server, HardDrive } from "lucide-react"

import type { SysInfo } from "@/api/types"
import type { RealtimeSample } from "@/features/realtime/types"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

interface InfraStatusRowProps {
  latest: RealtimeSample | null
  previous: RealtimeSample | null
  sysInfo: SysInfo | null
}

function ResourceRow({ label, pct }: { label: string; pct: number }) {
  const tone = pct >= 90 ? "bg-destructive" : pct >= 70 ? "bg-amber-500" : "bg-primary"
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between text-[13px]">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-medium tabular-nums text-foreground">{pct.toFixed(1)}%</span>
      </div>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
        <div className={`h-full rounded-full transition-all ${tone}`} style={{ width: `${Math.min(pct, 100)}%` }} />
      </div>
    </div>
  )
}

export function InfraStatusRow({ latest, sysInfo }: InfraStatusRowProps) {
  const regularCards = [
    {
      key: "port",
      title: "端口可用率",
      icon: Activity,
      value: latest ? `${latest.portRate.toFixed(2)}%` : "-",
    },
    {
      key: "http",
      title: "HTTP 健康度",
      icon: Gauge,
      value: latest ? `${latest.httpRate.toFixed(2)}%` : "-",
    },
    {
      key: "containers",
      title: "集群可用容量",
      icon: Server,
      value: latest ? `${latest.runningContainers} / ${latest.totalContainers} 实例` : "-",
    },
  ]

  return (
    <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-5">
      {/* CPU */}
      <Card className="shadow-sm border border-border/40 hover:border-border/80 transition-colors">
        <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
          <CardTitle className="text-sm font-medium text-muted-foreground">CPU 负载</CardTitle>
          <Cpu className="size-4 text-muted-foreground opacity-70" />
        </CardHeader>
        <CardContent>
          {sysInfo ? (
            <div className="flex flex-col gap-3 mt-1">
              <div className="text-2xl font-bold tabular-nums tracking-tight text-foreground">
                {sysInfo.cpu.usage_pct.toFixed(1)}
                <span className="ml-0.5 text-sm font-normal text-muted-foreground">%</span>
              </div>
              <ResourceRow label={`${sysInfo.cpu.cores} 核心`} pct={sysInfo.cpu.usage_pct} />
            </div>
          ) : (
            <div className="text-xs text-muted-foreground/80 mt-1 truncate">采集中...</div>
          )}
        </CardContent>
      </Card>

      {/* Memory */}
      <Card className="shadow-sm border border-border/40 hover:border-border/80 transition-colors">
        <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
          <CardTitle className="text-sm font-medium text-muted-foreground">系统内存</CardTitle>
          <HardDrive className="size-4 text-muted-foreground opacity-70" />
        </CardHeader>
        <CardContent>
          {sysInfo ? (
            <div className="flex flex-col gap-3 mt-1">
              <div className="text-2xl font-bold tabular-nums tracking-tight text-foreground">
                {sysInfo.memory.usage_pct.toFixed(1)}
                <span className="ml-0.5 text-sm font-normal text-muted-foreground">%</span>
              </div>
              <ResourceRow
                label={`${(sysInfo.memory.total_bytes / 1073741824).toFixed(1)} GB 总量`}
                pct={sysInfo.memory.usage_pct}
              />
            </div>
          ) : (
            <div className="text-xs text-muted-foreground/80 mt-1 truncate">采集中...</div>
          )}
        </CardContent>
      </Card>

      {/* Routine metrics */}
      {regularCards.map((item) => (
        <Card key={item.key} className="shadow-sm border border-border/40 hover:border-border/80 transition-colors">
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-sm font-medium text-muted-foreground">{item.title}</CardTitle>
            <item.icon className="size-4 text-muted-foreground opacity-70" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold tabular-nums tracking-tight">{item.value}</div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
