import type { PromQueryResult } from "@/api/types"
import type { RealtimeSample } from "@/features/realtime/types"

export type MetricsProfile = "overview" | "burst" | "replay"
export type PrecisionReason = "traffic" | "latency" | "errors" | null

export interface ServerMetricSample {
  timestamp: number
  label: string
  promQps?: number | null
  promP99LatencyMs?: number | null
  promErrorRate?: number | null
}

export interface PrecisionMachineState {
  active: boolean
  reason: PrecisionReason
  lastSatisfiedAtMs: number | null
  sampleInsufficient: boolean
}

export const OVERVIEW_LOOKBACK_SECONDS = 300
export const BURST_LOOKBACK_SECONDS = 30
export const PRECISION_HOLD_MS = 120_000
export const MIN_EXPECTED_REQUESTS_FOR_P99 = 100
export const PRECISION_TRAFFIC_QPS = 20
export const PRECISION_LATENCY_MS = 200
export const MAX_SERVER_SAMPLE_MATCH_GAP_MS = 5_000

export function resolveReplayLookbackSeconds(rangeMs: number): number | null {
  if (rangeMs <= 15 * 60 * 1000) {
    return 30
  }
  if (rangeMs <= 2 * 60 * 60 * 1000) {
    return 60
  }
  if (rangeMs <= 24 * 60 * 60 * 1000) {
    return 300
  }
  return null
}

export function replayMetricStep(rangeMs: number): string | null {
  if (rangeMs <= 15 * 60 * 1000) {
    return "2s"
  }
  if (rangeMs <= 2 * 60 * 60 * 1000) {
    return "10s"
  }
  if (rangeMs <= 24 * 60 * 60 * 1000) {
    return "30s"
  }
  return null
}

export function hasSufficientP99Samples(
  qps: number | null | undefined,
  lookbackSeconds: number
): boolean {
  return typeof qps === "number" && Number.isFinite(qps) && qps * lookbackSeconds >= MIN_EXPECTED_REQUESTS_FOR_P99
}

export function maskP99Latency(
  qps: number | null | undefined,
  p99LatencyMs: number | null | undefined,
  lookbackSeconds: number
): number | null {
  if (typeof p99LatencyMs !== "number" || !Number.isFinite(p99LatencyMs)) {
    return null
  }
  if (!hasSufficientP99Samples(qps, lookbackSeconds)) {
    return null
  }
  return p99LatencyMs
}

export function normalizeServerMetrics<
  T extends {
    promQps?: number | null
    promP99LatencyMs?: number | null
    promErrorRate?: number | null
  },
>(sample: T, lookbackSeconds: number): T {
  return {
    ...sample,
    promP99LatencyMs: maskP99Latency(sample.promQps, sample.promP99LatencyMs, lookbackSeconds),
  }
}

export function detectPrecisionReason(
  sample: Pick<ServerMetricSample, "promQps" | "promP99LatencyMs" | "promErrorRate">,
  lookbackSeconds: number
): { reason: PrecisionReason; sampleInsufficient: boolean } {
  const errorRate = finiteNumber(sample.promErrorRate)
  if (typeof errorRate === "number" && errorRate > 0) {
    return { reason: "errors", sampleInsufficient: false }
  }

  const qps = finiteNumber(sample.promQps)
  if (typeof qps === "number" && qps >= PRECISION_TRAFFIC_QPS) {
    return { reason: "traffic", sampleInsufficient: false }
  }

  const sampleInsufficient = !hasSufficientP99Samples(qps, lookbackSeconds)
  const p99 = maskP99Latency(qps, sample.promP99LatencyMs, lookbackSeconds)
  if (typeof p99 === "number" && p99 >= PRECISION_LATENCY_MS) {
    return { reason: "latency", sampleInsufficient: false }
  }

  return { reason: null, sampleInsufficient }
}

export function advancePrecisionMode(
  previous: PrecisionMachineState,
  sample: Pick<ServerMetricSample, "promQps" | "promP99LatencyMs" | "promErrorRate"> | null,
  nowMs: number,
  lookbackSeconds: number
): PrecisionMachineState {
  if (!sample) {
    return previous
  }

  const { reason, sampleInsufficient } = detectPrecisionReason(sample, lookbackSeconds)
  if (reason) {
    return {
      active: true,
      reason,
      lastSatisfiedAtMs: nowMs,
      sampleInsufficient,
    }
  }

  if (!previous.active) {
    return {
      active: false,
      reason: null,
      lastSatisfiedAtMs: previous.lastSatisfiedAtMs,
      sampleInsufficient,
    }
  }

  if (previous.lastSatisfiedAtMs != null && nowMs-previous.lastSatisfiedAtMs < PRECISION_HOLD_MS) {
    return {
      active: true,
      reason: previous.reason,
      lastSatisfiedAtMs: previous.lastSatisfiedAtMs,
      sampleInsufficient,
    }
  }

  return {
    active: false,
    reason: null,
    lastSatisfiedAtMs: previous.lastSatisfiedAtMs,
    sampleInsufficient,
  }
}

