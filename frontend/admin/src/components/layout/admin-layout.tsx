import { Outlet, useNavigate, useLocation } from "react-router-dom"
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
  ChevronsUpDown,
} from "lucide-react"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useAdminSessionState, useAdminLogoutMutation, useMyAdminProfileQuery } from "@/hooks/auth/use-admin-session"
import { useAdminPermission } from "@/hooks/auth/use-admin-permission"
import { cn } from "@/lib/utils"

interface NavItem {
  label: string
  path: string
  icon: React.ReactNode
  domain?: string
  exact?: boolean
}

const navItems: NavItem[] = [
  { label: "工作台", path: "/", icon: <LayoutDashboard className="size-4" />, exact: true },
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
  const location = useLocation()
  const { profile } = useAdminSessionState()
  const { can } = useAdminPermission()
  const logoutMutation = useAdminLogoutMutation()
  useMyAdminProfileQuery()
  const [collapsed, setCollapsed] = useState(false)

  const visibleItems = navItems.filter((item) => {
    if (!item.domain) return true
    return can(item.domain)
  })

  const isActive = (item: NavItem) => {
    if (item.exact) return location.pathname === item.path
    return location.pathname === item.path || location.pathname.startsWith(item.path + "/")
  }

  const handleLogout = () => {
    logoutMutation.mutate(undefined, {
      onSettled: () => navigate("/login"),
    })
  }

  // 取显示名首字母作为头像
  const displayName = profile?.display_name || profile?.username || "?"
  const avatarLetter = displayName.charAt(0).toUpperCase()

  return (
    <div className="flex h-screen bg-background">
      <aside
        className={cn(
          "flex flex-col bg-card transition-all duration-200",
          collapsed ? "w-14" : "w-56"
        )}
      >
        {/* Logo / Title */}
        <div className={cn(
          "flex h-14 items-center px-3",
          collapsed ? "justify-center" : "justify-between"
        )}>
          {!collapsed && (
            <span className="text-sm font-semibold tracking-tight">FlashSale 管理</span>
          )}
          <Button
            variant="ghost"
            size="icon"
            className="size-7 shrink-0 text-muted-foreground hover:text-foreground"
            onClick={() => setCollapsed(!collapsed)}
          >
            {collapsed ? <ChevronRight className="size-4" /> : <ChevronLeft className="size-4" />}
          </Button>
        </div>

        {/* Nav */}
        <nav className="flex-1 space-y-0.5 px-2 py-1">
          {visibleItems.map((item) => {
            const active = isActive(item)
            return (
              <Button
                key={item.path}
                variant="ghost"
                className={cn(
                  "w-full h-9 justify-start gap-2.5 rounded-md text-sm font-normal transition-colors",
                  collapsed && "justify-center px-0",
                  active
                    ? "bg-accent text-accent-foreground font-medium"
                    : "text-muted-foreground hover:text-foreground hover:bg-accent/50"
                )}
                onClick={() => navigate(item.path)}
              >
                {item.icon}
                {!collapsed && <span>{item.label}</span>}
              </Button>
            )
          })}
        </nav>

        {/* User card footer */}
        <div className="p-2">
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <button
                  className={cn(
                    "flex w-full items-center gap-2.5 rounded-md px-2 py-2",
                    "hover:bg-accent/60 transition-colors outline-none",
                    collapsed && "justify-center px-0"
                  )}
                />
              }
            >
              {/* Avatar */}
              <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-primary text-[11px] font-semibold text-primary-foreground">
                {avatarLetter}
              </span>

              {!collapsed && (
                <>
                  <span className="flex min-w-0 flex-1 flex-col text-start">
                    <span className="truncate text-xs font-medium leading-tight">
                      {displayName}
                    </span>
                    {profile?.username && profile.display_name && (
                      <span className="truncate text-[11px] leading-tight text-muted-foreground">
                        {profile.username}
                      </span>
                    )}
                  </span>
                  <ChevronsUpDown className="size-3.5 shrink-0 text-muted-foreground" />
                </>
              )}
            </DropdownMenuTrigger>

            <DropdownMenuContent
              side="top"
              sideOffset={6}
              align="start"
              className="min-w-48"
            >
              <div className="px-2 py-1.5 flex flex-col gap-0.5">
                <span className="text-xs font-medium text-foreground">{displayName}</span>
                {profile?.username && (
                  <span className="text-[11px] text-muted-foreground">{profile.username}</span>
                )}
              </div>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                variant="destructive"
                onClick={handleLogout}
                disabled={logoutMutation.isPending}
              >
                <LogOut className="size-3.5" />
                {logoutMutation.isPending ? "退出中…" : "退出登录"}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </aside>

      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
