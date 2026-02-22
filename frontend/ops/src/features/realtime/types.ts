// RealtimeSample 定义实时监测页面的单次采样点。
export interface RealtimeSample {
  timestamp: number
  label: string
  // 基础设施健康
  portRate: number
  httpRate: number
  replicaRate: number
  runningContainers: number
  totalContainers: number
  runningReplicas: number
  totalReplicas: number
  // Prometheus 服务端视角（7×24 持续有值）
  promQps?: number | null
  promP99LatencyMs?: number | null
  promErrorRate?: number | null
  // 压测客户端视角（仅压测进行中有值）
  qps?: number | null
  successRate?: number | null
  /** 业务拒绝率 — 库存不足、限购冲突等正常竞争结果 */
  rejectRate?: number | null
  /** 系统异常率 — 非 OK 且非已知业务拒绝的 code */
  systemErrorRate?: number | null
  p95LatencyMs?: number | null
  networkErrorRate?: number | null
  stockDeductionRate?: number | null
  // 秒杀异步购买队列（服务端 Prometheus）
  purchaseTaskQueueDepth?: number | null
  purchaseTaskQueueCap?: number | null
  purchaseTaskDropped?: number | null
}
