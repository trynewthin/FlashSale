import type { RealtimeSample } from "@/features/realtime/types"

export interface PerfTestMetricPoint {
  timestamp: number
  label: string
  qps: number
  successRate: number
  errorRate: number
  p95LatencyMs: number
  networkErrorRate: number
  stockDeductionRate: number
}

interface PerfSummaryPayload {
  generated_at?: string
  summary?: {
    rps?: number
    success_rate?: number
    network_error_rate?: number
    latency_p95_ms?: number
    success?: number
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
  const successRate = asPercent(payload.summary.success_rate)
  return {
    timestamp,
    label: formatTickLabel(timestamp),
    qps: Math.max(0, toFiniteNumber(payload.summary.rps)),
    successRate,
    errorRate: Math.max(0, Math.min(100, Math.round((100 - successRate) * 100) / 100)),
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
      errorRate: null,
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
      errorRate: current?.errorRate ?? null,
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
      hit.errorRate = point.errorRate
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
      errorRate: point.errorRate,
      p95LatencyMs: point.p95LatencyMs,
      networkErrorRate: point.networkErrorRate,
      stockDeductionRate: point.stockDeductionRate,
    }
    merged.push(copy)
  }

  merged.sort((left, right) => left.timestamp - right.timestamp)
  return merged
}
