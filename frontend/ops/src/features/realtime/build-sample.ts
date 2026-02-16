import type { ContainerRuntimeSnapshot, StatusSnapshot } from "@/api/types"
import type { RealtimeSample } from "@/features/realtime/types"

function safeRate(okCount: number, total: number): number {
  if (total <= 0) {
    return 0
  }
  return Math.round((okCount / total) * 10000) / 100
}

function formatTickLabel(ts: number): string {
  return new Date(ts).toLocaleTimeString("zh-CN", { hour12: false })
}

// buildRealtimeSample 将状态快照转换为前端图表采样点。
export function buildRealtimeSample(
  status: StatusSnapshot,
  containers: ContainerRuntimeSnapshot
): RealtimeSample {
  const now = Date.now()
  const portsTotal = status.ports.length
  const portsUp = status.ports.filter((item) => item.ok).length
  const httpTotal = status.http.length
  const httpUp = status.http.filter((item) => item.ok).length
  const runningContainers = containers.containers.filter((item) => item.running).length
  const totalContainers = containers.containers.length
  const runningReplicas = containers.services.reduce((sum, item) => sum + item.running_replicas, 0)
  const totalReplicas = containers.services.reduce((sum, item) => sum + item.replicas, 0)

  return {
    timestamp: now,
    label: formatTickLabel(now),
    portRate: safeRate(portsUp, portsTotal),
    httpRate: safeRate(httpUp, httpTotal),
    replicaRate: safeRate(runningReplicas, totalReplicas),
    runningContainers,
    totalContainers,
    runningReplicas,
    totalReplicas,
  }
}
