import {
  Activity,
  AlertTriangle,
  Box,
  CheckCircle2,
  Database,
  Globe,
  Layers3,
  type LucideIcon,
  RefreshCcw,
  Server,
  ShieldCheck,
  Telescope,
  Workflow,
  XCircle,
} from "lucide-react"
import { useCallback, useEffect, useMemo, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import type {
  ContainerRuntimeSnapshot,
  HTTPStatus,
  ObservabilityLink,
  ObservabilityLinks,
  PortStatus,
  ServiceRuntime,
  StatusSnapshot,
} from "@/api/types"
import { PageShell } from "@/components/layout/page-shell"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useOpsApiError } from "@/hooks/use-ops-api-error"
import { cn } from "@/lib/utils"

type ServiceTone = "healthy" | "partial" | "down" | "absent"

interface ModuleGroupDef {
  key: string
  title: string
  description: string
  icon: LucideIcon
  iconClassName: string
  services: string[]
}

interface ServiceSignal {
  label: string
  ok: boolean
}

interface ServiceViewModel {
  key: string
  label: string
  code: string
  tone: ServiceTone
  statusText: string
  detailText: string
  signals: ServiceSignal[]
}

interface IssueItem {
  key: string
  title: string
  detail: string
  severity: "warn" | "error"
}

const SERVICE_LABELS: Record<string, string> = {
  nginx: "Nginx 入口代理",
  cdn: "静态资源代理",
  "media-store": "媒体存储",
  "ops-control": "运维控制台",
  "user-gateway": "用户网关",
  "admin-gateway": "管理网关",
  "user-rpc": "用户 RPC",
  "product-rpc": "商品 RPC",
  "order-rpc": "订单 RPC",
  "seckill-rpc": "秒杀 RPC",
  "admin-rpc": "管理 RPC",
  mysql: "MySQL",
  redis: "Redis",
  kafka: "Kafka",
  etcd: "etcd",
  prometheus: "Prometheus",
  grafana: "Grafana",
  jaeger: "Jaeger",
}

const MODULE_GROUPS: ModuleGroupDef[] = [
  {
    key: "edge",
    title: "入口与控制面",
    description: "用户流量入口、管理入口与运维控制面。",
    icon: Globe,
    iconClassName: "bg-sky-500/10 text-sky-500",
    services: ["nginx", "cdn", "media-store", "user-gateway", "admin-gateway", "ops-control"],
  },
  {
    key: "core",
    title: "核心业务服务",
    description: "承接核心业务流量的 RPC 服务。",
    icon: Workflow,
    iconClassName: "bg-emerald-500/10 text-emerald-500",
    services: ["user-rpc", "product-rpc", "order-rpc", "seckill-rpc", "admin-rpc"],
  },
  {
    key: "infra",
    title: "基础设施",
    description: "存储、缓存、消息与注册中心。",
    icon: Database,
    iconClassName: "bg-amber-500/10 text-amber-500",
    services: ["mysql", "redis", "kafka", "etcd"],
  },
  {
    key: "observability",
    title: "可观测组件",
    description: "指标、链路与可视化入口。",
    icon: Telescope,
    iconClassName: "bg-violet-500/10 text-violet-500",
    services: ["prometheus", "grafana", "jaeger"],
  },
]

function toneClass(tone: ServiceTone): string {
  switch (tone) {
    case "healthy":
      return "border-emerald-500/20 bg-emerald-500/8 text-emerald-600 dark:text-emerald-400"
    case "partial":
      return "border-amber-500/20 bg-amber-500/8 text-amber-600 dark:text-amber-400"
    case "absent":
      return "border-border bg-muted/40 text-muted-foreground"
    case "down":
    default:
      return "border-destructive/20 bg-destructive/8 text-destructive"
  }
}

function signalClass(ok: boolean): string {
  return ok
    ? "border-emerald-500/20 bg-emerald-500/8 text-emerald-600 dark:text-emerald-400"
    : "border-destructive/20 bg-destructive/8 text-destructive"
}

function moduleItemsClass(itemCount: number): string {
  if (itemCount <= 1) {
    return "grid gap-3"
  }
  if (itemCount === 2) {
    return "grid gap-3 sm:grid-cols-2"
  }
  return "grid auto-rows-fr gap-3 sm:grid-cols-2"
}

