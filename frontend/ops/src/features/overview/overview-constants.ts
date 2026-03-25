import type { LucideIcon } from "lucide-react"
import {
  Database,
  Globe,
  Telescope,
  Workflow,
} from "lucide-react"

import type {
  HTTPStatus,
  ObservabilityLink,
  ObservabilityLinks,
  PortStatus,
  ServiceRuntime,
} from "@/api/types"

// ── Types ──

export type ServiceTone = "healthy" | "partial" | "down" | "absent"

export interface ModuleGroupDef {
  key: string
  title: string
  icon: LucideIcon
  iconClassName: string
  services: string[]
}

export interface ServiceSignal {
  label: string
  ok: boolean
}

export interface ServiceViewModel {
  key: string
  label: string
  code: string
  tone: ServiceTone
  statusText: string
  detailText: string
  signals: ServiceSignal[]
}

export interface IssueItem {
  key: string
  title: string
  detail: string
  severity: "warn" | "error"
}

// ── Constants ──

export const SERVICE_LABELS: Record<string, string> = {
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

export const MODULE_GROUPS: ModuleGroupDef[] = [
  {
    key: "edge",
    title: "入口与控制面",
    icon: Globe,
    iconClassName: "bg-muted/60 text-muted-foreground",
    services: ["nginx", "cdn", "media-store", "user-gateway", "admin-gateway", "ops-control"],
  },
  {
    key: "core",
    title: "核心业务服务",
    icon: Workflow,
    iconClassName: "bg-muted/60 text-muted-foreground",
    services: ["user-rpc", "product-rpc", "order-rpc", "seckill-rpc", "admin-rpc"],
  },
  {
    key: "infra",
    title: "基础设施",
    icon: Database,
    iconClassName: "bg-muted/60 text-muted-foreground",
    services: ["mysql", "redis", "kafka", "etcd"],
  },
  {
    key: "observability",
    title: "可观测组件",
    icon: Telescope,
    iconClassName: "bg-muted/60 text-muted-foreground",
    services: ["prometheus", "grafana", "jaeger"],
  },
]

// ── Utility functions ──

export function toneClass(tone: ServiceTone): string {
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

export function signalClass(ok: boolean): string {
  return ok
    ? "border-emerald-500/20 bg-emerald-500/8 text-emerald-600 dark:text-emerald-400"
    : "border-destructive/20 bg-destructive/8 text-destructive"
}

export function moduleItemsClass(itemCount: number): string {
  if (itemCount <= 1) {
    return "grid gap-3"
  }
  if (itemCount === 2) {
    return "grid gap-3 sm:grid-cols-2"
  }
  return "grid auto-rows-fr gap-3 sm:grid-cols-2"
}

export function serviceLabel(name: string): string {
  return SERVICE_LABELS[name] ?? name
}

export function getServiceTone(service?: ServiceRuntime): ServiceTone {
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

export function getServiceStatusText(service?: ServiceRuntime): string {
  if (!service) return "未部署"
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

export function buildServiceViewModel(
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
      ? "未部署"
      : runtime.replicas > 0
        ? `${runtime.running_replicas}/${runtime.replicas} 副本`
        : "未配置副本"
    : "未部署"

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

export function issueToneClass(severity: "warn" | "error"): string {
  return severity === "error"
    ? "border-destructive/20 bg-destructive/8"
    : "border-amber-500/20 bg-amber-500/8"
}
