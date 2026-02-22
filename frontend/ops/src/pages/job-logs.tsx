import { Pause, Play, RefreshCcw } from "lucide-react"
import { useCallback, useEffect, useMemo, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import type { JobSummary, SSELogLineFrame } from "@/api/types"
import { LogStreamViewer } from "@/components/common/log-stream-viewer"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { PageShell } from "@/components/layout/page-shell"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useEventSource } from "@/hooks/use-event-source"
import { useOpsApiError } from "@/hooks/use-ops-api-error"
import { useOpsUIStore } from "@/store/ops-ui-store"

function statusVariant(status: string): "secondary" | "destructive" | "outline" {
  if (status === "success") {
    return "secondary"
  }
  if (status === "failed") {
    return "destructive"
  }
  return "outline"
}

// JobLogsPage 展示任务历史与日志实时流。
export function JobLogsPage() {
  const showApiError = useOpsApiError()
  const showNotice = useOpsUIStore((state) => state.showNotice)
  const selectedJobId = useOpsUIStore((state) => state.selectedJobId)
  const setSelectedJobId = useOpsUIStore((state) => state.setSelectedJobId)

  const [jobs, setJobs] = useState<JobSummary[]>([])
  const [loading, setLoading] = useState(false)
  const [streaming, setStreaming] = useState(false)
  const [logText, setLogText] = useState("")

  const loadJobs = useCallback(async () => {
    try {
      setLoading(true)
      const data = await opsApi.listJobs(30)
      setJobs(data)
      if (!selectedJobId && data.length > 0) {
        setSelectedJobId(data[0].id)
      }
    } catch (error) {
      showApiError(error, "任务列表加载失败")
    } finally {
      setLoading(false)
    }
  }, [selectedJobId, setSelectedJobId, showApiError])

  const loadJobLog = useCallback(async (jobId: string) => {
    if (!jobId) {
      return
    }
    try {
      const text = await opsApi.getJobLog(jobId)
      setLogText(text)
      setStreaming(true)
    } catch (error) {
      showApiError(error, "任务日志加载失败")
    }
  }, [showApiError])

  useEffect(() => {
    void loadJobs()
    const timer = window.setInterval(() => {
      void loadJobs()
    }, 3000)
    return () => window.clearInterval(timer)
  }, [loadJobs])

  useEffect(() => {
    if (!selectedJobId) {
      return
    }
    void loadJobLog(selectedJobId)
  }, [loadJobLog, selectedJobId])

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
      onEvent: (eventType, payload) => {
        if (eventType !== "log") {
          return
        }
        const frame = payload as SSELogLineFrame
        if (typeof frame.chunk === "string") {
          setLogText((prev) => prev + frame.chunk)
          return
        }
        if (typeof frame.line === "string") {
          setLogText((prev) => `${prev}${prev.endsWith("\n") || prev.length === 0 ? "" : "\n"}${frame.line}\n`)
        }
      },
      onError: () => {
        showNotice("error", "任务日志流已断开，请重新开始")
        setStreaming(false)
      },
    },
  })

  return (
    <PageShell
      title="任务日志"
      actions={
        <>
          <Button variant="secondary" size="sm" onClick={() => void loadJobs()} disabled={loading}>
            <RefreshCcw className="size-4" />
            刷新列表
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setStreaming((v) => !v)}
            disabled={!selectedJobId}
          >
            {streaming ? <Pause className="size-4" /> : <Play className="size-4" />}
            {streaming ? "停止流" : "开始流"}
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => { if (selectedJobId) void loadJobLog(selectedJobId) }}
            disabled={!selectedJobId}
          >
            <RefreshCcw className="size-4" />
            重载快照
          </Button>
        </>
      }
    >
      <div className="grid gap-4 xl:grid-cols-[1.2fr_1.8fr]">
        <Card>
          <CardContent className="pt-5">
            <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">最近任务</p>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Job ID</TableHead>
                  <TableHead>Task</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {jobs.map((job) => (
                  <TableRow key={job.id} data-state={selectedJobId === job.id ? "selected" : undefined}>
                    <TableCell className="max-w-[220px] truncate">{job.id}</TableCell>
                    <TableCell>{job.task_id}</TableCell>
                    <TableCell>
                      <Badge variant={statusVariant(job.status)}>{job.status}</Badge>
                    </TableCell>
                    <TableCell>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setSelectedJobId(job.id)
                          void loadJobLog(job.id)
                        }}
                      >
                        查看日志
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-5 space-y-3">
            <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {selectedJobId ? `Job: ${selectedJobId}` : "请选择任务"}
            </p>
            <LogStreamViewer value={logText} emptyText="请选择任务并查看日志" />
          </CardContent>
        </Card>
      </div>
    </PageShell>
  )
}
