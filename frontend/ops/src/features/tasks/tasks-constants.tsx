import type { JobSummary } from "@/api/types"
import { cn } from "@/lib/utils"

export function parseArgs(raw: string): string[] {
  const trimmed = (raw || "").trim()
  if (!trimmed) {
    return []
  }
  return trimmed.split(/\s+/)
}

export function statusVariant(status: string): "secondary" | "destructive" | "outline" {
  if (status === "success") {
    return "secondary"
  }
  if (status === "failed") {
    return "destructive"
  }
  return "outline"
}

export function formatDateTime(value?: string): string {
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

export function formatDuration(job?: Pick<JobSummary, "started_at" | "finished_at"> | null): string {
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

export function MetaField({
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
