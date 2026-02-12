// 用户端 API 错误归一：统一抽取错误码并生成面向用户的提示文案。
import { ApiError, toApiError } from "@/api/core/error"

const CODE_TO_MESSAGE: Record<string, string> = {
  AUTH_UNAUTHORIZED: "登录已失效，请重新登录",
  AUTH_FORBIDDEN: "当前无权限执行该操作",
  SYS_BAD_REQUEST: "请求参数不正确，请检查后重试",
  SYS_INTERNAL: "系统繁忙，请稍后再试",
  PRODUCT_NOT_FOUND: "商品不存在或已下架",
  ORDER_NOT_FOUND: "订单不存在",
  ORDER_ALREADY_CLOSED: "订单已关闭",
  ORDER_ALREADY_PAID: "订单已支付",
  SECKILL_ACTIVITY_NOT_PUBLISHED: "活动未发布",
  SECKILL_ACTIVITY_NOT_STARTED: "活动未开始",
  SECKILL_ACTIVITY_ENDED: "活动已结束",
  SECKILL_OUT_OF_STOCK: "库存不足",
  SECKILL_LIMIT_EXCEEDED: "超出限购范围",
  SECKILL_PURCHASE_CONFLICT: "请求处理中，请稍后在订单页查看结果",
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

