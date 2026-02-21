import { useNavigate, useLocation, useOutlet } from "react-router-dom"
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
import { Toaster } from "sonner"
import { AnimatePresence } from "motion/react"
import { PageTransition } from "./page-transition"

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

  const currentOutlet = useOutlet()

  return (
    <div className="flex min-h-screen flex-col bg-background overflow-x-hidden">
      {/* 居中悬浮的 Glassmorphism 药丸岛 Header */}
      <header className="fixed top-4 inset-x-0 w-full z-50 flex justify-center px-4 pointer-events-none">
        <div className="pointer-events-auto flex h-14 items-center justify-between px-5 bg-background/50 dark:bg-zinc-900/50 backdrop-blur-xl supports-backdrop-filter:bg-background/40 border border-border/40 shadow-xl shadow-black/5 dark:shadow-black/20 rounded-full w-full max-w-4xl transition-all duration-300">
          {/* Logo */}
          <span
            className="text-base font-extrabold tracking-tight cursor-pointer select-none bg-clip-text text-transparent bg-linear-to-r from-foreground to-foreground/70"
            onClick={() => navigate("/")}
          >
            FlashSale
          </span>

          <div className="flex items-center gap-1">
            {/* 快捷导航 */}
            {navItems.map((item) => (
              <Button
                key={item.path}
                variant={location.pathname.startsWith(item.path) ? "secondary" : "ghost"}
                size="sm"
                className={`gap-1.5 rounded-full px-3 transition-all ${location.pathname.startsWith(item.path)
                  ? "bg-foreground/10 hover:bg-foreground/15 text-foreground font-semibold"
                  : "text-muted-foreground hover:text-foreground hover:bg-foreground/5"
                  }`}
                onClick={() => navigate(item.path)}
              >
                {/* 隐藏图标以追求更极致干净的文字排版 */}
                <span className="inline">{item.label}</span>
              </Button>
            ))}

            {/* 分隔线 */}
            <div className="w-px h-4 bg-border/60 mx-1" />

            {/* 用户菜单 */}
            <DropdownMenu>
              <DropdownMenuTrigger
                className="inline-flex items-center gap-1.5 h-9 rounded-full px-3 text-sm font-medium hover:bg-foreground/5 text-muted-foreground hover:text-foreground transition-all focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                <div className="size-6 rounded-full bg-linear-to-tr from-red-500/20 to-orange-500/20 flex items-center justify-center border border-border/50">
                  <User className="size-3.5 text-foreground" />
                </div>
                <span className="hidden sm:inline max-w-[80px] truncate">
                  {profile?.nickname ?? "我的"}
                </span>
                <ChevronDown className="size-3 opacity-60 ml-0.5" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48 rounded-2xl p-2 shadow-2xl border-border/50">
                <DropdownMenuItem onClick={() => navigate("/orders")} className="rounded-xl mt-1 cursor-pointer">
                  <ShoppingCart className="size-4 mr-2" />
                  管理订单
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => navigate("/profile")} className="rounded-xl cursor-pointer">
                  <User className="size-4 mr-2" />
                  个人中心
                </DropdownMenuItem>
                <DropdownMenuSeparator className="my-1.5 opacity-50" />
                <DropdownMenuItem
                  onClick={handleLogout}
                  className="rounded-xl text-destructive focus:text-destructive cursor-pointer"
                >
                  <LogOut className="size-4 mr-2" />
                  退出登录
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </header>
      <main className="mx-auto w-full max-w-4xl flex-1 px-4 py-6 pt-20">
        <AnimatePresence mode="wait" initial={false}>
          <PageTransition key={location.pathname} className="min-h-full">
            {currentOutlet}
          </PageTransition>
        </AnimatePresence>
      </main>
      <Toaster richColors position="bottom-center" />
    </div>
  )
}
