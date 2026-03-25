import { describe, expect, it } from "vitest"

import type { JobDetail } from "@/api/types"
import { analyzePerfDiagnostics } from "@/features/perf-test/perf-diagnosis"
import type { RealtimeSample } from "@/features/realtime/types"

function makeJob(config: Record<string, unknown> = {}): JobDetail {
  return {
    id: "job-1",
    task_id: "perf.purchase_open",
    task_name: "purchase-open",
    status: "success",
    created_at: "2026-03-21T10:00:00Z",
    started_at: "2026-03-21T10:00:01Z",
    finished_at: "2026-03-21T10:00:31Z",
    exit_code: 0,
    args: [],
    perf_report: {
      generated_at: "2026-03-21T10:00:31Z",
      config,
      summary: {},
      samples: [],
    },
  }
}

function makeSample(
  timestamp: number,
  overrides: Partial<RealtimeSample> = {}
): RealtimeSample {
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
    promQps: 50,
    promP99LatencyMs: 120,
    promErrorRate: 0,
    qps: 120,
    successRate: 100,
    rejectRate: 0,
    systemErrorRate: 0,
    p95LatencyMs: 15,
    networkErrorRate: 0,
    stockDeductionRate: 100,
    purchaseKafkaPublishRate: 100,
    purchaseKafkaPublishFailed: 0,
    orderStateConsumeRate: 100,
    ...overrides,
  }
}

describe("perf-diagnosis", () => {
  it("prioritizes kafka pipeline risk when kafka publish starts failing", () => {
    const samples = [
      makeSample(1_000, { qps: 2200, p95LatencyMs: 220, promP99LatencyMs: 320, purchaseKafkaPublishRate: 2100, orderStateConsumeRate: 1800, purchaseKafkaPublishFailed: 0 }),
      makeSample(2_000, { qps: 2300, p95LatencyMs: 260, promP99LatencyMs: 420, purchaseKafkaPublishRate: 2200, orderStateConsumeRate: 1700, purchaseKafkaPublishFailed: 1 }),
      makeSample(3_000, { qps: 2250, p95LatencyMs: 310, promP99LatencyMs: 480, purchaseKafkaPublishRate: 2150, orderStateConsumeRate: 1600, purchaseKafkaPublishFailed: 3 }),
    ]

    const report = analyzePerfDiagnostics(samples, makeJob({ open_rate: 3000, concurrency: 1500, timeout_ms: 7000 }))

    expect(report.primary.kind).toBe("kafka_pipeline_risk")
    expect(report.primary.level).toBe("critical")
    expect(report.primary.evidence.some((item) => item.includes("Kafka 发布失败峰值 3"))).toBe(true)
  })

  it("identifies load generation limit when target rate is far above actual but latency stays low", () => {
    const samples = [
      makeSample(1_000, { qps: 250, p95LatencyMs: 40, promP99LatencyMs: 60 }),
      makeSample(2_000, { qps: 260, p95LatencyMs: 45, promP99LatencyMs: 65 }),
      makeSample(3_000, { qps: 255, p95LatencyMs: 42, promP99LatencyMs: 58 }),
    ]

    const report = analyzePerfDiagnostics(samples, makeJob({ open_rate: 1000, concurrency: 3000, timeout_ms: 7000 }))

    expect(report.primary.kind).toBe("load_generation_limit")
    expect(report.primary.module).toContain("压测端")
  })

  it("returns healthy when latency, errors, and rejection stay normal", () => {
    const samples = [
      makeSample(1_000),
      makeSample(2_000, { qps: 121, p95LatencyMs: 16, promP99LatencyMs: 22 }),
      makeSample(3_000, { qps: 119, p95LatencyMs: 15, promP99LatencyMs: 24 }),
    ]

    const report = analyzePerfDiagnostics(samples, makeJob({ open_rate: 120, concurrency: 300, timeout_ms: 7000 }))

    expect(report.primary.kind).toBe("healthy")
    expect(report.notes.some((note) => note.includes("内存不足"))).toBe(true)
  })
})
