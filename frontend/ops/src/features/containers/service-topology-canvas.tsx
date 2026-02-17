import "@xyflow/react/dist/style.css"

import dagre from "dagre"
import { Component, useMemo, useRef, useEffect, type ReactNode, type ErrorInfo } from "react"
import {
  Background,
  Controls,
  MarkerType,
  MiniMap,
  ReactFlow,
  useReactFlow,
  type Edge,
  type Node,
  type NodeTypes,
  ReactFlowProvider,
} from "@xyflow/react"

import type { ContainerRuntimeSnapshot } from "@/api/types"
import { runtimeStatus, serviceEtcdKey } from "@/features/containers/shared"
import {
  ServiceGroupNodeView,
  type ReplicaItem,
  type ServiceGroupNodeData,
} from "@/features/containers/topology-nodes"

// ─── Layout Estimation Constants ───
// These must approximate the rendered pixel heights for dagre layout.

const SERVICE_GROUP_MIN_WIDTH = 324
const HEADER_EST = 68 // Area 1 (name+actions) + Area 2 (badges) + spacing
const REPLICA_CARD_EST_H = 82 // rendered height of one replica card
const REPLICA_GAP_EST = 6 // gap-1.5
const REPLICA_AREA_PAD = 18 // py-2 + border/rounding buffer
const MAX_COLUMNS = 2
const MAX_VISIBLE_ROWS = 3

// ─── Node Types (stable reference outside component) ───

const topologyNodeTypes: NodeTypes = {
  serviceGroup: ServiceGroupNodeView,
}

// ─── Props ───

interface ServiceTopologyCanvasProps {
  snapshot: ContainerRuntimeSnapshot | null
  etcdInstanceMap: Map<string, number>
  focusService: string
  actioningKey: string
  onActionContainer: (containerName: string, action: "start" | "stop" | "restart") => void
  onScaleUpService: (serviceName: string, targetReplicas: number) => void
  onScaleDownService: (serviceName: string, targetReplicas: number) => void
  onOpenReplicaLogs: (serviceName: string, containerName: string) => void
  onFocusServiceChange: (serviceName: string) => void
}

// ─── Helpers ───

function serviceNodeID(name: string): string {
  return `service:${name}`
}

function resolveServiceNameFromNode(node: Node): string {
  const data = node.data as { serviceName?: unknown }
  if (typeof data?.serviceName === "string") {
    return data.serviceName
  }
  return ""
}

function calcGroupHeight(replicaCount: number): number {
  if (replicaCount <= 0) {
    return HEADER_EST + 20 // minimum for empty services
  }
  const rows = Math.ceil(replicaCount / MAX_COLUMNS)
  const visibleRows = Math.min(rows, MAX_VISIBLE_ROWS)
  const replicaAreaHeight =
    visibleRows * REPLICA_CARD_EST_H + Math.max(0, visibleRows - 1) * REPLICA_GAP_EST + REPLICA_AREA_PAD
  return HEADER_EST + replicaAreaHeight
}

// ─── Build Flow ───

