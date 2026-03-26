import { useEffect, useMemo, useRef, useState } from "react"

import { opsApi, type PersistedSample } from "@/api/modules/ops"
import type { JobDetail, PerfProgress } from "@/api/types"

import {
    mergeRealtimeSamplesWithPerfPoints,
    parsePerfMetricPointsFromLog,
} from "@/features/perf-test/perf-report-parser"
import type { RealtimeSample } from "@/features/realtime/types"

function perfProgressToSample(p: PerfProgress): RealtimeSample {
    return {
        timestamp: p.timestamp,
        label: p.label,
        portRate: 0,
        httpRate: 0,
        replicaRate: 0,
        runningContainers: 0,
        totalContainers: 0,
        runningReplicas: 0,
        totalReplicas: 0,
        promQps: p.promQps ?? null,
        promP99LatencyMs: p.promP99LatencyMs ?? null,
        promErrorRate: p.promErrorRate ?? null,
        qps: p.qps,
        successRate: p.successRate,
        rejectRate: p.rejectRate,
        systemErrorRate: p.systemErrorRate,
        p95LatencyMs: p.p95LatencyMs,
        networkErrorRate: p.networkErrorRate,
        stockDeductionRate: p.stockDeductionRate,
        purchaseKafkaPublishRate: p.purchaseKafkaPublishRate ?? null,
        purchaseKafkaPublishFailed: p.purchaseKafkaPublishFailed ?? null,
        orderStateConsumeRate: p.orderStateConsumeRate ?? null,
    }
}

