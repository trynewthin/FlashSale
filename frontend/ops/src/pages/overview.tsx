import {
  Activity,
  Box,
  CheckCircle2,
  Database,
  GitBranch,
  GitCommitHorizontal,
  Info,
  Network,
  RefreshCcw,
  Server,
  XCircle,
} from "lucide-react"
import { useCallback, useEffect, useMemo, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import type { ObservabilityLink, ObservabilityLinks, StatusSnapshot } from "@/api/types"
import { useOpsApiError } from "@/hooks/use-ops-api-error"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { PageShell } from "@/components/layout/page-shell"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { cn } from "@/lib/utils"

function statusBg(ok: boolean) {
  return ok ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border-emerald-500/20" : "bg-destructive/15 text-destructive border-destructive/20"
}

function StatusIcon({ ok, className }: { ok: boolean; className?: string }) {
  return ok ? <CheckCircle2 className={cn("size-4 text-emerald-500", className)} /> : <XCircle className={cn("size-4 text-destructive", className)} />
}

function modeLabel(mode?: string): string {
  if (mode === "docker_app") return "Docker 编排"
  if (mode === "host_process") return "Host 直连"
  return "未知模式"
}

function ObsLinkButton({ link }: { link: ObservabilityLink }) {
  if (!link.url) {
    return (
      <Button variant="outline" size="sm" disabled className="gap-2 shrink-0 opacity-50 w-full justify-start rounded-xl h-10 border-border/50">
        <div className="flex size-6 items-center justify-center rounded-md bg-muted">
          <Activity className="size-3.5" />
        </div>
        <span className="flex-1 text-left">{link.name}</span>
        <Badge variant="secondary" className="text-[10px] font-normal h-5 border-none bg-muted-foreground/10 text-muted-foreground">未配置</Badge>
      </Button>
    )
  }
  if (link.available) {
    return (
      <a
        href={link.url}
        target="_blank"
        rel="noopener noreferrer"
        className="group inline-flex w-full items-center gap-2 rounded-xl border border-primary/20 bg-primary/5 hover:bg-primary/10 px-3 h-10 text-sm font-medium transition-all shadow-sm hover:shadow-md hover:border-primary/30"
      >
        <div className="flex size-6 items-center justify-center rounded-md bg-background shadow-xs group-hover:scale-105 transition-transform border border-border/50 text-primary">
          <Activity className="size-3.5" />
        </div>
        <span className="flex-1 text-left text-foreground">{link.name}</span>
        <Badge variant="outline" className="text-[10px] font-medium h-5 border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 pb-0">Available</Badge>
      </a>
    )
  }
  return (
    <Button variant="outline" size="sm" disabled className="gap-2 shrink-0 w-full justify-start rounded-xl h-10 border-destructive/20 bg-destructive/5 text-destructive hover:bg-destructive/5">
      <div className="flex size-6 items-center justify-center rounded-md bg-destructive/10 text-destructive">
        <Activity className="size-3.5" />
      </div>
      <span className="flex-1 text-left">{link.name}</span>
      <Badge variant="outline" className="text-[10px] font-normal h-5 border-destructive/30 bg-destructive/10 text-destructive">Offline</Badge>
    </Button>
  )
}

// OverviewPage 展示服务状态快照。
export function OverviewPage() {
  const showApiError = useOpsApiError()
  const [status, setStatus] = useState<StatusSnapshot | null>(null)
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    try {
      setLoading(true)
      const data = await opsApi.getStatus()
      setStatus(data)
    } catch (error) {
      showApiError(error, "加载状态失败")
    } finally {
      setLoading(false)
    }
  }, [showApiError])

  useEffect(() => {
    void refresh()
    const timer = window.setInterval(() => {
      void refresh()
    }, 5000)
    return () => window.clearInterval(timer)
  }, [refresh])

  // 可观测性工具入口
  const [obsLinks, setObsLinks] = useState<ObservabilityLinks | null>(null)
  useEffect(() => {
    opsApi.getObservabilityLinks().then(setObsLinks).catch(() => { })
  }, [])

  const dockerContainers = useMemo(() => status?.docker.containers || [], [status])

  return (
    <PageShell
      title="系统运行概览"
      actions={
        <Button
          onClick={() => void refresh()}
          disabled={loading}
          variant="secondary"
          size="sm"
          className={cn("gap-2", loading && "opacity-80")}
        >
          <RefreshCcw className={cn("size-4 text-primary", loading && "animate-spin")} />
          {loading ? "获取中..." : "立即刷新"}
        </Button>
      }
    >
      <div className="space-y-6">
        <div className="grid gap-4 md:grid-cols-3">
          {/* 核心依赖 */}
          <Card className="shadow-sm border-border/50 bg-background/50 backdrop-blur-sm hover:border-primary/20 transition-colors">
            <CardHeader className="pb-3 border-b border-border/30">
              <CardTitle className="flex items-center gap-2 text-base">
                <div className="p-1.5 rounded-md bg-blue-500/10 text-blue-500">
                  <Database className="size-4" />
                </div>
                核心级依赖
              </CardTitle>
            </CardHeader>
            <CardContent className="pt-4 space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground flex items-center gap-2">
                  <Box className="size-4" />
                  Docker 引擎
                </span>
                <Badge variant="outline" className={cn("px-2.5 py-0.5", statusBg(status?.docker.ok ?? false))}>
                  <StatusIcon ok={status?.docker.ok ?? false} className="mr-1.5 size-3" />
                  {(status?.docker.ok ?? false) ? "正常工作" : "发生异常"}
                </Badge>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground flex items-center gap-2">
                  <Server className="size-4" />
                  Seed 数据状态
                </span>
                <Badge variant="outline" className={cn("px-2.5 py-0.5", statusBg(status?.files.seed_result_ok ?? false))}>
                  <StatusIcon ok={status?.files.seed_result_ok ?? false} className="mr-1.5 size-3" />
                  {(status?.files.seed_result_ok ?? false) ? "数据可用" : "缺失数据"}
                </Badge>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground flex items-center gap-2">
                  <Network className="size-4" />
                  服务部署模式
                </span>
                <Badge variant="outline" className="px-2.5 py-0.5 bg-muted/50 border-border text-foreground">
                  {modeLabel(status?.deployment_mode)}
                </Badge>
              </div>
            </CardContent>
          </Card>

          {/* 版本控制 */}
          <Card className="shadow-sm border-border/50 bg-background/50 backdrop-blur-sm hover:border-primary/20 transition-colors">
            <CardHeader className="pb-3 border-b border-border/30">
              <CardTitle className="flex items-center gap-2 text-base">
                <div className="p-1.5 rounded-md bg-purple-500/10 text-purple-500">
                  <GitBranch className="size-4" />
                </div>
                代码与部署版本
              </CardTitle>
            </CardHeader>
            <CardContent className="pt-4 space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground flex items-center gap-2">
                  <GitBranch className="size-4" />
                  当前分支
                </span>
                <div className="text-sm font-medium bg-muted/50 px-2 py-0.5 rounded border border-border/50">
                  {status?.git.branch || "未知"}
                </div>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground flex items-center gap-2">
                  <GitCommitHorizontal className="size-4" />
                  提交哈希
                </span>
                <div className="text-sm font-mono text-muted-foreground bg-muted/50 px-2 py-0.5 rounded border border-border/50">
                  {status?.git.commit ? status.git.commit.substring(0, 7) : "未知"}
                </div>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground flex items-center gap-2">
                  <Info className="size-4" />
                  工作区干净度
                </span>
                <Badge variant="outline" className={cn("px-2.5 py-0.5", statusBg(!(status?.git.dirty ?? true)))}>
                  <StatusIcon ok={!(status?.git.dirty ?? true)} className="mr-1.5 size-3" />
                  {!(status?.git.dirty ?? true) ? "Clean" : "Dirty"}
                </Badge>
              </div>
            </CardContent>
          </Card>

          {/* 可观测性 */}
          <Card className="shadow-sm border-border/50 bg-background/50 backdrop-blur-sm hover:border-primary/20 transition-colors relative overflow-hidden group">
            {/* 炫光装饰 */}
            <div className="absolute top-0 right-0 w-32 h-32 bg-primary/5 rounded-full blur-3xl -mr-16 -mt-16 group-hover:bg-primary/10 transition-colors" />
            <CardHeader className="pb-3 border-b border-border/30 relative z-10">
              <CardTitle className="flex items-center gap-2 text-base">
                <div className="p-1.5 rounded-md bg-orange-500/10 text-orange-500">
                  <Activity className="size-4" />
                </div>
                可观测性设施
              </CardTitle>
            </CardHeader>
            <CardContent className="pt-4 relative z-10">
              {obsLinks ? (
                <div className="flex flex-col gap-2.5">
                  <ObsLinkButton link={obsLinks.prometheus} />
                  <ObsLinkButton link={obsLinks.grafana} />
                  <ObsLinkButton link={obsLinks.jaeger} />
                </div>
              ) : (
                <div className="flex h-full min-h-[140px] items-center justify-center text-sm text-muted-foreground/60 flex-col gap-2">
                  <Activity className="size-6 opacity-30 animate-pulse" />
                  正在推断可观测性链路...
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* 底层细节面板 */}
        <div className="grid gap-6 xl:grid-cols-3">
          {/* 端口连通性 (TCP) */}
          <Card className="shadow-sm border-border/50 bg-background/50 flex flex-col pt-0 overflow-hidden">
            <div className="px-5 py-4 border-b border-border/50 bg-muted/20 shrink-0">
              <h3 className="font-semibold text-sm flex items-center gap-2">
                <Network className="size-4 text-primary" />
                端口连通性 (TCP)
              </h3>
            </div>
            <div className="flex-1 overflow-auto max-h-[300px]">
              <Table className="max-w-full">
                <TableHeader className="sticky top-0 bg-background/95 backdrop-blur z-10 shadow-sm">
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="w-1/3">目标服务</TableHead>
                    <TableHead>绑定地址</TableHead>
                    <TableHead className="w-[80px] text-right">状态</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(!status?.ports || status.ports.length === 0) ? (
                    <TableRow>
                      <TableCell colSpan={3} className="h-24 text-center text-muted-foreground/60">无可用端口数据</TableCell>
                    </TableRow>
                  ) : (
                    status.ports.map((item) => (
                      <TableRow key={`${item.name}-${item.addr}`} className="group hover:bg-muted/30">
                        <TableCell className="font-medium">{item.name}</TableCell>
                        <TableCell className="font-mono text-xs text-muted-foreground">{item.addr}</TableCell>
                        <TableCell className="text-right">
                          <StatusIcon ok={item.ok} />
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </Card>

          {/* 应用探活 (HTTP Probes) */}
          <Card className="shadow-sm border-border/50 bg-background/50 flex flex-col pt-0 overflow-hidden">
            <div className="px-5 py-4 border-b border-border/50 bg-muted/20 shrink-0">
              <h3 className="font-semibold text-sm flex items-center gap-2">
                <Activity className="size-4 text-emerald-500" />
                应用探活 (HTTP Probes)
              </h3>
            </div>
            <div className="flex-1 overflow-auto max-h-[300px]">
              <Table className="max-w-full">
                <TableHeader className="sticky top-0 bg-background/95 backdrop-blur z-10 shadow-sm">
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="w-1/3">探活对象</TableHead>
                    <TableHead>断言路径</TableHead>
                    <TableHead className="w-[80px] text-right">响应码</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(!status?.http || status.http.length === 0) ? (
                    <TableRow>
                      <TableCell colSpan={3} className="h-24 text-center text-muted-foreground/60">无应用探活数据</TableCell>
                    </TableRow>
                  ) : (
                    status.http.map((item) => (
                      <TableRow key={`${item.name}-${item.url}`} className="group hover:bg-muted/30">
                        <TableCell className="font-medium">{item.name}</TableCell>
                        <TableCell className="font-mono text-[10px] text-muted-foreground truncate max-w-[150px]" title={item.url}>{item.url}</TableCell>
                        <TableCell className="text-right">
                          <Badge variant="outline" className={cn("px-1.5 py-0 font-mono text-[11px]", statusBg(item.ok))}>
                            {item.status_code || "FAIL"}
                          </Badge>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </Card>

          {/* 右侧：Docker 容器状态 */}
          <Card className="shadow-sm border-border/50 bg-background/50 flex flex-col pt-0 overflow-hidden">
            <div className="px-5 py-4 border-b border-border/50 bg-muted/20 shrink-0 flex items-center justify-between">
              <h3 className="font-semibold text-sm flex items-center gap-2">
                <Box className="size-4 text-blue-500" />
                Docker 生命周期快照 (docker ps)
              </h3>
              <Badge variant="secondary" className="font-mono text-xs px-2">{dockerContainers.length} 实例</Badge>
            </div>

            {!status?.docker.ok && status?.docker.error ? (
              <div className="m-4 rounded-lg border border-destructive/30 bg-destructive/10 p-4 shrink-0">
                <div className="flex items-start gap-3 text-sm text-destructive font-mono">
                  <XCircle className="size-5 shrink-0 mt-0.5" />
                  <span className="break-all whitespace-pre-wrap leading-relaxed">{status.docker.error}</span>
                </div>
              </div>
            ) : null}

            <div className="flex-1 overflow-auto max-h-[300px]">
              <Table>
                <TableHeader className="sticky top-0 bg-background/95 backdrop-blur z-10 shadow-sm border-b">
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="w-[200px]">微服务容器</TableHead>
                    <TableHead>运行时状态</TableHead>
                    <TableHead>端口挂载映射</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {dockerContainers.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={3} className="h-32 text-center text-muted-foreground/60">
                        当前环境无运行中的 Docker 容器或未侦测到信息
                      </TableCell>
                    </TableRow>
                  ) : (
                    dockerContainers.map((item) => {
                      const isUp = item.status.toLowerCase().includes("up")
                      const isHealthy = item.status.toLowerCase().includes("healthy")
                      return (
                        <TableRow key={`${item.name}-${item.status}`} className="group hover:bg-muted/30">
                          <TableCell className="font-medium text-[13px]">{item.name}</TableCell>
                          <TableCell>
                            <span className={cn(
                              "inline-flex items-center gap-1.5 text-xs px-2 py-0.5 rounded-full font-medium border",
                              isUp
                                ? isHealthy
                                  ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
                                  : "bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20"
                                : "bg-muted text-muted-foreground border-border"
                            )}>
                              {isHealthy && <CheckCircle2 className="size-3" />}
                              {!isHealthy && isUp && <Activity className="size-3" />}
                              {item.status}
                            </span>
                          </TableCell>
                          <TableCell className="font-mono text-[10px] text-muted-foreground max-w-[200px] truncate" title={item.ports}>
                            {item.ports || "无挂载"}
                          </TableCell>
                        </TableRow>
                      )
                    })
                  )}
                </TableBody>
              </Table>
            </div>
          </Card>
        </div>
      </div>
    </PageShell>
  )
}
