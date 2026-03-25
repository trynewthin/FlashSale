import { Activity } from "lucide-react"

import type { JobSummary } from "@/api/types"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

import { formatDateTime, formatDuration, statusVariant } from "./tasks-constants"

export function JobListCard({
  jobs,
  selectedJobId,
  streaming,
  onSelectJob,
}: {
  jobs: JobSummary[]
  selectedJobId: string
  streaming: boolean
  onSelectJob: (jobId: string) => void
}) {
  return (
    <Card className="border-border/50 bg-background/70 shadow-sm">
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center justify-between gap-3 text-base">
          <span>最近任务</span>
          <Badge variant="outline">{jobs.length}</Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="max-h-[72vh] min-h-[420px] space-y-2 overflow-y-auto pr-1">
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
                  onClick={() => onSelectJob(job.id)}
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
  )
}
