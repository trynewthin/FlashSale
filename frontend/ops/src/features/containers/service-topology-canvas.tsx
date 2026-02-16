import "@xyflow/react/dist/style.css"

import dagre from "dagre"
import { useMemo } from "react"
import {
  Background,
  Controls,
  MarkerType,
  MiniMap,
  ReactFlow,
  type Edge,
  type Node,
  type NodeTypes,
} from "@xyflow/react"

import type { ContainerRuntimeSnapshot } from "@/api/types"
import { runtimeStatus } from "@/features/containers/shared"
import { ReplicaNodeView, ServiceGroupNodeView } from "@/features/containers/topology-nodes"

const SERVICE_GROUP_MIN_WIDTH = 320
const SERVICE_GROUP_HEADER_HEIGHT = 84
const SERVICE_GROUP_PADDING = 12
const REPLICA_NODE_WIDTH = 240
const REPLICA_NODE_HEIGHT = 88
const REPLICA_GAP = 10

interface ServiceGroupLayout {
  width: number
  height: number
  columns: number
}

interface ServiceGroupNodeData extends Record<string, unknown> {
  serviceName: string
  role: string
  runningReplicas: number
  replicas: number
  scalable: boolean
  scaleLoading: boolean
  statusText: string
  statusTone: "running" | "partial" | "stopped"
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

const topologyNodeTypes: NodeTypes = {
  serviceGroup: ServiceGroupNodeView,
  replicaNode: ReplicaNodeView,
}

interface ServiceTopologyCanvasProps {
  snapshot: ContainerRuntimeSnapshot | null
  focusService: string
  actioningKey: string
  onActionContainer: (containerName: string, action: "start" | "stop" | "restart") => void
  onScaleUpService: (serviceName: string, targetReplicas: number) => void
  onScaleDownService: (serviceName: string, targetReplicas: number) => void
  onOpenReplicaLogs: (serviceName: string, containerName: string) => void
  onFocusServiceChange: (serviceName: string) => void
}

function serviceNodeID(name: string): string {
  return `service:${name}`
}

function resolveServiceNameFromNode(node: Node): string {
  const data = node.data as { serviceName?: unknown }
  if (typeof data?.serviceName === "string") {
    return data.serviceName
  }
  if (typeof node.parentId === "string" && node.parentId.startsWith("service:")) {
    return node.parentId.replace("service:", "")
  }
  return ""
}

function calcReplicaColumns(replicaCount: number): number {
  if (replicaCount <= 1) {
    return 1
  }
  if (replicaCount <= 4) {
    return 2
  }
  return 3
}

// buildServiceGroupLayout 根据副本数决定服务分组框尺寸，确保每个副本独立节点可见。
function buildServiceGroupLayout(replicaCount: number): ServiceGroupLayout {
  const safeReplicas = Math.max(0, replicaCount)
  const columns = calcReplicaColumns(safeReplicas)
  const rows = safeReplicas > 0 ? Math.ceil(safeReplicas / columns) : 0
  const contentHeight =
    rows > 0 ? rows * REPLICA_NODE_HEIGHT + (rows - 1) * REPLICA_GAP : REPLICA_NODE_HEIGHT - 24
  const width = Math.max(
    SERVICE_GROUP_MIN_WIDTH,
    columns * REPLICA_NODE_WIDTH + (columns - 1) * REPLICA_GAP + SERVICE_GROUP_PADDING * 2
  )
  return {
    width,
    height: SERVICE_GROUP_HEADER_HEIGHT + contentHeight + SERVICE_GROUP_PADDING,
    columns,
  }
}

function buildFlow(
  snapshot: ContainerRuntimeSnapshot | null,
  focusService: string,
  actioningKey: string,
  onActionContainer: (containerName: string, action: "start" | "stop" | "restart") => void,
  onScaleUpService: (serviceName: string, targetReplicas: number) => void,
  onScaleDownService: (serviceName: string, targetReplicas: number) => void,
  onOpenReplicaLogs: (serviceName: string, containerName: string) => void
): {
  nodes: Node<ServiceGroupNodeData | ReplicaNodeData>[]
  edges: Edge[]
} {
  if (!snapshot) {
    return { nodes: [], edges: [] }
  }

  const containersByService = new Map<string, typeof snapshot.containers>()
  for (const container of snapshot.containers) {
    const list = containersByService.get(container.service) ?? []
    list.push(container)
    containersByService.set(container.service, list)
  }
  for (const list of containersByService.values()) {
    list.sort((a, b) => a.name.localeCompare(b.name))
  }

  const layoutByService = new Map<string, ServiceGroupLayout>()
  const graph = new dagre.graphlib.Graph()
  graph.setDefaultEdgeLabel(() => ({}))
  graph.setGraph({
    rankdir: "LR",
    nodesep: 76,
    ranksep: 168,
    marginx: 48,
    marginy: 48,
  })

  for (const service of snapshot.services) {
    const containerCount = containersByService.get(service.name)?.length ?? 0
    const replicas = Math.max(service.replicas, containerCount)
    const layout = buildServiceGroupLayout(replicas)
    layoutByService.set(service.name, layout)
    graph.setNode(serviceNodeID(service.name), { width: layout.width, height: layout.height })
  }
  for (const edge of snapshot.edges) {
    graph.setEdge(serviceNodeID(edge.from), serviceNodeID(edge.to))
  }
  dagre.layout(graph)

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

  const serviceNodes: Node<ServiceGroupNodeData>[] = snapshot.services.map((service) => {
    const serviceID = serviceNodeID(service.name)
    const layout = graph.node(serviceID)
    const size = layoutByService.get(service.name) ?? buildServiceGroupLayout(service.replicas)
    const status = runtimeStatus(service.running_replicas, service.replicas)
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
        focused: focusService ? focusedNodeSet.has(service.name) : false,
        onScaleUp: onScaleUpService,
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

  const replicaNodes: Node<ReplicaNodeData>[] = []
  for (const service of snapshot.services) {
    const serviceID = serviceNodeID(service.name)
    const containers = containersByService.get(service.name) ?? []
    const targetReplicaCount = Math.max(service.replicas, containers.length)
    const size = layoutByService.get(service.name) ?? buildServiceGroupLayout(targetReplicaCount)
    for (let index = 0; index < targetReplicaCount; index += 1) {
      const container = containers[index]
      const isPlaceholder = !container
      const containerName = isPlaceholder ? `${service.name}-replica-${index + 1}` : container.name
      const column = index % size.columns
      const row = Math.floor(index / size.columns)
      replicaNodes.push({
        id: isPlaceholder ? `replica:virtual:${service.name}:${index + 1}` : `replica:${container.name}`,
        type: "replicaNode",
        parentId: serviceID,
        extent: "parent",
        position: {
          x: SERVICE_GROUP_PADDING + column * (REPLICA_NODE_WIDTH + REPLICA_GAP),
          y: SERVICE_GROUP_HEADER_HEIGHT + row * (REPLICA_NODE_HEIGHT + REPLICA_GAP),
        },
        style: {
          width: REPLICA_NODE_WIDTH,
          height: REPLICA_NODE_HEIGHT,
        },
        data: {
          serviceName: service.name,
          serviceReplicas: service.replicas,
          serviceScalable: service.scalable,
          scaleLoading: actioningKey === `scale:${service.name}`,
          containerName,
          replicaIndex: index + 1,
          running: isPlaceholder ? false : container.running,
          operable: !isPlaceholder,
          health: isPlaceholder ? "missing" : container.health,
          status: isPlaceholder ? "未创建容器实例" : container.status,
          actioningKey,
          onActionContainer,
          onScaleDownService,
          onOpenLogs: onOpenReplicaLogs,
        },
        draggable: false,
        selectable: true,
        zIndex: 2,
      })
    }
  }

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

  return { nodes: [...serviceNodes, ...replicaNodes], edges }
}

// ServiceTopologyCanvas 提供全画布拓扑视图，并将副本拆分为独立节点。
export function ServiceTopologyCanvas({
  snapshot,
  focusService,
  actioningKey,
  onActionContainer,
  onScaleUpService,
  onScaleDownService,
  onOpenReplicaLogs,
  onFocusServiceChange,
}: ServiceTopologyCanvasProps) {
  const flowData = useMemo(
    () =>
      buildFlow(
        snapshot,
        focusService,
        actioningKey,
        onActionContainer,
        onScaleUpService,
        onScaleDownService,
        onOpenReplicaLogs
      ),
    [
      snapshot,
      focusService,
      actioningKey,
      onActionContainer,
      onScaleUpService,
      onScaleDownService,
      onOpenReplicaLogs,
    ]
  )

  return (
    <div className="h-full w-full">
      <ReactFlow
        nodes={flowData.nodes}
        edges={flowData.edges}
        nodeTypes={topologyNodeTypes}
        fitView
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
    </div>
  )
}
