import { Outlet, useNavigate } from "react-router-dom"
import {
  Users,
  ShieldCheck,
  FileText,
  UserCog,
  Package,
  ClipboardList,
  Zap,
  LayoutDashboard,
  LogOut,
  ChevronLeft,
  ChevronRight,
} from "lucide-react"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { useAdminSessionState, useAdminLogoutMutation } from "@/hooks/auth/use-admin-session"
import { useAdminPermission } from "@/hooks/auth/use-admin-permission"
import { cn } from "@/lib/utils"

interface NavItem {
  label: string
  path: string
  icon: React.ReactNode
  domain?: string
  requireAll?: boolean
}

const navItems: NavItem[] = [
  { label: "工作台", path: "/", icon: <LayoutDashboard className="size-4" /> },
  { label: "管理员", path: "/admins", icon: <UserCog className="size-4" />, domain: "admin_management" },
  { label: "角色", path: "/roles", icon: <ShieldCheck className="size-4" />, domain: "admin_management" },
  { label: "审计日志", path: "/audit-logs", icon: <FileText className="size-4" />, domain: "admin_management" },
  { label: "用户", path: "/users", icon: <Users className="size-4" />, domain: "user_management" },
  { label: "商品", path: "/products", icon: <Package className="size-4" />, domain: "product_management" },
  { label: "订单", path: "/orders", icon: <ClipboardList className="size-4" />, domain: "order_management" },
  { label: "秒杀活动", path: "/seckill", icon: <Zap className="size-4" />, domain: "seckill_management" },
]

export function AdminLayout() {
  const navigate = useNavigate()
  const { profile } = useAdminSessionState()
  const { can } = useAdminPermission()
  const logoutMutation = useAdminLogoutMutation()
  const [collapsed, setCollapsed] = useState(false)

  const visibleItems = navItems.filter((item) => {
    if (!item.domain) return true
    return can(item.domain)
  })

  const handleLogout = () => {
    logoutMutation.mutate(undefined, {
      onSettled: () => navigate("/login"),
    })
  }

  return (
    <div className="flex h-screen bg-background">
      <aside
        className={cn(
          "flex flex-col border-r bg-card transition-all duration-200",
          collapsed ? "w-16" : "w-56"
        )}
      >
        <div className="flex h-14 items-center justify-between px-4">
          {!collapsed && (
            <span className="text-sm font-semibold tracking-tight">FlashSale 管理</span>
          )}
          <Button variant="ghost" size="icon" className="size-7" onClick={() => setCollapsed(!collapsed)}>
            {collapsed ? <ChevronRight className="size-4" /> : <ChevronLeft className="size-4" />}
          </Button>
        </div>
        <Separator />
        <nav className="flex-1 space-y-1 p-2">
          {visibleItems.map((item) => (
            <Button
              key={item.path}
              variant="ghost"
              className={cn("w-full justify-start gap-2", collapsed && "justify-center px-0")}
              onClick={() => navigate(item.path)}
            >
              {item.icon}
              {!collapsed && <span>{item.label}</span>}
            </Button>
          ))}
        </nav>
        <Separator />
        <div className="p-2 space-y-1">
          {!collapsed && profile && (
            <p className="truncate px-3 text-xs text-muted-foreground">
              {profile.display_name || profile.username}
            </p>
          )}
          <Button
            variant="ghost"
            className={cn("w-full justify-start gap-2 text-destructive", collapsed && "justify-center px-0")}
            onClick={handleLogout}
            disabled={logoutMutation.isPending}
          >
            <LogOut className="size-4" />
            {!collapsed && <span>退出登录</span>}
          </Button>
        </div>
      </aside>
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
