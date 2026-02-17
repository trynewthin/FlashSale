import { useCallback, useEffect, useMemo, useRef, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import { buildRealtimeSample } from "@/features/realtime/build-sample"
import type { RealtimeSample } from "@/features/realtime/types"
import { useOpsApiError } from "@/hooks/use-ops-api-error"

const SAMPLE_INTERVAL_MS = 2000
const DEFAULT_WINDOW_SECONDS = 180
const ERROR_TOAST_COOLDOWN_MS = 15000

interface UseRealtimeMonitorResult {
  samples: RealtimeSample[]
  latest: RealtimeSample | null
  previous: RealtimeSample | null
  loading: boolean
  running: boolean
  windowSeconds: number
  setWindowSeconds: (value: number) => void
  setRunning: (value: boolean) => void
  clearSamples: () => void
  refreshNow: () => Promise<void>
}

// useRealtimeMonitor 负责轮询采样、窗口裁剪与错误节流。
export function useRealtimeMonitor(): UseRealtimeMonitorResult {
  const showApiError = useOpsApiError()
  const [samples, setSamples] = useState<RealtimeSample[]>([])
  const [loading, setLoading] = useState(false)
  const [running, setRunning] = useState(true)
  const [windowSeconds, setWindowSeconds] = useState(DEFAULT_WINDOW_SECONDS)
  const lastErrorToastAtRef = useRef(0)
  const hasLoadedRef = useRef(false)

  const maxPoints = useMemo(
    () => Math.max(10, Math.floor((windowSeconds * 1000) / SAMPLE_INTERVAL_MS)),
    [windowSeconds]
  )

  const appendSample = useCallback(
    (sample: RealtimeSample) => {
      setSamples((prev) => [...prev, sample].slice(-maxPoints))
    },
    [maxPoints]
  )

  const poll = useCallback(
    async (manual: boolean) => {
      try {
        if (manual) {
          setLoading(true)
        }
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
        if (manual) {
          setLoading(false)
        }
      }
    },
    [appendSample, showApiError]
  )

  useEffect(() => {
    void poll(false)
  }, [poll])

  useEffect(() => {
    if (!running) {
      return
    }
    const timer = window.setInterval(() => {
      void poll(false)
    }, SAMPLE_INTERVAL_MS)
    return () => window.clearInterval(timer)
  }, [poll, running])

  useEffect(() => {
    setSamples((prev) => prev.slice(-maxPoints))
  }, [maxPoints])

  return {
    samples,
    latest: samples.at(-1) ?? null,
    previous: samples.length > 1 ? samples[samples.length - 2] : null,
    loading,
    running,
    windowSeconds,
    setWindowSeconds,
    setRunning,
    clearSamples: () => setSamples([]),
    refreshNow: async () => {
      await poll(true)
    },
  }
}
