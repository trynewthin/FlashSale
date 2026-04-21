import {
  Activity,
  Pause,
  Play,
  RefreshCcw,
  TerminalSquare,
} from "lucide-react"
import { useCallback, useEffect, useMemo, useRef, useState } from "react"

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
import { useEventSource } from "@/hooks/use-event-source"
import { useOpsApiError } from "@/hooks/use-ops-api-error"
import { useOpsUIStore } from "@/store/ops-ui-store"

import { JobListCard } from "./job-list-card"
import { TaskExecutorCard } from "./task-executor-card"
import { formatDuration, parseArgs, statusVariant } from "./tasks-constants"

export function TasksPageFeature() {
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
  const doneReceivedRef = useRef(false)

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
      doneReceivedRef.current = false
      return
    }

    doneReceivedRef.current = false
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
            doneReceivedRef.current = true
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
        setStreaming(false)
        if (doneReceivedRef.current || !selectedJobId) {
          return
        }
        void loadJobLog(selectedJobId)
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
        <TaskExecutorCard
          tasks={tasks}
          selectedTask={selectedTask}
          selectedTaskDef={selectedTaskDef}
          argsText={argsText}
          runningTask={runningTask}
          loadingTasks={loadingTasks}
          onSelectTask={setSelectedTask}
          onArgsChange={setArgsText}
          onRunTask={() => void runTask()}
        />

        <div className="grid items-stretch gap-4 xl:grid-cols-[360px_minmax(0,1fr)]">
          <JobListCard
            jobs={jobs}
            selectedJobId={selectedJobId}
            streaming={streaming}
            onSelectJob={(jobId) => {
              setSelectedJobId(jobId)
              void loadJobLog(jobId)
            }}
          />

          <div className="flex flex-col gap-0 overflow-hidden rounded-xl border border-border/50 bg-background/70 shadow-sm">
            {/* 元数据头部 */}
            <div className="flex flex-wrap items-center gap-x-5 gap-y-1.5 border-b border-border/40 px-4 py-3">
              <div className="flex items-center gap-2">
                <TerminalSquare className="size-3.5 text-primary" />
                <span className="text-xs font-medium text-muted-foreground">
                  {selectedJobId || "未选择任务"}
                </span>
              </div>
              <div className="flex items-center gap-2 ml-auto">
                {selectedJob ? (
                  <Badge variant={statusVariant(displayStatus)} className="text-xs">{displayStatus}</Badge>
                ) : null}
                {displayExitCode !== null ? (
                  <span className="text-[11px] text-muted-foreground">exit {displayExitCode}</span>
                ) : null}
                {selectedJob?.started_at ? (
                  <span className="text-[11px] text-muted-foreground">{formatDuration(selectedJob)}</span>
                ) : null}
                {streaming ? (
                  <Badge variant="outline" className="gap-1 border-emerald-500/30 bg-emerald-500/8 text-emerald-600 text-[10px]">
                    <Activity className="size-3" />
                    实时流
                  </Badge>
                ) : null}
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 px-2 text-xs text-muted-foreground"
                  onClick={() => { if (selectedJobId) void loadJobLog(selectedJobId) }}
                  disabled={!selectedJobId}
                >
                  <RefreshCcw className="size-3" />
                  刷新
                </Button>
              </div>
            </div>
            {/* 日志区 */}
            <LogStreamViewer
              value={logText}
              emptyText="请选择任务并查看日志"
              className="h-[72vh] min-h-[420px] rounded-none border-0 bg-muted/20"
            />
          </div>
        </div>
      </div>
    </PageShell>
  )
}
