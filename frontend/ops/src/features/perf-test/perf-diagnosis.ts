import type { JobDetail } from "@/api/types"
import type { RealtimeSample } from "@/features/realtime/types"
import { computeAvg, computeLatest, computeMax } from "@/features/perf-test/perf-metric-cards"

export type PerfDiagnosisLevel = "critical" | "high" | "medium" | "info"
export type PerfDiagnosisKind =
  | "kafka_pipeline_risk"
  | "service_saturation"
  | "system_errors"
  | "network_instability"
  | "inventory_competition"
  | "load_generation_limit"
  | "healthy"

export interface PerfDiagnosis {
  kind: PerfDiagnosisKind
  level: PerfDiagnosisLevel
  title: string
  summary: string
  module: string
  confidence: number
  suggestions: string[]
  evidence: string[]
}

export interface PerfDiagnosisReport {
  primary: PerfDiagnosis
  secondary: PerfDiagnosis[]
  notes: string[]
}

interface PerfConfigContext {
  scenario: string | null
  targetRps: number | null
  requests: number | null
  concurrency: number | null
  timeoutMs: number | null
}

interface PerfSnapshot {
  sampleCount: number
  avgQps: number
  latestQps: number
  avgClientP95: number
  latestClientP95: number
  avgServerP99: number
  latestServerP99: number
  avgSuccessRate: number
  latestSuccessRate: number
  avgRejectRate: number
  latestRejectRate: number
  avgSystemErrorRate: number
  latestSystemErrorRate: number
  avgNetworkErrorRate: number
  latestNetworkErrorRate: number
  avgServerErrorRate: number
  latestServerErrorRate: number
  avgStockDeductionRate: number
  latestStockDeductionRate: number
  avgKafkaPublishRate: number
  latestKafkaPublishRate: number
  avgOrderStateConsumeRate: number
  latestOrderStateConsumeRate: number
  maxKafkaPublishFailed: number
  latestKafkaPublishFailed: number
}

function parseNumericLike(value: unknown): number | null {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value
  }
  if (typeof value !== "string") {
    return null
  }
  const trimmed = value.trim()
  if (trimmed.length === 0) {
    return null
  }
  if (/^\d+(\.\d+)?ms$/i.test(trimmed)) {
    const parsedMs = Number.parseFloat(trimmed.slice(0, -2))
    return Number.isFinite(parsedMs) ? parsedMs : null
  }
  if (/^\d+(\.\d+)?s$/i.test(trimmed)) {
    const parsedSeconds = Number.parseFloat(trimmed.slice(0, -1))
    return Number.isFinite(parsedSeconds) ? parsedSeconds * 1000 : null
  }
  const parsed = Number.parseFloat(trimmed)
  return Number.isFinite(parsed) ? parsed : null
}

function readFlag(args: string[], names: string[]): string | null {
  for (let index = 0; index < args.length; index += 1) {
    const token = String(args[index] ?? "").trim()
    for (const name of names) {
      if (token === `-${name}` || token === `--${name}`) {
        return args[index + 1] ?? null
      }
      if (token.startsWith(`-${name}=`)) {
        return token.slice(name.length + 2)
      }
      if (token.startsWith(`--${name}=`)) {
        return token.slice(name.length + 3)
      }
    }
  }
  return null
}

function configFromJob(job: JobDetail | null): PerfConfigContext {
  const config = (job?.perf_report?.config ?? {}) as Record<string, unknown>
  const args = job?.args ?? []

  const scenario = typeof config.scenario === "string"
    ? config.scenario
    : readFlag(args, ["scenario"])

  const targetRps = parseNumericLike(config.open_rate)
    ?? parseNumericLike(readFlag(args, ["rate"]))

  const requests = parseNumericLike(config.requests)
    ?? parseNumericLike(readFlag(args, ["requests"]))

  const concurrency = parseNumericLike(config.concurrency)
    ?? parseNumericLike(readFlag(args, ["concurrency"]))

  const timeoutMs = parseNumericLike(config.timeout_ms)
    ?? parseNumericLike(readFlag(args, ["timeout"]))

  return {
    scenario,
    targetRps,
    requests,
    concurrency,
    timeoutMs,
  }
}

