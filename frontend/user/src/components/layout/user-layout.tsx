import { Outlet, useNavigate, useLocation } from "react-router-dom"
import { Package, Zap, ShoppingCart, User, LogOut, ChevronDown } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useLogoutAction, useUserSessionState } from "@/hooks/user/use-auth-hooks"

interface NavItem {
  label: string
  path: string
  icon: React.ReactNode
}

const navItems: NavItem[] = [
  { label: "商品", path: "/products", icon: <Package className="size-4" /> },
  { label: "秒杀", path: "/seckill", icon: <Zap className="size-4" /> },
]

export function UserLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const logout = useLogoutAction()
  const { profile } = useUserSessionState()

  const handleLogout = () => {
    logout()
    navigate("/login")
  }

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="sticky top-0 z-40 border-b bg-card">
        <div className="flex h-14 w-full items-center justify-between px-4">
          {/* Logo */}
          <span
            className="text-sm font-semibold tracking-tight cursor-pointer select-none"
            onClick={() => navigate("/")}
          >
            FlashSale
          </span>

          <div className="flex items-center gap-1">
            {/* 快捷导航 */}
            {navItems.map((item) => (
              <Button
                key={item.path}
                variant={location.pathname.startsWith(item.path) ? "default" : "ghost"}
                size="sm"
                className="gap-1.5"
                onClick={() => navigate(item.path)}
              >
                {item.icon}
                <span className="hidden sm:inline">{item.label}</span>
              </Button>
            ))}

            {/* 用户菜单 */}
            <DropdownMenu>
              <DropdownMenuTrigger
                className="inline-flex items-center gap-1.5 ml-1 h-8 rounded-md px-3 text-sm font-medium hover:bg-accent hover:text-accent-foreground transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                <User className="size-4" />
                <span className="hidden sm:inline max-w-[80px] truncate">
                  {profile?.nickname ?? "我的"}
                </span>
                <ChevronDown className="size-3 opacity-60" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-44">
                <DropdownMenuItem onClick={() => navigate("/orders")}>
                  <ShoppingCart className="size-4 mr-2" />
                  管理订单
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => navigate("/profile")}>
                  <User className="size-4 mr-2" />
                  个人中心
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={handleLogout}
                  className="text-destructive focus:text-destructive"
                >
                  <LogOut className="size-4 mr-2" />
                  退出登录
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </header>
      <main className="mx-auto w-full max-w-4xl flex-1 px-4 py-6">
        <Outlet />
      </main>
    </div>
  )
}
