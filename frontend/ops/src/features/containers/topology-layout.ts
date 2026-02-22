import { MarkerType, type Edge, type Node } from "@xyflow/react"

import type { ContainerRuntimeSnapshot } from "@/api/types"
import { roleLabel, runtimeStatus, serviceEtcdKey } from "@/features/containers/shared"
import type { ReplicaItem, ServiceGroupNodeData } from "@/features/containers/topology-nodes"
import {
    SERVICE_GROUP_MIN_WIDTH,
    HEADER_EST,
    REPLICA_CARD_EST_H,
    REPLICA_GAP_EST,
    REPLICA_AREA_PAD,
    MAX_COLUMNS,
    MAX_VISIBLE_ROWS,
    serviceNodeID,
} from "@/features/containers/topology-types"

// ─── Grid Configuration ───

const COLUMN_GAP = 96
const ROW_GAP = 60
const NODE_GAP_X = 32
const NODE_GAP_Y = 24
const GROUP_PAD_X = 32
const GROUP_PAD_Y = 32
const LABEL_H = 32
const CANVAS_MARGIN = 48
const INNER_COLS = 1

// 二维布局表：[col, row]
// 第 0 行: ingress(0) | gateway(1) | rpc(2) | infra(3) | job(4)
// 第 1 行: observability(0)
const ROLE_GRID: { role: string; col: number; row: number }[] = [
    { role: "ingress", col: 0, row: 0 },
    { role: "gateway", col: 1, row: 0 },
    { role: "rpc", col: 2, row: 0 },
    { role: "infra", col: 3, row: 0 },
    { role: "job", col: 4, row: 0 },
    { role: "observability", col: 0, row: 1 },
    { role: "other", col: 4, row: 1 },
]

// ─── Height Estimation ───

function calcGroupHeight(replicaCount: number): number {
    if (replicaCount <= 0) {
        return HEADER_EST + 20
    }
    const rows = Math.ceil(replicaCount / MAX_COLUMNS)
    const visibleRows = Math.min(rows, MAX_VISIBLE_ROWS)
    const replicaAreaHeight =
        visibleRows * REPLICA_CARD_EST_H + Math.max(0, visibleRows - 1) * REPLICA_GAP_EST + REPLICA_AREA_PAD
    return HEADER_EST + replicaAreaHeight
}

// ─── Nearest-side Handle Routing ───

type Side = "top" | "right" | "bottom" | "left"

const SOURCE_HANDLES: { side: Side; id: string }[] = [
    { side: "top", id: "top-s" },
    { side: "right", id: "right-s" },
    { side: "bottom", id: "bottom-s" },
    { side: "left", id: "left-s" },
]
const TARGET_HANDLES: { side: Side; id: string }[] = [
    { side: "top", id: "top-t" },
    { side: "right", id: "right-t" },
    { side: "bottom", id: "bottom-t" },
    { side: "left", id: "left-t" },
]

// 跨组仅允许水平连接
const HORIZONTAL_SOURCE: { side: Side; id: string }[] = [
    { side: "right", id: "right-s" },
    { side: "left", id: "left-s" },
]
const HORIZONTAL_TARGET: { side: Side; id: string }[] = [
    { side: "right", id: "right-t" },
    { side: "left", id: "left-t" },
]

/** 计算节点某一边的中心坐标 */
function sideCenter(
    pos: { x: number; y: number },
    size: { width: number; height: number },
    side: Side
): { x: number; y: number } {
    switch (side) {
        case "top": return { x: pos.x + size.width / 2, y: pos.y }
        case "bottom": return { x: pos.x + size.width / 2, y: pos.y + size.height }
        case "left": return { x: pos.x, y: pos.y + size.height / 2 }
        case "right": return { x: pos.x + size.width, y: pos.y + size.height / 2 }
    }
}

/** 选取两个节点之间距离最短的 source/target handle 对 */
function pickNearestHandles(
    fromName: string,
    toName: string,
    posMap: Map<string, { x: number; y: number }>,
    sizeMap: Map<string, { width: number; height: number }>,
    sameGroup: boolean
): { sourceHandle: string; targetHandle: string } {
    const posA = posMap.get(fromName)
    const sizeA = sizeMap.get(fromName)
    const posB = posMap.get(toName)
    const sizeB = sizeMap.get(toName)

    if (!posA || !sizeA || !posB || !sizeB) {
        return { sourceHandle: "right-s", targetHandle: "left-t" }
    }

    // 同组：四向均可；跨组：仅左右
    const srcHandles = sameGroup ? SOURCE_HANDLES : HORIZONTAL_SOURCE
    const tgtHandles = sameGroup ? TARGET_HANDLES : HORIZONTAL_TARGET

    let bestDist = Infinity
    let bestSH = "right-s"
    let bestTH = "left-t"

    for (const sh of srcHandles) {
        const sp = sideCenter(posA, sizeA, sh.side)
        for (const th of tgtHandles) {
            const tp = sideCenter(posB, sizeB, th.side)
            const dist = (sp.x - tp.x) ** 2 + (sp.y - tp.y) ** 2
            if (dist < bestDist) {
                bestDist = dist
                bestSH = sh.id
                bestTH = th.id
            }
        }
    }

    return { sourceHandle: bestSH, targetHandle: bestTH }
}

