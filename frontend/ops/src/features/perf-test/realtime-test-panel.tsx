import { Pause, Play, RefreshCcw } from "lucide-react"

import type { JobDetail, JobSummary } from "@/api/types"
import { LogStreamViewer } from "@/components/common/log-stream-viewer"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { getRealtimeTestPreset } from "@/features/perf-test/realtime-test-presets"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { jobDurationSeconds } from "@/features/perf-test/realtime-test-utils"

interface RealtimeTestPanelProps {
  activeJob: JobDetail | null
  activeLog: string
  streamEnabled: boolean
  recentTestJobs: JobSummary[]
  onToggleStream: () => void
  onRefreshActiveJob: () => void
  onRefreshRecentJobs: () => void
  onClearActiveLog: () => void
  onSwitchActiveJob: (jobID: string) => void
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

// RealtimeTestPanel 展示测试执行状态、最近任务与实时日志。
export function RealtimeTestPanel({
  activeJob,
  activeLog,
  streamEnabled,
  recentTestJobs,
  onToggleStream,
  onRefreshActiveJob,
  onRefreshRecentJobs,
  onClearActiveLog,
  onSwitchActiveJob,
}: RealtimeTestPanelProps) {
  const durationSeconds = jobDurationSeconds(activeJob)
  const logLines = activeLog ? activeLog.split("\n").filter((line) => line.trim() !== "").length : 0

  return (
    <div className="grid gap-4 xl:grid-cols-[1.1fr_1.9fr]">
      <Card>
        <CardHeader>
          <CardTitle>测试任务</CardTitle>
          <CardDescription>选择最近任务，切换当前实时日志追踪对象。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" size="sm" onClick={onRefreshRecentJobs}>
              <RefreshCcw className="size-4" />
              刷新任务
            </Button>
          </div>
          {activeJob ? (
            <div className="grid gap-2 rounded-md border bg-muted/20 p-3 text-xs">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-muted-foreground">当前任务</span>
                <Badge variant={statusVariant(activeJob.status)}>{activeJob.status}</Badge>
              </div>
              <div className="break-all font-mono">{activeJob.id}</div>
              <div className="text-muted-foreground">
                运行时长 {durationSeconds}s · 退出码 {activeJob.exit_code} · 日志行 {logLines}
              </div>
            </div>
          ) : (
            <div className="rounded-md border bg-muted/20 p-3 text-xs text-muted-foreground">
              暂无运行中的测试任务
            </div>
          )}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Task</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {recentTestJobs.map((job) => (
                <TableRow key={job.id}>
                  <TableCell className="max-w-36">
                    <div className="truncate text-xs font-medium">
                      {getRealtimeTestPreset(job.task_id)?.title || job.task_id}
                    </div>
                    <div className="truncate text-[11px] text-muted-foreground">{job.task_id}</div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant(job.status)}>{job.status}</Badge>
                  </TableCell>
                  <TableCell>
                    <Button size="sm" variant="outline" onClick={() => onSwitchActiveJob(job.id)}>
                      追踪
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>测试实时日志</CardTitle>
          <CardDescription>{activeJob ? `Job: ${activeJob.id}` : "启动测试后自动显示"}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" onClick={onToggleStream} disabled={!activeJob}>
              {streamEnabled ? <Pause className="size-4" /> : <Play className="size-4" />}
              {streamEnabled ? "停止流" : "开始流"}
            </Button>
            <Button variant="outline" onClick={onRefreshActiveJob} disabled={!activeJob}>
              <RefreshCcw className="size-4" />
              刷新状态
            </Button>
            <Button variant="outline" onClick={onClearActiveLog}>
              清空日志
            </Button>
          </div>
          <LogStreamViewer value={activeLog} emptyText="暂无测试日志" className="h-[320px] rounded-lg border bg-muted/20" />
        </CardContent>
      </Card>
    </div>
  )
}
