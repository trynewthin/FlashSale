import { useMemo } from "react"

import type { PerfProgress } from "@/api/types"
import type { RealtimeSample } from "@/features/realtime/types"

// perfProgressToSample 将后端推送的 PerfProgress 转为 RealtimeSample。
// 系统指标置零（后端驱动模式下不再由前端轮询系统状态）。
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
        promQps: null,
        promP99LatencyMs: null,
        promErrorRate: null,
        purchaseTaskQueueDepth: null,
        purchaseTaskQueueCap: null,
        purchaseTaskDropped: null,
        qps: p.qps,
        successRate: p.successRate,
        rejectRate: p.rejectRate,
        systemErrorRate: p.systemErrorRate,
        p95LatencyMs: p.p95LatencyMs,
        networkErrorRate: p.networkErrorRate,
        stockDeductionRate: p.stockDeductionRate,
    }
}

export interface UsePerfTestSamplesResult {
    /** 图表数据 */
    chartSamples: RealtimeSample[]
    /** 是否有压测数据 */
    hasData: boolean
}

// usePerfTestSamples 将后端驱动的 PerfProgress[] 转换为图表 RealtimeSample[]。
// 不再轮询系统指标或解析日志，所有数据来自 useRealtimeTestRunner 的 SSE 推送。
export function usePerfTestSamples(perfSamples: PerfProgress[]): UsePerfTestSamplesResult {
    const chartSamples = useMemo(
        () => perfSamples.map(perfProgressToSample),
        [perfSamples]
    )

    const hasData = chartSamples.length > 0

    return {
        chartSamples,
        hasData,
    }
}
