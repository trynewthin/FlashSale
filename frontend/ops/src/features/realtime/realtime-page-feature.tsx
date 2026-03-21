import {
  AlertTriangle,
  Box,
  CalendarRange,
  Filter,
  Pause,
  Play,
  RefreshCcw,
  SearchCheck,
  Server,
  SlidersHorizontal,
  Trash2,
  TrendingUp,
  Zap,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  MetricFilterPanel,
  RealtimeUnifiedChart,
  useChartMetrics,
} from "@/features/realtime/realtime-unified-chart"
import {
  useRealtimeMonitor,
  TIME_WINDOWS,
  type ReplayPreset,
  type TimeWindow,
} from "@/features/realtime/use-realtime-monitor"

const REPLAY_PRESETS: Array<{ value: ReplayPreset; label: string }> = [
  { value: "5m", label: "最近 5 分钟" },
  { value: "15m", label: "最近 15 分钟" },
  { value: "1h", label: "最近 1 小时" },
]

function precisionReasonLabel(reason: "traffic" | "latency" | "errors" | null): string {
  switch (reason) {
    case "traffic":
      return "检测到流量峰值"
    case "latency":
      return "检测到延迟抬升"
    case "errors":
      return "检测到错误率异常"
    default:
      return "已检测到峰值/异常，服务端指标切换为短窗视图。"
  }
}

