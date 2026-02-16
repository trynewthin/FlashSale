import { getOpsAccessKey } from "@/api/core/auth"
import { buildApiUrl, requestJSON } from "@/api/core/http"
import type {
  ContainerRuntimeSnapshot,
  JobDetail,
  JobSummary,
  ServiceLogFile,
  ServiceLogTail,
  StatusSnapshot,
  TaskDef,
} from "@/api/types"

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

  async listServiceLogFiles(): Promise<ServiceLogFile[]> {
    const data = await requestJSON<{ files: ServiceLogFile[] }>("api/v1/service-logs/files")
    return data.files || []
  },

  async getServiceLogTail(fileId: string, lines: number): Promise<ServiceLogTail> {
    const safeFileId = encodeURIComponent(fileId)
    return requestJSON<ServiceLogTail>(
      `api/v1/service-logs/${safeFileId}/tail?lines=${encodeURIComponent(String(lines))}`
    )
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

  buildServiceLogStreamURL(fileId: string, fromEnd = true): string {
    const safeFileId = encodeURIComponent(fileId)
    const params = new URLSearchParams()
    const key = getOpsAccessKey()
    if (key) {
      params.set("key", key)
    }
    params.set("from_end", fromEnd ? "true" : "false")
    return buildApiUrl(`api/v1/service-logs/${safeFileId}/stream`, params)
  },

  async getContainersStatus(): Promise<ContainerRuntimeSnapshot> {
    const data = await requestJSON<{ status: ContainerRuntimeSnapshot }>("api/v1/containers/status")
    return data.status
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
}
