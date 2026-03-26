import { useCallback, useEffect, useMemo, useRef, useState } from "react"

import { opsApi, type PersistedSample } from "@/api/modules/ops"
import type { SysInfo } from "@/api/types"
import type { RealtimeSample } from "@/features/realtime/types"
import { useOpsApiError } from "@/hooks/use-ops-api-error"

// --------------- 常量 ---------------

/** 尾部实时轮询间隔（ms） */
const TAIL_POLL_INTERVAL_MS = 5_000

export type TimeWindow = "1m" | "5m" | "10m" | "30m" | "1h" | "5h" | "12h" | "1d" | "3d" | "7d"

export interface TimeWindowOption {
  value: TimeWindow
  label: string
  seconds: number
  /** 传给后端 range 参数 */
  apiRange: string
}

export const TIME_WINDOWS: TimeWindowOption[] = [
  { value: "1m", label: "最近 1 分钟", seconds: 60, apiRange: "1m" },
  { value: "5m", label: "最近 5 分钟", seconds: 300, apiRange: "5m" },
  { value: "10m", label: "最近 10 分钟", seconds: 600, apiRange: "10m" },
  { value: "30m", label: "最近 30 分钟", seconds: 1800, apiRange: "30m" },
  { value: "1h", label: "最近 1 小时", seconds: 3600, apiRange: "1h" },
  { value: "5h", label: "最近 5 小时", seconds: 18000, apiRange: "5h" },
  { value: "12h", label: "最近 12 小时", seconds: 43200, apiRange: "12h" },
  { value: "1d", label: "最近 1 天", seconds: 86400, apiRange: "1d" },
  { value: "3d", label: "最近 3 天", seconds: 259200, apiRange: "3d" },
  { value: "7d", label: "最近 7 天", seconds: 604800, apiRange: "7d" },
]

// --------------- 工具函数 ---------------

function findWindow(value: TimeWindow): TimeWindowOption {
  return TIME_WINDOWS.find((w) => w.value === value) ?? TIME_WINDOWS[1]
}

function persistedToRealtime(sample: PersistedSample): RealtimeSample {
  return {
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
  }
}

export function toDatetimeLocalValue(timestampMs: number): string {
  const date = new Date(timestampMs)
  const offsetMs = date.getTimezoneOffset() * 60 * 1000
  return new Date(timestampMs - offsetMs).toISOString().slice(0, 16)
}

export function fromDatetimeLocalValue(value: string): number | null {
  if (!value) return null
  const ts = Date.parse(value)
  return Number.isFinite(ts) ? ts : null
}

// --------------- 自定义范围状态 ---------------

export interface CustomRangeState {
  open: boolean
  startInput: string
  endInput: string
}

// --------------- Hook 返回值 ---------------

interface UseRealtimeMonitorResult {
  sysInfo: SysInfo | null
  samples: RealtimeSample[]
  latest: RealtimeSample | null
  previous: RealtimeSample | null
  loading: boolean
  timeWindow: TimeWindow
  /** 是否使用自定义时间范围（而非预设粒度） */
  isCustomRange: boolean
  setTimeWindow: (value: TimeWindow) => void
  refreshNow: () => Promise<void>
  customRange: CustomRangeState
  openCustomRange: () => void
  closeCustomRange: () => void
  setCustomRangeStart: (v: string) => void
  setCustomRangeEnd: (v: string) => void
  applyCustomRange: () => Promise<void>
}

// --------------- Hook 实现 ---------------

