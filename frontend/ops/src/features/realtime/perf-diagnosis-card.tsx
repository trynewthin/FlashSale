import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Gauge,
  Info,
  Layers3,
  Sparkles,
  Target,
} from "lucide-react"

import type { JobDetail } from "@/api/types"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  analyzePerfDiagnostics,
  type PerfDiagnosis,
  type PerfDiagnosisLevel,
} from "@/features/realtime/perf-diagnosis"
import type { RealtimeSample } from "@/features/realtime/types"
import { cn } from "@/lib/utils"

interface PerfDiagnosisCardProps {
  job: JobDetail | null
  samples: RealtimeSample[]
  className?: string
}

const LEVEL_ICONS = {
  critical: AlertTriangle,
  high: Gauge,
  medium: Activity,
  info: CheckCircle2,
} satisfies Record<PerfDiagnosisLevel, typeof AlertTriangle>

function levelBadgeVariant(level: PerfDiagnosisLevel) {
  switch (level) {
    case "critical":
      return "destructive" as const
    case "high":
      return "secondary" as const
    default:
      return "outline" as const
  }
}

function levelLabel(level: PerfDiagnosisLevel): string {
  switch (level) {
    case "critical":
      return "优先处理"
    case "high":
      return "高优先级"
    case "medium":
      return "关注"
    default:
      return "信息"
  }
}

function confidenceLabel(confidence: number): string {
  return `${Math.round(confidence * 100)}%`
}

function levelAccentClass(level: PerfDiagnosisLevel): string {
  switch (level) {
    case "critical":
      return "border-red-200/80 bg-red-50 text-red-900"
    case "high":
      return "border-amber-200/80 bg-amber-50 text-amber-900"
    case "medium":
      return "border-sky-200/80 bg-sky-50 text-sky-900"
    default:
      return "border-emerald-200/80 bg-emerald-50 text-emerald-900"
  }
}

function SecondaryDiagnosisBadge({ diagnosis }: { diagnosis: PerfDiagnosis }) {
  return (
    <div className="rounded-xl border bg-muted/20 px-3 py-3">
      <div className="flex items-center gap-2">
        <Badge variant={levelBadgeVariant(diagnosis.level)} className="text-[10px]">
          {levelLabel(diagnosis.level)}
        </Badge>
        <span className="truncate text-xs font-medium text-foreground">
          {diagnosis.title}
        </span>
      </div>
      <div className="mt-2 text-[12px] leading-5 text-muted-foreground">
        {diagnosis.summary}
      </div>
    </div>
  )
}

export function PerfDiagnosisCard({ job, samples, className }: PerfDiagnosisCardProps) {
  const report = analyzePerfDiagnostics(samples, job)
  const { primary, secondary, notes } = report
  const Icon = LEVEL_ICONS[primary.level]
  const condensedNotes = notes.slice(0, 2)

  return (
    <Card
      className={cn(
        "row-span-2 overflow-hidden border shadow-sm xl:col-span-2 2xl:col-span-2",
        className
      )}
    >
      <CardHeader className="border-b pb-4">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0 space-y-1">
            <div className="flex items-center gap-3">
              <div
                className={cn(
                  "flex size-10 shrink-0 items-center justify-center rounded-xl border",
                  levelAccentClass(primary.level)
                )}
              >
                <Icon className="size-4.5" />
              </div>
              <div>
                <CardTitle className="text-base">自动诊断</CardTitle>
                <CardDescription className="text-xs">
                  基于当前压测样本给出瓶颈判断和处理建议
                </CardDescription>
              </div>
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Badge variant={levelBadgeVariant(primary.level)} className="px-2.5 text-[10px]">
              {levelLabel(primary.level)}
            </Badge>
            <Badge variant="outline" className="px-2.5 text-[10px]">
              置信度 {confidenceLabel(primary.confidence)}
            </Badge>
          </div>
        </div>
      </CardHeader>

      <CardContent className="grid min-h-0 flex-1 grid-cols-1 gap-4 py-4 lg:grid-cols-[1.15fr_0.85fr]">
        <div className="space-y-4">
          <div className={cn("rounded-2xl border px-4 py-4", levelAccentClass(primary.level))}>
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={levelBadgeVariant(primary.level)} className="px-2.5 text-[10px]">
                主要结论
              </Badge>
              <span className="text-xs opacity-80">当前压测批次的自动判因结果</span>
            </div>
            <div className="mt-3 text-xl font-semibold leading-tight text-foreground">
              {primary.title}
            </div>
            <div className="mt-2 text-sm leading-6 text-foreground/80">
              {primary.summary}
            </div>
          </div>

          <div className="grid gap-3 sm:grid-cols-3">
            <div className="rounded-xl border bg-background px-3 py-3">
              <div className="flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                <Layers3 className="size-3.5" />
                关键模块
              </div>
              <div className="mt-2 text-base font-semibold text-foreground">
                {primary.module}
              </div>
            </div>

            <div className="rounded-xl border bg-background px-3 py-3">
              <div className="flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                <Sparkles className="size-3.5" />
                置信度
              </div>
              <div className="mt-2 text-2xl font-semibold tabular-nums text-foreground">
                {confidenceLabel(primary.confidence)}
              </div>
            </div>

            <div className="rounded-xl border bg-background px-3 py-3">
              <div className="flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                <Target className="size-3.5" />
                处理级别
              </div>
              <div className="mt-2 text-base font-semibold text-foreground">
                {levelLabel(primary.level)}
              </div>
            </div>
          </div>

          <div>
            <div className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
              建议动作
            </div>
            <div className="mt-2 space-y-2">
              {primary.suggestions.map((suggestion, index) => (
                <div
                  key={suggestion}
                  className="flex gap-3 rounded-xl border bg-background px-3 py-3 text-sm leading-relaxed text-foreground"
                >
                  <div className="flex size-6 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold text-foreground">
                    {index + 1}
                  </div>
                  <div>{suggestion}</div>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="space-y-3">
          <div className="rounded-xl border bg-background px-3 py-3">
            <div className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
              证据指标
            </div>
            <div className="mt-2 flex flex-wrap gap-2">
              {primary.evidence.map((item) => (
                <Badge key={item} variant="outline" className="rounded-md px-2 py-1 text-[11px]">
                  {item}
                </Badge>
              ))}
            </div>
          </div>

          {secondary.length > 0 ? (
            <div className="rounded-xl border bg-background px-3 py-3">
              <div className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                次要信号
              </div>
              <div className="mt-2 space-y-2">
                {secondary.map((diagnosis) => (
                  <SecondaryDiagnosisBadge key={diagnosis.kind} diagnosis={diagnosis} />
                ))}
              </div>
            </div>
          ) : null}

          <div className="rounded-xl border border-dashed bg-muted/10 px-3 py-3">
            <div className="flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
              <Info className="size-3.5" />
              判定备注
            </div>
            <div className="mt-2 space-y-2">
              {condensedNotes.map((note) => (
                <div key={note} className="text-[11px] leading-5 text-muted-foreground">
                  {note}
                </div>
              ))}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