export function RealtimePageFeature() {
  const {
    samples,
    latest,
    loading,
    running,
    mode,
    timeWindow,
    precisionActive,
    precisionReason,
    sampleInsufficient,
    setTimeWindow,
    setRunning,
    clearSamples,
    refreshNow,
    replayState,
    openReplay,
    closeReplay,
    setReplayStartInput,
    setReplayEndInput,
    applyReplayPreset,
    runReplay,
  } = useRealtimeMonitor()

  const {
    activeGroup,
    groupMetrics,
    visibleKeys,
    handleGroupChange,
    toggleMetric,
  } = useChartMetrics()

  const runningCont = latest?.runningContainers ?? 0
  const totalCont = latest?.totalContainers ?? 0
  const runningRep = latest?.runningReplicas ?? 0
  const totalRep = latest?.totalReplicas ?? 0

  return (
    <>
      <div className="flex h-full flex-col">
        <div className="shrink-0 border-b border-border/30 bg-background/60 px-6 py-3">
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-2">
              <Popover>
                <PopoverTrigger className="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-border/50 bg-background px-3 py-2 text-sm font-medium shadow-sm transition-colors hover:bg-muted/50">
                  <SlidersHorizontal className="size-4" />
                  采样控制
                </PopoverTrigger>
                <PopoverContent side="bottom" align="start" sideOffset={8} className="w-[340px] p-4">
                  <div className="space-y-3">
                    <div className="flex flex-wrap items-center gap-2">
                      <Button
                        size="sm"
                        variant={running ? "outline" : "secondary"}
                        onClick={() => setRunning(!running)}
                      >
                        {running ? <Pause className="size-3.5" /> : <Play className="size-3.5" />}
                        {running ? "暂停" : "继续"}
                      </Button>
                      <Button size="sm" variant="outline" disabled={loading} onClick={() => void refreshNow()}>
                        <RefreshCcw className="size-3.5" />
                        刷新
                      </Button>
                      <Button size="sm" variant="outline" onClick={clearSamples}>
                        <Trash2 className="size-3.5" />
                        清空
                      </Button>
                    </div>
                    <Select value={timeWindow} onValueChange={(value) => setTimeWindow(value as TimeWindow)}>
                      <SelectTrigger className="h-8 w-full text-xs">
                        <SelectValue placeholder="时间窗口" />
                      </SelectTrigger>
                      <SelectContent className="rounded-xl">
                        {TIME_WINDOWS.map((option) => (
                          <SelectItem key={option.value} value={option.value} className="rounded-lg text-xs">
                            {option.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </PopoverContent>
              </Popover>

              <Popover>
                <PopoverTrigger className="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-border/50 bg-background px-3 py-2 text-sm font-medium shadow-sm transition-colors hover:bg-muted/50">
                  <Filter className="size-4" />
                  指标筛选
                </PopoverTrigger>
                <PopoverContent side="bottom" align="start" sideOffset={8} className="w-[420px] p-4">
                  <MetricFilterPanel
                    activeGroup={activeGroup}
                    groupMetrics={groupMetrics}
                    visibleKeys={visibleKeys}
                    onGroupChange={handleGroupChange}
                    onToggleMetric={toggleMetric}
                  />
                </PopoverContent>
              </Popover>

              <Button
                size="sm"
                variant={mode === "replay" ? "secondary" : "outline"}
                className="gap-1.5"
                onClick={() => openReplay()}
              >
                <CalendarRange className="size-4" />
                异常回放
              </Button>
            </div>

            <div className="flex flex-wrap items-center justify-end gap-2">
              <ModeBadges
                mode={mode}
                precisionActive={precisionActive}
                precisionReason={precisionReason}
                sampleInsufficient={sampleInsufficient}
                replayActive={replayState.active}
              />
              <Badge variant="outline" className="gap-1.5 px-2.5 py-1 text-xs">
                <Box className="size-3.5 text-blue-500" />
                容器 {runningCont}/{totalCont}
              </Badge>
              <Badge variant="outline" className="gap-1.5 px-2.5 py-1 text-xs">
                <Server className="size-3.5 text-emerald-500" />
                副本 {runningRep}/{totalRep}
              </Badge>
            </div>
          </div>

          {mode === "live" && precisionActive ? (
            <div className="mt-3 flex items-center gap-2 rounded-lg border border-amber-200/70 bg-amber-50 px-3 py-2 text-xs text-amber-900">
              <Zap className="size-3.5 shrink-0" />
              <span>{precisionReasonLabel(precisionReason)}</span>
            </div>
          ) : null}
        </div>

        <div className="min-h-0 flex-1 bg-card p-4">
          <RealtimeUnifiedChart
            samples={samples}
            visibleKeys={visibleKeys}
            mode={mode}
            className="h-full"
          />
        </div>
      </div>

      <Sheet open={replayState.open} onOpenChange={(open) => {
        if (open) {
          openReplay()
          return
        }
        closeReplay()
      }}>
        <SheetContent side="right" className="w-full sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>异常片段回放</SheetTitle>
            <SheetDescription>
              为任意自然时间段拉取高精度 RPC 指标，并与基础设施状态同屏对齐。
            </SheetDescription>
          </SheetHeader>

          <div className="flex min-h-0 flex-1 flex-col gap-4 px-4 pb-4">
            <div className="space-y-3 rounded-xl border bg-muted/20 p-3">
              <div className="flex flex-wrap items-center gap-2">
                {REPLAY_PRESETS.map((preset) => (
                  <Button
                    key={preset.value}
                    size="sm"
                    variant="outline"
                    className="text-xs"
                    onClick={() => applyReplayPreset(preset.value)}
                  >
                    {preset.label}
                  </Button>
                ))}
              </div>

              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-1.5">
                  <Label htmlFor="replay-start">开始时间</Label>
                  <Input
                    id="replay-start"
                    type="datetime-local"
                    value={replayState.startInput}
                    onChange={(event) => setReplayStartInput(event.target.value)}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="replay-end">结束时间</Label>
                  <Input
                    id="replay-end"
                    type="datetime-local"
                    value={replayState.endInput}
                    onChange={(event) => setReplayEndInput(event.target.value)}
                  />
                </div>
              </div>

              <div className="flex items-center gap-2">
                <Button size="sm" className="gap-1.5" onClick={() => void runReplay()} disabled={replayState.loading}>
                  <SearchCheck className="size-3.5" />
                  开始回放
                </Button>
                <Button size="sm" variant="outline" onClick={closeReplay}>
                  返回实时
                </Button>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={replayState.active ? "secondary" : "outline"} className="gap-1.5 text-xs">
                {replayState.active ? <CalendarRange className="size-3.5" /> : <TrendingUp className="size-3.5" />}
                {replayState.active ? "回放模式" : "未加载回放"}
              </Badge>
              {replayState.sampleInsufficient ? (
                <Badge variant="outline" className="gap-1.5 text-xs">
                  <AlertTriangle className="size-3.5 text-amber-600" />
                  样本不足
                </Badge>
              ) : null}
            </div>

            {replayState.error ? (
              <div className="rounded-xl border border-destructive/40 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {replayState.error}
              </div>
            ) : null}

            <div className="min-h-0 flex-1 rounded-xl border bg-background p-3">
              <RealtimeUnifiedChart
                samples={replayState.active ? replayState.samples : []}
                visibleKeys={visibleKeys}
                mode="replay"
                className="h-full"
              />
            </div>
          </div>
        </SheetContent>
      </Sheet>
    </>
  )
}

interface ModeBadgesProps {
  mode: "live" | "replay"
  precisionActive: boolean
  precisionReason: "traffic" | "latency" | "errors" | null
  sampleInsufficient: boolean
  replayActive: boolean
}

function ModeBadges({
  mode,
  precisionActive,
  precisionReason,
  sampleInsufficient,
  replayActive,
}: ModeBadgesProps) {
  if (replayActive || mode === "replay") {
    return (
      <Badge variant="secondary" className="gap-1.5 px-2.5 py-1 text-xs">
        <CalendarRange className="size-3.5 text-blue-600" />
        回放模式
      </Badge>
    )
  }

  if (sampleInsufficient) {
    return (
      <Badge variant="outline" className="gap-1.5 px-2.5 py-1 text-xs">
        <AlertTriangle className="size-3.5 text-amber-600" />
        样本不足
      </Badge>
    )
  }

  if (precisionActive) {
    const icon = precisionReason === "errors"
      ? <AlertTriangle className="size-3.5 text-red-600" />
      : <Zap className="size-3.5 text-amber-600" />
    return (
      <Badge variant="secondary" className="gap-1.5 px-2.5 py-1 text-xs">
        {icon}
        高精度模式
      </Badge>
    )
  }

  return (
    <Badge variant="outline" className="gap-1.5 px-2.5 py-1 text-xs">
      <TrendingUp className="size-3.5 text-emerald-600" />
      趋势模式
    </Badge>
  )
}