function snapshotFromSamples(samples: RealtimeSample[]): PerfSnapshot {
  return {
    sampleCount: samples.length,
    avgQps: computeAvg(samples, "qps"),
    latestQps: computeLatest(samples, "qps"),
    avgClientP95: computeAvg(samples, "p95LatencyMs"),
    latestClientP95: computeLatest(samples, "p95LatencyMs"),
    avgServerP99: computeAvg(samples, "promP99LatencyMs"),
    latestServerP99: computeLatest(samples, "promP99LatencyMs"),
    avgSuccessRate: computeAvg(samples, "successRate"),
    latestSuccessRate: computeLatest(samples, "successRate"),
    avgRejectRate: computeAvg(samples, "rejectRate"),
    latestRejectRate: computeLatest(samples, "rejectRate"),
    avgSystemErrorRate: computeAvg(samples, "systemErrorRate"),
    latestSystemErrorRate: computeLatest(samples, "systemErrorRate"),
    avgNetworkErrorRate: computeAvg(samples, "networkErrorRate"),
    latestNetworkErrorRate: computeLatest(samples, "networkErrorRate"),
    avgServerErrorRate: computeAvg(samples, "promErrorRate"),
    latestServerErrorRate: computeLatest(samples, "promErrorRate"),
    avgStockDeductionRate: computeAvg(samples, "stockDeductionRate"),
    latestStockDeductionRate: computeLatest(samples, "stockDeductionRate"),
    avgKafkaPublishRate: computeAvg(samples, "purchaseKafkaPublishRate"),
    latestKafkaPublishRate: computeLatest(samples, "purchaseKafkaPublishRate"),
    avgOrderStateConsumeRate: computeAvg(samples, "orderStateConsumeRate"),
    latestOrderStateConsumeRate: computeLatest(samples, "orderStateConsumeRate"),
    maxKafkaPublishFailed: computeMax(samples, "purchaseKafkaPublishFailed"),
    latestKafkaPublishFailed: computeLatest(samples, "purchaseKafkaPublishFailed"),
  }
}

function round(value: number): number {
  return Math.round(value * 10) / 10
}

function pct(value: number | null): string {
  if (value == null || !Number.isFinite(value)) {
    return "-"
  }
  return `${round(value)}%`
}

function ms(value: number): string {
  return `${Math.round(value)} ms`
}

function req(value: number | null): string {
  if (value == null || !Number.isFinite(value)) {
    return "-"
  }
  return `${round(value)} req/s`
}

function clampConfidence(value: number): number {
  return Math.max(0.1, Math.min(0.98, Number.parseFloat(value.toFixed(2))))
}

function diagnosisPriority(level: PerfDiagnosisLevel): number {
  switch (level) {
    case "critical":
      return 0
    case "high":
      return 1
    case "medium":
      return 2
    default:
      return 3
  }
}

