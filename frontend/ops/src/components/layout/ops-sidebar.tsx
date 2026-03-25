import {
  Activity,
  Boxes,
  ChevronLeft,
  ChevronRight,
  LayoutDashboard,
  ListChecks,
  Zap,
} from "lucide-react"
import { NavLink } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { cn } from "@/lib/utils"

interface OpsSidebarProps {
  collapsed: boolean
  onToggle: () => void
}

const navItems = [
  { to: "/overview", label: "概览", icon: LayoutDashboard },
  { to: "/realtime", label: "实时监测", icon: Activity },
  { to: "/perf-test", label: "压力测试", icon: Zap },
  { to: "/containers", label: "容器编排", icon: Boxes },
  { to: "/tasks", label: "任务中心", icon: ListChecks },
]

export function OpsSidebar({ collapsed, onToggle }: OpsSidebarProps) {
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
    </aside>
  )
}
