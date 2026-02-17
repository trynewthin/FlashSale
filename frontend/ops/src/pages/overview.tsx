import { ExternalLink, RefreshCcw } from "lucide-react"
import { useCallback, useEffect, useMemo, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import type { ObservabilityLink, ObservabilityLinks, StatusSnapshot } from "@/api/types"
import { useOpsApiError } from "@/hooks/use-ops-api-error"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

function boolBadge(ok: boolean, okText = "ok", failText = "fail") {
  return <Badge variant={ok ? "secondary" : "destructive"}>{ok ? okText : failText}</Badge>
}

function modeLabel(mode?: string): string {
  if (mode === "docker_app") {
    return "Docker 模式"
  }
  if (mode === "host_process") {
    return "Host 进程模式"
  }
  return "未知模式"
}

function ObsLinkButton({ link }: { link: ObservabilityLink }) {
  if (!link.url) {
    return (
      <Button variant="outline" size="sm" disabled className="gap-1.5 opacity-50">
        <ExternalLink className="size-3.5" />
        {link.name}
        <Badge variant="secondary" className="ml-1 text-[10px]">未配置</Badge>
      </Button>
    )
  }
  if (link.available) {
    return (
      <a
        href={link.url}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-1.5 rounded-md border border-input bg-background px-3 py-1.5 text-sm font-medium shadow-xs hover:bg-accent hover:text-accent-foreground transition-colors"
      >
        <ExternalLink className="size-3.5" />
        {link.name}
        <Badge variant="secondary" className="ml-1 text-[10px]">可用</Badge>
      </a>
    )
  }
  return (
    <Button variant="secondary" size="sm" disabled className="gap-1.5">
      <ExternalLink className="size-3.5" />
      {link.name}
      <Badge variant="destructive" className="ml-1 text-[10px]">离线</Badge>
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
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>概览</CardTitle>
          <CardDescription>服务端口、HTTP 健康、Docker 与种子结果状态。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" onClick={() => void refresh()} disabled={loading}>
              <RefreshCcw className="size-4" />
              刷新
            </Button>
            {boolBadge(status?.docker.ok ?? false, "Docker 正常", "Docker 异常")}
            {boolBadge(status?.files.seed_result_ok ?? false, "Seed 可用", "Seed 缺失")}
            <Badge variant="outline">{modeLabel(status?.deployment_mode)}</Badge>
            <Badge variant="outline">{status?.git.branch || "-"}</Badge>
            <Badge variant="outline">{status?.git.commit || "-"}</Badge>
            {status ? boolBadge(!status.git.dirty, "Git Clean", "Git Dirty") : null}
          </div>
          <div className="text-xs text-muted-foreground">Seed 文件：{status?.files.seed_result_path || "-"}</div>
        </CardContent>
      </Card>

      {obsLinks && (
        <Card>
          <CardHeader>
            <CardTitle>可观测性入口</CardTitle>
            <CardDescription>由后端动态推断的可观测性工具链接。</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-3">
              {([obsLinks.jaeger, obsLinks.prometheus, obsLinks.grafana] as ObservabilityLink[]).map(
                (link) => (
                  <ObsLinkButton key={link.name} link={link} />
                )
              )}
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>端口监听</CardTitle>
            <CardDescription>
              {status?.deployment_mode === "docker_app"
                ? "Docker 模式下按容器运行状态判定服务可用性。"
                : "通过 TCP 拨测判断监听状态。"}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>服务</TableHead>
                  <TableHead>地址</TableHead>
                  <TableHead>结果</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(status?.ports || []).map((item) => (
                  <TableRow key={`${item.name}-${item.addr}`}>
                    <TableCell>{item.name}</TableCell>
                    <TableCell>{item.addr}</TableCell>
                    <TableCell>{boolBadge(item.ok, "ok", "down")}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>HTTP 检查</CardTitle>
            <CardDescription>
              {status?.deployment_mode === "docker_app"
                ? "Docker 模式下重点检查 Nginx 代理与 Ops 健康。"
                : "2xx 视为成功。"}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>检查项</TableHead>
                  <TableHead>URL</TableHead>
                  <TableHead>状态</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(status?.http || []).map((item) => (
                  <TableRow key={`${item.name}-${item.url}`}>
                    <TableCell>{item.name}</TableCell>
                    <TableCell>{item.url}</TableCell>
                    <TableCell>{boolBadge(item.ok, String(item.status_code || 0), "fail")}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Docker 容器</CardTitle>
          <CardDescription>来自 docker ps 快照。</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>容器</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>端口</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {dockerContainers.map((item) => (
                <TableRow key={`${item.name}-${item.status}`}>
                  <TableCell>{item.name}</TableCell>
                  <TableCell>{item.status}</TableCell>
                  <TableCell>{item.ports}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {!status?.docker.ok && (
            <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/10 p-2 text-xs text-destructive">
              {status?.docker.error || "docker 状态异常"}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
