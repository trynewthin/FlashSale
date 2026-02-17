import type { RealtimeSample } from "@/features/realtime/types"

export interface PerfTestMetricPoint {
  timestamp: number
  label: string
  qps: number
  successRate: number
  /** 业务拒绝率 — 库存不足、限购冲突等正常竞争结果 */
  rejectRate: number
  /** 系统异常率 — 非 OK 且非已知业务拒绝的 code */
  systemErrorRate: number
  p95LatencyMs: number
  networkErrorRate: number
  stockDeductionRate: number
}

interface PerfSummaryPayload {
  generated_at?: string
  summary?: {
    total?: number
    success?: number
    rps?: number
    success_rate?: number
    network_error_rate?: number
    latency_p95_ms?: number
    business_code?: Record<string, number>
  }
}

function toFiniteNumber(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value
  }
  if (typeof value === "string") {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) {
      return parsed
    }
  }
  return 0
}

function asPercent(value: unknown): number {
  const num = toFiniteNumber(value)
  const percent = num <= 1 ? num * 100 : num
  return Math.max(0, Math.min(100, Math.round(percent * 100) / 100))
}

function formatTickLabel(ts: number): string {
  return new Date(ts).toLocaleTimeString("zh-CN", { hour12: false })
}

// ─── 业务 code 分类 ───

/** 已知的「正常业务拒绝」code，这些不算系统错误 */
const KNOWN_REJECT_CODES = new Set([
  "SECKILL_OUT_OF_STOCK",
  "SECKILL_PURCHASE_CONFLICT",
  "SECKILL_LIMIT_EXCEEDED",
  "SECKILL_ACTIVITY_ENDED",
  "SECKILL_ACTIVITY_NOT_STARTED",
  "SECKILL_ACTIVITY_NOT_PUBLISHED",
  "SECKILL_ACTIVITY_NOT_FOUND",
  "SECKILL_ITEM_NOT_FOUND",
])

/** 从 business_code 字典计算各类占比 */
function classifyBusinessCodes(
  businessCode: Record<string, number> | undefined,
  total: number
): { rejectRate: number; systemErrorRate: number } {
  if (!businessCode || total <= 0) {
    return { rejectRate: 0, systemErrorRate: 0 }
  }
  let rejectCount = 0
  let systemErrorCount = 0

  for (const [code, count] of Object.entries(businessCode)) {
    const safeCount = Math.max(0, toFiniteNumber(count))
    if (code === "OK" || code === "<empty>") {
      continue // 成功或空 code，跳过
    }
    if (KNOWN_REJECT_CODES.has(code)) {
      rejectCount += safeCount
    } else {
      systemErrorCount += safeCount
    }
  }

  return {
    rejectRate: Math.round((rejectCount / total) * 10000) / 100,
    systemErrorRate: Math.round((systemErrorCount / total) * 10000) / 100,
  }
}

function calcStockDeductionRate(summary: PerfSummaryPayload["summary"]): number {
  const success = Math.max(0, toFiniteNumber(summary?.success))
  const businessCode = summary?.business_code || {}
  const outOfStock = Math.max(0, toFiniteNumber(businessCode["SECKILL_OUT_OF_STOCK"]))
  const purchaseConflict = Math.max(0, toFiniteNumber(businessCode["SECKILL_PURCHASE_CONFLICT"]))
  const denominator = success + outOfStock + purchaseConflict
  if (denominator <= 0) {
    return asPercent(summary?.success_rate)
  }
  return Math.round((success / denominator) * 10000) / 100
}

function parseOnePerfPayload(line: string): PerfTestMetricPoint | null {
  const raw = String(line || "").trim()
  if (!raw) {
    return null
  }
  const jsonStart = raw.indexOf("{")
  if (jsonStart < 0) {
    return null
  }
  const jsonPart = raw.slice(jsonStart)
  let payload: PerfSummaryPayload
  try {
    payload = JSON.parse(jsonPart) as PerfSummaryPayload
  } catch {
    return null
  }
  if (!payload || typeof payload !== "object" || !payload.summary) {
    return null
  }
  const generatedAt = Date.parse(payload.generated_at || "")
  const timestamp = Number.isFinite(generatedAt) ? generatedAt : Date.now()
  const total = Math.max(0, toFiniteNumber(payload.summary.total))
  const successRate = asPercent(payload.summary.success_rate)
  const { rejectRate, systemErrorRate } = classifyBusinessCodes(payload.summary.business_code, total)

  return {
    timestamp,
    label: formatTickLabel(timestamp),
    qps: Math.max(0, toFiniteNumber(payload.summary.rps)),
    successRate,
    rejectRate,
    systemErrorRate,
    p95LatencyMs: Math.max(0, toFiniteNumber(payload.summary.latency_p95_ms)),
    networkErrorRate: asPercent(payload.summary.network_error_rate),
    stockDeductionRate: calcStockDeductionRate(payload.summary),
  }
}

