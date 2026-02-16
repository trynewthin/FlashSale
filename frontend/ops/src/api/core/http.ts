import { getOpsAccessKey } from "@/api/core/auth"
import type { ApiEnvelope } from "@/api/types"

const OPS_API_BASE_URL = String(import.meta.env.VITE_OPS_API_BASE_URL ?? "").trim()

// OpsApiError 表示 ops 接口调用失败。
export class OpsApiError extends Error {
  code: string
  status: number

  constructor(message: string, code = "ERR", status = 500) {
    super(message)
    this.name = "OpsApiError"
    this.code = code
    this.status = status
  }
}

function normalizePath(path: string): string {
  return path.replace(/^\/+/, "")
}

function joinBase(base: string, path: string): string {
  const cleanPath = normalizePath(path)
  if (!base) {
    return cleanPath
  }
  if (base.endsWith("/")) {
    return `${base}${cleanPath}`
  }
  return `${base}/${cleanPath}`
}

function asEnvelope<T>(payload: unknown): ApiEnvelope<T> {
  if (!payload || typeof payload !== "object") {
    throw new OpsApiError("响应格式非法", "ERR_RESPONSE", 500)
  }
  const envelope = payload as Partial<ApiEnvelope<T>>
  if (typeof envelope.code !== "string") {
    throw new OpsApiError("响应格式非法", "ERR_RESPONSE", 500)
  }
  return {
    code: envelope.code,
    message: typeof envelope.message === "string" ? envelope.message : "",
    data: envelope.data as T,
  }
}

// buildApiUrl 构造 HTTP 与 SSE 共用 URL。
export function buildApiUrl(path: string, params?: URLSearchParams): string {
  const url = joinBase(OPS_API_BASE_URL, path)
  if (!params || Array.from(params.entries()).length === 0) {
    return url
  }
  const query = params.toString()
  return query ? `${url}?${query}` : url
}

// verifyOpsAccessKey 用候选密钥探测鉴权是否通过。
export async function verifyOpsAccessKey(candidateKey: string): Promise<boolean> {
  const accessKey = (candidateKey || "").trim()
  if (!accessKey) {
    return false
  }
  try {
    const response = await fetch(buildApiUrl("api/v1/status"), {
      headers: {
        "X-Ops-Key": accessKey,
      },
    })
    if (!response.ok) {
      return false
    }
    const payload = (await response.json()) as unknown
    const envelope = asEnvelope<Record<string, unknown>>(payload)
    return envelope.code === "OK"
  } catch {
    return false
  }
}

// requestJSON 统一处理 `{ code, message, data }` 响应包装。
export async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  const accessKey = getOpsAccessKey()
  if (accessKey) {
    headers.set("X-Ops-Key", accessKey)
  }
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }

  const response = await fetch(buildApiUrl(path), {
    ...init,
    headers,
  })

  let payload: unknown
  try {
    payload = await response.json()
  } catch {
    throw new OpsApiError("响应解析失败", "ERR_RESPONSE", response.status)
  }

  const envelope = asEnvelope<T>(payload)
  if (envelope.code !== "OK") {
    throw new OpsApiError(envelope.message || "请求失败", envelope.code, response.status)
  }

  return envelope.data
}
