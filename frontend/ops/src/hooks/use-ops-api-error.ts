import { useCallback } from "react"

import { OpsApiError } from "@/api/core/http"
import { useOpsUIStore } from "@/store/ops-ui-store"

function toMessage(error: unknown, fallback: string): string {
  if (error instanceof OpsApiError) {
    return error.message
  }
  if (error instanceof Error) {
    return error.message
  }
  return fallback
}

// useOpsApiError 统一把异常转成全局提示。
export function useOpsApiError() {
  const showNotice = useOpsUIStore((state) => state.showNotice)

  return useCallback(
    (error: unknown, fallback = "请求失败") => {
      showNotice("error", toMessage(error, fallback))
    },
    [showNotice]
  )
}
