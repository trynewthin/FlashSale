import { useCallback, useEffect, useRef, useState } from "react"
import {
    Activity,
    ArrowLeft,
    ChevronRight,
    Clock,
    Download,
    FileText,
    Loader2,
    Play,
    RefreshCcw,
    Zap,
} from "lucide-react"

import type { JobDetail, JobSummary } from "@/api/types"
import { LogStreamViewer } from "@/components/common/log-stream-viewer"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import {
    Sheet,
    SheetContent,
    SheetHeader,
    SheetTitle,
    SheetDescription,
} from "@/components/ui/sheet"
import {
    PerfMetricChartCard,
    PERF_METRICS,
    computeAvg,
    computeMax,
    computeLatest,
} from "@/features/realtime/perf-metric-cards"
import { PerfSummaryCard } from "@/features/realtime/perf-summary-card"
import { RealtimeTestLauncher } from "@/features/realtime/realtime-test-launcher"
import { getRealtimeTestPreset } from "@/features/realtime/realtime-test-presets"
import { usePerfTestSamples } from "@/features/realtime/use-perf-test-samples"
import { useRealtimeTestRunner } from "@/features/realtime/use-realtime-test-runner"

function statusBadgeVariant(status: string) {
    if (status === "running") return "secondary" as const
    if (status === "success") return "outline" as const
    if (status === "failed") return "destructive" as const
    return "outline" as const
}

const MODE_LABELS: Record<string, string> = {
    running: "采集中",
    success: "已完成",
    failed: "已失败",
    queued: "排队中",
}

// ─── 视图类型 ───

type ViewMode = "guide" | "chart"

