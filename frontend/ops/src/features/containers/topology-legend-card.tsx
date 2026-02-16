import { useState } from "react"

import type { ContainerRuntimeSnapshot } from "@/api/types"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { ChevronDown, ChevronUp, RefreshCcw } from "lucide-react"

interface TopologyLegendCardProps {
  snapshot: ContainerRuntimeSnapshot | null
  loading: boolean
  onRefresh: () => void
}

// TopologyLegendCard 在画布外层展示固定图例，替代原侧边操作菜单中的图例区。
export function TopologyLegendCard({ snapshot, loading, onRefresh }: TopologyLegendCardProps) {
  const [expanded, setExpanded] = useState(false)

  if (!expanded) {
    return (
      <Button variant="secondary" size="sm" className="shadow" onClick={() => setExpanded(true)}>
        <ChevronDown className="size-4" />
        图例
      </Button>
    )
  }

  return (
    <div className="w-[340px] max-w-[calc(100vw-2rem)] rounded-lg border bg-background/94 p-3 shadow-lg backdrop-blur-md">
      <div className="mb-2 flex items-center justify-between">
        <div className="text-sm font-semibold">关系图图例</div>
        <div className="flex items-center gap-1">
          <Button variant="outline" size="sm" onClick={onRefresh} disabled={loading}>
            <RefreshCcw className="size-4" />
            刷新
          </Button>
          <Button variant="outline" size="icon-sm" onClick={() => setExpanded(false)} title="收起图例">
            <ChevronUp className="size-4" />
          </Button>
        </div>
      </div>

      <div className="space-y-2 text-xs">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="secondary">部署模式：{snapshot?.deployment_mode || "-"}</Badge>
          <Badge variant="outline">项目：{snapshot?.compose.project || "-"}</Badge>
        </div>

        <div className="font-medium text-slate-700">服务类型</div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="inline-flex items-center gap-1 rounded border px-1">
            <i className="inline-block h-3 w-3 rounded-sm border bg-sky-50" />入口
          </span>
          <span className="inline-flex items-center gap-1 rounded border px-1">
            <i className="inline-block h-3 w-3 rounded-sm border bg-indigo-50" />网关
          </span>
          <span className="inline-flex items-center gap-1 rounded border px-1">
            <i className="inline-block h-3 w-3 rounded-sm border bg-emerald-50" />业务
          </span>
          <span className="inline-flex items-center gap-1 rounded border px-1">
            <i className="inline-block h-3 w-3 rounded-sm border bg-amber-50" />基础设施
          </span>
          <span className="inline-flex items-center gap-1 rounded border px-1">
            <i className="inline-block h-3 w-3 rounded-sm border bg-zinc-100" />任务
          </span>
        </div>

        <div className="font-medium text-slate-700">副本状态</div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="inline-flex items-center gap-1 rounded border px-1">
            <i className="inline-flex h-4 min-w-4 items-center justify-center rounded-sm border border-emerald-600 bg-emerald-500 px-1 text-[10px] font-semibold text-white">
              1
            </i>
            运行
          </span>
          <span className="inline-flex items-center gap-1 rounded border px-1">
            <i className="inline-flex h-4 min-w-4 items-center justify-center rounded-sm border border-slate-400 bg-slate-200 px-1 text-[10px] font-semibold text-slate-700">
              1
            </i>
            未运行
          </span>
        </div>
      </div>
    </div>
  )
}