export function useRealtimeMonitor(): UseRealtimeMonitorResult {
  const showApiError = useOpsApiError()

  const [samples, setSamples] = useState<RealtimeSample[]>([])
  const [sysInfo, setSysInfo] = useState<SysInfo | null>(null)
  const [loading, setLoading] = useState(false)
  const [timeWindow, setTimeWindowRaw] = useState<TimeWindow>("5m")
  const [isCustomRange, setIsCustomRange] = useState(false)
  const [customRange, setCustomRange] = useState<CustomRangeState>(() => {
    const now = Date.now()
    return {
      open: false,
      startInput: toDatetimeLocalValue(now - 15 * 60 * 1000),
      endInput: toDatetimeLocalValue(now),
    }
  })

  // 用 ref 记录当前查询参数供尾部轮询复用
  const currentQueryRef = useRef<{ mode: "preset"; range: string } | { mode: "custom"; startMs: number; endMs: number }>({
    mode: "preset",
    range: "5m",
  })

  // ---- 核心加载函数 ----

  const loadSamples = useCallback(
    async (query: typeof currentQueryRef.current, opts?: { silent?: boolean }) => {
      if (!opts?.silent) setLoading(true)
      try {
        const response =
          query.mode === "preset"
            ? await opsApi.getSamples(query.range)
            : await opsApi.getSamplesByTimeRange(query.startMs, query.endMs)
        setSamples(response.samples.map(persistedToRealtime))
      } catch (error) {
        if (!opts?.silent) showApiError(error, "加载监控数据失败")
      } finally {
        if (!opts?.silent) setLoading(false)
      }
    },
    [showApiError],
  )

  const loadSysInfo = useCallback(async () => {
    try {
      const info = await opsApi.getSysInfo()
      setSysInfo(info)
    } catch {
      // sysInfo 失败不影响主流程
    }
  }, [])

  // ---- 初始加载 ----

  useEffect(() => {
    const query = currentQueryRef.current
    void loadSamples(query)
    void loadSysInfo()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // ---- 尾部轮询：预设时间窗口（包含当前）持续刷新 ----

  useEffect(() => {
    // 只有预设模式才持续轮询（预设窗口总是包含当前时间）
    if (isCustomRange) return

    const timer = window.setInterval(() => {
      void loadSamples(currentQueryRef.current, { silent: true })
      void loadSysInfo()
    }, TAIL_POLL_INTERVAL_MS)

    return () => window.clearInterval(timer)
  }, [isCustomRange, loadSamples, loadSysInfo])

  // ---- 对外 API ----

  const setTimeWindow = useCallback(
    (value: TimeWindow) => {
      setTimeWindowRaw(value)
      setIsCustomRange(false)
      const query = { mode: "preset" as const, range: findWindow(value).apiRange }
      currentQueryRef.current = query
      void loadSamples(query)
      void loadSysInfo()
    },
    [loadSamples, loadSysInfo],
  )

  const refreshNow = useCallback(async () => {
    setLoading(true)
    try {
      await loadSamples(currentQueryRef.current)
      await loadSysInfo()
    } finally {
      setLoading(false)
    }
  }, [loadSamples, loadSysInfo])

  const openCustomRange = useCallback(() => {
    const now = Date.now()
    setCustomRange((prev) => ({
      ...prev,
      open: true,
      startInput: toDatetimeLocalValue(now - 15 * 60 * 1000),
      endInput: toDatetimeLocalValue(now),
    }))
  }, [])

  const closeCustomRange = useCallback(() => {
    setCustomRange((prev) => ({ ...prev, open: false }))
  }, [])

  const applyCustomRange = useCallback(async () => {
    const startMs = fromDatetimeLocalValue(customRange.startInput)
    const endMs = fromDatetimeLocalValue(customRange.endInput)
    if (startMs == null || endMs == null || endMs <= startMs) {
      showApiError(new Error("请选择合法的时间段"), "时间范围无效")
      return
    }
    setIsCustomRange(true)
    const query = { mode: "custom" as const, startMs, endMs }
    currentQueryRef.current = query
    setCustomRange((prev) => ({ ...prev, open: false }))
    await loadSamples(query)
    await loadSysInfo()
  }, [customRange.startInput, customRange.endInput, loadSamples, loadSysInfo, showApiError])

  // ---- 派生数据 ----

  const latest = useMemo(() => samples.at(-1) ?? null, [samples])
  const previous = useMemo(() => (samples.length > 1 ? samples[samples.length - 2] : null), [samples])

  return {
    sysInfo,
    samples,
    latest,
    previous,
    loading,
    timeWindow,
    isCustomRange,
    setTimeWindow,
    refreshNow,
    customRange,
    openCustomRange,
    closeCustomRange,
    setCustomRangeStart: (v: string) => setCustomRange((prev) => ({ ...prev, startInput: v })),
    setCustomRangeEnd: (v: string) => setCustomRange((prev) => ({ ...prev, endInput: v })),
    applyCustomRange,
  }
}
