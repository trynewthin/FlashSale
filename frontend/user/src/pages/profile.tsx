import { useState, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { motion } from "motion/react"
import { toast } from "sonner"
import { Pencil, Trash2, User, Phone, Shield, LogOut, ChevronRight } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"

import { useUserSessionState, useUserProfileQuery, useLogoutAction } from "@/hooks/user/use-auth-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

import { NicknameDialog, DeleteAccountDialog } from "@/components/profile"

export function ProfilePage() {
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()
  const { profile: sessionProfile } = useUserSessionState()
  const logout = useLogoutAction()

  const profileQuery = useUserProfileQuery()
  const user = profileQuery.data

  const [nicknameOpen, setNicknameOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  // 处理全局错误提示
  useEffect(() => {
    if (profileQuery.isError) {
      toast.error(toUserMessage(profileQuery.error))
    }
  }, [profileQuery.isError, profileQuery.error, toUserMessage])

  return (
    <div className="relative max-w-2xl mx-auto space-y-8 pt-6 pb-12">
      {/* 动态背景光晕 */}
      <div className="absolute top-0 right-0 -translate-y-1/2 translate-x-1/3 w-[500px] h-[500px] bg-blue-500/10 dark:bg-blue-500/5 blur-[100px] rounded-full pointer-events-none" />
      <div className="absolute top-1/2 left-0 -translate-y-1/2 -translate-x-1/2 w-[400px] h-[400px] bg-purple-500/10 dark:bg-purple-500/5 blur-[100px] rounded-full pointer-events-none" />

      {/* 页面标题 */}
      <motion.div
        initial={{ opacity: 0, y: -10 }}
        animate={{ opacity: 1, y: 0 }}
        className="relative z-10 flex items-center justify-between"
      >
        <h1 className="text-3xl font-bolder tracking-tight">个人中心</h1>
      </motion.div>

      {/* 用户基础信息卡片 (Bento Style) */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1, duration: 0.4, ease: "easeOut" }}
        className="relative z-10 flex flex-col sm:flex-row items-center sm:items-stretch gap-6 p-6 sm:p-8 rounded-[2rem] bg-card border border-border/50 shadow-xl shadow-black/5 overflow-hidden group"
      >
        <div className="absolute top-0 right-0 w-32 h-32 bg-linear-to-bl from-foreground/5 to-transparent rounded-full blur-2xl -mr-10 -mt-10 pointer-events-none" />

        {/* 聚合头像 */}
        <div className="shrink-0 relative">
          <div className="size-24 rounded-full bg-linear-to-tr from-blue-500 to-indigo-500 flex items-center justify-center border-4 border-background shadow-lg ring-1 ring-border/50 group-hover:scale-105 transition-transform duration-500">
            <User className="size-10 text-white" />
          </div>
          <div className="absolute bottom-1 right-1 size-5 bg-emerald-500 border-2 border-background rounded-full" />
        </div>

        {/* 字段信息 */}
        <div className="flex-1 flex flex-col justify-center text-center sm:text-left space-y-4 w-full">
          {profileQuery.isLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-8 w-40 sm:mx-0 mx-auto" />
              <Skeleton className="h-4 w-24 sm:mx-0 mx-auto" />
            </div>
          ) : user ? (
            <>
              <div>
                <h2 className="text-2xl font-bold tracking-tight text-foreground flex items-center justify-center sm:justify-start gap-2">
                  {user.nickname || "闪购会员"}
                </h2>
                <span className="text-sm text-muted-foreground font-mono mt-1 block">ID: {user.user_id}</span>
              </div>

              <div className="flex flex-wrap gap-3 justify-center sm:justify-start">
                <div className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-muted/50 border border-border/50 text-xs font-medium text-muted-foreground">
                  <Phone className="size-3.5" />
                  {user.phone.replace(/(\d{3})\d{4}(\d{4})/, '$1****$2')}
                </div>
                <div className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-blue-50 dark:bg-blue-500/10 border border-blue-100 dark:border-blue-500/20 text-xs font-medium text-blue-600 dark:text-blue-400">
                  <Shield className="size-3.5" />
                  已实名
                </div>
              </div>
            </>
          ) : (
            <div className="text-muted-foreground py-4">数据加载失败</div>
          )}
        </div>

        <div className="absolute top-6 right-6 hidden sm:block">
          <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-foreground rounded-full h-10 w-10 relative overflow-hidden group/edit" onClick={() => setNicknameOpen(true)}>
            <span className="absolute inset-0 bg-foreground/10 translate-y-full group-hover/edit:translate-y-0 transition-transform duration-300 rounded-full" />
            <Pencil className="size-4 relative z-10" />
          </Button>
        </div>
      </motion.div>

      {/* 账号操作台 */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.2, duration: 0.4, ease: "easeOut" }}
        className="relative z-10 rounded-[2rem] bg-card border border-border/50 shadow-xl shadow-black/5 p-2 sm:p-4"
      >
        <div className="px-4 py-3 text-sm font-semibold text-muted-foreground flex items-center gap-2">
          账号与安全
        </div>

        <div className="space-y-1 mt-2">
          <button
            onClick={() => setNicknameOpen(true)}
            className="w-full flex items-center justify-between p-4 rounded-2xl hover:bg-accent hover:text-accent-foreground transition-all group"
          >
            <div className="flex items-center gap-3">
              <div className="size-10 rounded-full bg-blue-50 dark:bg-blue-500/10 flex items-center justify-center shrink-0 text-blue-500">
                <Pencil className="size-4" />
              </div>
              <div className="text-left">
                <div className="font-semibold text-sm">修改昵称</div>
                <div className="text-xs text-muted-foreground mt-0.5">定制你在平台中的专属名称</div>
              </div>
            </div>
            <ChevronRight className="size-4 text-muted-foreground opacity-50 group-hover:opacity-100 group-hover:translate-x-1 transition-all" />
          </button>

          <Separator className="mx-4 my-1 w-auto opacity-50" />

          <button
            onClick={() => setDeleteOpen(true)}
            className="w-full flex items-center justify-between p-4 rounded-2xl hover:bg-red-50 dark:hover:bg-red-500/10 text-muted-foreground hover:text-red-500 transition-all group"
          >
            <div className="flex items-center gap-3">
              <div className="size-10 rounded-full bg-muted group-hover:bg-red-100 dark:group-hover:bg-red-500/20 flex items-center justify-center shrink-0 group-hover:text-red-500 transition-colors">
                <Trash2 className="size-4" />
              </div>
              <div className="text-left">
                <div className="font-semibold text-sm">注销账号</div>
                <div className="text-xs opacity-70 mt-0.5">永久删除所有信息与交易记录</div>
              </div>
            </div>
            <ChevronRight className="size-4 opacity-50 group-hover:opacity-100 group-hover:translate-x-1 transition-all" />
          </button>

          <Separator className="mx-4 my-1 w-auto opacity-50" />

          <button
            onClick={() => {
              logout()
              navigate("/login")
            }}
            className="w-full flex items-center justify-between p-4 rounded-2xl hover:bg-accent hover:text-accent-foreground transition-all group"
          >
            <div className="flex items-center gap-3">
              <div className="size-10 rounded-full bg-muted flex items-center justify-center shrink-0">
                <LogOut className="size-4" />
              </div>
              <div className="text-left">
                <div className="font-semibold text-sm">退出登录</div>
                <div className="text-xs text-muted-foreground mt-0.5">切断当前会话</div>
              </div>
            </div>
          </button>
        </div>
      </motion.div>

      {/* 弹窗区 */}
      {nicknameOpen && (
        <NicknameDialog currentNickname={user?.nickname ?? sessionProfile?.nickname ?? ""} onClose={() => setNicknameOpen(false)} />
      )}

      {deleteOpen && (
        <DeleteAccountDialog onClose={() => setDeleteOpen(false)} onDeleted={() => { logout(); navigate("/login") }} />
      )}
    </div>
  )
}

