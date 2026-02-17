import type { ContainerRuntimeSnapshot, PromQueryResult, StatusSnapshot } from "@/api/types"
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

// extractScalar 从 Prometheus instant query 结果提取标量值。
function extractScalar(result: PromQueryResult | { error: string } | undefined): number | null {
  if (!result || 'error' in result) return null
  const r = result.data?.result
  if (!r || r.length === 0) return null
  const val = r[0].value
  if (!val || val.length < 2) return null
  const num = parseFloat(val[1])
  return isNaN(num) ? null : num
}

// buildRealtimeSample 将状态快照转换为前端图表采样点。
// metricsSnapshot 可选：如果 Prometheus 可用，真实填充 KPI。
export function buildRealtimeSample(
  status: StatusSnapshot,
  containers: ContainerRuntimeSnapshot,
  metricsSnapshot?: Record<string, PromQueryResult | { error: string }>
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

  const sample: RealtimeSample = {
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

  // 填充 Prometheus 服务端视角指标（独立字段，不与压测客户端字段冲突）
  if (metricsSnapshot) {
    sample.promQps = extractScalar(metricsSnapshot.rpc_request_rate)
    sample.promErrorRate = extractScalar(metricsSnapshot.rpc_error_rate)
    sample.promP99LatencyMs = extractScalar(metricsSnapshot.rpc_p99_latency)
  }

  return sample
}