function StatusIcon({ ok, className }: { ok: boolean; className?: string }) {
  return ok
    ? <CheckCircle2 className={cn("size-4 text-emerald-500", className)} />
    : <XCircle className={cn("size-4 text-destructive", className)} />
}

function serviceLabel(name: string): string {
  return SERVICE_LABELS[name] ?? name
}

function getServiceTone(service?: ServiceRuntime): ServiceTone {
  if (!service) {
    return "down"
  }
  if (service.absent) {
    return "absent"
  }
  if (service.replicas <= 0 || service.running_replicas <= 0) {
    return "down"
  }
  if (service.running_replicas < service.replicas) {
    return "partial"
  }
  return "healthy"
}

function getServiceStatusText(service?: ServiceRuntime): string {
  if (!service) return "未发现"
  if (service.absent) return "未部署"
  if (service.replicas <= 0) return "未启动"
  if (service.running_replicas <= 0) return "不可用"
  if (service.running_replicas < service.replicas) return "部分可用"
  return "可用"
}

function buildServiceSignals(
  serviceName: string,
  portMap: Map<string, PortStatus>,
  httpMap: Map<string, HTTPStatus>,
  obsLinks: ObservabilityLinks | null,
): ServiceSignal[] {
  const signals: ServiceSignal[] = []
  const portStatus = portMap.get(serviceName)
  if (portStatus) {
    signals.push({ label: "TCP", ok: portStatus.ok })
  }

  const httpChecks: Record<string, string[]> = {
    nginx: ["nginx-healthz"],
    "ops-control": ["ops-control/healthz"],
    cdn: ["proxy user /healthz"],
    "media-store": ["proxy admin /admin-healthz"],
  }

  for (const checkName of httpChecks[serviceName] ?? []) {
    const check = httpMap.get(checkName)
    if (check) {
      signals.push({ label: "HTTP", ok: check.ok })
      break
    }
  }

  const obsMap: Partial<Record<string, ObservabilityLink | undefined>> = {
    prometheus: obsLinks?.prometheus,
    grafana: obsLinks?.grafana,
    jaeger: obsLinks?.jaeger,
  }
  const obsLink = obsMap[serviceName]
  if (obsLink) {
    signals.push({ label: "入口", ok: obsLink.available })
  }

  return signals
}

function buildServiceViewModel(
  serviceName: string,
  serviceMap: Map<string, ServiceRuntime>,
  portMap: Map<string, PortStatus>,
  httpMap: Map<string, HTTPStatus>,
  obsLinks: ObservabilityLinks | null,
): ServiceViewModel {
  const runtime = serviceMap.get(serviceName)
  const tone = getServiceTone(runtime)
  const detailText = runtime
    ? runtime.absent
      ? "可选服务未部署"
      : runtime.replicas > 0
        ? `${runtime.running_replicas}/${runtime.replicas} 副本`
        : "未配置副本"
    : "未纳入当前编排"

  return {
    key: serviceName,
    label: serviceLabel(serviceName),
    code: serviceName,
    tone,
    statusText: getServiceStatusText(runtime),
    detailText,
    signals: buildServiceSignals(serviceName, portMap, httpMap, obsLinks),
  }
}

function SectionCard({
  title,
  description,
  icon: Icon,
  iconClassName,
  action,
  children,
}: {
  title: string
  description?: string
  icon: LucideIcon
  iconClassName: string
  action?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <Card className="border-border/50 bg-background/70 shadow-sm">
      <CardHeader className="border-b border-border/40 pb-4">
        <div className="flex items-start justify-between gap-3">
          <div className="space-y-1">
            <CardTitle className="flex items-center gap-2 text-base">
              <div className={cn("rounded-md p-1.5", iconClassName)}>
                <Icon className="size-4" />
              </div>
              {title}
            </CardTitle>
            {description ? <p className="text-sm text-muted-foreground">{description}</p> : null}
          </div>
          {action}
        </div>
      </CardHeader>
      <CardContent className="pt-4">{children}</CardContent>
    </Card>
  )
}

