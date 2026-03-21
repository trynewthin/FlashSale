import {
  Activity,
  Clock3,
  Pause,
  Play,
  RefreshCcw,
  ShieldAlert,
  TerminalSquare,
} from "lucide-react"
import { useCallback, useEffect, useMemo, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import type {
  JobStreamDonePayload,
  JobStreamEnvelope,
  JobStreamErrorPayload,
  JobStreamLogLinePayload,
  JobStreamSnapshotPayload,
  JobStreamStatePayload,
  JobSummary,
  TaskDef,
} from "@/api/types"
import { LogStreamViewer } from "@/components/common/log-stream-viewer"
import { PageShell } from "@/components/layout/page-shell"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useEventSource } from "@/hooks/use-event-source"
import { useOpsApiError } from "@/hooks/use-ops-api-error"
import { cn } from "@/lib/utils"
import { useOpsUIStore } from "@/store/ops-ui-store"

function parseArgs(raw: string): string[] {
  const trimmed = (raw || "").trim()
  if (!trimmed) {
    return []
  }
  return trimmed.split(/\s+/)
}

function statusVariant(status: string): "secondary" | "destructive" | "outline" {
  if (status === "success") {
    return "secondary"
  }
  if (status === "failed") {
    return "destructive"
  }
  return "outline"
}

