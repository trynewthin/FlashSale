import { type Int64, type Int64Like, type PaginatedList, type PaginationParams } from "@/api/core/types"
import { userApiClient } from "@/api/core/http"
import { toInt64String } from "@/api/core/int64"

export const OrderSource = {
  Normal: 1,
  Seckill: 2,
} as const

export interface OrderView {
  order_id: Int64
  order_no: string
  user_id: Int64
  order_source: number
  product_id: Int64
  sku_code: string
  product_name: string
  main_image: string
  unit_price_cent: number
  quantity: number
  total_amount_cent: number
  order_status: number
  payment_status: number
  review_status: number
  shipping_status: number
  refund_status: number
  review_mode: number
  paid_at_unix: number
  pay_channel: string
  pay_reference: string
  receiver_name: string
  receiver_phone: string
  receiver_address: string
  buyer_remark: string
  review_due_at_unix: number
  reviewed_at_unix: number
  reviewed_by: Int64
  review_reason: string
  shipped_at_unix: number
  shipped_by: Int64
  tracking_no: string
  refund_due_at_unix: number
  refunded_at_unix: number
  closed_at_unix: number
  close_reason: string
  created_at_unix: number
  updated_at_unix: number
  seckill_activity_id: Int64
  seckill_activity_item_id: Int64
}

export interface CreateOrderReq {
  product_id: Int64Like
  order_source?: number
}

export interface ConfirmPaymentAndInfoReq {
  pay_channel: string
  pay_reference: string
  receiver_name: string
  receiver_phone: string
  receiver_address: string
  buyer_remark?: string
}

export interface CancelOrderReq {
  reason?: string
}

export interface CreateOrderResp {
  order: OrderView
}

export interface ConfirmPaymentAndInfoResp {
  order: OrderView
}

export interface CancelOrderResp {
  order: OrderView
}

export interface ConfirmReceiptResp {
  order: OrderView
}

export interface GetOrderResp {
  order: OrderView
}

export interface ListOrdersQuery extends PaginationParams {
  order_status?: number
  order_no?: string
}

export type ListOrdersResp = PaginatedList<OrderView>

export const userOrderApi = {
  createOrder(req: CreateOrderReq): Promise<CreateOrderResp> {
    return userApiClient.post<CreateOrderResp, { product_id: string; order_source?: number }>("/orders", {
      ...req,
      product_id: toInt64String(req.product_id, "product_id"),
    })
  },

  confirmPaymentAndInfo(orderId: Int64Like, req: ConfirmPaymentAndInfoReq): Promise<ConfirmPaymentAndInfoResp> {
    return userApiClient.post<ConfirmPaymentAndInfoResp, ConfirmPaymentAndInfoReq>(
      `/orders/${orderId}/pay-confirm`,
      req
    )
  },

  cancelOrder(orderId: Int64Like, req: CancelOrderReq): Promise<CancelOrderResp> {
    return userApiClient.post<CancelOrderResp, CancelOrderReq>(`/orders/${orderId}/cancel`, req)
  },

  confirmReceipt(orderId: Int64Like): Promise<ConfirmReceiptResp> {
    return userApiClient.post<ConfirmReceiptResp>(`/orders/${orderId}/confirm-receipt`)
  },

  getOrder(orderId: Int64Like): Promise<GetOrderResp> {
    return userApiClient.get<GetOrderResp>(`/orders/${orderId}`)
  },

  listOrders(query?: ListOrdersQuery): Promise<ListOrdersResp> {
    return userApiClient.get<ListOrdersResp>("/orders", {
      params: query,
    })
  },
}