export function analyzePerfDiagnostics(
  samples: RealtimeSample[],
  job: JobDetail | null
): PerfDiagnosisReport {
  const snapshot = snapshotFromSamples(samples)
  const config = configFromJob(job)
  const diagnoses: PerfDiagnosis[] = []
  const notes = [
    "当前建议基于压测结果与服务端趋势规则生成，不等于替代日志排障。",
  ]

  if (snapshot.sampleCount < 3) {
    return {
      primary: {
        kind: "healthy",
        level: "info",
        title: "样本不足，暂不判定瓶颈",
        summary: "当前采样点太少，建议至少等待数个进度点后再做自动诊断。",
        module: "监测层",
        confidence: 0.45,
        suggestions: [
          "继续运行压测，至少保留 5 个以上采样点。",
          "优先观察客户端 P95、成功率和系统异常率是否开始偏离。",
        ],
        evidence: [`采样点数 ${snapshot.sampleCount}`],
      },
      secondary: [],
      notes,
    }
  }

  if (snapshot.maxKafkaPublishFailed > 0) {
    diagnoses.push({
      kind: "kafka_pipeline_risk",
      level: "critical",
      title: "Kafka 建单链路已出现发布失败",
      summary: "秒杀建单消息在发布到 Kafka 时已经出现失败，这说明异步建单链路存在真实故障，继续提压只会放大丢单或延迟问题。",
      module: "seckill-rpc -> Kafka",
      confidence: clampConfidence(0.84 + Math.min(0.1, snapshot.maxKafkaPublishFailed / 10)),
      suggestions: [
        "先检查 seckill-rpc 到 Kafka broker 的错误日志、网络连通性、topic 配置和 broker 健康状态。",
        "再核对 Kafka broker 资源、磁盘、ISR 和配额限制，确认不是基础设施层拒绝写入。",
        "如果发布恢复正常但订单状态消费仍长期偏低，再继续排查 order-rpc 消费与回流链路。",
      ],
      evidence: [
        `Kafka 发布失败峰值 ${Math.round(snapshot.maxKafkaPublishFailed)}`,
        `Kafka 发布速率 ${req(snapshot.latestKafkaPublishRate)}`,
        `订单状态消费速率 ${req(snapshot.latestOrderStateConsumeRate)}`,
      ],
    })
  }

  if (
    snapshot.avgSystemErrorRate >= 1
    || snapshot.latestSystemErrorRate >= 1
    || snapshot.avgServerErrorRate >= 1
    || snapshot.latestServerErrorRate >= 1
  ) {
    diagnoses.push({
      kind: "system_errors",
      level: "critical",
      title: "链路已出现真实系统错误",
      summary: "当前不只是秒杀竞争拒绝，而是已经出现 RPC 或服务内部错误，继续加压只会放大失败率。",
      module: "seckill-rpc / order-rpc / 依赖组件",
      confidence: clampConfidence(0.82 + Math.min(0.12, snapshot.avgSystemErrorRate / 10)),
      suggestions: [
        "先停止继续提压，检查 seckill-rpc、order-rpc 以及 Redis/MySQL/Kafka 的错误日志。",
        "如果错误集中在某一个服务，再优先扩容该模块，而不是同时盲目扩全部副本。",
        "当前指标不足以直接断言是内存不足；若要自动给出内存结论，需要补 per-service 内存、OOM 和重启指标。",
      ],
      evidence: [
        `客户端系统异常率 ${pct(snapshot.latestSystemErrorRate)}`,
        `服务端 RPC 异常率 ${pct(snapshot.latestServerErrorRate)}`,
        `客户端成功率 ${pct(snapshot.latestSuccessRate)}`,
      ],
    })
  }

  if (snapshot.avgNetworkErrorRate >= 2 || snapshot.latestNetworkErrorRate >= 2) {
    diagnoses.push({
      kind: "network_instability",
      level: "high",
      title: "网络或入口层已开始丢请求",
      summary: "结果里已经出现连接/超时类失败，这更像入口链路或客户端连接能力不足，而不是纯业务竞争。",
      module: "压测端 / 网关入口",
      confidence: clampConfidence(0.72 + Math.min(0.16, snapshot.avgNetworkErrorRate / 20)),
      suggestions: [
        "先检查压测端连接池、并发上限和机器资源，确认不是 load generator 自身打满。",
        "再检查网关或入口限流配置，避免把入口限流误判成后端处理能力不足。",
        "如果网络错误同时伴随服务端错误率升高，再回到对应服务日志确认是否是后端不可用导致的超时。",
      ],
      evidence: [
        `网络错误率 ${pct(snapshot.latestNetworkErrorRate)}`,
        `客户端 P95 ${ms(snapshot.latestClientP95)}`,
        `服务端 P99 ${ms(snapshot.latestServerP99)}`,
      ],
    })
  }

  const latencyElevated = snapshot.avgClientP95 >= 150 || snapshot.latestClientP95 >= 250
  const serverLatencyElevated = snapshot.avgServerP99 >= 200 || snapshot.latestServerP99 >= 300
  if (
    latencyElevated
    && serverLatencyElevated
    && snapshot.avgSystemErrorRate < 1
    && snapshot.avgNetworkErrorRate < 2
  ) {
    diagnoses.push({
      kind: "service_saturation",
      level: snapshot.latestClientP95 >= 400 || snapshot.latestServerP99 >= 500 ? "high" : "medium",
      title: "服务端已进入处理饱和区间",
      summary: "客户端尾延迟和服务端尾延迟同步抬升，更像秒杀主链路或其依赖被打满，而不是单纯指标口径波动。",
      module: "seckill-rpc / order-rpc",
      confidence: clampConfidence(0.68 + Math.min(0.18, snapshot.latestServerP99 / 2000)),
      suggestions: [
        "优先扩容 seckill-rpc 和 order-rpc，再观察 P95/P99 是否回落。",
        "如果扩容后延迟仍高，继续检查 Redis、MySQL、Kafka 的慢操作和超时日志。",
        "结合库存扣减率与成功率的差值，确认瓶颈更偏向库存落库还是建单/回写。",
      ],
      evidence: [
        `客户端 P95 ${ms(snapshot.latestClientP95)}`,
        `服务端 P99 ${ms(snapshot.latestServerP99)}`,
        `成功率 ${pct(snapshot.latestSuccessRate)}`,
      ],
    })
  }

  if (
    snapshot.avgRejectRate >= 20
    && snapshot.avgSystemErrorRate < 1
    && snapshot.avgNetworkErrorRate < 1
  ) {
    diagnoses.push({
      kind: "inventory_competition",
      level: snapshot.avgRejectRate >= 60 ? "medium" : "info",
      title: "业务拒绝占主导，更像库存或资格竞争",
      summary: "当前失败主要来自秒杀正常竞争，而不是系统错误；如果库存本来就少，这是符合预期的结果。",
      module: "库存与秒杀规则",
      confidence: clampConfidence(0.65 + Math.min(0.2, snapshot.avgRejectRate / 100)),
      suggestions: [
        "先确认测试库存、限购规则和活动窗口是否符合预期，不要把正常竞争误判成系统故障。",
        "如果目标是提高成交量，应增加库存或优化流量分配，而不是只扩服务实例。",
        "若库存扣减率明显低于成功率，再进一步检查 Redis 预扣到订单落库的转化链路。",
      ],
      evidence: [
        `业务拒绝率 ${pct(snapshot.latestRejectRate)}`,
        `系统异常率 ${pct(snapshot.latestSystemErrorRate)}`,
        `库存扣减率 ${pct(snapshot.latestStockDeductionRate)}`,
      ],
    })
  }

  const targetGap = config.targetRps != null && config.targetRps > 0
    ? snapshot.avgQps / config.targetRps
    : null

  if (
    targetGap != null
    && targetGap < 0.7
    && snapshot.avgClientP95 < 150
    && snapshot.avgServerP99 < 250
    && snapshot.avgSystemErrorRate < 1
    && snapshot.avgNetworkErrorRate < 1
    && snapshot.avgRejectRate < 10
  ) {
    diagnoses.push({
      kind: "load_generation_limit",
      level: "medium",
      title: "更像压测端或入口限流未把目标流量打满",
      summary: "目标到达率明显高于实际完成量，但服务端延迟和错误并不高，瓶颈更可能在压测端能力或入口限流。",
      module: "压测端 / 网关入口",
      confidence: clampConfidence(0.7 + Math.min(0.12, (1 - targetGap) * 0.2)),
      suggestions: [
        "先核对压测机 CPU、连接池、并发上限和本地丢弃情况，确认 load generator 没有先打满。",
        "再核对网关限流、来源限制和用户数配置，确认请求没有在入口层提前被拦下。",
        "只有在确认流量真的进入后端后，再根据服务端延迟决定是否扩容业务服务。",
      ],
      evidence: [
        `目标到达率 ${req(config.targetRps)}`,
        `实际平均 QPS ${req(snapshot.avgQps)}`,
        `客户端 P95 ${ms(snapshot.avgClientP95)}`,
      ],
    })
  }

  if (diagnoses.length === 0) {
    diagnoses.push({
      kind: "healthy",
      level: "info",
      title: "当前压测结果没有显示出明显瓶颈",
      summary: "在现有采样窗口内，延迟、错误率和拒绝率都没有表现出异常失衡，系统处于相对健康状态。",
      module: "秒杀主链路",
      confidence: 0.72,
      suggestions: [
        "继续按阶梯提升 RPS，找到下一处延迟或成功率拐点。",
        "若目标是容量评估，优先记录当前稳定区间作为基线，再做分模块扩容实验。",
      ],
      evidence: [
        `平均 QPS ${req(snapshot.avgQps)}`,
        `客户端 P95 ${ms(snapshot.avgClientP95)}`,
        `系统异常率 ${pct(snapshot.avgSystemErrorRate)}`,
      ],
    })
  }

  if (config.targetRps != null) {
    notes.push(`目标到达率 ${req(config.targetRps)}，当前平均完成量 ${req(snapshot.avgQps)}。`)
  } else if (config.requests != null) {
    notes.push(`本次测试配置了总请求数 ${Math.round(config.requests)}。`)
  }

  if (config.concurrency != null) {
    notes.push(`并发上限 ${Math.round(config.concurrency)}，请求超时 ${config.timeoutMs != null ? `${Math.round(config.timeoutMs)} ms` : "-" }。`)
  }

  notes.push("当前未接入按服务内存、GC、OOM 与重启次数，所以不会自动给出“内存不足”结论。")

  diagnoses.sort((left, right) => {
    const priorityDiff = diagnosisPriority(left.level) - diagnosisPriority(right.level)
    if (priorityDiff !== 0) {
      return priorityDiff
    }
    return right.confidence - left.confidence
  })

  return {
    primary: diagnoses[0],
    secondary: diagnoses.slice(1, 3),
    notes,
  }
}