// PerfTestPageFeature — 双视图切换
export function PerfTestPageFeature() {
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
        perfSamples,
        recentTestJobs,
        startTest,
        refreshTasks,
        refreshTestJobs,
        refreshActiveJob,
        switchActiveJob,
        clearActiveLog,
    } = useRealtimeTestRunner()

    const { chartSamples, hasData } = usePerfTestSamples(perfSamples)

    const isTestRunning = activeJob?.status === "running"
    const statusMode = activeJob?.status ?? "idle"

    // ─── 视图管理 ───
    // 默认引导页；有正在运行的任务或选中了终态任务时自动切换到图表页
    const [viewMode, setViewMode] = useState<ViewMode>("guide")

    // 初始化时：如果有正在运行的测试，自动进入图表页
    const initialCheckRef = useRef(false)
    useEffect(() => {
        if (initialCheckRef.current) return
        if (recentTestJobs.length === 0) return // 还没加载完
        initialCheckRef.current = true

        // 查找正在运行的任务
        const runningJob = recentTestJobs.find((j) => j.status === "running")
        if (runningJob) {
            void switchActiveJob(runningJob.id)
            setViewMode("chart")
        }
    }, [recentTestJobs]) // eslint-disable-line react-hooks/exhaustive-deps

    // 启动测试后自动进入图表页
    const handleStartTest = useCallback(async () => {
        const ok = await startTest()
        if (ok) {
            setLauncherOpen(false)
            setViewMode("chart")
        }
    }, [startTest, setLauncherOpen])

    // 选择历史任务 → 进入图表页
    const handleSelectJob = useCallback(
        async (jobId: string) => {
            await switchActiveJob(jobId)
            setViewMode("chart")
        },
        [switchActiveJob]
    )

    // 返回引导页
    const handleGoBack = useCallback(() => {
        setViewMode("guide")
    }, [])

    // Sheet 日志面板状态
    const [logSheetOpen, setLogSheetOpen] = useState(false)





    // 当前任务显示名
    const activeJobTitle = activeJob
        ? (getRealtimeTestPreset(activeJob.task_id)?.title ?? activeJob.task_id)
        : "压力测试"

    return (
        <div className="flex h-full flex-col gap-3 overflow-y-auto p-3">
            {viewMode === "guide" ? (
                <GuideView
                    creating={creating}
                    isTestRunning={isTestRunning}
                    recentTestJobs={recentTestJobs}
                    onOpenLauncher={() => setLauncherOpen(true)}
                    onSelectJob={handleSelectJob}
                    onRefreshTestJobs={() => void refreshTestJobs()}
                />
            ) : (
                <ChartView
                    title={activeJobTitle}
                    statusMode={statusMode}
                    hasData={hasData}
                    isTestRunning={isTestRunning}
                    activeJob={activeJob}
                    chartSamples={chartSamples}
                    onGoBack={handleGoBack}
                    onOpenLogSheet={() => setLogSheetOpen(true)}
                    onRefreshActiveJob={() => void refreshActiveJob()}
                />
            )}

            {/* ─── 日志 Sheet（右侧滑出面板） ─── */}
            <Sheet open={logSheetOpen} onOpenChange={setLogSheetOpen}>
                <SheetContent side="right" className="flex w-full flex-col sm:max-w-lg">
                    <SheetHeader>
                        <SheetTitle>测试日志</SheetTitle>
                        <SheetDescription>
                            {activeJob
                                ? `${getRealtimeTestPreset(activeJob.task_id)?.title ?? activeJob.task_id} — ${activeJob.status}`
                                : "无活跃任务"}
                        </SheetDescription>
                    </SheetHeader>

                    {/* 日志控制 */}
                    <div className="flex items-center gap-2 px-4">
                        <Badge variant={activeJob ? statusBadgeVariant(activeJob.status) : "outline"} className="text-[11px]">
                            {activeJob?.status === "running" && <Activity className="mr-1 size-3 animate-pulse" />}
                            {activeJob?.status ?? "idle"}
                        </Badge>
                        <div className="ml-auto flex items-center gap-1">
                            <Button
                                size="sm"
                                variant="ghost"
                                className="h-6 px-2 text-[11px]"
                                disabled={!activeJob}
                                onClick={() => setStreamEnabled(!streamEnabled)}
                            >
                                {streamEnabled ? "暂停" : "恢复"}
                            </Button>
                            <Button
                                size="sm"
                                variant="ghost"
                                className="h-6 px-2 text-[11px]"
                                onClick={clearActiveLog}
                            >
                                清空
                            </Button>
                        </div>
                    </div>

                    {/* 日志内容 */}
                    <div className="min-h-0 flex-1 px-4 pb-4">
                        <LogStreamViewer
                            value={activeLog}
                            emptyText="暂无测试日志"
                            className="h-full rounded-lg border bg-muted/20"
                        />
                    </div>
                </SheetContent>
            </Sheet>

            {/* 测试启动器 Dialog */}
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
                onStartTest={handleStartTest}
                showFloatingTrigger={false}
            />
        </div>
    )
}

// ─── 引导视图 ───

interface GuideViewProps {
    creating: boolean
    isTestRunning: boolean
    recentTestJobs: JobSummary[]
    onOpenLauncher: () => void
    onSelectJob: (jobId: string) => Promise<void>
    onRefreshTestJobs: () => void
}

