import { memo, useState, type MouseEvent } from "react"
import { Handle, Position, type Node, type NodeProps } from "@xyflow/react"
import { FileText, MoreHorizontal, Play, Plus, RotateCw, Square, Trash2 } from "lucide-react"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { cn } from "@/lib/utils"
import {
  roleBgColor,
  runtimeStatusClass,
  serviceDisplayName,
  type RuntimeStatusTone,
} from "@/features/containers/shared"

// ─── Types (exported for canvas) ───

export interface ReplicaItem {
  containerName: string
  replicaIndex: number
  running: boolean
  operable: boolean
  health: string
  status: string
}

export interface ServiceGroupNodeData extends Record<string, unknown> {
  serviceName: string
  role: string
  runningReplicas: number
  replicas: number
  scalable: boolean
  scaleLoading: boolean
  statusText: string
  statusTone: RuntimeStatusTone
  etcdRegistered?: number
  dependsOn: string[]
  requiredBy: string[]
  focused: boolean
  replicaItems: ReplicaItem[]
  actioningKey: string
  onScaleUp: (serviceName: string, targetReplicas: number) => void
  onActionContainer: (containerName: string, action: "start" | "stop" | "restart") => void
  onScaleDownService: (serviceName: string, targetReplicas: number) => void
  onOpenLogs: (serviceName: string, containerName: string) => void
}

// ─── Replica Card (inline HTML, not a ReactFlow node) ───

