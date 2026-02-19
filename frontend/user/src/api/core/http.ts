import axios, { AxiosHeaders, type AxiosRequestConfig } from "axios"
import JSONbig from "json-bigint"

import { ApiError, toApiError } from "@/api/core/error"
import { userTokenStore } from "@/api/core/token-store"
import { isApiEnvelope } from "@/api/core/types"
import { useUserAuthStore } from "@/stores/auth-store"

const USER_API_BASE_URL = import.meta.env.VITE_USER_API_BASE_URL ?? "/api/v1"
const REQUEST_TIMEOUT_MS = 10000
const jsonParser = JSONbig({ storeAsString: true })

function parseJSONWithInt64Safety(data: unknown): unknown {
  if (typeof data !== "string" || data.trim() === "") {
    return data
  }
  try {
    return jsonParser.parse(data)
  } catch {
    return data
  }
}

// ── 会话失效自动跳登录 ───────────────────────────────────────────
// 这些业务 code 表示当前会话已不可用，需要重新登录：
// - AUTH_UNAUTHORIZED: JWT 无效/过期
// - USER_NOT_FOUND: token 有效但用户已被删除（如 reset 后数据库清空）
const SESSION_INVALID_CODES = new Set([
  "AUTH_UNAUTHORIZED",
  "USER_NOT_FOUND",
])

let isRedirectingToLogin = false

function handleSessionInvalid() {
  if (isRedirectingToLogin) return
  isRedirectingToLogin = true

  // 同时清 localStorage 和 zustand store（确保 AuthGuard 立即响应）
  userTokenStore.clear()
  useUserAuthStore.getState().clearSession()

  // 延迟跳转，让当前渲染周期完成
  setTimeout(() => {
    isRedirectingToLogin = false
    const current = window.location.pathname + window.location.search
    const loginPath = current === "/login" ? "/login" : `/login?redirect=${encodeURIComponent(current)}`
    window.location.href = loginPath
  }, 50)
}

const userHttp = axios.create({
  baseURL: USER_API_BASE_URL,
  timeout: REQUEST_TIMEOUT_MS,
  responseType: "text",
  transformResponse: [(data) => parseJSONWithInt64Safety(data)],
})

userHttp.interceptors.request.use((config) => {
  const token = userTokenStore.getAccessToken()
  if (token) {
    config.headers = AxiosHeaders.from(config.headers)
    config.headers.set("Authorization", `Bearer ${token}`)
  }
  return config
})

userHttp.interceptors.response.use(
  (response) => {
    const payload = response.data
    if (!isApiEnvelope(payload)) {
      return payload
    }
    if (payload.code !== "OK") {
      // 会话失效：JWT 过期、用户被删等
      if (SESSION_INVALID_CODES.has(payload.code)) {
        handleSessionInvalid()
      }
      throw new ApiError({
        message: payload.message || "请求失败",
        code: payload.code,
        status: response.status,
        traceId: payload.trace_id,
        details: payload,
      })
    }
    return payload.data
  },
  (error) => {
    if (axios.isAxiosError(error)) {
      const status = error.response?.status ?? 0
      // HTTP 401 直接判定
      if (status === 401) {
        handleSessionInvalid()
      }
      // 非 2xx 响应也检查 body 中的业务 code（如 404 + USER_NOT_FOUND）
      let payload = error.response?.data
      // responseType:"text" 时 error path 的 data 可能仍是原始 JSON 字符串
      if (typeof payload === "string") {
        try { payload = JSON.parse(payload) } catch { /* ignore */ }
      }
      if (isApiEnvelope(payload) && SESSION_INVALID_CODES.has(payload.code)) {
        handleSessionInvalid()
      }
    }
    return Promise.reject(toApiError(error))
  }
)

async function request<TResponse, TBody = unknown>(
  config: AxiosRequestConfig<TBody>
): Promise<TResponse> {
  return userHttp.request<TResponse, TResponse, TBody>(config)
}

export const userApiClient = {
  get<TResponse>(url: string, config?: AxiosRequestConfig): Promise<TResponse> {
    return request<TResponse>({
      ...config,
      method: "GET",
      url,
    })
  },

  post<TResponse, TBody = unknown>(
    url: string,
    data?: TBody,
    config?: AxiosRequestConfig<TBody>
  ): Promise<TResponse> {
    return request<TResponse, TBody>({
      ...config,
      method: "POST",
      url,
      data,
    })
  },

  patch<TResponse, TBody = unknown>(
    url: string,
    data?: TBody,
    config?: AxiosRequestConfig<TBody>
  ): Promise<TResponse> {
    return request<TResponse, TBody>({
      ...config,
      method: "PATCH",
      url,
      data,
    })
  },

  put<TResponse, TBody = unknown>(
    url: string,
    data?: TBody,
    config?: AxiosRequestConfig<TBody>
  ): Promise<TResponse> {
    return request<TResponse, TBody>({
      ...config,
      method: "PUT",
      url,
      data,
    })
  },

  delete<TResponse>(url: string, config?: AxiosRequestConfig): Promise<TResponse> {
    return request<TResponse>({
      ...config,
      method: "DELETE",
      url,
    })
  },
}
