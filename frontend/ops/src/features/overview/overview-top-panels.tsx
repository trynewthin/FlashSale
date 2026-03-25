import { Cpu, MemoryStick, Telescope } from "lucide-react"

import type { ObservabilityLinks, SysInfo } from "@/api/types"

import { ObservabilityLinkButton, SectionCard } from "./overview-cards"

function ResourceRow({ label, pct }: { label: string; pct: number }) {
  const tone =
    pct >= 90 ? "bg-destructive" : pct >= 70 ? "bg-amber-500" : "bg-primary"
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-medium tabular-nums text-foreground">{pct.toFixed(1)}%</span>
      </div>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
        <div className={`h-full rounded-full transition-all ${tone}`} style={{ width: `${Math.min(pct, 100)}%` }} />
      </div>
    </div>
  )
}

export function OverviewTopPanels({
  obsLinks,
  sysInfo,
}: {
  obsLinks: ObservabilityLinks | null
  sysInfo: SysInfo | null
}) {
  return (
    <div className="grid gap-4 xl:grid-cols-3">
      {/* CPU */}
      <SectionCard title="CPU 用量" icon={Cpu}>
        <div className="flex h-[116px] flex-col justify-center gap-4">
          {sysInfo ? (
            <>
              <ResourceRow label={`${sysInfo.cpu.cores} 核心`} pct={sysInfo.cpu.usage_pct} />
              <div className="text-center text-2xl font-semibold tabular-nums text-foreground">
                {sysInfo.cpu.usage_pct.toFixed(1)}
                <span className="ml-1 text-sm font-normal text-muted-foreground">%</span>
              </div>
            </>
          ) : (
            <div className="text-center text-sm text-muted-foreground/60">采集中...</div>
          )}
        </div>
      </SectionCard>

      {/* 内存 */}
      <SectionCard title="内存用量" icon={MemoryStick}>
        <div className="flex h-[116px] flex-col justify-center gap-4">
          {sysInfo ? (
            <>
              <ResourceRow
                label={`${(sysInfo.memory.total_bytes / 1073741824).toFixed(1)} GB 总量`}
                pct={sysInfo.memory.usage_pct}
              />
              <div className="text-center text-2xl font-semibold tabular-nums text-foreground">
                {sysInfo.memory.usage_pct.toFixed(1)}
                <span className="ml-1 text-sm font-normal text-muted-foreground">%</span>
              </div>
            </>
          ) : (
            <div className="text-center text-sm text-muted-foreground/60">采集中...</div>
          )}
        </div>
      </SectionCard>

      {/* 观测入口 */}
      <SectionCard title="观测入口" icon={Telescope}>
        {obsLinks ? (
          <div className="flex flex-col gap-2">
            <ObservabilityLinkButton link={obsLinks.prometheus} />
            <ObservabilityLinkButton link={obsLinks.grafana} />
            <ObservabilityLinkButton link={obsLinks.jaeger} />
          </div>
        ) : (
          <div className="flex h-[116px] items-center justify-center text-sm text-muted-foreground/60">
            正在加载...
          </div>
        )}
      </SectionCard>
    </div>
  )
}
