import { Activity, CheckCircle2, type LucideIcon, XCircle } from "lucide-react"

import type { ObservabilityLink } from "@/api/types"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

export function StatusIcon({ ok, className }: { ok: boolean; className?: string }) {
  return ok
    ? <CheckCircle2 className={cn("size-4 text-emerald-500", className)} />
    : <XCircle className={cn("size-4 text-destructive", className)} />
}

export function SectionCard({
  title,
  description,
  icon: Icon,
  action,
  children,
}: {
  title: string
  description?: string
  icon: LucideIcon
  iconClassName?: string
  action?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <Card className="border-border/40 bg-background/50 shadow-sm">
      <CardHeader>
        <CardTitle className="text-sm font-semibold text-muted-foreground">{title}</CardTitle>
        <CardAction className="flex items-center gap-2">
          {action}
          <Icon className="size-5 text-foreground/70" />
        </CardAction>
        {description && <CardDescription>{description}</CardDescription>}
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  )
}

export function StatCard({
  title,
  value,
  total,
  tone = "neutral",
  icon: Icon,
}: {
  title: string
  value: number
  total?: number
  tone?: "healthy" | "warn" | "danger" | "neutral" | "primary"
  icon: LucideIcon
}) {
  const isProgress = total !== undefined
  const validTotal = total || 1
  const percentage = isProgress ? (value / validTotal) * 100 : 0
  const displayPercent = isProgress ? Math.round(percentage * 10) / 10 : 0

  const toneColors = {
    healthy: "text-emerald-500",
    warn: "text-amber-500",
    danger: "text-destructive",
    neutral: "text-muted-foreground",
    primary: "text-primary",
  }

  const barColors = {
    healthy: "bg-emerald-500",
    warn: "bg-amber-500",
    danger: "bg-destructive",
    neutral: "bg-muted-foreground",
    primary: "bg-primary",
  }

  const textColor = toneColors[tone] || toneColors.neutral
  const barColor = barColors[tone] || barColors.neutral

  return (
    <Card className="border-border/40 bg-background/50 shadow-sm transition-colors hover:bg-background/80">
      <CardHeader>
        <CardTitle className="text-sm font-semibold text-muted-foreground">{title}</CardTitle>
        <CardAction>
          <Icon className={cn("size-5", textColor)} />
        </CardAction>
      </CardHeader>
      
      <CardContent className="space-y-4">
        <div className="flex items-baseline gap-2">
          <h2 className="text-[32px] font-bold tracking-tight text-foreground leading-none">
            {isProgress ? `${value}/${total}` : value}
          </h2>
          {isProgress && (
            <span className={cn("text-sm font-medium", textColor)}>
              {displayPercent}%
            </span>
          )}
        </div>

        {isProgress ? (
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-secondary/60">
            <div
              className={cn("h-full transition-all duration-500 ease-in-out", barColor)}
              style={{ width: `${Math.min(100, Math.max(0, percentage))}%` }}
            />
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}

export function ObservabilityLinkButton({ link }: { link: ObservabilityLink }) {
  if (!link.url) {
    return (
      <Button variant="outline" size="sm" disabled className="h-10 w-full justify-start gap-3 rounded-xl border-dashed opacity-50">
        <Activity className="size-4" />
        <span className="flex-1 text-left">{link.name}</span>
        <span className="text-[10px] text-muted-foreground">Unset</span>
      </Button>
    )
  }

  const isUp = link.available
  return (
    <a
      href={link.url}
      target="_blank"
      rel="noopener noreferrer"
      className={cn(
        "group flex h-10 w-full items-center justify-between rounded-xl border px-3 text-sm font-medium transition-colors",
        isUp ? "border-transparent bg-muted/40 hover:bg-muted/60" : "border-destructive/20 bg-destructive/5 hover:bg-destructive/10"
      )}
    >
      <div className="flex items-center gap-3">
        <Activity className={cn("size-4", isUp ? "text-primary opacity-70 group-hover:opacity-100" : "text-destructive")} />
        <span className={isUp ? "text-foreground" : "text-destructive"}>{link.name}</span>
      </div>
      <div className="flex items-center gap-2">
        <span className={cn("text-[10px] uppercase tracking-wider", isUp ? "text-muted-foreground" : "text-destructive")}>
          {isUp ? "Online" : "Offline"}
        </span>
        <div className={cn("size-2 rounded-full", isUp ? "bg-emerald-500" : "bg-destructive")} />
      </div>
    </a>
  )
}
