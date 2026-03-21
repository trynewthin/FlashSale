import { Box, Filter, Pause, Play, RefreshCcw, Server, SlidersHorizontal, Trash2 } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
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
  type TimeWindow,
} from "@/features/realtime/use-realtime-monitor"

// RealtimePageFeature: top control row, chart area below.
export function RealtimePageFeature() {
  const {
    samples,
    loading,
    running,
    timeWindow,
    setTimeWindow,
    setRunning,
    clearSamples,
    refreshNow,
  } = useRealtimeMonitor()

  const {
    activeGroup,
    groupMetrics,
    visibleKeys,
    handleGroupChange,
    toggleMetric,
  } = useChartMetrics()

  const latest = samples.length > 0 ? samples[samples.length - 1] : null
  const runningCont = latest?.runningContainers ?? 0
  const totalCont = latest?.totalContainers ?? 0
  const runningRep = latest?.runningReplicas ?? 0
  const totalRep = latest?.totalReplicas ?? 0

  return (
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
                  <Select value={timeWindow} onValueChange={(v) => setTimeWindow(v as TimeWindow)}>
                    <SelectTrigger className="h-8 w-full text-xs">
                      <SelectValue placeholder="时间窗口" />
                    </SelectTrigger>
                    <SelectContent className="rounded-xl">
                      {TIME_WINDOWS.map((opt) => (
                        <SelectItem key={opt.value} value={opt.value} className="rounded-lg text-xs">
                          {opt.label}
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
          </div>

          <div className="flex items-center gap-2">
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
      </div>

      <div className="min-h-0 flex-1 bg-card p-4">
        <RealtimeUnifiedChart
          samples={samples}
          visibleKeys={visibleKeys}
          className="h-full"
        />
      </div>
    </div>
  )
}