export function buildServerSeries(
  qpsResult: PromQueryResult | null | undefined,
  errorResult: PromQueryResult | null | undefined,
  p99Result: PromQueryResult | null | undefined,
  lookbackSeconds: number
): ServerMetricSample[] {
  const points = new Map<number, ServerMetricSample>()

  mergeRangeMetric(points, qpsResult, "promQps")
  mergeRangeMetric(points, errorResult, "promErrorRate")
  mergeRangeMetric(points, p99Result, "promP99LatencyMs")

  return [...points.values()]
    .sort((left, right) => left.timestamp - right.timestamp)
    .map((point) => normalizeServerMetrics(point, lookbackSeconds))
}

export function mergeServerMetricsIntoSamples(
  baseSamples: RealtimeSample[],
  serverSamples: ServerMetricSample[]
): RealtimeSample[] {
  if (baseSamples.length === 0) {
    return serverSamples.map(serverPointToRealtimeSample)
  }
  if (serverSamples.length === 0) {
    return baseSamples
  }

  const sortedServer = [...serverSamples].sort((left, right) => left.timestamp - right.timestamp)
  let serverIndex = 0
  let previous: ServerMetricSample | null = null

  return baseSamples.map((sample) => {
    for (; serverIndex < sortedServer.length; serverIndex += 1) {
      if (sortedServer[serverIndex].timestamp <= sample.timestamp) {
        previous = sortedServer[serverIndex]
        continue
      }
      break
    }

    const next = serverIndex < sortedServer.length ? sortedServer[serverIndex] : null
    const matched = chooseClosestServerSample(sample.timestamp, previous, next)
    if (!matched) {
      return sample
    }

    return {
      ...sample,
      promQps: matched.promQps ?? null,
      promP99LatencyMs: matched.promP99LatencyMs ?? null,
      promErrorRate: matched.promErrorRate ?? null,
    }
  })
}

export function toDatetimeLocalValue(timestampMs: number): string {
  const date = new Date(timestampMs)
  const offsetMs = date.getTimezoneOffset() * 60 * 1000
  return new Date(timestampMs - offsetMs).toISOString().slice(0, 16)
}

export function fromDatetimeLocalValue(value: string): number | null {
  if (!value) {
    return null
  }
  const timestamp = Date.parse(value)
  return Number.isFinite(timestamp) ? timestamp : null
}

function mergeRangeMetric(
  points: Map<number, ServerMetricSample>,
  result: PromQueryResult | null | undefined,
  key: "promQps" | "promP99LatencyMs" | "promErrorRate"
) {
  const series = result?.data?.result?.[0]?.values ?? []
  for (const [ts, rawValue] of series) {
    const value = parseNumber(rawValue)
    const timestamp = Math.round(ts * 1000)
    const existing = points.get(timestamp) ?? {
      timestamp,
      label: new Date(timestamp).toLocaleTimeString("zh-CN", { hour12: false }),
    }
    existing[key] = value
    points.set(timestamp, existing)
  }
}

function chooseClosestServerSample(
  timestamp: number,
  previous: ServerMetricSample | null,
  next: ServerMetricSample | null
): ServerMetricSample | null {
  let matched: ServerMetricSample | null
  if (!previous) {
    matched = next
  } else if (!next) {
    matched = previous
  } else {
    matched = Math.abs(previous.timestamp - timestamp) <= Math.abs(next.timestamp - timestamp) ? previous : next
  }
  if (!matched) {
    return null
  }
  if (Math.abs(matched.timestamp - timestamp) > MAX_SERVER_SAMPLE_MATCH_GAP_MS) {
    return null
  }
  return matched
}

function serverPointToRealtimeSample(point: ServerMetricSample): RealtimeSample {
  return {
    timestamp: point.timestamp,
    label: point.label,
    portRate: 0,
    httpRate: 0,
    replicaRate: 0,
    runningContainers: 0,
    totalContainers: 0,
    runningReplicas: 0,
    totalReplicas: 0,
    promQps: point.promQps ?? null,
    promP99LatencyMs: point.promP99LatencyMs ?? null,
    promErrorRate: point.promErrorRate ?? null,
  }
}

function parseNumber(raw: string): number | null {
  const value = Number.parseFloat(raw)
  return Number.isFinite(value) ? value : null
}

function finiteNumber(value: number | null | undefined): number | null {
  return typeof value === "number" && Number.isFinite(value) ? value : null
}
