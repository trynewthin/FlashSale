import {
  Activity,
  Boxes,
  Check,
  ChevronLeft,
  ChevronRight,
  KeyRound,
  LayoutDashboard,
  ListChecks,
  Pencil,
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
  { to: "/tasks", label: "任务中心", icon: ListChecks },
]

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
        "relative flex h-full flex-col border-r border-border/50 bg-background/60 backdrop-blur-xl shadow-[4px_0_24px_-12px_rgba(0,0,0,0.1)] transition-[width] duration-300 ease-in-out z-20",
        collapsed ? "w-[68px]" : "w-64"
      )}
    >
      <div className="flex h-14 shrink-0 items-center justify-between px-3">
        {!collapsed && (
          <span className="px-2 text-[15px] font-bold tracking-tight bg-linear-to-br from-foreground to-foreground/70 bg-clip-text text-transparent truncate">
            FlashSale Ops
          </span>
        )}
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={onToggle}
          className={cn(
            "rounded-lg hover:bg-secondary/80 transition-transform active:scale-95 text-muted-foreground hover:text-foreground",
            collapsed && "mx-auto"
          )}
        >
          {collapsed ? <ChevronRight className="size-4" /> : <ChevronLeft className="size-4" />}
        </Button>
      </div>

      <Separator className="opacity-50" />

      <nav className="flex-1 space-y-1.5 overflow-y-auto p-3">
        {navItems.map((item) => {
          const Icon = item.icon
          return (
            <NavLink key={item.to} to={item.to}>
              {({ isActive }) => (
                <Button
                  variant="ghost"
                  className={cn(
                    "relative w-full justify-start gap-3 h-10 transition-all duration-200 overflow-hidden group",
                    collapsed ? "justify-center px-0 h-11" : "px-3 rounded-lg",
                    isActive
                      ? "bg-primary/10 text-primary font-medium hover:bg-primary/15"
                      : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
                  )}
                >
                  {/* Vertical highlight strip */}
                  <div
                    className={cn(
                      "absolute left-0 top-1/2 -translate-y-1/2 w-1 bg-primary rounded-r-full transition-all duration-300",
                      isActive ? "h-5 opacity-100" : "h-0 opacity-0"
                    )}
                  />

                  <Icon
                    className={cn(
                      "shrink-0 transition-colors duration-200",
                      collapsed ? "size-5" : "size-[18px]",
                      isActive ? "text-primary" : "text-muted-foreground group-hover:text-foreground"
                    )}
                  />
                  {!collapsed && <span className="truncate">{item.label}</span>}
                </Button>
              )}
            </NavLink>
          )
        })}
      </nav>

      {/* Access Key Card (Floating Island) */}
      <div className="shrink-0 p-3 pb-4">
        {!collapsed ? (
          <Card
            size="sm"
            className={cn(
              "relative overflow-hidden transition-all duration-300 shadow-sm border border-border/50",
              keySaved && !editingKey
                ? "bg-linear-to-br from-emerald-50/80 to-emerald-100/50 dark:from-emerald-950/30 dark:to-emerald-900/10 border-emerald-200/50 dark:border-emerald-800/50"
                : "bg-background/80 backdrop-blur-sm"
            )}
          >
            <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
              <CardTitle className="text-xs font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-1.5">
                <KeyRound className="size-3.5" />
                访问密钥
              </CardTitle>
              {!editingKey && (
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="size-6 rounded-full hover:bg-background/80"
                  onClick={onBeginEdit}
                  title="修改密钥"
                >
                  <Pencil className="size-3" />
                </Button>
              )}
            </CardHeader>
            <CardContent>
              {editingKey ? (
                <div className="flex items-center gap-2 animate-in fade-in slide-in-from-bottom-1 duration-200">
                  <Input
                    value={accessKey}
                    onChange={(event) => onAccessKeyChange(event.target.value)}
                    placeholder="X-Ops-Key..."
                    type="password"
                    className="h-8 text-xs font-mono bg-background shadow-inner"
                    disabled={savingKey}
                  />
                  <Button
                    size="icon-sm"
                    className="h-8 w-8 shrink-0 hover:scale-105 active:scale-95 transition-all shadow-sm rounded-lg"
                    onClick={onSaveAccessKey}
                    title="保存密钥"
                    disabled={savingKey}
                  >
                    <Check className="size-4" />
                  </Button>
                </div>
              ) : (
                <div className="inline-flex items-center gap-1.5 text-xs font-medium text-emerald-600 dark:text-emerald-400 animate-in fade-in duration-300">
                  <div className="flex size-4 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-900/50">
                    <Check className="size-2.5" />
                  </div>
                  已保存并生效
                </div>
              )}
            </CardContent>
          </Card>
        ) : (
          <Button
            variant="ghost"
            className={cn(
              "w-full h-11 flex items-center justify-center rounded-xl transition-all px-0",
              keySaved ? "text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50 dark:hover:bg-emerald-950/50" : "text-muted-foreground"
            )}
            onClick={() => {
              onToggle()
            }}
            title="访问密钥"
          >
            <KeyRound className="size-5" />
          </Button>
        )}
      </div>
    </aside>
  )
}