// ─── Build Flow ───

export function buildFlow(
    snapshot: ContainerRuntimeSnapshot | null,
    etcdInstanceMap: Map<string, number>,
    focusService: string,
    actioningKey: string,
    onActionContainer: (containerName: string, action: "start" | "stop" | "restart") => void,
    onScaleUpService: (serviceName: string, targetReplicas: number) => void,
    onScaleDownService: (serviceName: string, targetReplicas: number) => void,
    onOpenReplicaLogs: (serviceName: string, containerName: string) => void
): {
    nodes: Node[]
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

    // ─── Layout: role groups as vertical columns ───

    // 按角色分组
    const servicesByRole = new Map<string, typeof snapshot.services>()
    for (const service of snapshot.services) {
        const role = service.role || "other"
        const list = servicesByRole.get(role) ?? []
        list.push(service)
        servicesByRole.set(role, list)
    }

    // 只保留实际有服务的角色
    const activeGrid = ROLE_GRID.filter((g) => servicesByRole.has(g.role))

    // 计算每个节点的尺寸
    const sizeByService = new Map<string, { width: number; height: number }>()
    for (const service of snapshot.services) {
        const containerCount = containersByService.get(service.name)?.length ?? 0
        const replicas = Math.max(service.replicas, containerCount)
        sizeByService.set(service.name, {
            width: SERVICE_GROUP_MIN_WIDTH,
            height: calcGroupHeight(replicas),
        })
    }

    // 第一步：计算每列宽度、每个分组尺寸
    const colWidths = new Map<number, number>()
    const groupSizes = new Map<string, { w: number; h: number }>()

    for (const g of activeGrid) {
        const services = servicesByRole.get(g.role) ?? []
        const nodeW = SERVICE_GROUP_MIN_WIDTH
        const actualCols = Math.min(services.length, INNER_COLS)
        const innerW = actualCols * nodeW + Math.max(0, actualCols - 1) * NODE_GAP_X
        const groupW = innerW + GROUP_PAD_X * 2

        const rowCount = Math.ceil(services.length / INNER_COLS)
        let totalH = LABEL_H + GROUP_PAD_Y
        for (let r = 0; r < rowCount; r++) {
            let maxRowH = 0
            for (let c = 0; c < INNER_COLS; c++) {
                const idx = r * INNER_COLS + c
                if (idx >= services.length) break
                const size = sizeByService.get(services[idx].name) ?? { width: nodeW, height: HEADER_EST }
                maxRowH = Math.max(maxRowH, size.height)
            }
            totalH += maxRowH + NODE_GAP_Y
        }
        totalH -= NODE_GAP_Y
        totalH += GROUP_PAD_Y
        groupSizes.set(g.role, { w: groupW, h: totalH })

        const prevColW = colWidths.get(g.col) ?? 0
        colWidths.set(g.col, Math.max(prevColW, groupW))
    }

    // 第二步：计算每行高度
    const rowHeights = new Map<number, number>()
    for (const g of activeGrid) {
        const gs = groupSizes.get(g.role)!
        const prevRowH = rowHeights.get(g.row) ?? 0
        rowHeights.set(g.row, Math.max(prevRowH, gs.h))
    }

    // 第三步：计算每列/每行起始坐标
    const sortedCols = [...colWidths.keys()].sort((a, b) => a - b)
    const colStartX = new Map<number, number>()
    let cx = CANVAS_MARGIN
    for (const col of sortedCols) {
        colStartX.set(col, cx)
        cx += colWidths.get(col)! + COLUMN_GAP
    }

    const sortedRows = [...rowHeights.keys()].sort((a, b) => a - b)
    const rowStartY = new Map<number, number>()
    let ry = CANVAS_MARGIN
    for (const row of sortedRows) {
        rowStartY.set(row, ry)
        ry += rowHeights.get(row)! + ROW_GAP
    }

    // 第四步：放置节点
    const positionByService = new Map<string, { x: number; y: number }>()
    const groupBounds = new Map<string, { x: number; y: number; width: number; height: number }>()

    for (const g of activeGrid) {
        const services = servicesByRole.get(g.role) ?? []
        const gx = colStartX.get(g.col)!
        const rowH = rowHeights.get(g.row)!
        const gs = groupSizes.get(g.role)!
        // 垂直居中：行高减去组高，偏移一半
        const gy = rowStartY.get(g.row)! + (rowH - gs.h) / 2
        const nodeW = SERVICE_GROUP_MIN_WIDTH

        let cursorY = gy + LABEL_H + GROUP_PAD_Y
        const rowCount = Math.ceil(services.length / INNER_COLS)
        for (let r = 0; r < rowCount; r++) {
            let maxRowH = 0
            for (let c = 0; c < INNER_COLS; c++) {
                const idx = r * INNER_COLS + c
                if (idx >= services.length) break
                const service = services[idx]
                const size = sizeByService.get(service.name) ?? { width: nodeW, height: HEADER_EST }
                positionByService.set(service.name, {
                    x: gx + GROUP_PAD_X + c * (nodeW + NODE_GAP_X),
                    y: cursorY,
                })
                maxRowH = Math.max(maxRowH, size.height)
            }
            cursorY += maxRowH + NODE_GAP_Y
        }

        groupBounds.set(g.role, {
            x: gx,
            y: gy,
            width: gs.w,
            height: gs.h,
        })
    }

    // ─── Focus/highlight sets ───
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

    // ─── Build service nodes ───
    const serviceNodes: Node<ServiceGroupNodeData>[] = snapshot.services.map((service) => {
        const serviceID = serviceNodeID(service.name)
        const size = sizeByService.get(service.name) ?? { width: SERVICE_GROUP_MIN_WIDTH, height: calcGroupHeight(0) }
        const pos = positionByService.get(service.name) ?? { x: 0, y: 0 }
        const status = runtimeStatus(service.running_replicas, service.replicas, service.absent)

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
            position: pos,
            style: { width: size.width, height: size.height },
            draggable: false,
            selectable: true,
            zIndex: 1,
        }
    })

    // ─── Edges (nearest-side routing) ───
    const edges: Edge[] = []

    // Build role lookup for sameGroup check
    const roleByService = new Map<string, string>()
    for (const service of snapshot.services) {
        roleByService.set(service.name, service.role || "other")
    }

    // ─── Role-group edges (always visible, no arrows) ───
    const roleEdgeSet = new Set<string>()
    for (const edge of snapshot.edges) {
        const fromRole = roleByService.get(edge.from)
        const toRole = roleByService.get(edge.to)
        if (fromRole && toRole && fromRole !== toRole) {
            const key = `${fromRole}->${toRole}`
            if (!roleEdgeSet.has(key)) {
                roleEdgeSet.add(key)
                edges.push({
                    id: `role-edge:${key}`,
                    source: `role-group:${fromRole}`,
                    target: `role-group:${toRole}`,
                    type: "default",
                    zIndex: -1,
                    style: {
                        stroke: "#cbd5e1",
                        strokeWidth: 1.5,
                        strokeDasharray: "6 4",
                    },
                })
            }
        }
    }

    // ─── Focused service-level edges ───
    if (focusService) {
        for (const edge of snapshot.edges) {
            if (edge.from === focusService || edge.to === focusService) {
                const sameGroup = roleByService.get(edge.from) === roleByService.get(edge.to)
                const { sourceHandle, targetHandle } = pickNearestHandles(
                    edge.from, edge.to, positionByService, sizeByService, sameGroup
                )
                edges.push({
                    id: `focus-edge:${edge.from}->${edge.to}`,
                    source: serviceNodeID(edge.from),
                    target: serviceNodeID(edge.to),
                    sourceHandle,
                    targetHandle,
                    type: "default",
                    markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14 },
                    zIndex: 10,
                    style: {
                        stroke: "#0ea5e9",
                        strokeWidth: 3,
                        filter: "drop-shadow(0 0 6px rgba(14, 165, 233, 0.6))",
                    },
                })
            }
        }
    }

    // ─── Role group background nodes ───
    const roleGroupNodes: Node[] = []
    for (const [role, bounds] of groupBounds.entries()) {
        roleGroupNodes.push({
            id: `role-group:${role}`,
            type: "roleGroup",
            data: { label: roleLabel(role), role },
            position: { x: bounds.x, y: bounds.y },
            style: { width: bounds.width, height: bounds.height },
            draggable: false,
            selectable: false,
            zIndex: -1,
        })
    }

    return { nodes: [...roleGroupNodes, ...serviceNodes], edges }
}
