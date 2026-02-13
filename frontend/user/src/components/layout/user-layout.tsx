import { Outlet, useNavigate, useLocation } from "react-router-dom"
import { Package, ShoppingCart, Zap, User, LogOut } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { useLogoutAction } from "@/hooks/user/use-auth-hooks"

interface NavItem {
  label: string
  path: string
  icon: React.ReactNode
}

const navItems: NavItem[] = [
  { label: "商品", path: "/products", icon: <Package className="size-4" /> },
  { label: "秒杀", path: "/seckill", icon: <Zap className="size-4" /> },
  { label: "订单", path: "/orders", icon: <ShoppingCart className="size-4" /> },
  { label: "我的", path: "/profile", icon: <User className="size-4" /> },
]

export function UserLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const logout = useLogoutAction()

  const handleLogout = () => {
    logout()
    navigate("/login")
  }

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="sticky top-0 z-40 border-b bg-card">
        <div className="mx-auto flex h-14 max-w-4xl items-center justify-between px-4">
          <span className="text-sm font-semibold tracking-tight cursor-pointer" onClick={() => navigate("/products")}>
            FlashSale
          </span>
          <nav className="flex items-center gap-1">
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
            <Separator orientation="vertical" className="mx-1 h-6" />
            <Button variant="ghost" size="icon" className="size-8 text-destructive" onClick={handleLogout}>
              <LogOut className="size-4" />
            </Button>
          </nav>
        </div>
      </header>
      <main className="mx-auto w-full max-w-4xl flex-1 px-4 py-6">
        <Outlet />
      </main>
    </div>
  )
}
