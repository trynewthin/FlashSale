import { ApiError } from "@/api/core/error"
import { type Int64Like } from "@/api/core/types"

const INT64_MIN = -9223372036854775808n
const INT64_MAX = 9223372036854775807n

export function toInt64String(value: Int64Like, fieldName: string): string {
  const normalized = typeof value === "string" ? value.trim() : value
  if (normalized === "") {
    throw new ApiError({
      code: "SYS_BAD_REQUEST",
      status: 400,
      message: `${fieldName} 不能为空`,
    })
  }
  if (typeof normalized === "number") {
    if (!Number.isInteger(normalized) || !Number.isSafeInteger(normalized)) {
      throw new ApiError({
        code: "SYS_BAD_REQUEST",
        status: 400,
        message: `${fieldName} 非安全整数，请使用字符串传递`,
      })
    }
    return normalized.toString()
  }
  if (!/^-?\d+$/.test(normalized)) {
    throw new ApiError({
      code: "SYS_BAD_REQUEST",
      status: 400,
      message: `${fieldName} 必须为整数`,
    })
  }
  let parsed: bigint
  try {
    parsed = BigInt(normalized)
  } catch {
    throw new ApiError({
      code: "SYS_BAD_REQUEST",
      status: 400,
      message: `${fieldName} 非法`,
    })
  }
  if (parsed < INT64_MIN || parsed > INT64_MAX) {
    throw new ApiError({
      code: "SYS_BAD_REQUEST",
      status: 400,
      message: `${fieldName} 超出 int64 范围`,
    })
  }
  return parsed.toString()
}
