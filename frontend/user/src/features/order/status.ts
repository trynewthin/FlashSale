import type { OrderView } from "@/api/modules/order"

export const ORDER_STATUS = {
  pending_pay: 10,
  pending_review: 20,
  pending_ship: 30,
  shipped: 40,
  closed: 90,
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
  0: "无需审核",
  1: "待审核",
  2: "审核通过",
  3: "审核拒绝",
  4: "审核超时拒绝",
}

export const SHIPPING_STATUS_LABELS: Record<number, string> = {
  0: "未发货",
  1: "已发货",
  2: "已收货",
}

function toStatusNumber(value: unknown, fallback = -1): number {
  const n = Number(value)
  return Number.isFinite(n) ? n : fallback
}

export function canPayOrder(order: OrderView): boolean {
  return toStatusNumber(order.order_status) === ORDER_STATUS.pending_pay
}

export function canCancelOrder(order: OrderView): boolean {
  return toStatusNumber(order.order_status) === ORDER_STATUS.pending_pay
}

export function canConfirmReceipt(order: OrderView): boolean {
  return toStatusNumber(order.order_status) === ORDER_STATUS.shipped
}