function ReplicaCard({ item, data }: { item: ReplicaItem; data: ServiceGroupNodeData }) {
  const [scaleDownOpen, setScaleDownOpen] = useState(false)
  const allowScaleDown = data.scalable && data.replicas > 1
  const targetReplicas = Math.max(0, data.replicas - 1)

  const startKey = `${item.containerName}:start`
  const stopKey = `${item.containerName}:stop`
  const restartKey = `${item.containerName}:restart`
  const menuDisabled =
    !item.operable || (data.actioningKey !== "" && !data.actioningKey.startsWith(`${item.containerName}:`))

  const handleScaleDown = (e: MouseEvent) => {
    e.stopPropagation()
    if (!allowScaleDown || data.scaleLoading) return
    setScaleDownOpen(true)
  }
  const confirmScaleDown = (e: MouseEvent) => {
    e.stopPropagation()
    setScaleDownOpen(false)
    data.onScaleDownService(data.serviceName, targetReplicas)
  }
  const openLogs = (e: MouseEvent) => {
    e.stopPropagation()
    if (!item.operable) return
    data.onOpenLogs(data.serviceName, item.containerName)
  }

  return (
    <div className="rounded-lg border border-slate-200 bg-white/90 p-1.5 shadow-sm">
      {/* Area 1: Name + Buttons */}
      <div className="flex items-center justify-between gap-1">
        <div className="truncate text-[11px] font-semibold text-slate-800">副本 #{item.replicaIndex}</div>
        <div
          className="flex shrink-0 items-center gap-0.5"
          onPointerDown={(e) => e.stopPropagation()}
          onMouseDown={(e) => e.stopPropagation()}
          onClick={(e) => e.stopPropagation()}
        >
          <Button
            variant="ghost"
            size="icon-xs"
            className="size-5"
            disabled={!item.operable}
            title="查看日志"
            onClick={openLogs}
          >
            <FileText className="size-3" />
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger
              disabled={menuDisabled}
              render={<Button variant="ghost" size="icon-xs" className="size-5" />}
            >
              <MoreHorizontal className="size-3" />
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="end"
              className="min-w-36"
              onPointerDown={(e) => e.stopPropagation()}
              onMouseDown={(e) => e.stopPropagation()}
              onClick={(e) => e.stopPropagation()}
            >
              <DropdownMenuItem
                disabled={!item.operable || (data.actioningKey !== "" && data.actioningKey !== startKey)}
                onClick={() => data.onActionContainer(item.containerName, "start")}
              >
                <Play className="size-3" /> 启动
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={!item.operable || (data.actioningKey !== "" && data.actioningKey !== stopKey)}
                onClick={() => data.onActionContainer(item.containerName, "stop")}
              >
                <Square className="size-3" /> 停止
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={!item.operable || (data.actioningKey !== "" && data.actioningKey !== restartKey)}
                onClick={() => data.onActionContainer(item.containerName, "restart")}
              >
                <RotateCw className="size-3" /> 重启
              </DropdownMenuItem>
              {allowScaleDown ? (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    disabled={data.scaleLoading}
                    variant="destructive"
                    onClick={handleScaleDown}
                  >
                    <Trash2 className="size-3" /> 删除副本
                  </DropdownMenuItem>
                </>
              ) : null}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* Area 2: Container ID */}
      <div className="mt-0.5 truncate text-[9px] text-slate-400" title={item.containerName}>
        {item.containerName}
      </div>

      {/* Area 3: Badges — running status + health */}
      <div className="mt-1 flex items-center gap-1">
        <span
          className={cn(
            "inline-flex h-[18px] items-center rounded-full border px-1.5 text-[9px] font-medium",
            !item.operable
              ? "border-slate-200 text-slate-400"
              : item.running
                ? "border-emerald-200 bg-emerald-50 text-emerald-700"
                : "border-rose-200 bg-rose-50 text-rose-600"
          )}
        >
          {!item.operable ? "未创建" : item.running ? "运行中" : "已停止"}
        </span>
        <span
          className={cn(
            "inline-flex h-[18px] items-center rounded-full border px-1.5 text-[9px] font-medium",
            item.health === "healthy" || item.health === "running"
              ? "border-emerald-200 bg-emerald-50 text-emerald-700"
              : item.health === "missing" || item.health === "stopped"
                ? "border-slate-200 text-slate-400"
                : "border-rose-200 bg-rose-50 text-rose-600"
          )}
        >
          {item.health}
        </span>
      </div>

      {/* Scale Down Confirm */}
      <AlertDialog open={scaleDownOpen} onOpenChange={setScaleDownOpen}>
        <AlertDialogContent
          onPointerDown={(e) => e.stopPropagation()}
          onMouseDown={(e) => e.stopPropagation()}
          onClick={(e) => e.stopPropagation()}
        >
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除副本</AlertDialogTitle>
            <AlertDialogDescription>
              将「{serviceDisplayName(data.serviceName)}」从 {data.replicas} 个副本缩容到 {targetReplicas} 个副本。确认继续吗？
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction disabled={data.scaleLoading} onClick={confirmScaleDown}>
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

// ─── Service Group Node (ReactFlow node) ───
// Layout:
//  ┌───────────────────────────────────────────┐
//  │ Area 1: Display Name         [⋯] [+]     │
//  ├───────────────────────────────────────────┤
//  │ Area 2: [ID] [运行中] [etcd 1/1] [2副本]  │
//  ├───────────────────────────────────────────┤
//  │ Area 3: Replica Grid (scrollable if >3行)  │
//  │ ┌─────────┐ ┌─────────┐                   │
//  │ │ Replica1 │ │ Replica2 │                   │
//  │ └─────────┘ └─────────┘                   │
//  └───────────────────────────────────────────┘

export const ServiceGroupNodeView = memo(({ data }: NodeProps<Node<ServiceGroupNodeData>>) => {
  const [scaleConfirmOpen, setScaleConfirmOpen] = useState(false)
  const targetReplicas = data.replicas + 1
  const detailMenuID = `service-detail-${data.serviceName}`

  const openScaleConfirm = (e: MouseEvent) => {
    e.stopPropagation()
    if (!data.scalable || data.scaleLoading) return
    setScaleConfirmOpen(true)
  }
  const confirmScaleUp = (e: MouseEvent) => {
    e.stopPropagation()
    setScaleConfirmOpen(false)
    data.onScaleUp(data.serviceName, targetReplicas)
  }

  const replicaCount = data.replicaItems.length
  const rows = Math.ceil(replicaCount / 2)
  const maxVisibleRows = 3
  const needsScroll = rows > maxVisibleRows

  return (
    <div
      className={cn(
        "flex h-full w-full flex-col rounded-xl border-2 shadow-sm transition-colors",
        roleBgColor(data.role),
        data.focused ? "border-sky-500 ring-2 ring-sky-200" : "border-slate-300"
      )}
    >
      {/* Invisible handles for ReactFlow edges (4 sides for smart routing) */}
      <Handle id="top-s" type="source" position={Position.Top} className="opacity-0!" />
      <Handle id="right-s" type="source" position={Position.Right} className="opacity-0!" />
      <Handle id="bottom-s" type="source" position={Position.Bottom} className="opacity-0!" />
      <Handle id="left-s" type="source" position={Position.Left} className="opacity-0!" />

      <Handle id="top-t" type="target" position={Position.Top} className="opacity-0!" />
      <Handle id="right-t" type="target" position={Position.Right} className="opacity-0!" />
      <Handle id="bottom-t" type="target" position={Position.Bottom} className="opacity-0!" />
      <Handle id="left-t" type="target" position={Position.Left} className="opacity-0!" />

      {/* ── Area 1: Name + Actions ── */}
      <div className="flex items-center justify-between gap-2 px-3 pb-1 pt-2.5">
        <div className="min-w-0">
          <div className="truncate text-sm font-bold text-slate-900">
            {serviceDisplayName(data.serviceName)}
          </div>
        </div>
        <div
          className="nodrag nopan nowheel flex shrink-0 items-center gap-1"
          onPointerDown={(e) => e.stopPropagation()}
          onMouseDown={(e) => e.stopPropagation()}
          onClick={(e) => e.stopPropagation()}
        >
          <DropdownMenu>
            <DropdownMenuTrigger
              aria-label={`查看${serviceDisplayName(data.serviceName)}详情`}
              render={
                <Button
                  variant="outline"
                  size="icon-xs"
                  className="size-6"
                  aria-controls={detailMenuID}
                />
              }
            >
              <MoreHorizontal className="size-3.5" />
            </DropdownMenuTrigger>
            <DropdownMenuContent
              id={detailMenuID}
              align="end"
              className="nodrag nopan nowheel min-w-64"
              onPointerDown={(e) => e.stopPropagation()}
              onMouseDown={(e) => e.stopPropagation()}
              onClick={(e) => e.stopPropagation()}
            >
              <DropdownMenuItem disabled>我需要（依赖）</DropdownMenuItem>
              <DropdownMenuSeparator />
              {(data.dependsOn ?? []).length > 0 ? (
                (data.dependsOn ?? []).map((dep) => (
                  <DropdownMenuItem key={`${data.serviceName}-dep-${dep}`} disabled>
                    <div className="min-w-0">
                      <div className="truncate text-xs font-medium">{serviceDisplayName(dep)}</div>
                      <div className="truncate text-[10px] text-slate-500">{dep}</div>
                    </div>
                  </DropdownMenuItem>
                ))
              ) : (
                <DropdownMenuItem disabled>无依赖</DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem disabled>需要我（被依赖）</DropdownMenuItem>
              <DropdownMenuSeparator />
              {(data.requiredBy ?? []).length > 0 ? (
                (data.requiredBy ?? []).map((consumer) => (
                  <DropdownMenuItem key={`${data.serviceName}-consumer-${consumer}`} disabled>
                    <div className="min-w-0">
                      <div className="truncate text-xs font-medium">{serviceDisplayName(consumer)}</div>
                      <div className="truncate text-[10px] text-slate-500">{consumer}</div>
                    </div>
                  </DropdownMenuItem>
                ))
              ) : (
                <DropdownMenuItem disabled>无人依赖</DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
          <Button
            variant="outline"
            size="icon-xs"
            className="nodrag nopan nowheel size-6"
            disabled={!data.scalable || data.scaleLoading}
            title={
              data.scalable
                ? data.scaleLoading
                  ? "扩容执行中"
                  : "新增一个副本（需二次确认）"
                : "该服务不支持扩容"
            }
            onClick={openScaleConfirm}
          >
            <Plus className="size-3.5" />
          </Button>
          <AlertDialog open={scaleConfirmOpen} onOpenChange={setScaleConfirmOpen}>
            <AlertDialogContent
              className="nodrag nopan nowheel"
              onPointerDown={(e) => e.stopPropagation()}
              onMouseDown={(e) => e.stopPropagation()}
              onClick={(e) => e.stopPropagation()}
            >
              <AlertDialogHeader>
                <AlertDialogTitle>确认新增副本</AlertDialogTitle>
                <AlertDialogDescription>
                  确认将「{serviceDisplayName(data.serviceName)}」扩容到 {targetReplicas} 个副本吗？
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>取消</AlertDialogCancel>
                <AlertDialogAction disabled={data.scaleLoading} onClick={confirmScaleUp}>
                  确认扩容
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </div>

      {/* ── Area 2: Badge Bar ── */}
      <div className="flex flex-wrap items-center gap-1 px-3 py-1.5">
        <span className="inline-flex h-[18px] items-center rounded-full border border-slate-200 px-1.5 text-[9px] font-medium text-slate-500">
          {data.serviceName}
        </span>
        <span
          className={cn(
            "inline-flex h-[18px] items-center rounded-full border px-1.5 text-[9px] font-semibold",
            runtimeStatusClass(data.statusTone)
          )}
        >
          {data.statusText}
        </span>
        {data.etcdRegistered !== undefined && (
          <span
            className={cn(
              "inline-flex h-[18px] items-center gap-0.5 rounded-full border px-1.5 text-[9px] font-semibold",
              data.etcdRegistered >= data.replicas
                ? "border-blue-200 bg-blue-50 text-blue-700"
                : data.etcdRegistered > 0
                  ? "border-amber-200 bg-amber-50 text-amber-700"
                  : "border-rose-200 bg-rose-50 text-rose-600"
            )}
            title={`etcd 注册实例数: ${data.etcdRegistered}/${data.replicas}`}
          >
            <span
              className={cn(
                "inline-block size-1.5 rounded-full",
                data.etcdRegistered >= data.replicas
                  ? "bg-blue-500"
                  : data.etcdRegistered > 0
                    ? "bg-amber-500"
                    : "bg-rose-500"
              )}
            />
            etcd {data.etcdRegistered}/{data.replicas}
          </span>
        )}
        <span className="inline-flex h-[18px] items-center rounded-full border border-slate-200 px-1.5 text-[9px] font-medium text-slate-500">
          {data.runningReplicas}/{data.replicas} 副本
        </span>
      </div>

      {/* ── Area 3: Replica Grid ── */}
      {replicaCount > 0 && (
        <div
          className={cn(
            "nodrag nopan nowheel grid flex-1 grid-cols-2 gap-1.5 px-2 py-2",
            needsScroll && "overflow-y-auto"
          )}
          style={
            needsScroll
              ? { maxHeight: `${maxVisibleRows * 82 + (maxVisibleRows - 1) * 6}px` }
              : undefined
          }
          onPointerDown={(e) => e.stopPropagation()}
          onMouseDown={(e) => e.stopPropagation()}
          onWheel={(e) => e.stopPropagation()}
        >
          {data.replicaItems.map((item) => (
            <ReplicaCard key={item.containerName} item={item} data={data} />
          ))}
        </div>
      )}
    </div>
  )
})
