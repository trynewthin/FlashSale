import {
  AlertTriangle,
  Layers3,
  RefreshCcw,
  Server,
  ShieldCheck,
} from "lucide-react"
import { useCallback, useEffect, useMemo, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import type {
  ContainerRuntimeSnapshot,
  ObservabilityLinks,
  StatusSnapshot,
  SysInfo,
} from "@/api/types"
import { PageShell } from "@/components/layout/page-shell"
import { Button } from "@/components/ui/button"
import { useOpsApiError } from "@/hooks/use-ops-api-error"
import { cn } from "@/lib/utils"

import {
  MODULE_GROUPS,
  buildServiceViewModel,
  getServiceTone,
  type IssueItem,
} from "./overview-constants"
import { SectionCard, StatCard } from "./overview-cards"
import { OverviewTopPanels } from "./overview-top-panels"
import { ServiceStatusTile } from "./service-status-tile"

export function OverviewPageFeature() {
  const showApiError = useOpsApiError()
  const [status, setStatus] = useState<StatusSnapshot | null>(null)
  const [snapshot, setSnapshot] = useState<ContainerRuntimeSnapshot | null>(null)
  const [obsLinks, setObsLinks] = useState<ObservabilityLinks | null>(null)
  const [sysInfo, setSysInfo] = useState<SysInfo | null>(null)
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    try {
      setLoading(true)
      const [statusData, containerData, obsData, sysData] = await Promise.all([
        opsApi.getStatus(),
        opsApi.getContainersStatus().catch(() => null),
        opsApi.getObservabilityLinks().catch(() => null),
        opsApi.getSysInfo().catch(() => null),
      ])
      setStatus(statusData)
      setSnapshot(containerData)
      setObsLinks(obsData)
      setSysInfo(sysData)
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
        <OverviewTopPanels obsLinks={obsLinks} sysInfo={sysInfo} />


        {/* Stats Row */}
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <StatCard
            title="模块就绪"
            value={healthyServiceCount}
            total={servicesForAvailability.length || 0}
            tone={healthyServiceCount === servicesForAvailability.length ? "primary" : healthyServiceCount > 0 ? "primary" : "danger"}
            icon={Layers3}
          />
          <StatCard
            title="TCP 端口"
            value={tcpOkCount}
            total={status?.ports.length ?? 0}
            tone={tcpOkCount === (status?.ports.length ?? 0) ? "primary" : "warn"}
            icon={Server}
          />
          <StatCard
            title="HTTP 探活"
            value={httpOkCount}
            total={status?.http.length ?? 0}
            tone={httpOkCount === (status?.http.length ?? 0) ? "primary" : "warn"}
            icon={ShieldCheck}
          />
          <StatCard
            title="当前异常"
            value={issues.length}
            tone={issues.length === 0 ? "primary" : issues.length < 4 ? "warn" : "danger"}
            icon={AlertTriangle}
          />
        </div>

        {/* Module Groups */}
        <div className="grid gap-4 md:grid-cols-2 2xl:grid-cols-4">
          {moduleGroups.map((group) => (
            <SectionCard
              key={group.key}
              title={group.title}
              icon={group.icon}
            >
              <div className="grid gap-2">
                {group.items.map((item) => (
                  <ServiceStatusTile key={item.key} item={item} />
                ))}
              </div>
            </SectionCard>
          ))}
        </div>
      </div>
    </PageShell>
  )
}
