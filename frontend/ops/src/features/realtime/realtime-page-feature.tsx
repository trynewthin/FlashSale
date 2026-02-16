import { Play, ScrollText, SlidersHorizontal } from "lucide-react"
import { useMemo, useState } from "react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { RealtimeTestLauncher } from "@/features/realtime/realtime-test-launcher"
import { RealtimeTestPanel } from "@/features/realtime/realtime-test-panel"
import { RealtimeUnifiedChart } from "@/features/realtime/realtime-unified-chart"
import { mergeRealtimeSamplesWithPerfPoints, parsePerfMetricPointsFromLog } from "@/features/realtime/perf-report-parser"
import { useRealtimeMonitor } from "@/features/realtime/use-realtime-monitor"
import { useRealtimeTestRunner } from "@/features/realtime/use-realtime-test-runner"

// RealtimePageFeature 重构为“上图下卡片”：上方统一图表，下方聚合操作卡片。
export function RealtimePageFeature() {
  const [logSheetOpen, setLogSheetOpen] = useState(false)
  const {
    samples,
    latest,
    loading,
    running,
    windowSeconds,
    setWindowSeconds,
    setRunning,
    clearSamples,
    refreshNow,
  } = useRealtimeMonitor()

  const {
    launcherOpen,
    setLauncherOpen,
    tasks,
    selectedTaskID,
    setSelectedTaskID,
    selectedPreset,
    formValues,
    setFormValue,
    extraArgsText,
    setExtraArgsText,
    creating,
    activeJob,
    activeLog,
    streamEnabled,
    setStreamEnabled,
    recentTestJobs,
    startTest,
    refreshTasks,
    refreshTestJobs,
    refreshActiveJob,
    switchActiveJob,
    clearActiveLog,
  } = useRealtimeTestRunner()

  const testMetricPoints = useMemo(() => parsePerfMetricPointsFromLog(activeLog), [activeLog])
  const chartSamples = useMemo(() => mergeRealtimeSamplesWithPerfPoints(samples, testMetricPoints), [samples, testMetricPoints])

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="space-y-1">
            <div className="text-base font-semibold">实时监测</div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={running ? "secondary" : "outline"}>{running ? "采样中" : "已暂停"}</Badge>
            <Badge variant="outline">采样点 {samples.length}</Badge>
            {latest ? <Badge variant="outline">最新 {latest.label}</Badge> : null}
            <Badge variant={activeJob && activeJob.status === "running" ? "secondary" : "outline"}>
              测试 {activeJob ? activeJob.status : "idle"}
            </Badge>
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button variant="outline" size="sm">
                    <SlidersHorizontal className="size-4" />
                    采样控制
                  </Button>
                }
              />
              <DropdownMenuContent align="end" className="w-56">
                <DropdownMenuItem disabled>采样控制</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setRunning(!running)}>
                  {running ? "暂停采样" : "继续采样"}
                </DropdownMenuItem>
                <DropdownMenuItem disabled={loading} onClick={() => void refreshNow()}>
                  立即采样
                </DropdownMenuItem>
                <DropdownMenuItem onClick={clearSamples}>清空曲线</DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem disabled>时间窗口</DropdownMenuItem>
                <DropdownMenuRadioGroup
                  value={String(windowSeconds)}
                  onValueChange={(value) => setWindowSeconds(Number(value))}
                >
                  <DropdownMenuRadioItem value="60">最近 1 分钟</DropdownMenuRadioItem>
                  <DropdownMenuRadioItem value="180">最近 3 分钟</DropdownMenuRadioItem>
                  <DropdownMenuRadioItem value="300">最近 5 分钟</DropdownMenuRadioItem>
                </DropdownMenuRadioGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
        <RealtimeUnifiedChart samples={chartSamples} />
      </div>

      <div className="grid gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">测试操作</CardTitle>
            <CardDescription>统一操作面板。</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-wrap items-center gap-2">
            <Button onClick={() => setLauncherOpen(true)} disabled={creating}>
              <Play className="size-4" />
              {creating ? "启动中..." : "开始测试"}
            </Button>
            <Button variant="outline" onClick={() => setLogSheetOpen(true)}>
              <ScrollText className="size-4" />
              日志
            </Button>
          </CardContent>
        </Card>
      </div>
      <Sheet open={logSheetOpen} onOpenChange={setLogSheetOpen}>
        <SheetContent side="right" className="w-[96vw] max-w-none sm:max-w-5xl">
          <SheetHeader className="p-4 pb-2">
            <SheetTitle>测试日志</SheetTitle>
          </SheetHeader>
          <div className="px-4 pb-4">
            <RealtimeTestPanel
              activeJob={activeJob}
              activeLog={activeLog}
              streamEnabled={streamEnabled}
              recentTestJobs={recentTestJobs}
              onToggleStream={() => setStreamEnabled(!streamEnabled)}
              onRefreshActiveJob={() => void refreshActiveJob()}
              onRefreshRecentJobs={() => void refreshTestJobs()}
              onClearActiveLog={clearActiveLog}
              onSwitchActiveJob={(jobID) => void switchActiveJob(jobID)}
            />
          </div>
        </SheetContent>
      </Sheet>

      <RealtimeTestLauncher
        open={launcherOpen}
        onOpenChange={setLauncherOpen}
        tasks={tasks}
        selectedTaskID={selectedTaskID}
        selectedPreset={selectedPreset}
        formValues={formValues}
        extraArgsText={extraArgsText}
        creating={creating}
        onSelectTask={setSelectedTaskID}
        onSetFormValue={setFormValue}
        onChangeExtraArgs={setExtraArgsText}
        onRefreshTasks={() => void refreshTasks()}
        onStartTest={() => void startTest()}
        showFloatingTrigger={false}
      />
    </div>
  )
}
