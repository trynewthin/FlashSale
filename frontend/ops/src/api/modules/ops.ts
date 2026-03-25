import { getOpsAccessKey } from "@/api/core/auth"
import { buildApiUrl, requestJSON } from "@/api/core/http"
import type {
  ContainerRuntimeSnapshot,
  EtcdRegistrySnapshot,
  JobDetail,
  JobSummary,
  MetricCatalogItem,
  MetricsSnapshotResponse,
  ObservabilityLinks,
  PromQueryResult,
  StatusSnapshot,
  SysInfo,
  TaskDef,
} from "@/api/types"
import type { MetricsProfile } from "@/features/realtime/precision-metrics"

export interface CreateJobReq {
  task: string
  args: string[]
}

export interface CreatePerfJobReq {
  task: string
  fields: Record<string, string>
  advanced_args_text?: string
}

export const opsApi = {
  async getStatus(): Promise<StatusSnapshot> {
    const data = await requestJSON<{ status: StatusSnapshot }>("api/v1/status")
    return data.status
  },

  async listTasks(): Promise<TaskDef[]> {
    const data = await requestJSON<{ tasks: TaskDef[] }>("api/v1/tasks")
    return data.tasks || []
  },

  async createJob(req: CreateJobReq): Promise<JobDetail> {
    const data = await requestJSON<{ job: JobDetail }>("api/v1/jobs", {
      method: "POST",
      body: JSON.stringify(req),
    })
    return data.job
  },

  async createPerfJob(req: CreatePerfJobReq): Promise<JobDetail> {
    const data = await requestJSON<{ job: JobDetail }>("api/v1/perf/jobs", {
      method: "POST",
      body: JSON.stringify(req),
    })
    return data.job
  },

  async listJobs(limit = 30): Promise<JobSummary[]> {
    const params = new URLSearchParams({ limit: String(limit) })
    const data = await requestJSON<{ jobs: JobSummary[] }>(`api/v1/jobs?${params.toString()}`)
    return data.jobs || []
  },

  async getJob(jobId: string): Promise<JobDetail> {
    const safeJobId = encodeURIComponent(jobId)
    const data = await requestJSON<{ job: JobDetail }>(`api/v1/jobs/${safeJobId}`)
    return data.job
  },

  async getJobLog(jobId: string): Promise<string> {
    const safeJobId = encodeURIComponent(jobId)
    const data = await requestJSON<{ log: string }>(`api/v1/jobs/${safeJobId}/log`)
    return data.log || ""
  },

  buildJobStreamURL(jobId: string): string {
    const safeJobId = encodeURIComponent(jobId)
    const params = new URLSearchParams()
    const key = getOpsAccessKey()
    if (key) {
      params.set("key", key)
    }
    return buildApiUrl(`api/v1/jobs/${safeJobId}/stream`, params)
  },

  async getContainersStatus(): Promise<ContainerRuntimeSnapshot> {
    const data = await requestJSON<{ status: ContainerRuntimeSnapshot }>("api/v1/containers/status")
    return data.status
  },

  async getContainerLogs(containerName: string, tail = 200): Promise<string> {
    const safeName = encodeURIComponent(containerName)
    const data = await requestJSON<{ text: string }>(`api/v1/containers/${safeName}/logs?tail=${tail}`)
    return data.text || ""
  },

  async getObservabilityLinks(): Promise<ObservabilityLinks> {
    const data = await requestJSON<{ links: ObservabilityLinks }>("api/v1/observability/links")
    return data.links
  },

  async getEtcdServices(): Promise<EtcdRegistrySnapshot> {
    const data = await requestJSON<{ registry: EtcdRegistrySnapshot }>("api/v1/etcd/services")
    return data.registry
  },

  async getMetricsCatalog(): Promise<MetricCatalogItem[]> {
    const data = await requestJSON<{ metrics: MetricCatalogItem[] }>("api/v1/metrics/catalog")
    return data.metrics
  },

  async getMetricsSnapshot(
    names: string[],
    profile: MetricsProfile = "overview"
  ): Promise<MetricsSnapshotResponse> {
    const params = new URLSearchParams({
      names: names.join(","),
      profile,
    })
    const data = await requestJSON<MetricsSnapshotResponse>(
      `api/v1/metrics/snapshot?${params.toString()}`
    )
    return data
  },

  async getMetricsRange(
    name: string,
    start: string,
    end: string,
    step = "15s",
    profile: MetricsProfile = "overview"
  ): Promise<{ name: string; unit: string; result: PromQueryResult }> {
    const params = new URLSearchParams({
      name,
      start,
      end,
      step,
      profile,
    })
    return requestJSON(`api/v1/metrics/range?${params.toString()}`)
  },

  async actionContainer(containerID: string, action: "start" | "stop" | "restart"): Promise<void> {
    const safeID = encodeURIComponent(containerID)
    await requestJSON<{ action: string }>(`api/v1/containers/${safeID}/action`, {
      method: "POST",
      body: JSON.stringify({ action }),
    })
  },

  async scaleService(service: string, replicas: number): Promise<void> {
    const safeService = encodeURIComponent(service)
    await requestJSON<{ service: string }>(`api/v1/services/${safeService}/scale`, {
      method: "POST",
      body: JSON.stringify({ replicas }),
    })
  },

  async getSamples(range_: string, step?: string): Promise<SamplesResponse> {
    let url = `api/v1/samples?range=${range_}`
    if (step) url += `&step=${step}`
    return requestJSON<SamplesResponse>(url)
  },

  async getSamplesByTimeRange(startMs: number, endMs: number, step?: string): Promise<SamplesResponse> {
    let url = `api/v1/samples?start=${startMs}&end=${endMs}`
    if (step) url += `&step=${step}`
    return requestJSON<SamplesResponse>(url)
  },

  async getAvailableDays(): Promise<{ days: string[] }> {
    return requestJSON<{ days: string[] }>("api/v1/samples/days")
  },

  async getSysInfo(): Promise<SysInfo> {
    const data = await requestJSON<{ sysinfo: SysInfo }>("api/v1/sysinfo")
    return data.sysinfo
  },
}

// 持久化采样点（后端字段名用 ts 代替 timestamp）
export interface PersistedSample {
  ts: number
  label: string
  portRate: number
  httpRate: number
  replicaRate: number
  runningContainers: number
  totalContainers: number
  runningReplicas: number
  totalReplicas: number
  promQps?: number | null
  promP99LatencyMs?: number | null
  promErrorRate?: number | null
  purchaseKafkaPublishRate?: number | null
  purchaseKafkaPublishFailed?: number | null
  orderStateConsumeRate?: number | null
}

export interface SamplesResponse {
  samples: PersistedSample[]
  meta: {
    range: string
    step: string
    stepMs: number
    count: number
    startMs: number
    endMs: number
  }
}
