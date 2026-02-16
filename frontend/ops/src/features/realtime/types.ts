// RealtimeSample 定义实时监测页面的单次采样点。
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
  qps?: number | null
  successRate?: number | null
  errorRate?: number | null
  p95LatencyMs?: number | null
  networkErrorRate?: number | null
  stockDeductionRate?: number | null
}