function persistedToRealtime(sample: PersistedSample): RealtimeSample {
    return {
        timestamp: sample.ts,
        label: sample.label,
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

function parseTimeMs(isoTime?: string): number | null {
    if (!isoTime) {
        return null
    }
    const ts = Date.parse(isoTime)
    return Number.isFinite(ts) ? ts : null
}

function chooseMonitorSample(
    timestamp: number,
    previous: RealtimeSample | null,
    next: RealtimeSample | null
): RealtimeSample | null {
    if (!previous) {
        return next
    }
    if (!next) {
        return previous
    }
    return Math.abs(previous.timestamp - timestamp) <= Math.abs(next.timestamp - timestamp)
        ? previous
        : next
}

function attachMonitorMetrics(
    perfSamples: RealtimeSample[],
    monitorSamples: RealtimeSample[]
): RealtimeSample[] {
    if (perfSamples.length === 0 || monitorSamples.length === 0) {
        return perfSamples
    }

    const sortedMonitorSamples = [...monitorSamples].sort((left, right) => left.timestamp - right.timestamp)
    let monitorIndex = 0
    let previousMonitorSample: RealtimeSample | null = null

    return perfSamples.map((sample) => {
        for (; monitorIndex < sortedMonitorSamples.length; monitorIndex += 1) {
            if (sortedMonitorSamples[monitorIndex].timestamp <= sample.timestamp) {
                previousMonitorSample = sortedMonitorSamples[monitorIndex]
                continue
            }
            break
        }

        const nextMonitorSample = monitorIndex < sortedMonitorSamples.length
            ? sortedMonitorSamples[monitorIndex]
            : null
        const matchedMonitorSample = chooseMonitorSample(sample.timestamp, previousMonitorSample, nextMonitorSample)
        if (!matchedMonitorSample) {
            return sample
        }

        return {
            ...sample,
            portRate: matchedMonitorSample.portRate,
            httpRate: matchedMonitorSample.httpRate,
            replicaRate: matchedMonitorSample.replicaRate,
            runningContainers: matchedMonitorSample.runningContainers,
            totalContainers: matchedMonitorSample.totalContainers,
            runningReplicas: matchedMonitorSample.runningReplicas,
            totalReplicas: matchedMonitorSample.totalReplicas,
            promQps: matchedMonitorSample.promQps,
            promP99LatencyMs: matchedMonitorSample.promP99LatencyMs,
            promErrorRate: matchedMonitorSample.promErrorRate,
            purchaseKafkaPublishRate: matchedMonitorSample.purchaseKafkaPublishRate,
            purchaseKafkaPublishFailed: matchedMonitorSample.purchaseKafkaPublishFailed,
            orderStateConsumeRate: matchedMonitorSample.orderStateConsumeRate,
        }
    })
}

const MONITOR_SAMPLE_STEP = "2s"

function buildMonitorStartMs(
    activeJob: JobDetail | null,
    perfStart: number | null
): number | null {
    const jobStart = parseTimeMs(activeJob?.started_at) ?? parseTimeMs(activeJob?.created_at)
    const startCandidates = [perfStart, jobStart].filter((value): value is number => typeof value === "number")
    if (startCandidates.length === 0) {
        return null
    }

    return Math.max(0, Math.min(...startCandidates) - 5000)
}

export interface UsePerfTestSamplesResult {
    chartSamples: RealtimeSample[]
    hasData: boolean
}

// Prefer SSE perf progress, fall back to progress JSON already present in logs,
// and backfill server-side Prometheus samples from persisted monitor data.
export function usePerfTestSamples(
    activeJob: JobDetail | null,
    perfSamples: PerfProgress[],
    logText = ""
): UsePerfTestSamplesResult {
    const perfChartSamples = useMemo(() => {
        const liveSamples = perfSamples.map(perfProgressToSample)
        const perfPoints = parsePerfMetricPointsFromLog(logText)
        return mergeRealtimeSamplesWithPerfPoints(liveSamples, perfPoints)
    }, [logText, perfSamples])
    const perfStartTs = perfChartSamples[0]?.timestamp ?? null
    const perfEndTs = perfChartSamples.length > 0
        ? perfChartSamples[perfChartSamples.length - 1].timestamp
        : null
    const monitorStartMs = useMemo(
        () => buildMonitorStartMs(activeJob, perfStartTs),
        [activeJob, perfStartTs]
    )
    const [monitorSamples, setMonitorSamples] = useState<RealtimeSample[]>([])
    const perfEndRef = useRef<number | null>(null)
    const jobFinishedAtRef = useRef<number | null>(null)
    const jobStatusRef = useRef<string>("")

    useEffect(() => {
        perfEndRef.current = perfEndTs
        jobFinishedAtRef.current = parseTimeMs(activeJob?.finished_at)
        jobStatusRef.current = activeJob?.status ?? ""
    }, [activeJob?.finished_at, activeJob?.status, perfEndTs])

    useEffect(() => {
        if (monitorStartMs == null) {
            return
        }

        let disposed = false

        const loadMonitorSamples = async () => {
            const endMs = Math.max(
                monitorStartMs + 1000,
                perfEndRef.current ?? 0,
                jobFinishedAtRef.current ?? 0,
                jobStatusRef.current === "running" ? Date.now() + 5000 : 0
            )

            try {
                const response = await opsApi.getSamplesByTimeRange(monitorStartMs, endMs, MONITOR_SAMPLE_STEP)
                if (disposed) {
                    return
                }
                setMonitorSamples(response.samples.map(persistedToRealtime))
            } catch {
                if (!disposed) {
                    setMonitorSamples([])
                }
            }
        }

        void loadMonitorSamples()

        if (activeJob?.status !== "running") {
            return () => {
                disposed = true
            }
        }

        const timer = window.setInterval(() => {
            void loadMonitorSamples()
        }, 2000)

        return () => {
            disposed = true
            window.clearInterval(timer)
        }
    }, [activeJob?.id, activeJob?.status, monitorStartMs])

    const chartSamples = useMemo(() => {
        if (perfChartSamples.length === 0) {
            return monitorStartMs == null ? [] : monitorSamples
        }
        if (monitorStartMs == null) {
            return perfChartSamples
        }
        return attachMonitorMetrics(perfChartSamples, monitorSamples)
    }, [monitorSamples, monitorStartMs, perfChartSamples])

    return {
        chartSamples,
        hasData: chartSamples.length > 0,
    }
}
