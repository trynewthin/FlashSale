import { useCallback, useEffect, useMemo, useRef, useState } from "react"

import { opsApi, type PersistedSample } from "@/api/modules/ops"
import type { PromQueryResult } from "@/api/types"
import { buildRealtimeSample } from "@/features/realtime/build-sample"
import {
  advancePrecisionMode,
  buildServerSeries,
  BURST_LOOKBACK_SECONDS,
  detectPrecisionReason,
  fromDatetimeLocalValue,
  mergeServerMetricsIntoSamples,
  normalizeServerMetrics,
  OVERVIEW_LOOKBACK_SECONDS,
  replayMetricStep,
  resolveReplayLookbackSeconds,
  toDatetimeLocalValue,
  type PrecisionReason,
  type ServerMetricSample,
} from "@/features/realtime/precision-metrics"
import type { RealtimeSample } from "@/features/realtime/types"
import { useOpsApiError } from "@/hooks/use-ops-api-error"

const SAMPLE_INTERVAL_MS = 2000
const ERROR_TOAST_COOLDOWN_MS = 15000
const BURST_FETCH_WINDOW_MS = 3 * 60 * 1000

export type TimeWindow = "1m" | "3m" | "5m" | "1h" | "3h" | "6h" | "12h" | "24h" | "3d" | "7d"
export type ReplayPreset = "5m" | "15m" | "1h"

export interface TimeWindowOption {
  value: TimeWindow
  label: string
  seconds: number
  persistent: boolean
  apiRange?: string
}

export interface ReplayState {
  open: boolean
  active: boolean
  loading: boolean
  error: string | null
  startInput: string
  endInput: string
  samples: RealtimeSample[]
  lookbackSeconds: number | null
  sampleInsufficient: boolean
}

export const TIME_WINDOWS: TimeWindowOption[] = [
  { value: "1m", label: "最近 1 分钟", seconds: 60, persistent: false },
  { value: "3m", label: "最近 3 分钟", seconds: 180, persistent: false },
  { value: "5m", label: "最近 5 分钟", seconds: 300, persistent: false },
  { value: "1h", label: "最近 1 小时", seconds: 3600, persistent: true, apiRange: "1h" },
  { value: "3h", label: "最近 3 小时", seconds: 10800, persistent: true, apiRange: "3h" },
  { value: "6h", label: "最近 6 小时", seconds: 21600, persistent: true, apiRange: "6h" },
  { value: "12h", label: "最近 12 小时", seconds: 43200, persistent: true, apiRange: "12h" },
  { value: "24h", label: "最近 24 小时", seconds: 86400, persistent: true, apiRange: "24h" },
  { value: "3d", label: "最近 3 天", seconds: 259200, persistent: true, apiRange: "3d" },
  { value: "7d", label: "最近 7 天", seconds: 604800, persistent: true, apiRange: "7d" },
]

const REALTIME_METRIC_NAMES = [
  "rpc_request_rate",
  "rpc_error_rate",
  "rpc_p99_latency",
  "seckill_purchase_kafka_publish_rate",
  "seckill_purchase_kafka_publish_failed",
  "seckill_order_state_consume_rate",
]

function findWindow(value: TimeWindow): TimeWindowOption {
  return TIME_WINDOWS.find((windowOption) => windowOption.value === value) ?? TIME_WINDOWS[1]
}

function persistedToRealtime(sample: PersistedSample): RealtimeSample {
  return normalizeServerMetrics({
    timestamp: sample.ts,
    label: new Date(sample.ts).toLocaleTimeString("zh-CN", { hour12: false }),
    portRate: sample.portRate,
    httpRate: sample.httpRate,
    replicaRate: sample.replicaRate,
    runningContainers: sample.runningContainers,
    totalContainers: sample.totalContainers,
    runningReplicas: sample.runningReplicas,
    totalReplicas: sample.totalReplicas,
    promQps: sample.promQps,
    promP99LatencyMs: sample.promP99LatencyMs,
    promErrorRate: sample.promErrorRate,
    purchaseKafkaPublishRate: sample.purchaseKafkaPublishRate,
    purchaseKafkaPublishFailed: sample.purchaseKafkaPublishFailed,
    orderStateConsumeRate: sample.orderStateConsumeRate,
  }, OVERVIEW_LOOKBACK_SECONDS)
}

function defaultReplayInputs(nowMs = Date.now()): { startInput: string; endInput: string } {
  return {
    startInput: toDatetimeLocalValue(nowMs - 15 * 60 * 1000),
    endInput: toDatetimeLocalValue(nowMs),
  }
}

