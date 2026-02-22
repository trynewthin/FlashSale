import "@xyflow/react/dist/style.css"

import { useMemo, useRef, useEffect } from "react"
import {
  Background,
  Controls,
  Handle,
  MiniMap,
  Position,
  ReactFlow,
  useReactFlow,
  type NodeTypes,
  ReactFlowProvider,
} from "@xyflow/react"

import { ServiceGroupNodeView } from "@/features/containers/topology-nodes"
import { buildFlow } from "@/features/containers/topology-layout"
import { TopologyErrorBoundary } from "@/features/containers/topology-error-boundary"
import { ROLE_GROUP_COLORS, resolveServiceNameFromNode, type ServiceTopologyCanvasProps } from "@/features/containers/topology-types"

// ─── Role Group Background Node ───

function RoleGroupNodeView({ data }: { data: { label: string; role: string } }) {
  const colorClass = ROLE_GROUP_COLORS[data.role] ?? "border-slate-200/60 bg-slate-50/30"
  return (
    <div className={`h-full w-full rounded-2xl border-3 border-dashed ${colorClass}`}>
      <Handle type="target" position={Position.Left} className="opacity-0!" />
      <Handle type="source" position={Position.Right} className="opacity-0!" />
      <div className="px-3 pt-1.5 text-[11px] font-semibold uppercase tracking-wider text-slate-400">
        {data.label}
      </div>
    </div>
  )
}

// ─── Node Types (stable reference outside component) ───

const topologyNodeTypes: NodeTypes = {
  serviceGroup: ServiceGroupNodeView,
  roleGroup: RoleGroupNodeView,
}

// ─── Inner Canvas (needs ReactFlowProvider context for useReactFlow) ───

function TopologyCanvasInner({
  snapshot,
  etcdInstanceMap,
  focusService,
  actioningKey,
  onActionContainer,
  onScaleUpService,
  onScaleDownService,
  onOpenReplicaLogs,
  onFocusServiceChange,
}: ServiceTopologyCanvasProps) {
  const { fitView } = useReactFlow()
  const didFitRef = useRef(false)

  const flowData = useMemo(
    () =>
      buildFlow(
        snapshot,
        etcdInstanceMap,
        focusService,
        actioningKey,
        onActionContainer,
        onScaleUpService,
        onScaleDownService,
        onOpenReplicaLogs
      ),
    [
      snapshot,
      etcdInstanceMap,
      focusService,
      actioningKey,
      onActionContainer,
      onScaleUpService,
      onScaleDownService,
      onOpenReplicaLogs,
    ]
  )

  // 只在首次有数据时 fitView，后续更新不再 fitView（避免用户缩放被覆盖）
  useEffect(() => {
    if (!didFitRef.current && flowData.nodes.length > 0) {
      const raf = requestAnimationFrame(() => {
        fitView({ padding: 0.22, minZoom: 0.42, maxZoom: 1.12, duration: 300 })
        didFitRef.current = true
      })
      return () => cancelAnimationFrame(raf)
    }
  }, [flowData.nodes.length, fitView])

  return (
    <ReactFlow
      nodes={flowData.nodes}
      edges={flowData.edges}
      nodeTypes={topologyNodeTypes}
      fitViewOptions={{ padding: 0.22, minZoom: 0.42, maxZoom: 1.12 }}
      minZoom={0.35}
      maxZoom={2}
      nodesDraggable={false}
      nodesConnectable={false}
      onNodeClick={(_, node) => {
        const serviceName = resolveServiceNameFromNode(node)
        if (!serviceName) return
        onFocusServiceChange(focusService === serviceName ? "" : serviceName)
      }}
      onPaneClick={() => onFocusServiceChange("")}
      proOptions={{ hideAttribution: true }}
    >
      <Background color="#d5dde8" gap={22} />
      <MiniMap zoomable pannable />
      <Controls />
    </ReactFlow>
  )
}

// ─── Canvas Component ───

export function ServiceTopologyCanvas(props: ServiceTopologyCanvasProps) {
  return (
    <div className="h-full w-full">
      <TopologyErrorBoundary>
        <ReactFlowProvider>
          <TopologyCanvasInner {...props} />
        </ReactFlowProvider>
      </TopologyErrorBoundary>
    </div>
  )
}
