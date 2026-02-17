import { useState } from "react"

import type { ContainerRuntimeSnapshot } from "@/api/types"
import { Button } from "@/components/ui/button"
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
          <span className="inline-flex h-[18px] items-center rounded-full border border-slate-200 px-1.5 text-[9px] font-medium text-slate-600">
            部署：{snapshot?.deployment_mode || "-"}
          </span>
          <span className="inline-flex h-[18px] items-center rounded-full border border-slate-200 px-1.5 text-[9px] font-medium text-slate-600">
            项目：{snapshot?.compose.project || "-"}
          </span>
        </div>

        <div className="font-medium text-slate-700">节点颜色 → 服务类型</div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="inline-flex items-center gap-1 rounded border px-1.5 py-0.5">
            <i className="inline-block h-3 w-3 rounded-sm border bg-sky-50" />入口
          </span>
          <span className="inline-flex items-center gap-1 rounded border px-1.5 py-0.5">
            <i className="inline-block h-3 w-3 rounded-sm border bg-indigo-50" />网关
          </span>
          <span className="inline-flex items-center gap-1 rounded border px-1.5 py-0.5">
            <i className="inline-block h-3 w-3 rounded-sm border bg-emerald-50" />业务
          </span>
          <span className="inline-flex items-center gap-1 rounded border px-1.5 py-0.5">
            <i className="inline-block h-3 w-3 rounded-sm border bg-amber-50" />基础设施
          </span>
          <span className="inline-flex items-center gap-1 rounded border px-1.5 py-0.5">
            <i className="inline-block h-3 w-3 rounded-sm border bg-violet-50" />可观测性
          </span>
          <span className="inline-flex items-center gap-1 rounded border px-1.5 py-0.5">
            <i className="inline-block h-3 w-3 rounded-sm border bg-zinc-100" />任务
          </span>
        </div>

        <div className="font-medium text-slate-700">徽章含义</div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="inline-flex h-[18px] items-center rounded-full border border-emerald-200 bg-emerald-50 px-1.5 text-[9px] font-medium text-emerald-700">
            运行中
          </span>
          <span className="inline-flex h-[18px] items-center rounded-full border border-rose-200 bg-rose-50 px-1.5 text-[9px] font-medium text-rose-600">
            已停止
          </span>
          <span className="inline-flex h-[18px] items-center gap-0.5 rounded-full border border-blue-200 bg-blue-50 px-1.5 text-[9px] font-semibold text-blue-700">
            <span className="inline-block size-1.5 rounded-full bg-blue-500" />
            etcd N/N
          </span>
          <span className="inline-flex h-[18px] items-center rounded-full border border-slate-200 px-1.5 text-[9px] font-medium text-slate-500">
            N/N 副本
          </span>
        </div>

        <div className="font-medium text-slate-700">连线</div>
        <div className="space-y-1 text-[11px] text-slate-600">
          <div className="flex items-center gap-1.5">
            <svg width="32" height="8"><line x1="0" y1="4" x2="32" y2="4" stroke="#94a3b8" strokeWidth="1.4" /><polygon points="28,1 32,4 28,7" fill="#94a3b8" /></svg>
            依赖方向（A → B 表示 A 依赖 B）
          </div>
          <div className="flex items-center gap-1.5">
            <svg width="32" height="8"><line x1="0" y1="4" x2="32" y2="4" stroke="#0284c7" strokeWidth="2.6" /><polygon points="28,1 32,4 28,7" fill="#0284c7" /></svg>
            选中节点的关联连线
          </div>
          <div className="text-[10px] text-slate-400">点击节点可聚焦查看其依赖关系</div>
        </div>
      </div>
    </div>
  )
}
