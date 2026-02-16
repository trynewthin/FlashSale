import { useEffect } from "react"

import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import { useOpsUIStore } from "@/store/ops-ui-store"

// StatusToast 显示全局成功/失败提示。
export function StatusToast() {
  const notice = useOpsUIStore((state) => state.notice)
  const clearNotice = useOpsUIStore((state) => state.clearNotice)

  useEffect(() => {
    if (!notice) {
      return
    }
    const timer = window.setTimeout(() => {
      clearNotice()
    }, 3200)
    return () => window.clearTimeout(timer)
  }, [notice, clearNotice])

  if (!notice) {
    return null
  }

  return (
    <div className="pointer-events-none fixed right-4 top-4 z-50">
      <div
        className={cn(
          "pointer-events-auto rounded-lg border px-3 py-2 text-sm shadow-md backdrop-blur-sm",
          notice.type === "error"
            ? "border-destructive/30 bg-destructive/10 text-destructive"
            : "border-emerald-500/30 bg-emerald-500/10 text-emerald-700"
        )}
      >
        <div className="flex items-center gap-2">
          <Badge variant={notice.type === "error" ? "destructive" : "secondary"}>
            {notice.type === "error" ? "错误" : "成功"}
          </Badge>
          <span>{notice.message}</span>
        </div>
      </div>
    </div>
  )
}
