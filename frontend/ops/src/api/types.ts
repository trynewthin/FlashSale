// ApiEnvelope 是 ops 服务统一响应包装。
export interface ApiEnvelope<T> {
  code: string
  message: string
  data: T
}

export interface TaskDef {
  id: string
  name: string
  description: string
  default_args?: string[]
  dangerous?: boolean
}

export type JobStatus = "queued" | "running" | "success" | "failed" | string

export interface JobSummary {
  id: string
  task_id: string
  status: JobStatus
  created_at: string
  started_at?: string
  finished_at?: string
  exit_code: number
}

export interface JobDetail extends JobSummary {
  task_name: string
  args: string[]
  perf_report?: PerfReport | null
}

export type JobStreamEventType =
  | "snapshot"
  | "log_line"
  | "job_state"
  | "perf_progress"
  | "done"
  | "error"
  | "ping"

export interface JobStreamEnvelope<T = unknown> {
  type: JobStreamEventType
  job_id: string
  ts: number
  seq: number
  payload?: T
}

export interface JobStreamSnapshotPayload {
  log: string
}

export interface JobStreamLogLinePayload {
  source: string
  line: string
}

export interface JobStreamStatePayload {
  status: JobStatus
  exit_code: number
  started_at?: string
  finished_at?: string
}

export interface JobStreamDonePayload {
  status: JobStatus
  exit_code: number
  finished_at?: string
  perf_report?: PerfReport | null
}

export interface JobStreamErrorPayload {
  message: string
}

export interface JobStreamPingPayload {
  ts: number
}

// ─── 压测数据（后端驱动） ───

export interface PerfProgress {
  timestamp: number
  label: string
  qps: number
  p95LatencyMs: number
  successRate: number
  rejectRate: number
  systemErrorRate: number
  networkErrorRate: number
  stockDeductionRate: number
}

export interface PerfReport {
  generated_at: string
  config: Record<string, unknown>
  summary: Record<string, unknown>
  samples: PerfProgress[]
}

export interface PortStatus {
  name: string
  addr: string
  ok: boolean
}

export interface HTTPStatus {
  name: string
  url: string
  ok: boolean
  status_code: number
}

export interface DockerContainer {
  name: string
  status: string
  ports: string
}

export interface StatusSnapshot {
  now_unix: number
  repo_root: string
  deployment_mode?: "host_process" | "docker_app" | string
  git: {
    branch: string
    commit: string
    dirty: boolean
  }
  ports: PortStatus[]
  http: HTTPStatus[]
  docker: {
    ok: boolean
    error?: string
    containers?: DockerContainer[]
    raw?: string
  }
  files: {
    seed_result_path: string
    seed_result_ok: boolean
  }
}

export interface ServiceLogFile {
  id: string
  rel_path: string
  name: string
  size_byte: number
  mod_unix: number
}

export interface ServiceLogTail {
  file_id: string
  rel_path: string
  lines: number
  truncated: boolean
  text: string
}

export interface SSELogLineFrame {
  line?: string
  chunk?: string
}

export interface ContainerState {
  id: string
  name: string
  service: string
  project: string
  image: string
  status: string
  health: string
  ports: string
  running: boolean
}

export interface ServiceRuntime {
  name: string
  role: string
  scalable: boolean
  optional: boolean
  absent: boolean
  replicas: number
  running_replicas: number
  depends_on: string[]
}

export interface ServiceEdge {
  from: string
  to: string
  type: string
}

export interface ComposeContext {
  compose_file: string
  env_file: string
  project: string
  detected: boolean
}

export interface ContainerRuntimeSnapshot {
  generated_at_unix: number
  deployment_mode: string
  compose: ComposeContext
  services: ServiceRuntime[]
  containers: ContainerState[]
  edges: ServiceEdge[]
  error?: string
}

export interface ObservabilityLink {
  name: string
  url: string
  available: boolean
}

export interface ObservabilityLinks {
  jaeger: ObservabilityLink
  prometheus: ObservabilityLink
  grafana: ObservabilityLink
}

export interface EtcdServiceInstance {
  key: string
  addr: string
}

export interface EtcdService {
  service_key: string
  instances: EtcdServiceInstance[]
}

export interface EtcdRegistrySnapshot {
  available: boolean
  endpoint: string
  services: EtcdService[]
  error?: string
}

// Prometheus 指标代理
export interface MetricCatalogItem {
  name: string
  unit: string
  desc: string
  format: "scalar" | "vector" | "matrix"
}

// Prometheus 原始响应结构 (简化)
export interface PromResult {
  metric: Record<string, string>
  value?: [number, string]      // instant query: [timestamp, value]
  values?: [number, string][]   // range query: [[ts, val], ...]
}

export interface PromQueryResult {
  status: string
  data: {
    resultType: string
    result: PromResult[]
  }
}

export interface MetricsSnapshotResponse {
  snapshot: Record<string, PromQueryResult | { error: string }>
}
