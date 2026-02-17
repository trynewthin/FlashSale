import { useCallback, useEffect, useMemo, useRef, useState } from "react"

import { opsApi, type PersistedSample } from "@/api/modules/ops"
import type { JobDetail } from "@/api/types"
import { buildRealtimeSample } from "@/features/realtime/build-sample"
import {
    mergeRealtimeSamplesWithPerfPoints,
    parsePerfMetricPointsFromLog,
} from "@/features/realtime/perf-report-parser"
import type { RealtimeSample } from "@/features/realtime/types"
import { useOpsApiError } from "@/hooks/use-ops-api-error"

const SAMPLE_INTERVAL_MS = 2000
const REALTIME_METRIC_NAMES = ["rpc_request_rate", "rpc_error_rate", "rpc_p99_latency"]

// 将持久化采样点转换为前端 RealtimeSample
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

export type PerfSamplesMode = "idle" | "collecting" | "frozen" | "history"

export interface UsePerfTestSamplesResult {
    /** 合并压测指标后的图表数据 */
    chartSamples: RealtimeSample[]
    /** 当前模式 */
    mode: PerfSamplesMode
    /** 是否正在加载 */
    loading: boolean
    /** 加载某个历史 job 的图表数据（系统采样 + 解析日志） */
    loadHistoricalJob: (job: JobDetail, log: string) => Promise<void>
}

// usePerfTestSamples 管理压测页面的采样生命周期。
//
// 行为说明:
// - idle: 无活跃任务，不采样，无图表数据
// - collecting: 测试运行中，实时轮询系统指标
// - frozen: 测试结束，图表冻结，不再追加
// - history: 查看历史任务，从后端加载该时段数据
export function usePerfTestSamples(
    activeJob: JobDetail | null,
    activeLog: string
): UsePerfTestSamplesResult {
    const showApiError = useOpsApiError()
    const [rawSamples, setRawSamples] = useState<RealtimeSample[]>([])
    const [mode, setMode] = useState<PerfSamplesMode>("idle")
    const [loading, setLoading] = useState(false)
    const prevJobIdRef = useRef<string | null>(null)
    const prevJobStatusRef = useRef<string | null>(null)

    // ─── 根据 activeJob 状态自动切换模式 ───
    const isJobActive = (s: string) => s === "running" || s === "queued"
    const isJobTerminal = (s: string) => s === "success" || s === "failed"

    useEffect(() => {
        if (!activeJob) {
            if (mode !== "history") {
                setMode("idle")
            }
            prevJobIdRef.current = null
            prevJobStatusRef.current = null
            return
        }

        const jobId = activeJob.id
        const jobStatus = activeJob.status
        const isNewJob = jobId !== prevJobIdRef.current

        // ── 场景 1：活跃 job（running/queued）→ 开始采集 ──
        if (isJobActive(jobStatus) && mode !== "collecting") {
            if (isNewJob) setRawSamples([]) // 新任务清空旧数据
            setMode("collecting")
        }

        // ── 场景 2：从活跃状态 → 终态（success/failed）→ 冻结 ──
        if (
            !isNewJob &&
            prevJobStatusRef.current &&
            isJobActive(prevJobStatusRef.current) &&
            isJobTerminal(jobStatus)
        ) {
            setMode("frozen")
        }

        prevJobIdRef.current = jobId
        prevJobStatusRef.current = jobStatus
    }, [activeJob?.id, activeJob?.status]) // eslint-disable-line react-hooks/exhaustive-deps

    // ─── 实时轮询（仅 collecting 模式） ───
    useEffect(() => {
        if (mode !== "collecting") return
        let cancelled = false

        const poll = async () => {
            try {
                const [status, containers, metricsResult] = await Promise.all([
                    opsApi.getStatus(),
                    opsApi.getContainersStatus(),
                    opsApi.getMetricsSnapshot(REALTIME_METRIC_NAMES).catch(() => null),
                ])
                if (cancelled) return
                const sample = buildRealtimeSample(status, containers, metricsResult?.snapshot)
                setRawSamples((prev) => [...prev, sample])
            } catch {
                // 静默失败，不中断采集
            }
        }

        // 立即采集一次
        void poll()
        const timer = window.setInterval(() => void poll(), SAMPLE_INTERVAL_MS)
        return () => {
            cancelled = true
            window.clearInterval(timer)
        }
    }, [mode])

    // ─── 加载历史任务数据 ───
    const loadHistoricalJob = useCallback(
        async (job: JobDetail, log: string) => {
            const startedAt = job.started_at || job.created_at
            const finishedAt = job.finished_at
            if (!startedAt) return

            const startMs = new Date(startedAt).getTime()
            // 给结束时间加 5 秒余量
            const endMs = finishedAt
                ? new Date(finishedAt).getTime() + 5000
                : Date.now()

            if (endMs <= startMs) return

            setLoading(true)
            setMode("history")

            try {
                const resp = await opsApi.getSamplesByTimeRange(startMs, endMs, "2s")
                const systemSamples = resp.samples.map(persistedToRealtime)

                // 解析日志中的压测指标并合并
                const perfPoints = parsePerfMetricPointsFromLog(log)
                const merged = mergeRealtimeSamplesWithPerfPoints(systemSamples, perfPoints)

                setRawSamples(merged)
            } catch (error) {
                showApiError(error, "加载历史测试数据失败")
            } finally {
                setLoading(false)
            }
        },
        [showApiError]
    )

    // ─── 合并压测指标（仅实时模式需要动态 merge） ───
    const testMetricPoints = useMemo(
        () => (mode === "collecting" || mode === "frozen" ? parsePerfMetricPointsFromLog(activeLog) : []),
        [activeLog, mode]
    )

    const chartSamples = useMemo(() => {
        if (mode === "history") {
            // history 模式已在 loadHistoricalJob 中完成 merge
            return rawSamples
        }
        if (testMetricPoints.length === 0) return rawSamples
        return mergeRealtimeSamplesWithPerfPoints(rawSamples, testMetricPoints)
    }, [rawSamples, testMetricPoints, mode])

    return {
        chartSamples,
        mode,
        loading,
        loadHistoricalJob,
    }
}