function formatDateTime(value?: string): string {
  if (!value) {
    return "-"
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(date)
}

function formatDuration(job?: Pick<JobSummary, "started_at" | "finished_at"> | null): string {
  if (!job?.started_at) {
    return "未开始"
  }

  const start = new Date(job.started_at).getTime()
  const end = job.finished_at ? new Date(job.finished_at).getTime() : Date.now()
  if (Number.isNaN(start) || Number.isNaN(end) || end < start) {
    return "计算中"
  }

  const totalSeconds = Math.floor((end - start) / 1000)
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (minutes <= 0) {
    return `${seconds}s`
  }
  return `${minutes}m ${seconds}s`
}

function MetaField({
  label,
  value,
  emphasis = false,
}: {
  label: string
  value: string
  emphasis?: boolean
}) {
  return (
    <div className="rounded-xl border border-border/50 bg-muted/20 px-3 py-3">
      <div className="text-[11px] uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={cn("mt-1 text-sm text-foreground", emphasis && "font-medium")}>{value}</div>
    </div>
  )
}

export function TasksPage() {
  const showApiError = useOpsApiError()
  const showNotice = useOpsUIStore((state) => state.showNotice)
  const selectedJobId = useOpsUIStore((state) => state.selectedJobId)
  const setSelectedJobId = useOpsUIStore((state) => state.setSelectedJobId)

  const [tasks, setTasks] = useState<TaskDef[]>([])
  const [jobs, setJobs] = useState<JobSummary[]>([])
  const [loadingTasks, setLoadingTasks] = useState(false)
  const [loadingJobs, setLoadingJobs] = useState(false)
  const [runningTask, setRunningTask] = useState(false)
  const [streaming, setStreaming] = useState(false)
  const [selectedTask, setSelectedTask] = useState("")
  const [argsText, setArgsText] = useState("")
  const [logText, setLogText] = useState("")
  const [jobStatus, setJobStatus] = useState("")
  const [jobExitCode, setJobExitCode] = useState<number | null>(null)

  const loadTasks = useCallback(async () => {
    try {
      setLoadingTasks(true)
      const data = await opsApi.listTasks()
      setTasks(data)
      if (!selectedTask && data.length > 0) {
        setSelectedTask(data[0].id)
      }
    } catch (error) {
      showApiError(error, "任务列表加载失败")
    } finally {
      setLoadingTasks(false)
    }
  }, [selectedTask, showApiError])

  const loadJobs = useCallback(async () => {
    try {
      setLoadingJobs(true)
      const data = await opsApi.listJobs(40)
      setJobs(data)

      if (data.length === 0) {
        setSelectedJobId("")
        return
      }

      if (!selectedJobId || !data.some((job) => job.id === selectedJobId)) {
        setSelectedJobId(data[0].id)
      }
    } catch (error) {
      showApiError(error, "任务记录加载失败")
    } finally {
      setLoadingJobs(false)
    }
  }, [selectedJobId, setSelectedJobId, showApiError])

  const loadJobLog = useCallback(async (jobId: string) => {
    if (!jobId) {
      return
    }

    try {
      const [text, detail] = await Promise.all([opsApi.getJobLog(jobId), opsApi.getJob(jobId)])
      setLogText(text)
      setJobStatus(detail.status)
      setJobExitCode(detail.exit_code)
      setStreaming(detail.status !== "success" && detail.status !== "failed")
    } catch (error) {
      showApiError(error, "任务日志加载失败")
    }
  }, [showApiError])

  useEffect(() => {
    void loadTasks()
    void loadJobs()

    const timer = window.setInterval(() => {
      void loadJobs()
    }, 3000)
    return () => window.clearInterval(timer)
  }, [loadJobs, loadTasks])

  useEffect(() => {
    if (!selectedJobId) {
      setLogText("")
      setJobStatus("")
      setJobExitCode(null)
      setStreaming(false)
      return
    }

    void loadJobLog(selectedJobId)
  }, [loadJobLog, selectedJobId])

  const selectedTaskDef = useMemo(
    () => tasks.find((item) => item.id === selectedTask) ?? null,
    [selectedTask, tasks],
  )

  const selectedJob = useMemo(
    () => jobs.find((job) => job.id === selectedJobId) ?? null,
    [jobs, selectedJobId],
  )

  const streamURL = useMemo(() => {
    if (!selectedJobId) {
      return ""
    }
    return opsApi.buildJobStreamURL(selectedJobId)
  }, [selectedJobId])

  useEventSource({
    enabled: streaming && !!selectedJobId,
    url: streamURL,
    handlers: {
      onMessageData: (payload) => {
        const envelope = payload as JobStreamEnvelope
        if (!envelope || typeof envelope.type !== "string") {
          return
        }

        switch (envelope.type) {
          case "snapshot": {
            const data = envelope.payload as JobStreamSnapshotPayload
            if (typeof data?.log === "string") {
              setLogText(data.log)
            }
            return
          }
          case "log_line": {
            const data = envelope.payload as JobStreamLogLinePayload
            if (typeof data?.line === "string") {
              setLogText((prev) => `${prev}${prev.endsWith("\n") || prev.length === 0 ? "" : "\n"}${data.line}\n`)
            }
            return
          }
          case "job_state": {
            const data = envelope.payload as JobStreamStatePayload
            setJobStatus(data?.status || "")
            setJobExitCode(typeof data?.exit_code === "number" ? data.exit_code : null)
            return
          }
          case "done": {
            const data = envelope.payload as JobStreamDonePayload
            setJobStatus(data?.status || "")
            setJobExitCode(typeof data?.exit_code === "number" ? data.exit_code : null)
            setStreaming(false)
            return
          }
          case "error": {
            const data = envelope.payload as JobStreamErrorPayload
            if (data?.message) {
              showNotice("error", data.message)
            }
            return
          }
        }
      },
      onError: () => {
        showNotice("error", "任务日志流已断开，请重新开启")
        setStreaming(false)
      },
    },
  })

  const runTask = useCallback(async () => {
    if (!selectedTask) {
      showNotice("error", "请选择任务")
      return
    }

    try {
      setRunningTask(true)
      const job = await opsApi.createJob({
        task: selectedTask,
        args: parseArgs(argsText),
      })

      setJobs((prev) => [job, ...prev.filter((item) => item.id !== job.id)].slice(0, 40))
      setSelectedJobId(job.id)
      setLogText("")
      setJobStatus(job.status)
      setJobExitCode(job.exit_code)
      setStreaming(job.status !== "success" && job.status !== "failed")
      showNotice("success", `任务已创建：${job.id}`)
    } catch (error) {
      showApiError(error, "任务执行失败")
    } finally {
      setRunningTask(false)
    }
  }, [argsText, selectedTask, setSelectedJobId, showApiError, showNotice])

  const displayStatus = jobStatus || selectedJob?.status || "-"
  const displayExitCode = jobExitCode ?? selectedJob?.exit_code ?? null
  const selectedArgsPreview = selectedTaskDef?.default_args?.join(" ") || "无默认参数"

  return (
    <PageShell
      title="任务中心"
      actions={(
        <>
          <Button variant="secondary" size="sm" onClick={() => void loadTasks()} disabled={loadingTasks}>
            <RefreshCcw className="size-4" />
            刷新任务
          </Button>
          <Button variant="secondary" size="sm" onClick={() => void loadJobs()} disabled={loadingJobs}>
            <RefreshCcw className="size-4" />
            刷新记录
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setStreaming((value) => !value)}
            disabled={!selectedJobId}
          >
            {streaming ? <Pause className="size-4" /> : <Play className="size-4" />}
            {streaming ? "停止流" : "开始流"}
          </Button>
        </>
      )}
    >
      <div className="space-y-4">
        <Card className="border-border/50 bg-background/70 shadow-sm">
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center justify-between gap-3 text-base">
              <span>任务执行器</span>
              <Badge variant="outline">{tasks.length} 个可执行任务</Badge>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-3 xl:grid-cols-[minmax(320px,420px)_minmax(0,1fr)_auto]">
              <Select value={selectedTask || undefined} onValueChange={(value) => setSelectedTask(value || "")}>
                <SelectTrigger className="h-10 w-full">
                  <SelectValue placeholder="选择任务" />
                </SelectTrigger>
                <SelectContent>
                  {tasks.map((task) => (
                    <SelectItem key={task.id} value={task.id}>
                      {task.id} - {task.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Input
                value={argsText}
                onChange={(event) => setArgsText(event.target.value)}
                placeholder="额外参数，例如: -rate 120 -open-duration 30s"
                className="h-10"
              />

              <Button className="h-10 min-w-28" onClick={() => void runTask()} disabled={runningTask || loadingTasks}>
                执行任务
              </Button>
            </div>

            {selectedTaskDef ? (
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_minmax(0,0.8fr)]">
                <div className="rounded-xl border border-border/50 bg-muted/20 px-3 py-3">
                  <div className="text-[11px] uppercase tracking-wider text-muted-foreground">任务说明</div>
                  <div className="mt-1 text-sm text-foreground">{selectedTaskDef.description}</div>
                </div>
                <div className="rounded-xl border border-border/50 bg-muted/20 px-3 py-3">
                  <div className="text-[11px] uppercase tracking-wider text-muted-foreground">默认参数</div>
                  <div className="mt-1 font-mono text-sm text-foreground">{selectedArgsPreview}</div>
                </div>
                <div className="rounded-xl border border-border/50 bg-muted/20 px-3 py-3">
                  <div className="text-[11px] uppercase tracking-wider text-muted-foreground">风险等级</div>
                  <div className="mt-1 flex items-center gap-2 text-sm text-foreground">
                    {selectedTaskDef.dangerous ? <ShieldAlert className="size-4 text-destructive" /> : null}
                    <Badge variant={selectedTaskDef.dangerous ? "destructive" : "secondary"}>
                      {selectedTaskDef.dangerous ? "危险任务" : "常规任务"}
                    </Badge>
                  </div>
                </div>
              </div>
            ) : null}
          </CardContent>
        </Card>

        <div className="grid items-start gap-4 xl:grid-cols-[360px_minmax(0,1fr)]">
          <Card className="border-border/50 bg-background/70 shadow-sm">
            <CardHeader className="pb-3">
              <CardTitle className="flex items-center justify-between gap-3 text-base">
                <span>最近任务</span>
                <Badge variant="outline">{jobs.length}</Badge>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="text-xs text-muted-foreground">
                执行任务后会自动选中新 job，右侧立即接管状态和日志查看。
              </div>
              <div className="max-h-[72vh] space-y-2 overflow-y-auto pr-1">
                {jobs.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-border/60 px-4 py-8 text-center text-sm text-muted-foreground">
                    暂无任务记录
                  </div>
                ) : (
                  jobs.map((job) => {
                    const active = selectedJobId === job.id
                    return (
                      <button
                        key={job.id}
                        type="button"
                        onClick={() => {
                          setSelectedJobId(job.id)
                          void loadJobLog(job.id)
                        }}
                        className={cn(
                          "w-full rounded-xl border px-3 py-3 text-left transition-colors",
                          active
                            ? "border-primary/40 bg-primary/8"
                            : "border-border/50 bg-muted/20 hover:bg-muted/35",
                        )}
                      >
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0">
                            <div className="truncate text-sm font-medium text-foreground">{job.task_id}</div>
                            <div className="mt-1 truncate font-mono text-[11px] text-muted-foreground">{job.id}</div>
                          </div>
                          <Badge variant={statusVariant(job.status)}>{job.status}</Badge>
                        </div>
                        <div className="mt-3 flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground">
                          <span>{formatDateTime(job.created_at)}</span>
                          <span>{formatDuration(job)}</span>
                          {typeof job.exit_code === "number" ? <span>exit={job.exit_code}</span> : null}
                          {active && streaming ? (
                            <Badge
                              variant="outline"
                              className="gap-1 border-emerald-500/30 bg-emerald-500/8 text-[10px] text-emerald-600"
                            >
                              <Activity className="size-3" />
                              直播中
                            </Badge>
                          ) : null}
                        </div>
                      </button>
                    )
                  })
                )}
              </div>
            </CardContent>
          </Card>

          <div className="grid gap-4">
            <Card className="border-border/50 bg-background/70 shadow-sm">
              <CardHeader className="pb-3">
                <CardTitle className="flex items-center gap-2 text-base">
                  <TerminalSquare className="size-4 text-primary" />
                  当前任务概览
                </CardTitle>
              </CardHeader>
              <CardContent className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                <MetaField label="任务 ID" value={selectedJobId || "未选择"} emphasis />
                <MetaField label="状态" value={String(displayStatus)} emphasis />
                <MetaField label="退出码" value={displayExitCode === null ? "-" : String(displayExitCode)} />
                <MetaField label="持续时间" value={formatDuration(selectedJob)} />
                <MetaField label="创建时间" value={formatDateTime(selectedJob?.created_at)} />
                <MetaField label="开始时间" value={formatDateTime(selectedJob?.started_at)} />
                <MetaField label="完成时间" value={formatDateTime(selectedJob?.finished_at)} />
                <MetaField label="流式状态" value={streaming ? "实时跟踪中" : "静态日志"} />
              </CardContent>
            </Card>

            <Card className="border-border/50 bg-background/70 shadow-sm">
              <CardHeader className="pb-3">
                <CardTitle className="flex items-center justify-between gap-3 text-base">
                  <span>日志输出</span>
                  <div className="flex items-center gap-2">
                    {selectedJob ? <Badge variant={statusVariant(displayStatus)}>{displayStatus}</Badge> : null}
                    {streaming ? (
                      <Badge variant="outline" className="gap-1 border-emerald-500/30 bg-emerald-500/8 text-emerald-600">
                        <Activity className="size-3" />
                        实时流
                      </Badge>
                    ) : null}
                  </div>
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
                  <span className="inline-flex items-center gap-1.5">
                    <Clock3 className="size-3.5" />
                    执行、切换任务和查看日志现在都在同一页完成，不再需要来回跳转。
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      if (selectedJobId) void loadJobLog(selectedJobId)
                    }}
                    disabled={!selectedJobId}
                  >
                    <RefreshCcw className="size-4" />
                    重新加载日志
                  </Button>
                </div>
                <LogStreamViewer
                  value={logText}
                  emptyText="请选择任务并查看日志"
                  className="h-[72vh] min-h-[420px] rounded-xl border bg-muted/20"
                />
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </PageShell>
  )
}
