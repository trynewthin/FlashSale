import { Badge } from "@/components/ui/badge"

const serviceNameMap: Record<string, string> = {
  nginx: "Nginx 入口代理",
  "grpc-lb": "gRPC 负载均衡",
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
  "mysql-init-user": "MySQL 初始化任务",
  "kafka-init": "Kafka 初始化任务",
  migrate: "数据库迁移任务",
}

export type RuntimeStatusTone = "running" | "partial" | "stopped"

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
export function runtimeStatus(runningReplicas: number, replicas: number): {
  text: string
  tone: RuntimeStatusTone
} {
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
    case "stopped":
    default:
      return "border-rose-200 bg-rose-50 text-rose-700"
  }
}

// healthBadge 统一渲染健康状态标签。
export function healthBadge(ok: boolean, okText = "正常", failText = "异常") {
  return <Badge variant={ok ? "secondary" : "destructive"}>{ok ? okText : failText}</Badge>
}