function StatCard({
  title,
  value,
  subtext,
  tone = "neutral",
  icon: Icon,
}: {
  title: string
  value: string
  subtext: string
  tone?: "healthy" | "warn" | "danger" | "neutral"
  icon: LucideIcon
}) {
  const toneClassName =
    tone === "healthy"
      ? "bg-emerald-500/10 text-emerald-500"
      : tone === "warn"
        ? "bg-amber-500/10 text-amber-500"
        : tone === "danger"
          ? "bg-destructive/10 text-destructive"
          : "bg-primary/10 text-primary"

  return (
    <Card className="border-border/50 bg-background/70 shadow-sm">
      <CardContent className="flex items-start justify-between gap-4 pt-5">
        <div className="space-y-1">
          <p className="text-sm text-muted-foreground">{title}</p>
          <p className="text-2xl font-semibold tracking-tight">{value}</p>
          <p className="text-xs text-muted-foreground">{subtext}</p>
        </div>
        <div className={cn("rounded-lg p-2", toneClassName)}>
          <Icon className="size-5" />
        </div>
      </CardContent>
    </Card>
  )
}

function ServiceStatusTile({ item }: { item: ServiceViewModel }) {
  return (
    <div className="flex h-full flex-col rounded-xl border border-border/50 bg-muted/20 p-3.5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate text-sm font-medium text-foreground">{item.label}</div>
          <div className="mt-0.5 text-xs text-muted-foreground">{item.code}</div>
        </div>
        <Badge variant="outline" className={cn("shrink-0 px-2 py-0.5", toneClass(item.tone))}>
          {item.statusText}
        </Badge>
      </div>
      <div className="mt-3 flex flex-1 flex-col justify-end gap-2.5">
        <span className="text-xs text-muted-foreground">{item.detailText}</span>
        <div className="flex flex-wrap items-center gap-1.5">
          {item.signals.length === 0 ? (
            <Badge variant="outline" className="px-1.5 py-0 text-[10px] text-muted-foreground">
              无附加探测
            </Badge>
          ) : (
            item.signals.map((signal) => (
              <Badge
                key={`${item.key}-${signal.label}`}
                variant="outline"
                className={cn("px-1.5 py-0 text-[10px]", signalClass(signal.ok))}
              >
                {signal.label}
              </Badge>
            ))
          )}
        </div>
      </div>
    </div>
  )
}

function ObservabilityLinkButton({ link }: { link: ObservabilityLink }) {
  if (!link.url) {
    return (
      <Button variant="outline" size="sm" disabled className="h-11 w-full justify-start gap-3 rounded-xl opacity-60">
        <Activity className="size-4" />
        <span className="flex-1 text-left">{link.name}</span>
        <Badge variant="secondary" className="text-[10px]">Unset</Badge>
      </Button>
    )
  }

  if (link.available) {
    return (
      <a
        href={link.url}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex h-11 w-full items-center gap-3 rounded-xl border border-emerald-500/20 bg-emerald-500/8 px-3 text-sm font-medium transition-colors hover:bg-emerald-500/12"
      >
        <Activity className="size-4 text-emerald-500" />
        <span className="flex-1 text-left text-foreground">{link.name}</span>
        <Badge variant="outline" className="border-emerald-500/30 bg-emerald-500/10 text-[10px] text-emerald-600 dark:text-emerald-400">
          Available
        </Badge>
      </a>
    )
  }

  return (
    <Button variant="outline" size="sm" disabled className="h-11 w-full justify-start gap-3 rounded-xl border-destructive/20 bg-destructive/8 text-destructive">
      <Activity className="size-4" />
      <span className="flex-1 text-left">{link.name}</span>
      <Badge variant="outline" className="border-destructive/30 bg-destructive/10 text-[10px] text-destructive">
        Offline
      </Badge>
    </Button>
  )
}

function issueToneClass(severity: "warn" | "error"): string {
  return severity === "error"
    ? "border-destructive/20 bg-destructive/8"
    : "border-amber-500/20 bg-amber-500/8"
}

