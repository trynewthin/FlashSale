export interface RealtimeSample {
  timestamp: number
  label: string
  portRate: number
  httpRate: number
  replicaRate: number
  runningContainers: number
  totalContainers: number
  runningReplicas: number
  totalReplicas: number
  promQps?: number | null
  promP99LatencyMs?: number | null
  promErrorRate?: number | null
  qps?: number | null
  successRate?: number | null
  rejectRate?: number | null
  systemErrorRate?: number | null
  p95LatencyMs?: number | null
  networkErrorRate?: number | null
  stockDeductionRate?: number | null
  purchaseTaskQueueDepth?: number | null
  purchaseTaskQueueCap?: number | null
  purchaseTaskDropped?: number | null
}