function buildFlow(
  snapshot: ContainerRuntimeSnapshot | null,
  etcdInstanceMap: Map<string, number>,
  focusService: string,
  actioningKey: string,
  onActionContainer: (containerName: string, action: "start" | "stop" | "restart") => void,
  onScaleUpService: (serviceName: string, targetReplicas: number) => void,
  onScaleDownService: (serviceName: string, targetReplicas: number) => void,
  onOpenReplicaLogs: (serviceName: string, containerName: string) => void
): {
  nodes: Node<ServiceGroupNodeData>[]
  edges: Edge[]
} {
  if (!snapshot || !snapshot.services || snapshot.services.length === 0) {
    return { nodes: [], edges: [] }
  }

  // Group containers by service
  const containersByService = new Map<string, typeof snapshot.containers>()
  for (const container of snapshot.containers) {
    const list = containersByService.get(container.service) ?? []
    list.push(container)
    containersByService.set(container.service, list)
  }
  for (const list of containersByService.values()) {
    list.sort((a, b) => a.name.localeCompare(b.name))
  }

  // Dagre layout — only service group nodes, replicas are rendered as HTML inside
  const graph = new dagre.graphlib.Graph()
  graph.setDefaultEdgeLabel(() => ({}))
  graph.setGraph({
    rankdir: "LR",
    nodesep: 76,
    ranksep: 168,
    marginx: 48,
    marginy: 48,
  })

  const sizeByService = new Map<string, { width: number; height: number }>()
  for (const service of snapshot.services) {
    const containerCount = containersByService.get(service.name)?.length ?? 0
    const replicas = Math.max(service.replicas, containerCount)
    const width = SERVICE_GROUP_MIN_WIDTH
    const height = calcGroupHeight(replicas)
    sizeByService.set(service.name, { width, height })
    graph.setNode(serviceNodeID(service.name), { width, height })
  }
  for (const edge of snapshot.edges) {
    graph.setEdge(serviceNodeID(edge.from), serviceNodeID(edge.to))
  }
  dagre.layout(graph)

  // Focus/highlight sets
  const focusedNodeSet = new Set<string>()
  const focusedEdgeSet = new Set<string>()
  const requiredByMap = new Map<string, string[]>()
  for (const edge of snapshot.edges) {
    const list = requiredByMap.get(edge.to) ?? []
    list.push(edge.from)
    requiredByMap.set(edge.to, list)
  }
  for (const [service, list] of requiredByMap.entries()) {
    list.sort((a, b) => a.localeCompare(b))
    requiredByMap.set(service, list)
  }
  if (focusService) {
    focusedNodeSet.add(focusService)
    for (const edge of snapshot.edges) {
      if (edge.from === focusService || edge.to === focusService) {
        focusedNodeSet.add(edge.from)
        focusedNodeSet.add(edge.to)
        focusedEdgeSet.add(`${edge.from}->${edge.to}`)
      }
    }
  }

  // Build service nodes with embedded replica data
  const serviceNodes: Node<ServiceGroupNodeData>[] = snapshot.services.map((service) => {
    const serviceID = serviceNodeID(service.name)
    const layout = graph.node(serviceID)
    const size = sizeByService.get(service.name) ?? {
      width: SERVICE_GROUP_MIN_WIDTH,
      height: calcGroupHeight(0),
    }
    const status = runtimeStatus(service.running_replicas, service.replicas, service.absent)

    // Build replica items inline
    const containers = containersByService.get(service.name) ?? []
    const targetReplicaCount = Math.max(service.replicas, containers.length)
    const replicaItems: ReplicaItem[] = []
    for (let index = 0; index < targetReplicaCount; index += 1) {
      const container = containers[index]
      const isPlaceholder = !container
      replicaItems.push({
        containerName: isPlaceholder ? `${service.name}-replica-${index + 1}` : container.name,
        replicaIndex: index + 1,
        running: isPlaceholder ? false : container.running,
        operable: !isPlaceholder,
        health: isPlaceholder ? "missing" : container.health,
        status: isPlaceholder ? "未创建" : container.status,
      })
    }

    return {
      id: serviceID,
      type: "serviceGroup",
      data: {
        serviceName: service.name,
        role: service.role,
        runningReplicas: service.running_replicas,
        replicas: service.replicas,
        scalable: service.scalable,
        scaleLoading: actioningKey === `scale:${service.name}`,
        statusText: status.text,
        statusTone: status.tone,
        dependsOn: service.depends_on,
        requiredBy: requiredByMap.get(service.name) ?? [],
        etcdRegistered: (() => {
          const ek = serviceEtcdKey(service.name)
          return ek !== undefined ? (etcdInstanceMap.get(ek) ?? 0) : undefined
        })(),
        focused: focusService ? focusedNodeSet.has(service.name) : false,
        replicaItems,
        actioningKey,
        onScaleUp: onScaleUpService,
        onActionContainer,
        onScaleDownService,
        onOpenLogs: onOpenReplicaLogs,
      },
      position: {
        x: (layout?.x ?? 0) - size.width / 2,
        y: (layout?.y ?? 0) - size.height / 2,
      },
      style: {
        width: size.width,
        height: size.height,
      },
      draggable: false,
      selectable: true,
      zIndex: 1,
    }
  })

  // Edges
  const edges: Edge[] = snapshot.edges.map((edge) => {
    const key = `${edge.from}->${edge.to}`
    const highlighted = focusService ? focusedEdgeSet.has(key) : false
    const muted = focusService ? !highlighted : false
    return {
      id: key,
      source: serviceNodeID(edge.from),
      target: serviceNodeID(edge.to),
      markerEnd: { type: MarkerType.ArrowClosed, width: 16, height: 16 },
      style: {
        stroke: highlighted ? "#0284c7" : "#94a3b8",
        opacity: muted ? 0.12 : highlighted ? 0.92 : 0.45,
        strokeWidth: highlighted ? 2.6 : 1.4,
      },
      animated: highlighted,
    }
  })

  return { nodes: serviceNodes, edges }
}

// ─── Error Boundary ───

interface ErrorBoundaryState {
  hasError: boolean
  errorMessage: string
}

class TopologyErrorBoundary extends Component<{ children: ReactNode }, ErrorBoundaryState> {
  constructor(props: { children: ReactNode }) {
    super(props)
    this.state = { hasError: false, errorMessage: "" }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, errorMessage: error.message }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("[TopologyErrorBoundary]", error, errorInfo)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex h-full w-full flex-col items-center justify-center gap-3 text-sm text-slate-500">
          <div className="text-lg font-semibold text-slate-700">拓扑渲染异常</div>
          <div className="max-w-md text-center text-xs">{this.state.errorMessage}</div>
          <button
            type="button"
            className="rounded-md bg-slate-800 px-4 py-1.5 text-xs text-white hover:bg-slate-700"
            onClick={() => this.setState({ hasError: false, errorMessage: "" })}
          >
            重新加载
          </button>
        </div>
      )
    }
    return this.props.children
  }
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
      // 延迟一帧等 React Flow 完成渲染
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
        if (!serviceName) {
          return
        }
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
