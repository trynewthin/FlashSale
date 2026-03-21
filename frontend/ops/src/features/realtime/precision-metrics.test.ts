import { describe, expect, it } from "vitest"

import {
  advancePrecisionMode,
  buildServerSeries,
  BURST_LOOKBACK_SECONDS,
  maskP99Latency,
  mergeServerMetricsIntoSamples,
  OVERVIEW_LOOKBACK_SECONDS,
  PRECISION_HOLD_MS,
  replayMetricStep,
  resolveReplayLookbackSeconds,
  type PrecisionMachineState,
} from "@/features/realtime/precision-metrics"
import type { PromQueryResult } from "@/api/types"
import type { RealtimeSample } from "@/features/realtime/types"

function makeRange(values: Array<[number, string]>): PromQueryResult {
  return {
    status: "success",
    data: {
      resultType: "matrix",
      result: [
        {
          metric: {},
          values,
        },
      ],
    },
  }
}

function makeSample(timestamp: number): RealtimeSample {
  return {
    timestamp,
    label: new Date(timestamp).toLocaleTimeString("zh-CN", { hour12: false }),
    portRate: 100,
    httpRate: 100,
    replicaRate: 100,
    runningContainers: 1,
    totalContainers: 1,
    runningReplicas: 1,
    totalReplicas: 1,
    promQps: 0,
    promP99LatencyMs: null,
    promErrorRate: 0,
  }
}

describe("precision-metrics", () => {
  it("masks p99 when expected requests are below threshold", () => {
    expect(maskP99Latency(0.1, 900, OVERVIEW_LOOKBACK_SECONDS)).toBeNull()
    expect(maskP99Latency(5, 900, BURST_LOOKBACK_SECONDS)).toBe(900)
  })

  it("keeps precision mode active during the 120s hold window", () => {
    const initial: PrecisionMachineState = {
      active: false,
      reason: null,
      lastSatisfiedAtMs: null,
      sampleInsufficient: false,
    }

    const activated = advancePrecisionMode(
      initial,
      { promQps: 50, promP99LatencyMs: 50, promErrorRate: 0 },
      10_000,
      OVERVIEW_LOOKBACK_SECONDS
    )
    expect(activated.active).toBe(true)
    expect(activated.reason).toBe("traffic")

    const held = advancePrecisionMode(
      activated,
      { promQps: 1, promP99LatencyMs: 50, promErrorRate: 0 },
      10_000 + PRECISION_HOLD_MS - 1,
      OVERVIEW_LOOKBACK_SECONDS
    )
    expect(held.active).toBe(true)
    expect(held.reason).toBe("traffic")

    const released = advancePrecisionMode(
      held,
      { promQps: 1, promP99LatencyMs: 50, promErrorRate: 0 },
      10_000 + PRECISION_HOLD_MS + 1,
      OVERVIEW_LOOKBACK_SECONDS
    )
    expect(released.active).toBe(false)
    expect(released.reason).toBeNull()
  })

  it("builds replay server series with masked low-sample p99 and merges by nearest timestamp", () => {
    const qps = makeRange([
      [1000, "2"],
      [1002, "2"],
    ])
    const errors = makeRange([
      [1000, "0"],
      [1002, "0"],
    ])
    const p99 = makeRange([
      [1000, "800"],
      [1002, "900"],
    ])

    const serverSeries = buildServerSeries(qps, errors, p99, BURST_LOOKBACK_SECONDS)
    expect(serverSeries).toHaveLength(2)
    expect(serverSeries[0].promP99LatencyMs).toBeNull()

    const merged = mergeServerMetricsIntoSamples(
      [makeSample(1_000_000), makeSample(1_002_000), makeSample(1_120_000)],
      [
        { timestamp: 1_000_500, label: "10:00:00", promQps: 20, promP99LatencyMs: 250, promErrorRate: 0 },
        { timestamp: 1_002_500, label: "10:00:02", promQps: 25, promP99LatencyMs: 300, promErrorRate: 0 },
      ]
    )

    expect(merged[0].promP99LatencyMs).toBe(250)
    expect(merged[1].promP99LatencyMs).toBe(300)
    expect(merged[2].promP99LatencyMs).toBeNull()
  })

  it("selects replay lookback and step from the requested time span", () => {
    expect(resolveReplayLookbackSeconds(10 * 60 * 1000)).toBe(30)
    expect(resolveReplayLookbackSeconds(90 * 60 * 1000)).toBe(60)
    expect(resolveReplayLookbackSeconds(12 * 60 * 60 * 1000)).toBe(300)
    expect(resolveReplayLookbackSeconds(25 * 60 * 60 * 1000)).toBeNull()

    expect(replayMetricStep(10 * 60 * 1000)).toBe("2s")
    expect(replayMetricStep(90 * 60 * 1000)).toBe("10s")
    expect(replayMetricStep(12 * 60 * 60 * 1000)).toBe("30s")
  })
})
