import { useCallback, useEffect, useMemo, useRef, useState } from "react"

import { opsApi, type PersistedSample } from "@/api/modules/ops"
import { buildRealtimeSample } from "@/features/realtime/build-sample"
import type { RealtimeSample } from "@/features/realtime/types"
import { useOpsApiError } from "@/hooks/use-ops-api-error"

const SAMPLE_INTERVAL_MS = 2000
const ERROR_TOAST_COOLDOWN_MS = 15000

// ─── 统一时间窗口 ───
// 短窗口：纯实时（内存轮询）
// 长窗口：自动从后端加载历史 + 续接实时
export type TimeWindow = "1m" | "3m" | "5m" | "1h" | "3h" | "6h" | "12h" | "24h" | "3d" | "7d"

export interface TimeWindowOption {
  value: TimeWindow
  label: string
  seconds: number
  /** 是否需要从后端加载持久化数据 */
  persistent: boolean
  /** 后端 API 的 range 参数（仅 persistent=true 时有效） */
  apiRange?: string
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

function findWindow(value: TimeWindow): TimeWindowOption {
  return TIME_WINDOWS.find((w) => w.value === value) ?? TIME_WINDOWS[1]
}

// 将持久化采样点转换为前端 RealtimeSample
// 注意：后端 label 可能是 UTC 时区，这里用 ts 重新生成本地时区的 label
function persistedToRealtime(p: PersistedSample): RealtimeSample {
  return {
    timestamp: p.ts,
    label: new Date(p.ts).toLocaleTimeString("zh-CN", { hour12: false }),
    portRate: p.portRate,
    httpRate: p.httpRate,
    replicaRate: p.replicaRate,
    runningContainers: p.runningContainers,
    totalContainers: p.totalContainers,
    runningReplicas: p.runningReplicas,
    totalReplicas: p.totalReplicas,
    promQps: p.promQps,
    promP99LatencyMs: p.promP99LatencyMs,
    promErrorRate: p.promErrorRate,
  }
}

interface UseRealtimeMonitorResult {
  samples: RealtimeSample[]
  latest: RealtimeSample | null
  previous: RealtimeSample | null
  loading: boolean
  running: boolean
  timeWindow: TimeWindow
  setTimeWindow: (value: TimeWindow) => void
  setRunning: (value: boolean) => void
  clearSamples: () => void
  refreshNow: () => Promise<void>
}

// useRealtimeMonitor 负责轮询采样、持久化数据加载、窗口裁剪。
// 用户只选一个时间窗口，系统自动决定数据来源：
// - 短窗口（≤5min）：纯内存轮询
// - 长窗口（≥1h）：从后端加载历史数据 + 继续实时追加
export function useRealtimeMonitor(): UseRealtimeMonitorResult {
  const showApiError = useOpsApiError()
  const [samples, setSamples] = useState<RealtimeSample[]>([])
  const [loading, setLoading] = useState(false)
  const [running, setRunning] = useState(true)
  const [timeWindow, setTimeWindowRaw] = useState<TimeWindow>("3m")
  const lastErrorToastAtRef = useRef(0)
  const hasLoadedRef = useRef(false)
  // 用于防止历史数据加载后被覆盖
  const historyLoadedRef = useRef(false)

  const windowDef = useMemo(() => findWindow(timeWindow), [timeWindow])

  const maxPoints = useMemo(
    () => Math.max(10, Math.floor((windowDef.seconds * 1000) / SAMPLE_INTERVAL_MS)),
    [windowDef.seconds]
  )

  // ─── 实时轮询 ───
  const appendSample = useCallback(
    (sample: RealtimeSample) => {
      setSamples((prev) => [...prev, sample].slice(-maxPoints))
    },
    [maxPoints]
  )

  const poll = useCallback(
    async (manual: boolean) => {
      try {
        if (manual) setLoading(true)
        const REALTIME_METRIC_NAMES = ["rpc_request_rate", "rpc_error_rate", "rpc_p99_latency"]
        const [status, containers, metricsResult] = await Promise.all([
          opsApi.getStatus(),
          opsApi.getContainersStatus(),
          opsApi.getMetricsSnapshot(REALTIME_METRIC_NAMES).catch(() => null),
        ])
        appendSample(buildRealtimeSample(status, containers, metricsResult?.snapshot))
        hasLoadedRef.current = true
      } catch (error) {
        const now = Date.now()
        const shouldToast =
          manual || !hasLoadedRef.current || now - lastErrorToastAtRef.current >= ERROR_TOAST_COOLDOWN_MS
        if (shouldToast) {
          showApiError(error, "实时监测采样失败")
          lastErrorToastAtRef.current = now
        }
      } finally {
        if (manual) setLoading(false)
      }
    },
    [appendSample, showApiError]
  )

  // ─── 历史数据加载 ───
  const loadHistoryData = useCallback(
    async (win: TimeWindowOption) => {
      if (!win.persistent || !win.apiRange) return
      setLoading(true)
      try {
        const resp = await opsApi.getSamples(win.apiRange)
        const converted = resp.samples.map(persistedToRealtime)
        setSamples(converted)
        historyLoadedRef.current = true
      } catch (error) {
        showApiError(error, "加载历史数据失败")
      } finally {
        setLoading(false)
      }
    },
    [showApiError]
  )

  // ─── 切换时间窗口 ───
  const setTimeWindow = useCallback(
    (value: TimeWindow) => {
      setTimeWindowRaw(value)
      const win = findWindow(value)
      historyLoadedRef.current = false
      if (win.persistent) {
        // 长窗口：先加载历史，然后继续实时追加
        setSamples([])
        void loadHistoryData(win)
      } else {
        // 短窗口：清空，从头开始实时采集
        setSamples([])
      }
    },
    [loadHistoryData]
  )

  // 首次挂载 / 窗口变化 → 初始拉取一次
  useEffect(() => {
    if (!windowDef.persistent) {
      void poll(false)
    }
    // persistent 窗口由 setTimeWindow 触发 loadHistoryData
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // 持续轮询（无论 persistent 与否都追加实时数据）
  useEffect(() => {
    if (!running) return
    const timer = window.setInterval(() => {
      void poll(false)
    }, SAMPLE_INTERVAL_MS)
    return () => window.clearInterval(timer)
  }, [poll, running])

  // 窗口裁剪
  useEffect(() => {
    setSamples((prev) => prev.slice(-maxPoints))
  }, [maxPoints])

  return {
    samples,
    latest: samples.at(-1) ?? null,
    previous: samples.length > 1 ? samples[samples.length - 2] : null,
    loading,
    running,
    timeWindow,
    setTimeWindow,
    setRunning,
    clearSamples: () => setSamples([]),
    refreshNow: async () => {
      if (windowDef.persistent) {
        await loadHistoryData(windowDef)
      } else {
        await poll(true)
      }
    },
  }
}