function GuideView({
    creating,
    isTestRunning,
    recentTestJobs,
    onOpenLauncher,
    onSelectJob,
    onRefreshTestJobs,
}: GuideViewProps) {
    return (
        <div className="mx-auto flex h-full w-full max-w-2xl flex-col items-center justify-center gap-8 p-6">
            {/* 主操作区 */}
            <div className="flex flex-col items-center gap-4">
                <div className="flex size-16 items-center justify-center rounded-2xl bg-primary/10">
                    <Zap className="size-8 text-primary" />
                </div>
                <div className="text-center">
                    <h2 className="text-lg font-semibold text-foreground">压力测试</h2>
                    <p className="mt-1 text-sm text-muted-foreground">
                        启动新测试或查看历史记录
                    </p>
                </div>
                <Button
                    size="lg"
                    onClick={onOpenLauncher}
                    disabled={creating || isTestRunning}
                    className="gap-2"
                >
                    {creating ? (
                        <Loader2 className="size-4 animate-spin" />
                    ) : (
                        <Play className="size-4" />
                    )}
                    {creating ? "启动中…" : isTestRunning ? "测试执行中" : "开始新测试"}
                </Button>
            </div>

            {/* 历史任务列表 */}
            {recentTestJobs.length > 0 && (
                <div className="w-full max-w-md">
                    <div className="mb-2 flex items-center justify-between">
                        <div className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                            <Clock className="size-4 text-muted-foreground" />
                            历史记录
                        </div>
                        <Button
                            size="sm"
                            variant="ghost"
                            className="h-6 text-xs text-muted-foreground"
                            onClick={onRefreshTestJobs}
                        >
                            <RefreshCcw className="mr-1 size-3" />
                            刷新
                        </Button>
                    </div>

                    <div className="max-h-64 overflow-y-auto rounded-xl border bg-card shadow-sm">
                        <div className="divide-y">
                            {recentTestJobs.map((job) => (
                                <button
                                    key={job.id}
                                    onClick={() => void onSelectJob(job.id)}
                                    className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-muted/50"
                                >
                                    <Badge
                                        variant={statusBadgeVariant(job.status)}
                                        className="h-5 shrink-0 text-[10px]"
                                    >
                                        {job.status === "running" && (
                                            <Loader2 className="mr-1 size-3 animate-spin" />
                                        )}
                                        {job.status}
                                    </Badge>
                                    <span className="flex-1 truncate text-sm">
                                        {getRealtimeTestPreset(job.task_id)?.title ?? job.task_id}
                                    </span>
                                    <span className="shrink-0 text-xs text-muted-foreground">
                                        {formatRelativeTime(job.created_at)}
                                    </span>
                                    <ChevronRight className="size-4 shrink-0 text-muted-foreground/50" />
                                </button>
                            ))}
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}

// ─── 导出测试报告 ───

function exportTestReport(
    job: JobDetail | null,
    samples: import("@/features/realtime/types").RealtimeSample[]
) {
    if (!job) return
    const preset = getRealtimeTestPreset(job.task_id)
    const report = {
        exportedAt: new Date().toISOString(),
        task: {
            id: job.task_id,
            name: preset?.title ?? job.task_name ?? job.task_id,
            jobId: job.id,
            status: job.status,
            exitCode: job.exit_code,
            args: job.args,
            createdAt: job.created_at,
            startedAt: job.started_at ?? null,
            finishedAt: job.finished_at ?? null,
            durationMs:
                job.started_at && job.finished_at
                    ? new Date(job.finished_at).getTime() - new Date(job.started_at).getTime()
                    : null,
        },
        summary: Object.fromEntries(
            PERF_METRICS.map((m) => [
                m.key,
                {
                    label: m.label,
                    unit: m.unit,
                    avg: +computeAvg(samples, m.key).toFixed(2),
                    max: +computeMax(samples, m.key).toFixed(2),
                    latest: +computeLatest(samples, m.key).toFixed(2),
                },
            ])
        ),
        sampleCount: samples.length,
    }

    const blob = new Blob([JSON.stringify(report, null, 2)], { type: "application/json" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = `perf-report-${job.task_id}-${job.id.slice(0, 8)}.json`
    a.click()
    URL.revokeObjectURL(url)
}

// ─── 图表视图 ───

interface ChartViewProps {
    title: string
    statusMode: string
    hasData: boolean
    isTestRunning: boolean
    activeJob: JobDetail | null
    chartSamples: import("@/features/realtime/types").RealtimeSample[]
    onGoBack: () => void
    onOpenLogSheet: () => void
    onRefreshActiveJob: () => void
}

function ChartView({
    title,
    statusMode,
    hasData,
    isTestRunning,
    activeJob,
    chartSamples,
    onGoBack,
    onOpenLogSheet,
    onRefreshActiveJob,
}: ChartViewProps) {
    return (
        <>
            {/* 顶部栏 */}
            <div className="sticky top-0 z-10 flex items-center gap-3 rounded-xl border bg-card px-4 py-2.5 shadow-sm">
                <Button
                    size="sm"
                    variant="ghost"
                    className="h-7 gap-1 px-2 text-xs"
                    onClick={onGoBack}
                    disabled={isTestRunning}
                    title={isTestRunning ? "测试运行中，无法返回" : "返回引导页"}
                >
                    <ArrowLeft className="size-3.5" />
                    返回
                </Button>

                <Separator orientation="vertical" className="h-5" />

                <div className="flex min-w-0 flex-1 items-center gap-2">
                    <span className="truncate text-sm font-medium text-foreground">
                        {title}
                    </span>
                    <Badge
                        variant={isTestRunning ? "secondary" : "outline"}
                        className="shrink-0 text-[10px]"
                    >
                        {isTestRunning && <Loader2 className="mr-1 size-3 animate-spin" />}
                        {MODE_LABELS[statusMode] ?? statusMode}
                    </Badge>
                    {isTestRunning && (
                        <Badge variant="secondary" className="shrink-0 text-[10px]">
                            <Activity className="mr-1 size-3 animate-pulse" />
                            执行中
                        </Badge>
                    )}
                </div>

                <div className="flex shrink-0 items-center gap-1.5">
                    <Button
                        size="sm"
                        variant="outline"
                        className="h-7 text-xs"
                        onClick={onRefreshActiveJob}
                        disabled={!activeJob}
                    >
                        <RefreshCcw className="size-3" />
                    </Button>
                    <Button
                        size="sm"
                        variant="outline"
                        className="h-7 text-xs"
                        onClick={() => exportTestReport(activeJob, chartSamples)}
                        disabled={!activeJob || chartSamples.length === 0}
                        title="导出测试报告"
                    >
                        <Download className="size-3" />
                    </Button>
                    <Button
                        size="sm"
                        variant="outline"
                        className="h-7 gap-1 text-xs"
                        onClick={onOpenLogSheet}
                    >
                        <FileText className="size-3" />
                        日志
                    </Button>
                </div>
            </div>

            {hasData ? (
                <div className="grid auto-rows-[180px] grid-cols-2 gap-3 xl:grid-cols-3 2xl:grid-cols-4">
                    {/* ① 测试摘要 */}
                    <PerfSummaryCard samples={chartSamples} />

                    {/* ② 各指标图表卡片 */}
                    {PERF_METRICS.map((metric) => (
                        <PerfMetricChartCard
                            key={metric.key}
                            metric={metric}
                            samples={chartSamples}
                        />
                    ))}
                </div>
            ) : (
                <div className="flex h-full min-h-[300px] flex-col items-center justify-center gap-3 rounded-xl border border-dashed bg-muted/20">
                    {isTestRunning ? (
                        <>
                            <Loader2 className="size-6 animate-spin text-muted-foreground" />
                            <span className="text-sm text-muted-foreground">
                                等待测试数据…
                            </span>
                        </>
                    ) : (
                        <span className="text-sm text-muted-foreground">
                            无测试数据
                        </span>
                    )}
                </div>
            )}
        </>
    )
}

// ─── 工具函数 ───

function formatRelativeTime(isoStr: string): string {
    try {
        const date = new Date(isoStr)
        const now = new Date()
        const diffMs = now.getTime() - date.getTime()
        const diffMin = Math.floor(diffMs / 60_000)

        if (diffMin < 1) return "刚刚"
        if (diffMin < 60) return `${diffMin} 分钟前`
        const diffHr = Math.floor(diffMin / 60)
        if (diffHr < 24) return `${diffHr} 小时前`
        const diffDay = Math.floor(diffHr / 24)
        return `${diffDay} 天前`
    } catch {
        return ""
    }
}
