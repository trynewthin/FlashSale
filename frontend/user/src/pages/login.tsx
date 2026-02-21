import { useState, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { motion, AnimatePresence } from "motion/react"
import { toast } from "sonner"
import { ArrowRight } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useLoginMutation, useRegisterMutation, useUserSessionState } from "@/hooks/user/use-auth-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

export function LoginPage() {
  const navigate = useNavigate()
  const { isAuthed } = useUserSessionState()
  const loginMutation = useLoginMutation()
  const registerMutation = useRegisterMutation()
  const { toUserMessage } = useApiError()

  const [isLogin, setIsLogin] = useState(true)
  const [phone, setPhone] = useState("")
  const [password, setPassword] = useState("")
  const [nickname, setNickname] = useState("") // Only for register

  useEffect(() => {
    if (isAuthed) {
      navigate("/", { replace: true })
    }
  }, [isAuthed, navigate])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    if (isLogin) {
      loginMutation.mutate(
        { phone, password },
        {
          onSuccess: () => {
            toast.success("登录成功，欢迎回来！")
            navigate("/", { replace: true })
          },
          onError: (error) => toast.error(toUserMessage(error))
        }
      )
    } else {
      registerMutation.mutate(
        { phone, password, nickname },
        {
          onSuccess: () => {
            toast.success("注册成功！即将进入商场")
            navigate("/", { replace: true })
          },
          onError: (error) => toast.error(toUserMessage(error))
        }
      )
    }
  }

  // 如果已经登录，在跳转发生前不渲染内容避免闪烁
  if (isAuthed) return null

  return (
    <div className="min-h-screen relative flex items-center justify-center p-4 overflow-hidden bg-background">
      {/* 动态唯美背景光晕 */}
      <div className="absolute inset-0 z-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-[20%] -right-[10%] w-[60vw] h-[60vw] max-w-[800px] max-h-[800px] bg-red-500/15 dark:bg-red-500/10 blur-[120px] rounded-full mix-blend-screen dark:mix-blend-lighten animate-pulse duration-1000" />
        <div className="absolute -bottom-[20%] -left-[10%] w-[60vw] h-[60vw] max-w-[800px] max-h-[800px] bg-orange-500/15 dark:bg-orange-500/10 blur-[120px] rounded-full mix-blend-screen dark:mix-blend-lighten animate-pulse duration-1000 delay-500" />
        <div className="absolute top-[20%] left-[20%] w-[30vw] h-[30vw] max-w-[400px] max-h-[400px] bg-purple-500/10 dark:bg-purple-500/5 blur-[100px] rounded-full mix-blend-screen dark:mix-blend-lighten" />
      </div>

      <motion.div
        initial={{ opacity: 0, scale: 0.95, y: 20 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
        className="relative z-10 w-full max-w-[420px]"
      >
        {/* Logo 区 */}
        <div className="flex flex-col items-center justify-center mb-8 space-y-3 text-center">
          <h1 className="text-3xl font-black tracking-tighter bg-clip-text text-transparent bg-linear-to-r from-foreground to-foreground/70">
            FlashSale
          </h1>
          <p className="text-sm font-medium text-muted-foreground w-4/5">
            极致低价天天抢，汇聚全网硬通货
          </p>
        </div>

        {/* 交互面板 (Glassmorphism) */}
        <div className="bg-card/60 backdrop-blur-2xl supports-backdrop-filter:bg-card/40 border border-border/50 shadow-2xl shadow-black/5 dark:shadow-black/20 rounded-[2rem] p-6 sm:p-8 overflow-hidden relative">
          <div className="absolute top-0 inset-x-0 h-px bg-linear-to-r from-transparent via-red-500/50 to-transparent" />

          {/* 登录/注册 切换拨片 */}
          <div className="flex bg-muted/50 p-1 rounded-2xl mb-8 relative border border-border/50">
            {/* 选中态滑块背景 */}
            <div
              className="absolute top-1 bottom-1 w-[calc(50%-4px)] bg-background rounded-xl shadow-sm border border-border/50 transition-transform duration-300 ease-in-out"
              style={{ transform: `translateX(${isLogin ? '0' : '100%'})` }}
            />

            <button
              type="button"
              className={`relative flex-1 flex items-center justify-center gap-2 h-10 text-sm font-bold z-10 transition-colors ${isLogin ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'}`}
              onClick={() => setIsLogin(true)}
            >
              安全登录
            </button>
            <button
              type="button"
              className={`relative flex-1 flex items-center justify-center gap-2 h-10 text-sm font-bold z-10 transition-colors ${!isLogin ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'}`}
              onClick={() => setIsLogin(false)}
            >
              极速注册
            </button>
          </div>

          {/* 表单体 */}
          <form onSubmit={handleSubmit} className="space-y-4">
            <AnimatePresence mode="popLayout" initial={false}>
              <motion.div
                key="phone"
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                className="relative group"
              >
                <Input
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  placeholder="手机号码 (例 13800138000)"
                  className="px-4 h-12 rounded-xl bg-background/50 focus-visible:bg-background border-border/50 border-2"
                  required
                  autoFocus
                />
              </motion.div>

              <motion.div
                key="password"
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: 0.05 }}
                className="relative group"
              >
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="账号密码"
                  className="px-4 h-12 rounded-xl bg-background/50 focus-visible:bg-background border-border/50 border-2"
                  required
                />
              </motion.div>

              {!isLogin && (
                <motion.div
                  key="nickname"
                  initial={{ opacity: 0, height: 0, scale: 0.9 }}
                  animate={{ opacity: 1, height: 'auto', scale: 1 }}
                  exit={{ opacity: 0, height: 0, scale: 0.9 }}
                  transition={{ duration: 0.2 }}
                  className="relative group"
                >
                  <Input
                    value={nickname}
                    onChange={(e) => setNickname(e.target.value)}
                    placeholder="您的昵称 (选填)"
                    className="px-4 h-12 rounded-xl bg-background/50 focus-visible:bg-background border-border/50 border-2"
                  />
                </motion.div>
              )}
            </AnimatePresence>

            <div className="pt-4">
              <Button
                type="submit"
                className="w-full h-12 rounded-xl text-base font-bold shadow-lg transition-transform active:scale-95 group/submit"
                disabled={loginMutation.isPending || registerMutation.isPending}
              >
                {isLogin ? (
                  loginMutation.isPending ? "登入中..." : "立即进入"
                ) : (
                  registerMutation.isPending ? "注册中..." : "注册并进入"
                )}
                <ArrowRight className="absolute right-4 size-5 opacity-0 -translate-x-4 group-hover/submit:opacity-100 group-hover/submit:translate-x-0 transition-all duration-300 pointer-events-none" />
              </Button>
            </div>
          </form>

          {/* 许可协议 */}
          <div className="mt-6 text-center text-xs text-muted-foreground font-medium">
            继续操作即代表您同意
            <a href="#" className="font-bold text-foreground hover:underline mx-1">服务条款</a>和
            <a href="#" className="font-bold text-foreground hover:underline mx-1">隐私政策</a>
          </div>
        </div>
      </motion.div>
    </div>
  )
}
