import { type Int64, type Int64Like, type PaginatedList, type PaginationParams } from "@/api/core/types"
import { adminApiClient } from "@/api/core/http"

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

export interface ListAdminOrdersQuery extends PaginationParams {
  order_status?: number
  review_status?: number
  user_id?: Int64Like
  order_no?: string
}

export interface ReviewOrderReq {
  approved: boolean
  reason?: string
}

export interface ShipOrderReq {
  tracking_no: string
}

export const adminOrderApi = {
  reviewOrder(orderId: Int64Like, req: ReviewOrderReq): Promise<{ order: OrderView }> {
    return adminApiClient.post<{ order: OrderView }, ReviewOrderReq>(`/orders/${orderId}/review`, req)
  },

  shipOrder(orderId: Int64Like, req: ShipOrderReq): Promise<{ order: OrderView }> {
    return adminApiClient.post<{ order: OrderView }, ShipOrderReq>(`/orders/${orderId}/ship`, req)
  },

  getOrder(orderId: Int64Like): Promise<{ order: OrderView }> {
    return adminApiClient.get<{ order: OrderView }>(`/orders/${orderId}`)
  },

  listOrders(query?: ListAdminOrdersQuery): Promise<PaginatedList<OrderView>> {
    return adminApiClient.get<PaginatedList<OrderView>>("/orders", {
      params: query,
    })
  },
}
