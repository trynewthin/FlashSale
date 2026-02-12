import axios, { AxiosHeaders, type AxiosRequestConfig } from "axios"
import JSONbig from "json-bigint"

import { ApiError, toApiError } from "@/api/core/error"
import { adminTokenStore } from "@/api/core/token-store"
import { isApiEnvelope } from "@/api/core/types"

const ADMIN_API_BASE_URL = import.meta.env.VITE_ADMIN_API_BASE_URL ?? "/api/v1/admin"
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

const adminHttp = axios.create({
  baseURL: ADMIN_API_BASE_URL,
  timeout: REQUEST_TIMEOUT_MS,
  responseType: "text",
  transformResponse: [(data) => parseJSONWithInt64Safety(data)],
})

adminHttp.interceptors.request.use((config) => {
  const token = adminTokenStore.getAccessToken()
  if (token) {
    config.headers = AxiosHeaders.from(config.headers)
    config.headers.set("Authorization", `Bearer ${token}`)
  }
  return config
})

adminHttp.interceptors.response.use(
  (response) => {
    const payload = response.data
    if (!isApiEnvelope(payload)) {
      return payload
    }
    if (payload.code !== "OK") {
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
    return Promise.reject(toApiError(error))
  }
)

async function request<TResponse, TBody = unknown>(
  config: AxiosRequestConfig<TBody>
): Promise<TResponse> {
  return adminHttp.request<TResponse, TResponse, TBody>(config)
}

export const adminApiClient = {
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