function presetToRange(preset: ReplayPreset, nowMs = Date.now()): { startMs: number; endMs: number } {
  const durationMs = preset === "5m" ? 5 * 60 * 1000 : preset === "15m" ? 15 * 60 * 1000 : 60 * 60 * 1000
  return {
    startMs: nowMs - durationMs,
    endMs: nowMs,
  }
}

async function fetchServerMetricSeries(
  startIso: string,
  endIso: string,
  step: string,
  profile: "burst" | "replay",
  lookbackSeconds: number
): Promise<ServerMetricSample[]> {
  const [qpsResponse, errorResponse, p99Response] = await Promise.all([
    opsApi.getMetricsRange("rpc_request_rate", startIso, endIso, step, profile),
    opsApi.getMetricsRange("rpc_error_rate", startIso, endIso, step, profile),
    opsApi.getMetricsRange("rpc_p99_latency", startIso, endIso, step, profile),
  ])

  return buildServerSeries(
    qpsResponse.result as PromQueryResult,
    errorResponse.result as PromQueryResult,
    p99Response.result as PromQueryResult,
    lookbackSeconds
  )
}

function latestServerSample(serverSamples: ServerMetricSample[]): ServerMetricSample | null {
  return serverSamples.length > 0 ? serverSamples[serverSamples.length - 1] : null
}

function replayIsActive(replayState: ReplayState): boolean {
  return replayState.open && replayState.active
}

interface UseRealtimeMonitorResult {
  samples: RealtimeSample[]
  latest: RealtimeSample | null
  previous: RealtimeSample | null
  loading: boolean
  running: boolean
  mode: "live" | "replay"
  timeWindow: TimeWindow
  precisionActive: boolean
  precisionReason: PrecisionReason
  sampleInsufficient: boolean
  setTimeWindow: (value: TimeWindow) => void
  setRunning: (value: boolean) => void
  clearSamples: () => void
  refreshNow: () => Promise<void>
  replayState: ReplayState
  openReplay: (preset?: ReplayPreset | { startMs: number; endMs: number }) => void
  closeReplay: () => void
  setReplayStartInput: (value: string) => void
  setReplayEndInput: (value: string) => void
  applyReplayPreset: (preset: ReplayPreset) => void
  runReplay: () => Promise<void>
}

