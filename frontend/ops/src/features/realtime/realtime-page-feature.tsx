import { Activity, Clock, Pause, Play, RefreshCcw, Trash2 } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
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

// RealtimePageFeature — 两段式布局：上方图表 + 下方操作区。
export function RealtimePageFeature() {
  const {
    samples,
    latest,
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

  return (
    <div className="flex h-full flex-col gap-4">
      {/* ─── 上方：图表区域 ─── */}
      <div className="min-h-0 flex-1 rounded-xl border bg-card p-4">
        <RealtimeUnifiedChart
          samples={samples}
          visibleKeys={visibleKeys}
          className="h-full"
        />
      </div>

      {/* ─── 下方：操作卡片区 ─── */}
      <div className="grid grid-cols-1 gap-4 pb-2 md:grid-cols-2 xl:grid-cols-3">
        {/* 采样控制 */}
        <Card className="flex flex-col">
          <CardContent className="flex flex-1 flex-col py-4">
            {/* 状态指示 */}
            <div className="flex flex-wrap items-center gap-1.5">
              <Badge variant={running ? "secondary" : "outline"} className="text-[11px]">
                <Activity className="mr-1 size-3" />
                {running ? "采样中" : "已暂停"}
              </Badge>
              {latest ? (
                <Badge variant="outline" className="text-[11px]">
                  <Clock className="mr-1 size-3" />
                  {latest.label}
                </Badge>
              ) : null}
              <Badge variant="outline" className="text-[11px]">{samples.length} 点</Badge>
            </div>

            <div className="mt-auto space-y-3">
              <Separator />
              {/* 操作区域 */}
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
                <Select value={timeWindow} onValueChange={(v) => setTimeWindow(v as TimeWindow)}>
                  <SelectTrigger className="ml-auto h-8 w-[130px] text-xs">
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
            </div>
          </CardContent>
        </Card>

        {/* 显示筛选 */}
        <Card className="md:col-span-1 xl:col-span-2">
          <CardContent className="py-4">
            <MetricFilterPanel
              activeGroup={activeGroup}
              groupMetrics={groupMetrics}
              visibleKeys={visibleKeys}
              onGroupChange={handleGroupChange}
              onToggleMetric={toggleMetric}
            />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