// parsePerfMetricPointsFromLog 从测试日志中提取压测汇总 JSON，生成测试维度采样点。
export function parsePerfMetricPointsFromLog(logText: string): PerfTestMetricPoint[] {
  if (!logText.trim()) {
    return []
  }
  const points: PerfTestMetricPoint[] = []
  const lines = logText.split(/\r?\n/)
  for (const line of lines) {
    const point = parseOnePerfPayload(line)
    if (!point) {
      continue
    }
    points.push(point)
  }
  points.sort((a, b) => a.timestamp - b.timestamp)
  return points
}

// attachPerfMetricsToSamples 把最新测试指标按时间挂载到实时采样点，形成统一图表数据。
export function attachPerfMetricsToSamples(samples: RealtimeSample[], points: PerfTestMetricPoint[]): RealtimeSample[] {
  if (samples.length === 0) {
    return []
  }
  if (points.length === 0) {
    return samples.map((sample) => ({
      ...sample,
      qps: null,
      successRate: null,
      rejectRate: null,
      systemErrorRate: null,
      p95LatencyMs: null,
      networkErrorRate: null,
      stockDeductionRate: null,
    }))
  }

  const sortedPoints = [...points].sort((a, b) => a.timestamp - b.timestamp)
  let pointIndex = 0
  let current: PerfTestMetricPoint | null = null

  return samples.map((sample) => {
    for (; pointIndex < sortedPoints.length; pointIndex += 1) {
      if (sortedPoints[pointIndex].timestamp <= sample.timestamp) {
        current = sortedPoints[pointIndex]
        continue
      }
      break
    }
    return {
      ...sample,
      qps: current?.qps ?? null,
      successRate: current?.successRate ?? null,
      rejectRate: current?.rejectRate ?? null,
      systemErrorRate: current?.systemErrorRate ?? null,
      p95LatencyMs: current?.p95LatencyMs ?? null,
      networkErrorRate: current?.networkErrorRate ?? null,
      stockDeductionRate: current?.stockDeductionRate ?? null,
    }
  })
}

// mergeRealtimeSamplesWithPerfPoints 合并基础采样与测试指标采样，保证测试点可实时入图。
export function mergeRealtimeSamplesWithPerfPoints(
  samples: RealtimeSample[],
  points: PerfTestMetricPoint[]
): RealtimeSample[] {
  if (samples.length === 0) {
    return []
  }
  const base = attachPerfMetricsToSamples(samples, points)
  if (points.length === 0) {
    return base
  }

  const merged = [...base]
  let lastBase = base[0]
  for (const sample of base) {
    if (sample.timestamp > lastBase.timestamp) {
      lastBase = sample
    }
  }

  for (const point of points) {
    const hit = merged.find((item) => Math.abs(item.timestamp - point.timestamp) <= 1000)
    if (hit) {
      hit.qps = point.qps
      hit.successRate = point.successRate
      hit.rejectRate = point.rejectRate
      hit.systemErrorRate = point.systemErrorRate
      hit.p95LatencyMs = point.p95LatencyMs
      hit.networkErrorRate = point.networkErrorRate
      hit.stockDeductionRate = point.stockDeductionRate
      continue
    }
    const copy: RealtimeSample = {
      ...lastBase,
      timestamp: point.timestamp,
      label: point.label,
      qps: point.qps,
      successRate: point.successRate,
      rejectRate: point.rejectRate,
      systemErrorRate: point.systemErrorRate,
      p95LatencyMs: point.p95LatencyMs,
      networkErrorRate: point.networkErrorRate,
      stockDeductionRate: point.stockDeductionRate,
    }
    merged.push(copy)
  }

  merged.sort((left, right) => left.timestamp - right.timestamp)
  return merged
}
