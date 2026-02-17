import {
  Activity,
  Boxes,
  Check,
  ChevronLeft,
  ChevronRight,
  FileText,
  KeyRound,
  LayoutDashboard,
  ListChecks,
  Pencil,
  ScrollText,
  Zap,
} from "lucide-react"
import { NavLink } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"
import { cn } from "@/lib/utils"

interface OpsSidebarProps {
  collapsed: boolean
  onToggle: () => void
  accessKey: string
  keySaved: boolean
  editingKey: boolean
  savingKey: boolean
  onAccessKeyChange: (value: string) => void
  onBeginEdit: () => void
  onSaveAccessKey: () => void
}

const navItems = [
  { to: "/overview", label: "概览", icon: LayoutDashboard },
  { to: "/realtime", label: "实时监测", icon: Activity },
  { to: "/perf-test", label: "压力测试", icon: Zap },
  { to: "/containers", label: "容器编排", icon: Boxes },
  { to: "/tasks", label: "任务执行", icon: ListChecks },
  { to: "/job-logs", label: "任务日志", icon: ScrollText },
  { to: "/service-logs", label: "服务日志", icon: FileText },
]

// OpsSidebar 渲染侧边栏导航。
export function OpsSidebar({
  collapsed,
  onToggle,
  accessKey,
  keySaved,
  editingKey,
  savingKey,
  onAccessKeyChange,
  onBeginEdit,
  onSaveAccessKey,
}: OpsSidebarProps) {
  return (
    <aside
      className={cn(
        "flex h-full flex-col border-r bg-card transition-all duration-200",
        collapsed ? "w-14" : "w-56"
      )}
    >
      <div className="flex h-14 items-center justify-between px-2">
        {!collapsed && <span className="px-2 text-sm font-semibold tracking-tight">FlashSale Ops</span>}
        <Button variant="ghost" size="icon-sm" onClick={onToggle}>
          {collapsed ? <ChevronRight className="size-4" /> : <ChevronLeft className="size-4" />}
        </Button>
      </div>
      <Separator />
      <nav className="flex-1 space-y-1 overflow-y-auto p-2">
        {navItems.map((item) => {
          const Icon = item.icon
          return (
            <NavLink key={item.to} to={item.to}>
              {({ isActive }) => (
                <Button
                  variant={isActive ? "secondary" : "ghost"}
                  className={cn("w-full justify-start gap-2", collapsed && "justify-center px-0")}
                >
                  <Icon className="size-4" />
                  {!collapsed && <span>{item.label}</span>}
                </Button>
              )}
            </NavLink>
          )
        })}
      </nav>
      <div className="p-2">
        {!collapsed ? (
          <Card
            size="sm"
            className={cn("gap-2", keySaved && !editingKey && "border-emerald-300 bg-emerald-50 text-emerald-900")}
          >
            <CardHeader className="pb-0">
              <CardTitle className="flex items-center justify-between gap-2 text-sm">
                <span className="inline-flex items-center gap-1">
                  <KeyRound className="size-4" />
                  访问密钥
                </span>
                {!editingKey ? (
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className="size-7"
                    title="输入密钥"
                    onClick={onBeginEdit}
                  >
                    <Pencil className="size-3.5" />
                  </Button>
                ) : null}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              {editingKey ? (
                <div className="flex items-center gap-2">
                  <Input
                    value={accessKey}
                    onChange={(event) => onAccessKeyChange(event.target.value)}
                    placeholder="X-Ops-Key"
                    type="password"
                    className="h-8"
                    disabled={savingKey}
                  />
                  <Button size="icon-sm" onClick={onSaveAccessKey} title="保存密钥" disabled={savingKey}>
                    <Check className="size-4" />
                  </Button>
                </div>
              ) : (
                <div className="inline-flex items-center gap-1 text-xs font-medium text-emerald-700">
                  <Check className="size-3.5" />
                  密钥已保存
                </div>
              )}
            </CardContent>
          </Card>
        ) : null}
      </div>
    </aside>
  )
}
