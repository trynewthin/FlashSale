import type { OrderView } from "@/api/modules/order"

export const ORDER_STATUS = {
  pending_pay: 10,
  pending_review: 20,
  pending_ship: 30,
  shipped: 40,
  closed: 90,
} as const

export const REVIEW_STATUS = {
  none: 0,
  manual_pending: 1,
  passed: 2,
  rejected: 3,
  timeout_reject: 4,
} as const

export const SHIPPING_STATUS = {
  not_shipped: 0,
  shipped: 1,
  received: 2,
} as const

export const ORDER_STATUS_LABELS: Record<number, string> = {
  [ORDER_STATUS.pending_pay]: "待支付",
  [ORDER_STATUS.pending_review]: "待审核",
  [ORDER_STATUS.pending_ship]: "待发货",
  [ORDER_STATUS.shipped]: "待收货",
  [ORDER_STATUS.closed]: "已关闭",
}

export const PAYMENT_STATUS_LABELS: Record<number, string> = {
  0: "未支付",
  1: "已支付",
  2: "已退款",
}

export const REVIEW_STATUS_LABELS: Record<number, string> = {
  [REVIEW_STATUS.none]: "无需审核",
  [REVIEW_STATUS.manual_pending]: "待审核",
  [REVIEW_STATUS.passed]: "通过",
  [REVIEW_STATUS.rejected]: "拒绝",
  [REVIEW_STATUS.timeout_reject]: "超时拒绝",
}

export const SHIPPING_STATUS_LABELS: Record<number, string> = {
  [SHIPPING_STATUS.not_shipped]: "未发货",
  [SHIPPING_STATUS.shipped]: "已发货",
  [SHIPPING_STATUS.received]: "已收货",
}

export const REVIEW_FILTER_OPTIONS: Record<number, string> = {
  [REVIEW_STATUS.manual_pending]: "待审核",
  [REVIEW_STATUS.passed]: "通过",
  [REVIEW_STATUS.rejected]: "拒绝",
  [REVIEW_STATUS.timeout_reject]: "超时拒绝",
}

function toStatusNumber(value: unknown, fallback = 0): number {
  const n = Number(value)
  return Number.isFinite(n) ? n : fallback
}

export function canReviewOrder(order: OrderView): boolean {
  return toStatusNumber(order.order_status) === ORDER_STATUS.pending_review
}

export function canShipOrder(order: OrderView): boolean {
  return toStatusNumber(order.order_status) === ORDER_STATUS.pending_ship
}
