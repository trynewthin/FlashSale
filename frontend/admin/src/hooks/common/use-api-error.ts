// 管理端 API 错误归一：按错误码提供稳定提示，便于统一反馈与分支处理。
import { ApiError, toApiError } from "@/api/core/error"

const CODE_TO_MESSAGE: Record<string, string> = {
  AUTH_UNAUTHORIZED: "登录已失效，请重新登录",
  AUTH_FORBIDDEN: "当前账号权限不足",
  SYS_BAD_REQUEST: "请求参数不正确",
  SYS_INTERNAL: "系统繁忙，请稍后再试",
  ADMIN_NOT_FOUND: "管理员不存在",
  ADMIN_USERNAME_ALREADY_EXISTS: "用户名已存在",
  ADMIN_ACCOUNT_DISABLED: "管理员账号已禁用",
  ADMIN_ACCOUNT_LOCKED: "管理员账号已锁定",
  ADMIN_ROLE_NOT_FOUND: "角色不存在",
  ADMIN_ROLE_CODE_ALREADY_EXISTS: "角色编码已存在",
  ADMIN_INVALID_SCOPE: "数据权限范围非法",
  ADMIN_REFRESH_TOKEN_INVALID: "登录状态已过期，请重新登录",
}

export function useApiError() {
  const asApiError = (error: unknown): ApiError => toApiError(error)

  const isCode = (error: unknown, code: string): boolean => {
    const normalized = asApiError(error)
    return normalized.code === code
  }

  const toUserMessage = (error: unknown): string => {
    const normalized = asApiError(error)
    const byCode = CODE_TO_MESSAGE[normalized.code]
    if (byCode) {
      return byCode
    }
    return normalized.message || "请求失败，请稍后重试"
  }

  return {
    asApiError,
    isCode,
    toUserMessage,
  }
}

