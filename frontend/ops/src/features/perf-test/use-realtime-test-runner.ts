import { useCallback, useEffect, useMemo, useRef, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import type {
  JobDetail,
  JobStreamDonePayload,
  JobStreamEnvelope,
  JobStreamErrorPayload,
  JobStreamLogLinePayload,
  JobStreamSnapshotPayload,
  JobStreamStatePayload,
  JobSummary,
  PerfProgress,
  PerfReport,
  TaskDef,
} from "@/api/types"
import {
  getRealtimeTestPreset,
  type RealtimeTestFieldDef,
  type RealtimeTestPreset,
  sortRealtimeTestTasks,
} from "@/features/perf-test/realtime-test-presets"
import {
  formatTaskArgs,
  getFlagArg,
  removeFlagArg,
} from "@/features/perf-test/realtime-test-utils"
import { useEventSource } from "@/hooks/use-event-source"
import { useOpsApiError } from "@/hooks/use-ops-api-error"
import { useOpsUIStore } from "@/store/ops-ui-store"

interface UseRealtimeTestRunnerResult {
  launcherOpen: boolean
  setLauncherOpen: (open: boolean) => void
  tasks: TaskDef[]
  selectedTaskID: string
  setSelectedTaskID: (value: string) => void
  selectedPreset: RealtimeTestPreset | null
  formValues: Record<string, string>
  setFormValue: (flag: string, value: string) => void
  extraArgsText: string
  setExtraArgsText: (value: string) => void
  creating: boolean
  activeJob: JobDetail | null
  activeLog: string
  streamEnabled: boolean
  setStreamEnabled: (value: boolean) => void
  perfSamples: PerfProgress[]
  perfReport: PerfReport | null
  recentTestJobs: JobSummary[]
  startTest: () => Promise<boolean>
  refreshTasks: () => Promise<void>
  refreshTestJobs: () => Promise<void>
  refreshActiveJob: () => Promise<void>
  switchActiveJob: (jobID: string) => Promise<void>
  clearActiveLog: () => void
}

function isTestTask(task: TaskDef): boolean {
  return task.id.startsWith("perf.")
}

function validateFieldValue(field: RealtimeTestFieldDef, rawValue: string): string | null {
  const value = rawValue.trim()
  if ((field.required ?? true) && !value) {
    return `${field.label} 不能为空`
  }
  if (!value) {
    return null
  }
  if (field.type === "number") {
    const parsed = Number(value)
    if (!Number.isFinite(parsed)) {
      return `${field.label} 必须是数字`
    }
    if (typeof field.min === "number" && parsed < field.min) {
      return `${field.label} 不能小于 ${field.min}`
    }
    if (typeof field.max === "number" && parsed > field.max) {
      return `${field.label} 不能大于 ${field.max}`
    }
  }
  return null
}

// useRealtimeTestRunner 管理实时监测页中的测试任务启动与日志流。
export function useRealtimeTestRunner(): UseRealtimeTestRunnerResult {
  const showApiError = useOpsApiError()
  const showNotice = useOpsUIStore((state) => state.showNotice)

  const [launcherOpen, setLauncherOpen] = useState(false)
  const [tasks, setTasks] = useState<TaskDef[]>([])
  const [selectedTaskID, setSelectedTaskID] = useState("")
  const [formValues, setFormValues] = useState<Record<string, string>>({})
  const [extraArgsText, setExtraArgsText] = useState("")
  const [creating, setCreating] = useState(false)
  const [activeJob, setActiveJob] = useState<JobDetail | null>(null)
  const [activeLog, setActiveLog] = useState("")
  const [streamEnabled, setStreamEnabled] = useState(false)
  const [recentTestJobs, setRecentTestJobs] = useState<JobSummary[]>([])
  const [perfSamples, setPerfSamples] = useState<PerfProgress[]>([])
  const [perfReport, setPerfReport] = useState<PerfReport | null>(null)

  const selectedTask = useMemo(
    () => tasks.find((item) => item.id === selectedTaskID) ?? null,
    [selectedTaskID, tasks]
  )
  const selectedPreset = useMemo(
    () => getRealtimeTestPreset(selectedTaskID),
    [selectedTaskID]
  )

  const refreshTasks = useCallback(async () => {
    try {
      const allTasks = await opsApi.listTasks()
      const testTasks = sortRealtimeTestTasks(allTasks.filter(isTestTask))
      setTasks(testTasks)
      if (testTasks.length === 0) {
        setSelectedTaskID("")
        setFormValues({})
        setExtraArgsText("")
        return
      }
      setSelectedTaskID((prev) => (prev && testTasks.some((item) => item.id === prev) ? prev : testTasks[0].id))
    } catch (error) {
      showApiError(error, "加载测试任务失败")
    }
  }, [showApiError])

  const refreshTestJobs = useCallback(async () => {
    try {
      const jobs = await opsApi.listJobs(30)
      const filtered = jobs.filter((item) => item.task_id.startsWith("perf."))
      setRecentTestJobs(filtered)
    } catch (error) {
      showApiError(error, "加载测试任务历史失败")
    }
  }, [showApiError])

  const refreshActiveJob = useCallback(async () => {
    if (!activeJob?.id) {
      return
    }
    try {
      const latest = await opsApi.getJob(activeJob.id)
      setActiveJob(latest)
      try {
        const logSnapshot = await opsApi.getJobLog(activeJob.id)
        setActiveLog(logSnapshot)
      } catch {
        // 日志快照失败不阻断状态刷新。
      }
      if (latest.status === "success" || latest.status === "failed") {
        setStreamEnabled(false)
      }
    } catch (error) {
      showApiError(error, "刷新测试任务状态失败")
    }
  }, [activeJob?.id, showApiError])

  const switchActiveJob = useCallback(
    async (jobID: string) => {
      if (!jobID) {
        return
      }
      try {
        const [detail, logSnapshot] = await Promise.all([opsApi.getJob(jobID), opsApi.getJobLog(jobID)])
        setActiveJob(detail)
        setActiveLog(logSnapshot)
        // 历史任务：直接从 job.perf_report 加载压测数据
        setPerfSamples(detail.perf_report?.samples ?? [])
        setPerfReport(detail.perf_report ?? null)
        setStreamEnabled(true)
      } catch (error) {
        showApiError(error, "切换测试任务失败")
      }
    },
    [showApiError]
  )

  const setFormValue = useCallback((flag: string, value: string) => {
    setFormValues((prev) => ({ ...prev, [flag]: value }))
  }, [])

  useEffect(() => {
    void refreshTasks()
    void refreshTestJobs()
  }, [refreshTasks, refreshTestJobs])

  useEffect(() => {
    const timer = window.setInterval(() => {
      void refreshTestJobs()
      void refreshActiveJob()
    }, 3000)
    return () => window.clearInterval(timer)
  }, [refreshActiveJob, refreshTestJobs])

  useEffect(() => {
    if (!selectedTask) {
      setFormValues({})
      setExtraArgsText("")
      return
    }
    const defaults = selectedTask.default_args ?? []
    const preset = getRealtimeTestPreset(selectedTask.id)
    if (!preset) {
      setFormValues({})
      setExtraArgsText(formatTaskArgs(defaults))
      return
    }

    const nextValues: Record<string, string> = {}
    for (const field of preset.fields) {
      let value = getFlagArg(defaults, field.flag)
      if (!value && field.type === "select" && field.options && field.options.length > 0) {
        value = field.options[0].value
      }
      nextValues[field.flag] = value
    }
    setFormValues(nextValues)

    let baseExtraArgs = [...defaults]
    for (const field of preset.fields) {
      baseExtraArgs = removeFlagArg(baseExtraArgs, field.flag)
    }
    setExtraArgsText(formatTaskArgs(baseExtraArgs))
  }, [selectedTask])

  const startTest = useCallback(async (): Promise<boolean> => {
    if (!selectedTaskID) {
      showNotice("error", "请选择测试项目")
      return false
    }
    if (!selectedTask) {
      showNotice("error", "测试任务不存在，请刷新后重试")
      return false
    }
    try {
      setCreating(true)
      const fields: Record<string, string> = {}

      if (selectedPreset) {
        for (const field of selectedPreset.fields) {
          const value = (formValues[field.flag] ?? "").trim()
          const errorText = validateFieldValue(field, value)
          if (errorText) {
            showNotice("error", errorText)
            return false
          }
          if (value) {
            fields[field.flag] = value
          }
        }
      }

      const job = await opsApi.createPerfJob({
        task: selectedTaskID,
        fields,
        advanced_args_text: extraArgsText,
      })
      setActiveJob(job)
      setActiveLog("")
      setPerfSamples([])
      setPerfReport(null)
      setStreamEnabled(true)
      setLauncherOpen(false)
      await refreshTestJobs()
      showNotice("success", `测试任务已启动：${job.id}`)
      return true
    } catch (error) {
      showApiError(error, "启动测试任务失败")
      return false
    } finally {
      setCreating(false)
    }
  }, [
    extraArgsText,
    formValues,
    refreshTestJobs,
    selectedPreset,
    selectedTask,
    selectedTaskID,
    showApiError,
    showNotice,
  ])

  const activeJobStreamURL = useMemo(() => {
    if (!activeJob?.id) {
      return ""
    }
    return opsApi.buildJobStreamURL(activeJob.id)
  }, [activeJob?.id])

  // 追踪是否已收到 done 事件（任务正常结束），避免 onerror 误报
  const doneReceivedRef = useRef(false)

  // 当 activeJob 变化时重置 done 标记
  useEffect(() => {
    doneReceivedRef.current = false
  }, [activeJob?.id])

  useEventSource({
    enabled: streamEnabled && !!activeJob?.id,
    url: activeJobStreamURL,
    handlers: {
      onMessageData: (payload) => {
        const envelope = payload as JobStreamEnvelope
        if (!envelope || typeof envelope.type !== "string") {
          return
        }
        if (envelope.type === "snapshot") {
          const frame = envelope.payload as JobStreamSnapshotPayload
          if (typeof frame?.log === "string" && frame.log.length > 0) {
            setActiveLog(frame.log)
          }
          return
        }
        if (envelope.type === "log_line") {
          const frame = envelope.payload as JobStreamLogLinePayload
          if (typeof frame?.line === "string") {
            setActiveLog((prev) => `${prev}${prev.endsWith("\n") || prev.length === 0 ? "" : "\n"}${frame.line}\n`)
          }
          return
        }
        if (envelope.type === "job_state") {
          const data = envelope.payload as JobStreamStatePayload
          setActiveJob((prev) => (
            prev
              ? {
                  ...prev,
                  status: data?.status || prev.status,
                  exit_code: typeof data?.exit_code === "number" ? data.exit_code : prev.exit_code,
                  started_at: data?.started_at || prev.started_at,
                  finished_at: data?.finished_at || prev.finished_at,
                }
              : prev
          ))
          return
        }
        if (envelope.type === "perf_progress") {
          const sample = envelope.payload as PerfProgress
          if (sample && typeof sample.timestamp === "number") {
            setPerfSamples((prev) => [...prev, sample])
          }
          return
        }
        if (envelope.type === "done") {
          doneReceivedRef.current = true
          const donePayload = envelope.payload as JobStreamDonePayload
          if (donePayload?.perf_report) {
            setPerfReport(donePayload.perf_report)
            if (donePayload.perf_report.samples?.length) {
              setPerfSamples(donePayload.perf_report.samples)
            }
          }
          setStreamEnabled(false)
          void refreshActiveJob()
          return
        }
        if (envelope.type === "error") {
          const data = envelope.payload as JobStreamErrorPayload
          if (data?.message) {
            showNotice("error", data.message)
          }
        }
      },
      onError: () => {
        // 如果已经收到 done 事件，这是后端正常关闭连接，不需要提示
        if (doneReceivedRef.current) return
        setStreamEnabled(false)
        showNotice("error", "测试日志流已断开，请手动重开")
      },
    },
  })

  return {
    launcherOpen,
    setLauncherOpen,
    tasks,
    selectedTaskID,
    setSelectedTaskID,
    selectedPreset,
    formValues,
    setFormValue,
    extraArgsText,
    setExtraArgsText,
    creating,
    activeJob,
    activeLog,
    streamEnabled,
    setStreamEnabled,
    perfSamples,
    perfReport,
    recentTestJobs,
    startTest,
    refreshTasks,
    refreshTestJobs,
    refreshActiveJob,
    switchActiveJob,
    clearActiveLog: () => setActiveLog(""),
  }
}
