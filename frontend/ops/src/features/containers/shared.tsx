import { Badge } from "@/components/ui/badge"

const serviceNameMap: Record<string, string> = {
  nginx: "Nginx 入口代理",
  "user-gateway": "用户网关服务",
  "admin-gateway": "管理网关服务",
  "user-rpc": "用户 RPC 服务",
  "admin-rpc": "管理员 RPC 服务",
  "product-rpc": "商品 RPC 服务",
  "order-rpc": "订单 RPC 服务",
  "seckill-rpc": "秒杀 RPC 服务",
  mysql: "MySQL 数据库",
  redis: "Redis 缓存",
  kafka: "Kafka 消息队列",
  etcd: "etcd 注册中心",
  jaeger: "Jaeger 链路追踪",
  prometheus: "Prometheus 指标采集",
  grafana: "Grafana 仪表盘",
  "mysql-init-user": "MySQL 初始化任务",
  "kafka-init": "Kafka 初始化任务",
  migrate: "数据库迁移任务",
}

export type RuntimeStatusTone = "running" | "partial" | "stopped" | "absent"

// roleLabel 将后端角色编码转换为中文展示文案。
export function roleLabel(role: string): string {
  switch (role) {
    case "ingress":
      return "入口层"
    case "gateway":
      return "网关层"
    case "rpc":
      return "业务服务层"
    case "infra":
      return "基础设施层"
    case "observability":
      return "可观测性"
    case "job":
      return "初始化/任务层"
    default:
      return "其他"
  }
}

// roleBgColor 为不同角色提供节点背景色。
export function roleBgColor(role: string): string {
  switch (role) {
    case "ingress":
      return "bg-sky-50"
    case "gateway":
      return "bg-indigo-50"
    case "rpc":
      return "bg-emerald-50"
    case "infra":
      return "bg-amber-50"
    case "observability":
      return "bg-violet-50"
    case "job":
      return "bg-zinc-100"
    default:
      return "bg-slate-50"
  }
}

// serviceDisplayName 将服务代码名转换为业务展示名。
export function serviceDisplayName(name: string): string {
  return serviceNameMap[name] || `${name} 服务`
}

// runtimeStatus 统一输出副本运行状态与语义色彩。
// absent 参数：catalog 中定义但无实际容器存在（可选服务未部署，不应报红）。
export function runtimeStatus(runningReplicas: number, replicas: number, absent = false): {
  text: string
  tone: RuntimeStatusTone
} {
  if (absent) {
    return { text: "未部署", tone: "absent" }
  }
  if (replicas <= 0) {
    return { text: "未部署", tone: "stopped" }
  }
  if (runningReplicas <= 0) {
    return { text: "已停止", tone: "stopped" }
  }
  if (runningReplicas < replicas) {
    return { text: "部分运行", tone: "partial" }
  }
  return { text: "运行中", tone: "running" }
}

// runtimeStatusClass 为运行状态提供统一配色。
export function runtimeStatusClass(tone: RuntimeStatusTone): string {
  switch (tone) {
    case "running":
      return "border-emerald-200 bg-emerald-50 text-emerald-700"
    case "partial":
      return "border-amber-200 bg-amber-50 text-amber-700"
    case "absent":
      return "border-slate-200 bg-slate-50 text-slate-400"
    case "stopped":
    default:
      return "border-rose-200 bg-rose-50 text-rose-700"
  }
}

// serviceEtcdKey 返回对应服务在 etcd 中的注册 Key，只有 RPC 服务有注册。
// 与 apps/*/rpc/etc/*.docker.yaml 中 Etcd.Key 保持一致。
const etcdKeyMap: Record<string, string> = {
  "user-rpc": "user.rpc",
  "admin-rpc": "admin.rpc",
  "product-rpc": "product.rpc",
  "order-rpc": "order.rpc",
  "seckill-rpc": "seckill.rpc",
}

export function serviceEtcdKey(serviceName: string): string | undefined {
  return etcdKeyMap[serviceName]
}

// healthBadge 统一渲染健康状态标签。
export function healthBadge(ok: boolean, okText = "正常", failText = "异常") {
  return <Badge variant={ok ? "secondary" : "destructive"}>{ok ? okText : failText}</Badge>
}
