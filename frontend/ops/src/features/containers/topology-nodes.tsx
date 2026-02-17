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
import { Badge } from "@/components/ui/badge"
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
  roleLabel,
  runtimeStatusClass,
  serviceDisplayName,
  type RuntimeStatusTone,
} from "@/features/containers/shared"

interface ServiceGroupNodeData extends Record<string, unknown> {
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
  onScaleUp: (serviceName: string, targetReplicas: number) => void
}

interface ReplicaNodeData extends Record<string, unknown> {
  serviceName: string
  serviceReplicas: number
  serviceScalable: boolean
  scaleLoading: boolean
  containerName: string
  replicaIndex: number
  running: boolean
  operable: boolean
  health: string
  status: string
  actioningKey: string
  onActionContainer: (containerName: string, action: "start" | "stop" | "restart") => void
  onScaleDownService: (serviceName: string, targetReplicas: number) => void
  onOpenLogs: (serviceName: string, containerName: string) => void
}

export const ServiceGroupNodeView = memo(({ data }: NodeProps<Node<ServiceGroupNodeData>>) => {
  const detailMenuID = `service-detail-${data.serviceName}`
  const [scaleConfirmOpen, setScaleConfirmOpen] = useState(false)
  const targetReplicas = data.replicas + 1

  const openScaleConfirm = (event: MouseEvent) => {
    event.stopPropagation()
    if (!data.scalable || data.scaleLoading) {
      return
    }
    setScaleConfirmOpen(true)
  }

  const confirmScaleUp = (event: MouseEvent) => {
    event.stopPropagation()
    setScaleConfirmOpen(false)
    data.onScaleUp(data.serviceName, targetReplicas)
  }

  return (
    <div
      className={cn(
        "h-full w-full rounded-xl border-2 p-3 shadow-sm transition-colors",
        roleBgColor(data.role),
        data.focused ? "border-sky-500 ring-2 ring-sky-200" : "border-slate-300"
      )}
    >
      <Handle type="target" position={Position.Left} className="!h-2.5 !w-2.5 !border-0 !bg-slate-400" />
      <Handle type="source" position={Position.Right} className="!h-2.5 !w-2.5 !border-0 !bg-slate-400" />

      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="truncate text-sm font-bold text-slate-900">{serviceDisplayName(data.serviceName)}</div>
          <div className="truncate text-[11px] text-slate-500">{data.serviceName}</div>
          {data.etcdRegistered !== undefined && (
            <div
              className="mt-0.5 text-[10px] font-medium text-blue-600"
              title={`etcd 注册实例数: ${data.etcdRegistered}`}
            >
              etcd: {data.etcdRegistered}/{data.replicas} registered
            </div>
          )}
        </div>
        <div
          className="nodrag nopan nowheel flex shrink-0 items-center gap-1"
          onPointerDown={(event) => event.stopPropagation()}
          onMouseDown={(event) => event.stopPropagation()}
          onClick={(event) => event.stopPropagation()}
        >
          <span
            className={cn(
              "inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold",
              runtimeStatusClass(data.statusTone)
            )}
          >
            {data.statusText}
          </span>
          <DropdownMenu>
            <DropdownMenuTrigger
              aria-label={`查看${serviceDisplayName(data.serviceName)}详情`}
              render={
                <Button
                  variant="outline"
                  size="icon-xs"
                  className="nodrag nopan nowheel size-6"
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
              onPointerDown={(event) => event.stopPropagation()}
              onMouseDown={(event) => event.stopPropagation()}
              onClick={(event) => event.stopPropagation()}
            >
              <DropdownMenuItem disabled>我需要（依赖）</DropdownMenuItem>
              <DropdownMenuSeparator />
              {data.dependsOn.length > 0 ? (
                data.dependsOn.map((dep) => (
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
              {data.requiredBy.length > 0 ? (
                data.requiredBy.map((consumer) => (
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
              onPointerDown={(event) => event.stopPropagation()}
              onMouseDown={(event) => event.stopPropagation()}
              onClick={(event) => event.stopPropagation()}
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

      <div className="mt-2 text-[11px] text-slate-700">
        {roleLabel(data.role)} · 运行副本 {data.runningReplicas}/{data.replicas}
      </div>
    </div>
  )
})

export const ReplicaNodeView = memo(({ data }: NodeProps<Node<ReplicaNodeData>>) => {
  const [scaleDownConfirmOpen, setScaleDownConfirmOpen] = useState(false)
  const statusClass = !data.operable
    ? "border-slate-200 bg-slate-100 text-slate-700"
    : data.running
      ? "border-emerald-200 bg-emerald-50 text-emerald-700"
      : "border-rose-200 bg-rose-50 text-rose-700"
  const startKey = `${data.containerName}:start`
  const stopKey = `${data.containerName}:stop`
  const restartKey = `${data.containerName}:restart`
  const menuDisabled =
    !data.operable || (data.actioningKey !== "" && !data.actioningKey.startsWith(`${data.containerName}:`))
  const allowScaleDown = data.serviceScalable && data.serviceReplicas > 1
  const targetReplicas = Math.max(0, data.serviceReplicas - 1)
  const handleScaleDown = (event: MouseEvent) => {
    event.stopPropagation()
    if (!allowScaleDown || data.scaleLoading) {
      return
    }
    setScaleDownConfirmOpen(true)
  }
  const confirmScaleDown = (event: MouseEvent) => {
    event.stopPropagation()
    setScaleDownConfirmOpen(false)
    data.onScaleDownService(data.serviceName, targetReplicas)
  }
  const openServiceLogs = (event: MouseEvent) => {
    event.stopPropagation()
    if (!data.operable) {
      return
    }
    data.onOpenLogs(data.serviceName, data.containerName)
  }

  return (
    <div className="h-full w-full rounded-lg border border-slate-300 bg-white/95 p-2 shadow-sm">
      <div className="flex items-start justify-between gap-2">
        <div className="truncate text-xs font-semibold text-slate-900">副本 #{data.replicaIndex}</div>
        <div
          className="nodrag nopan nowheel flex items-center gap-1"
          onPointerDown={(event) => event.stopPropagation()}
          onMouseDown={(event) => event.stopPropagation()}
          onClick={(event) => event.stopPropagation()}
        >
          <span className={cn("rounded-full border px-1.5 py-0.5 text-[10px] font-semibold", statusClass)}>
            {!data.operable ? "未创建" : data.running ? "运行中" : "已停止"}
          </span>
          <Button
            variant="outline"
            size="icon-xs"
            className="nodrag nopan nowheel size-6"
            disabled={!data.operable}
            title="查看运行日志"
            onClick={openServiceLogs}
          >
            <FileText className="size-3.5" />
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger
              disabled={menuDisabled}
              render={<Button variant="outline" size="icon-xs" className="nodrag nopan nowheel size-6" />}
            >
              <MoreHorizontal className="size-3.5" />
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="end"
              className="nodrag nopan nowheel"
              onPointerDown={(event) => event.stopPropagation()}
              onMouseDown={(event) => event.stopPropagation()}
              onClick={(event) => event.stopPropagation()}
            >
              <DropdownMenuItem
                disabled={!data.operable || (data.actioningKey !== "" && data.actioningKey !== startKey)}
                onClick={() => data.onActionContainer(data.containerName, "start")}
              >
                <Play className="size-3.5" />
                启动
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={!data.operable || (data.actioningKey !== "" && data.actioningKey !== stopKey)}
                onClick={() => data.onActionContainer(data.containerName, "stop")}
              >
                <Square className="size-3.5" />
                停止
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={!data.operable || (data.actioningKey !== "" && data.actioningKey !== restartKey)}
                onClick={() => data.onActionContainer(data.containerName, "restart")}
              >
                <RotateCw className="size-3.5" />
                重启
              </DropdownMenuItem>
              {allowScaleDown ? (
                <DropdownMenuItem
                  disabled={data.scaleLoading}
                  variant="destructive"
                  onClick={handleScaleDown}
                >
                  <Trash2 className="size-3.5" />
                  删除一个副本
                </DropdownMenuItem>
              ) : null}
            </DropdownMenuContent>
          </DropdownMenu>
          <AlertDialog open={scaleDownConfirmOpen} onOpenChange={setScaleDownConfirmOpen}>
            <AlertDialogContent
              className="nodrag nopan nowheel"
              onPointerDown={(event) => event.stopPropagation()}
              onMouseDown={(event) => event.stopPropagation()}
              onClick={(event) => event.stopPropagation()}
            >
              <AlertDialogHeader>
                <AlertDialogTitle>确认删除副本</AlertDialogTitle>
                <AlertDialogDescription>
                  将「{serviceDisplayName(data.serviceName)}」从 {data.serviceReplicas} 个副本缩容到 {targetReplicas} 个副本。确认继续吗？
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>取消</AlertDialogCancel>
                <AlertDialogAction
                  disabled={data.scaleLoading}
                  onClick={confirmScaleDown}
                >
                  确认删除
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </div>
      <div className="mt-1 truncate text-[10px] text-slate-500">{data.containerName}</div>

      <div className="mt-2 flex items-center gap-1 text-[10px] text-slate-600">
        <Badge variant={data.health === "healthy" || data.health === "running" ? "secondary" : "destructive"}>
          {data.health}
        </Badge>
        <span className="truncate">{data.status}</span>
      </div>
    </div>
  )
})
