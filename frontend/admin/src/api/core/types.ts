export interface ApiEnvelope<T> {
  code: string
  message: string
  data: T
  trace_id?: string
}

export type Int64 = string
export type Int64Like = Int64 | number

export interface PaginationParams {
  page?: number
  page_size?: number
}

export interface PaginatedList<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface PaginatedItems<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export type Dict = Record<string, unknown>

export function isRecord(value: unknown): value is Dict {
  return typeof value === "object" && value !== null
}

export function isApiEnvelope<T = unknown>(value: unknown): value is ApiEnvelope<T> {
  if (!isRecord(value)) {
    return false
  }
  return "code" in value && "message" in value && "data" in value
}
