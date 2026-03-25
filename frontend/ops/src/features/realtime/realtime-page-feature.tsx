import { useState } from "react"
import {
  ArrowLeftRight,
  Pause,
  Play,
  RefreshCcw,
  Settings2,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  RealtimeUnifiedChart,
  groupByKey,
  metricByKey,
  type MetricGroupKey,
} from "@/features/realtime/realtime-unified-chart"
import {
  useRealtimeMonitor,
  TIME_WINDOWS,
  type TimeWindow,
} from "@/features/realtime/use-realtime-monitor"
import { MetricMiniChart } from "@/features/realtime/metric-mini-chart"
import { InfraStatusRow } from "@/features/realtime/infra-status-row"



export function RealtimePageFeature() {
  const {
    sysInfo,
    samples,
    latest,
    previous,
    loading,
    running,
    timeWindow,
    setTimeWindow,
    setRunning,
    refreshNow,
  } = useRealtimeMonitor()

  const [isServerLeft, setIsServerLeft] = useState(true)

  const leftGroupKey: MetricGroupKey = isServerLeft ? "server_metrics" : "kafka_pipeline"
  const rightGroupKey: MetricGroupKey = isServerLeft ? "kafka_pipeline" : "server_metrics"

  const leftVisibleKeys = groupByKey(leftGroupKey).metrics
  const rightGroupMetrics = groupByKey(rightGroupKey).metrics.map(metricByKey)

  return (
    <div className="h-full overflow-y-auto bg-muted/10">
      {/* 内部 grid：xl 下 min-h-full + 1fr/auto 填满视口；窄屏 auto 高度自然滚动 */}
      <div className="grid gap-6 p-4 md:p-6 xl:min-h-full xl:grid-rows-[1fr_auto]">

        {/* Row 1: 主图 + 微图列 */}
        <div className="grid gap-6 grid-cols-1 xl:grid-cols-[1fr_280px] min-h-0">

          {/* 左侧主图 — 非 xl 用固定高度，xl 下 grid stretch 自动填满行高 */}
          <div className="relative bg-background rounded-xl border shadow-sm overflow-hidden h-[480px] xl:h-auto">
            {/* 右上角浮动控件 */}
            <div className="absolute right-4 top-4 sm:right-5 sm:top-5 z-10 flex items-center gap-1">


              {/* 设置 Popover */}
              <Popover>
                <PopoverTrigger className="inline-flex items-center justify-center h-7 sm:h-8 w-7 sm:w-8 rounded-md text-muted-foreground hover:text-accent-foreground hover:bg-accent/50 transition-colors">
                  <Settings2 className="size-3.5" />
                </PopoverTrigger>
                <PopoverContent align="end" className="w-64 p-3 space-y-2">
                  {/* 显示粒度 */}
                  <div className="flex items-center justify-between gap-3">
                    <span className="text-xs text-muted-foreground shrink-0">显示粒度</span>
                    <Select value={timeWindow} onValueChange={(v) => setTimeWindow(v as TimeWindow)}>
                      <SelectTrigger className="h-7 text-xs w-[120px]">
                        <SelectValue placeholder="时间窗口" />
                      </SelectTrigger>
                      <SelectContent>
                        {TIME_WINDOWS.map((option) => (
                          <SelectItem key={option.value} value={option.value} className="text-xs">
                            {option.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  {/* 左右切换 */}
                  <div className="flex items-center justify-between gap-3">
                    <span className="text-xs text-muted-foreground shrink-0">主副视图</span>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => setIsServerLeft(v => !v)}
                      className="h-7 text-xs gap-1.5"
                    >
                      <ArrowLeftRight className="size-3" />
                      {isServerLeft ? "服务端 / Kafka" : "Kafka / 服务端"}
                    </Button>
                  </div>

                  {/* 采集控制 */}
                  <div className="flex items-center justify-between gap-3">
                    <span className="text-xs text-muted-foreground shrink-0">采集状态</span>
                    <div className="flex gap-1.5">
                      <Button
                        size="sm"
                        variant={running ? "outline" : "secondary"}
                        onClick={() => setRunning(!running)}
                        className="h-7 text-xs px-2.5"
                      >
                        {running ? <Pause className="size-3 mr-1" /> : <Play className="size-3 mr-1" />}
                        {running ? "暂停" : "恢复"}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={loading}
                        onClick={() => void refreshNow()}
                        className="h-7 text-xs px-2.5"
                      >
                        <RefreshCcw className="size-3 mr-1" />
                        刷新
                      </Button>
                    </div>
                  </div>


                </PopoverContent>
              </Popover>
            </div>

            {/* 图表区域 — absolute 填充确保 ResponsiveContainer 始终有确定的宽高 */}
            <div className="absolute inset-0 pt-4 sm:pt-5 px-4 sm:px-5 pb-4">
              <RealtimeUnifiedChart
                samples={samples}
                visibleKeys={leftVisibleKeys}
                className="h-full w-full"
              />
            </div>
          </div>

          {/* 右侧微图列 */}
          <div className="grid grid-cols-2 xl:grid-cols-1 gap-4 xl:grid-rows-3">
            {rightGroupMetrics.map((metric) => (
              <div key={metric.key} className="min-h-[120px]">
                <MetricMiniChart metric={metric} samples={samples} />
              </div>
            ))}
          </div>
        </div>

        {/* Row 2: 基础设施与节点健康 */}
        <div className="pb-4">
        <InfraStatusRow latest={latest} previous={previous} sysInfo={sysInfo} />
      </div>


      </div>
    </div>
  )
}

