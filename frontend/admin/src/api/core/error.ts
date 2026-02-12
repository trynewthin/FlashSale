import axios, { type AxiosError } from "axios"

import { isApiEnvelope, isRecord } from "@/api/core/types"

export class ApiError extends Error {
  code: string
  status: number
  traceId?: string
  details?: unknown

  constructor(params: {
    message: string
    code?: string
    status?: number
    traceId?: string
    details?: unknown
  }) {
    super(params.message)
    this.name = "ApiError"
    this.code = params.code ?? "SYS_UNKNOWN"
    this.status = params.status ?? 0
    this.traceId = params.traceId
    this.details = params.details
  }
}

export function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error
  }

  if (axios.isAxiosError(error)) {
    return normalizeAxiosError(error)
  }

  if (error instanceof Error) {
    return new ApiError({
      message: error.message || "请求失败",
      details: error,
    })
  }

  return new ApiError({
    message: "请求失败",
    details: error,
  })
}

function normalizeAxiosError(error: AxiosError): ApiError {
  const status = error.response?.status ?? 0
  const payload = error.response?.data

  if (isApiEnvelope(payload)) {
    return new ApiError({
      message: payload.message || "请求失败",
      code: payload.code,
      status,
      traceId: payload.trace_id,
      details: payload,
    })
  }

  if (isRecord(payload)) {
    const message = typeof payload.message === "string" ? payload.message : error.message
    const code = typeof payload.code === "string" ? payload.code : undefined
    const traceId = typeof payload.trace_id === "string" ? payload.trace_id : undefined
    return new ApiError({
      message: message || "请求失败",
      code,
      status,
      traceId,
      details: payload,
    })
  }

  return new ApiError({
    message: error.message || "请求失败",
    status,
    details: payload,
  })
}

