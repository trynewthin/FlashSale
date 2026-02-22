import type { ContainerRuntimeSnapshot } from "@/api/types"

// ─── Canvas Props ───

export interface ServiceTopologyCanvasProps {
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

// ─── Layout Estimation Constants ───

export const SERVICE_GROUP_MIN_WIDTH = 324
export const HEADER_EST = 68
export const REPLICA_CARD_EST_H = 82
export const REPLICA_GAP_EST = 6
export const REPLICA_AREA_PAD = 18
export const MAX_COLUMNS = 2
export const MAX_VISIBLE_ROWS = 3

// ─── Role Group Visual Config ───

export const ROLE_GROUP_COLORS: Record<string, string> = {
    ingress: "border-sky-200/60 bg-sky-50/30",
    gateway: "border-indigo-200/60 bg-indigo-50/30",
    rpc: "border-emerald-200/60 bg-emerald-50/30",
    infra: "border-amber-200/60 bg-amber-50/30",
    observability: "border-violet-200/60 bg-violet-50/30",
    job: "border-zinc-200/60 bg-zinc-50/30",
}

// ─── Helpers ───

export function serviceNodeID(name: string): string {
    return `service:${name}`
}

export function resolveServiceNameFromNode(node: { data: unknown }): string {
    const data = node.data as { serviceName?: unknown }
    if (typeof data?.serviceName === "string") {
        return data.serviceName
    }
    return ""
}