export function OverviewPage() {
  const showApiError = useOpsApiError()
  const [status, setStatus] = useState<StatusSnapshot | null>(null)
  const [snapshot, setSnapshot] = useState<ContainerRuntimeSnapshot | null>(null)
  const [obsLinks, setObsLinks] = useState<ObservabilityLinks | null>(null)
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    try {
      setLoading(true)
      const [statusData, containerData, obsData] = await Promise.all([
        opsApi.getStatus(),
        opsApi.getContainersStatus().catch(() => null),
        opsApi.getObservabilityLinks().catch(() => null),
      ])
      setStatus(statusData)
      setSnapshot(containerData)
      setObsLinks(obsData)
    } catch (error) {
      showApiError(error, "加载概览失败")
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

  const serviceMap = useMemo(
    () => new Map((snapshot?.services ?? []).map((service) => [service.name, service])),
    [snapshot],
  )
  const portMap = useMemo(
    () => new Map((status?.ports ?? []).map((item) => [item.name, item])),
    [status],
  )
  const httpMap = useMemo(
    () => new Map((status?.http ?? []).map((item) => [item.name, item])),
    [status],
  )

  const moduleGroups = useMemo(
    () =>
      MODULE_GROUPS.map((group) => ({
        ...group,
        items: group.services.map((serviceName) =>
          buildServiceViewModel(serviceName, serviceMap, portMap, httpMap, obsLinks),
        ),
      })),
    [httpMap, obsLinks, portMap, serviceMap],
  )

  const servicesForAvailability = useMemo(
    () => (snapshot?.services ?? []).filter((service) => !(service.optional && service.absent)),
    [snapshot],
  )
  const healthyServiceCount = servicesForAvailability.filter((service) => getServiceTone(service) === "healthy").length
  const tcpOkCount = status?.ports.filter((item) => item.ok).length ?? 0
  const httpOkCount = status?.http.filter((item) => item.ok).length ?? 0
  const issues = useMemo<IssueItem[]>(() => {
    const nextIssues: IssueItem[] = []

    if (status && !status.files.seed_result_ok) {
      nextIssues.push({
        key: "seed",
        title: "Seed 数据不可用",
        detail: status.files.seed_result_path,
        severity: "warn",
      })
    }

    if (status && !status.docker.ok) {
      nextIssues.push({
        key: "docker",
        title: "Docker 状态异常",
        detail: status.docker.error || "docker ps 无法正常返回",
        severity: "error",
      })
    }

    for (const item of status?.ports ?? []) {
      if (!item.ok) {
        nextIssues.push({
          key: `port-${item.name}`,
          title: `${item.name} TCP 不可达`,
          detail: item.addr,
          severity: "error",
        })
      }
    }

    for (const item of status?.http ?? []) {
      if (!item.ok) {
        nextIssues.push({
          key: `http-${item.name}`,
          title: `${item.name} HTTP 探活失败`,
          detail: item.url,
          severity: "error",
        })
      }
    }

    if (obsLinks) {
      for (const link of [obsLinks.prometheus, obsLinks.grafana, obsLinks.jaeger]) {
        if (!link.available) {
          nextIssues.push({
            key: `obs-${link.name}`,
            title: `${link.name} 不可访问`,
            detail: link.url || "未配置访问地址",
            severity: "warn",
          })
        }
      }
    }

    return nextIssues.slice(0, 10)
  }, [obsLinks, status])

  return (
    <PageShell
      title="模块状态概览"
      actions={(
        <Button
          onClick={() => void refresh()}
          disabled={loading}
          variant="secondary"
          size="sm"
          className={cn("gap-2", loading && "opacity-80")}
        >
          <RefreshCcw className={cn("size-4 text-primary", loading && "animate-spin")} />
          {loading ? "刷新中..." : "立即刷新"}
        </Button>
      )}
    >
      <div className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <StatCard
            title="模块就绪"
            value={`${healthyServiceCount}/${servicesForAvailability.length || 0}`}
            subtext="按编排服务副本状态统计"
            tone={healthyServiceCount === servicesForAvailability.length ? "healthy" : healthyServiceCount > 0 ? "warn" : "danger"}
            icon={Layers3}
          />
          <StatCard
            title="TCP 端口"
            value={`${tcpOkCount}/${status?.ports.length ?? 0}`}
            subtext="核心网关与 RPC 端口可达性"
            tone={tcpOkCount === (status?.ports.length ?? 0) ? "healthy" : "warn"}
            icon={Server}
          />
          <StatCard
            title="HTTP 探活"
            value={`${httpOkCount}/${status?.http.length ?? 0}`}
            subtext="Nginx、代理与 ops healthz"
            tone={httpOkCount === (status?.http.length ?? 0) ? "healthy" : "warn"}
            icon={ShieldCheck}
          />
          <StatCard
            title="当前异常"
            value={String(issues.length)}
            subtext="汇总容器、探活、观测与数据问题"
            tone={issues.length === 0 ? "healthy" : issues.length < 4 ? "warn" : "danger"}
            icon={AlertTriangle}
          />
        </div>

        <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1.7fr)_minmax(340px,0.9fr)]">
          <div className="grid gap-6 2xl:grid-cols-2">
            {moduleGroups.map((group) => {
              const healthyCount = group.items.filter((item) => item.tone === "healthy").length
              return (
                <SectionCard
                  key={group.key}
                  title={group.title}
                  description={group.description}
                  icon={group.icon}
                  iconClassName={group.iconClassName}
                  action={<Badge variant="secondary">{healthyCount}/{group.items.length}</Badge>}
                >
                  <div className={moduleItemsClass(group.items.length)}>
                    {group.items.map((item) => (
                      <ServiceStatusTile key={item.key} item={item} />
                    ))}
                  </div>
                </SectionCard>
              )
            })}
          </div>

          <div className="space-y-6">
            <SectionCard
              title="当前异常"
              description="优先处理会直接影响可用性的模块问题。"
              icon={AlertTriangle}
              iconClassName="bg-destructive/10 text-destructive"
            >
              {issues.length === 0 ? (
                <div className="flex min-h-[220px] flex-col items-center justify-center gap-3 rounded-xl border border-emerald-500/20 bg-emerald-500/6 px-4 text-center">
                  <CheckCircle2 className="size-8 text-emerald-500" />
                  <div>
                    <div className="font-medium text-foreground">当前没有发现阻断性异常</div>
                    <div className="mt-1 text-sm text-muted-foreground">入口、模块与观测链路都在正常范围内。</div>
                  </div>
                </div>
              ) : (
                <div className="space-y-3">
                  {issues.map((issue) => (
                    <div key={issue.key} className={cn("rounded-xl border px-3 py-3", issueToneClass(issue.severity))}>
                      <div className="flex items-start gap-2">
                        {issue.severity === "error" ? (
                          <XCircle className="mt-0.5 size-4 shrink-0 text-destructive" />
                        ) : (
                          <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-500" />
                        )}
                        <div className="min-w-0">
                          <div className="text-sm font-medium text-foreground">{issue.title}</div>
                          <div className="mt-1 break-all text-xs text-muted-foreground">{issue.detail}</div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </SectionCard>

            <SectionCard
              title="观测入口"
              description="出现异常时，直接进入指标与链路页面继续排查。"
              icon={Telescope}
              iconClassName="bg-violet-500/10 text-violet-500"
            >
              {obsLinks ? (
                <div className="space-y-3">
                  <ObservabilityLinkButton link={obsLinks.prometheus} />
                  <ObservabilityLinkButton link={obsLinks.grafana} />
                  <ObservabilityLinkButton link={obsLinks.jaeger} />
                </div>
              ) : (
                <div className="flex min-h-[180px] items-center justify-center text-sm text-muted-foreground/60">
                  正在加载观测入口...
                </div>
              )}
            </SectionCard>

            <SectionCard
              title="运行背景"
              description="保留少量辅助信息，避免概览页被版本介绍占满。"
              icon={Box}
              iconClassName="bg-slate-500/10 text-slate-500"
            >
              <div className="grid gap-3">
                <div className="rounded-xl border border-border/50 bg-muted/20 px-3 py-3">
                  <div className="text-xs text-muted-foreground">部署模式</div>
                  <div className="mt-1 text-sm font-medium text-foreground">
                    {status?.deployment_mode === "docker_app"
                      ? "Docker 编排"
                      : status?.deployment_mode === "host_process"
                        ? "Host 直连"
                        : "未知"}
                  </div>
                </div>
                <div className="rounded-xl border border-border/50 bg-muted/20 px-3 py-3">
                  <div className="text-xs text-muted-foreground">Seed 数据</div>
                  <div className="mt-1 flex items-center gap-2 text-sm font-medium text-foreground">
                    <StatusIcon ok={status?.files.seed_result_ok ?? false} />
                    {(status?.files.seed_result_ok ?? false) ? "可用" : "缺失或无效"}
                  </div>
                </div>
              </div>
            </SectionCard>
          </div>
        </div>
      </div>
    </PageShell>
  )
}