export function useRealtimeMonitor(): UseRealtimeMonitorResult {
  const showApiError = useOpsApiError()
  const [overviewSamples, setOverviewSamples] = useState<RealtimeSample[]>([])
  const [precisionServerSamples, setPrecisionServerSamples] = useState<ServerMetricSample[]>([])
  const [loading, setLoading] = useState(false)
  const [running, setRunning] = useState(true)
  const [timeWindow, setTimeWindowRaw] = useState<TimeWindow>("3m")
  const [precisionState, setPrecisionState] = useState({
    active: false,
    reason: null as PrecisionReason,
    lastSatisfiedAtMs: null as number | null,
    sampleInsufficient: false,
  })
  const [replayState, setReplayState] = useState<ReplayState>(() => {
    const inputs = defaultReplayInputs()
    return {
      open: false,
      active: false,
      loading: false,
      error: null,
      startInput: inputs.startInput,
      endInput: inputs.endInput,
      samples: [],
      lookbackSeconds: null,
      sampleInsufficient: false,
    }
  })
  const lastErrorToastAtRef = useRef(0)
  const hasLoadedRef = useRef(false)

  const windowDef = useMemo(() => findWindow(timeWindow), [timeWindow])
  const maxPoints = useMemo(
    () => Math.max(10, Math.floor((windowDef.seconds * 1000) / SAMPLE_INTERVAL_MS)),
    [windowDef.seconds]
  )

  const appendOverviewSample = useCallback(
    (sample: RealtimeSample) => {
      setOverviewSamples((previous) => [...previous, sample].slice(-maxPoints))
    },
    [maxPoints]
  )

  const handleRealtimeError = useCallback((error: unknown, message: string, manual: boolean) => {
    const now = Date.now()
    const shouldToast =
      manual || !hasLoadedRef.current || now - lastErrorToastAtRef.current >= ERROR_TOAST_COOLDOWN_MS
    if (shouldToast) {
      showApiError(error, message)
      lastErrorToastAtRef.current = now
    }
  }, [showApiError])

  const pollOverview = useCallback(
    async (manual: boolean) => {
      try {
        if (manual) {
          setLoading(true)
        }
        const [status, containers, metricsResult] = await Promise.all([
          opsApi.getStatus(),
          opsApi.getContainersStatus(),
          opsApi.getMetricsSnapshot(REALTIME_METRIC_NAMES, "overview").catch(() => null),
        ])
        appendOverviewSample(buildRealtimeSample(status, containers, metricsResult?.snapshot))
        hasLoadedRef.current = true
      } catch (error) {
        handleRealtimeError(error, "实时监测采样失败", manual)
      } finally {
        if (manual) {
          setLoading(false)
        }
      }
    },
    [appendOverviewSample, handleRealtimeError]
  )

  const loadHistoryData = useCallback(
    async (windowOption: TimeWindowOption) => {
      if (!windowOption.persistent || !windowOption.apiRange) {
        return
      }
      setLoading(true)
      try {
        const response = await opsApi.getSamples(windowOption.apiRange)
        setOverviewSamples(response.samples.map(persistedToRealtime))
      } catch (error) {
        showApiError(error, "加载历史数据失败")
      } finally {
        setLoading(false)
      }
    },
    [showApiError]
  )

  const loadBurstPrecision = useCallback(async () => {
    const endMs = Date.now()
    const startMs = endMs - BURST_FETCH_WINDOW_MS
    try {
      const serverSeries = await fetchServerMetricSeries(
        new Date(startMs).toISOString(),
        new Date(endMs).toISOString(),
        "2s",
        "burst",
        BURST_LOOKBACK_SECONDS
      )
      setPrecisionServerSamples(serverSeries)
    } catch (error) {
      handleRealtimeError(error, "高精度服务端指标拉取失败", false)
    }
  }, [handleRealtimeError])

  const loadReplayRange = useCallback(async (startMs: number, endMs: number) => {
    const rangeMs = endMs - startMs
    const lookbackSeconds = resolveReplayLookbackSeconds(rangeMs)
    const step = replayMetricStep(rangeMs)
    if (lookbackSeconds == null || step == null) {
      setReplayState((previous) => ({
        ...previous,
        open: true,
        active: false,
        loading: false,
        error: "高精度回放仅支持 24 小时内的时间段，请改用趋势视图。",
        samples: [],
        lookbackSeconds: null,
        sampleInsufficient: false,
      }))
      return
    }

    setReplayState((previous) => ({
      ...previous,
      open: true,
      loading: true,
      error: null,
      startInput: toDatetimeLocalValue(startMs),
      endInput: toDatetimeLocalValue(endMs),
    }))

    try {
      const [infraResponse, serverSeries] = await Promise.all([
        opsApi.getSamplesByTimeRange(startMs, endMs),
        fetchServerMetricSeries(
          new Date(startMs).toISOString(),
          new Date(endMs).toISOString(),
          step,
          "replay",
          lookbackSeconds
        ),
      ])
      const infraSamples = infraResponse.samples.map(persistedToRealtime)
      const mergedSamples = mergeServerMetricsIntoSamples(infraSamples, serverSeries)
      const latestReplayServerSample = latestServerSample(serverSeries)
      const sampleInsufficient = latestReplayServerSample
        ? detectPrecisionReason(latestReplayServerSample, lookbackSeconds).sampleInsufficient
        : false
      setReplayState((previous) => ({
        ...previous,
        open: true,
        active: true,
        loading: false,
        error: null,
        samples: mergedSamples,
        lookbackSeconds,
        sampleInsufficient,
      }))
    } catch (error) {
      setReplayState((previous) => ({
        ...previous,
        open: true,
        active: false,
        loading: false,
        error: error instanceof Error ? error.message : "加载异常片段失败",
        samples: [],
        lookbackSeconds: null,
        sampleInsufficient: false,
      }))
      showApiError(error, "加载异常片段失败")
    }
  }, [showApiError])

  useEffect(() => {
    if (!windowDef.persistent) {
      void pollOverview(false)
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!running) {
      return
    }
    const timer = window.setInterval(() => {
      void pollOverview(false)
    }, SAMPLE_INTERVAL_MS)
    return () => window.clearInterval(timer)
  }, [pollOverview, running])

  useEffect(() => {
    setOverviewSamples((previous) => previous.slice(-maxPoints))
  }, [maxPoints])

  useEffect(() => {
    const latestOverviewSample = overviewSamples.at(-1) ?? null
    if (!latestOverviewSample || replayIsActive(replayState)) {
      return
    }
    setPrecisionState((previous) =>
      advancePrecisionMode(previous, latestOverviewSample, latestOverviewSample.timestamp, OVERVIEW_LOOKBACK_SECONDS)
    )
  }, [overviewSamples, replayState])

  useEffect(() => {
    if (!precisionState.active || replayIsActive(replayState)) {
      setPrecisionServerSamples([])
      return
    }

    void loadBurstPrecision()
    const timer = window.setInterval(() => {
      void loadBurstPrecision()
    }, SAMPLE_INTERVAL_MS)
    return () => window.clearInterval(timer)
  }, [loadBurstPrecision, precisionState.active, replayState])

  const setTimeWindow = useCallback(
    (value: TimeWindow) => {
      setTimeWindowRaw(value)
      setPrecisionServerSamples([])
      const nextWindow = findWindow(value)
      if (nextWindow.persistent) {
        setOverviewSamples([])
        void loadHistoryData(nextWindow)
        return
      }
      setOverviewSamples([])
      void pollOverview(false)
    },
    [loadHistoryData, pollOverview]
  )

  const openReplay = useCallback((preset?: ReplayPreset | { startMs: number; endMs: number }) => {
    if (!preset) {
      const inputs = defaultReplayInputs()
      setReplayState((previous) => ({
        ...previous,
        open: true,
        error: null,
        startInput: inputs.startInput,
        endInput: inputs.endInput,
      }))
      return
    }

    if (typeof preset === "string") {
      const range = presetToRange(preset)
      void loadReplayRange(range.startMs, range.endMs)
      return
    }

    void loadReplayRange(preset.startMs, preset.endMs)
  }, [loadReplayRange])

  const closeReplay = useCallback(() => {
    const inputs = defaultReplayInputs()
    setReplayState((previous) => ({
      ...previous,
      open: false,
      active: false,
      loading: false,
      error: null,
      samples: [],
      lookbackSeconds: null,
      sampleInsufficient: false,
      startInput: inputs.startInput,
      endInput: inputs.endInput,
    }))
  }, [])

  const applyReplayPreset = useCallback((preset: ReplayPreset) => {
    const range = presetToRange(preset)
    void loadReplayRange(range.startMs, range.endMs)
  }, [loadReplayRange])

  const runReplay = useCallback(async () => {
    const startMs = fromDatetimeLocalValue(replayState.startInput)
    const endMs = fromDatetimeLocalValue(replayState.endInput)
    if (startMs == null || endMs == null || endMs <= startMs) {
      setReplayState((previous) => ({
        ...previous,
        open: true,
        active: false,
        error: "请选择合法的回放时间段。",
      }))
      return
    }
    await loadReplayRange(startMs, endMs)
  }, [loadReplayRange, replayState.endInput, replayState.startInput])

  const liveSamples = useMemo(() => {
    if (!precisionState.active || precisionServerSamples.length === 0) {
      return overviewSamples
    }
    return mergeServerMetricsIntoSamples(overviewSamples, precisionServerSamples)
  }, [overviewSamples, precisionServerSamples, precisionState.active])

  const mode = replayState.active ? "replay" : "live"
  const samples = replayState.active ? replayState.samples : liveSamples
  const latest = samples.at(-1) ?? null
  const previous = samples.length > 1 ? samples[samples.length - 2] : null
  const liveSampleInsufficient = useMemo(() => {
    if (precisionState.active) {
      const latestPrecisionSample = latestServerSample(precisionServerSamples)
      if (!latestPrecisionSample) {
        return false
      }
      return detectPrecisionReason(latestPrecisionSample, BURST_LOOKBACK_SECONDS).sampleInsufficient
    }
    const latestOverviewSample = overviewSamples.at(-1) ?? null
    if (!latestOverviewSample) {
      return false
    }
    return detectPrecisionReason(latestOverviewSample, OVERVIEW_LOOKBACK_SECONDS).sampleInsufficient
  }, [overviewSamples, precisionServerSamples, precisionState.active])

  return {
    samples,
    latest,
    previous,
    loading: loading || replayState.loading,
    running,
    mode,
    timeWindow,
    precisionActive: precisionState.active,
    precisionReason: precisionState.reason,
    sampleInsufficient: replayState.active ? replayState.sampleInsufficient : liveSampleInsufficient,
    setTimeWindow,
    setRunning,
    clearSamples: () => {
      setOverviewSamples([])
      setPrecisionServerSamples([])
    },
    refreshNow: async () => {
      if (replayState.active) {
        await runReplay()
        return
      }
      if (windowDef.persistent) {
        await loadHistoryData(windowDef)
        return
      }
      await pollOverview(true)
    },
    replayState,
    openReplay,
    closeReplay,
    setReplayStartInput: (value: string) => {
      setReplayState((previousState) => ({ ...previousState, startInput: value }))
    },
    setReplayEndInput: (value: string) => {
      setReplayState((previousState) => ({ ...previousState, endInput: value }))
    },
    applyReplayPreset,
    runReplay,
  }
}
