import { type Int64, type Int64Like, type PaginatedList, type PaginationParams } from "@/api/core/types"
import { adminApiClient } from "@/api/core/http"
import { toInt64String } from "@/api/core/int64"

export interface ActivityAdmin {
  activity_id: Int64
  title: string
  description: string
  style_config_json: string
  start_at_unix: number
  end_at_unix: number
  status: number
  created_by: Int64
  updated_by: Int64
  created_at_unix: number
  updated_at_unix: number
  items: ActivityItemAdmin[]
}

export interface ActivityItemAdmin {
  item_id: Int64
  activity_id: Int64
  product_id: Int64
  sku_code: string
  snapshot_name: string
  snapshot_main_image: string
  origin_price_cent: number
  seckill_price_cent: number
  reserved_stock_total: number
  available_stock: number
  sold_stock: number
  user_limit_mode: number
  user_limit_window_sec: number
  user_limit_qty: number
  max_qty_per_order: number
  status: number
  created_at_unix: number
  updated_at_unix: number
}

export interface ActivityOrderView {
  link_id: Int64
  order_id: Int64
  order_no: string
  user_id: Int64
  activity_id: Int64
  activity_item_id: Int64
  quantity: number
  order_status: number
  payment_status: number
  close_reason: string
  last_synced_at_unix: number
  created_at_unix: number
  updated_at_unix: number
}

export interface TrafficBucket {
  bucket_minute_unix: number
  activity_id: Int64
  activity_item_id: Int64
  pv: number
  uv: number
  click: number
  purchase_attempt: number
  purchase_success: number
  purchase_fail: number
  pay_success: number
  order_closed: number
}

export interface ActivityUpsertReq {
  title: string
  description: string
  style_config_json?: string
  start_at_unix: number
  end_at_unix: number
}

export interface UpsertActivityItemReq {
  product_id: Int64Like
  seckill_price_cent: number
  reserved_stock_total: number
  user_limit_mode: number
  user_limit_window_sec: number
  user_limit_qty: number
  max_qty_per_order: number
  status: number
}

export interface ListActivitiesQuery extends PaginationParams {
  keyword?: string
  status?: number
  include_deleted?: boolean
}

export interface GetActivityTrafficQuery {
  activity_item_id?: Int64Like
  from_minute_unix: number
  to_minute_unix: number
}

export interface ListActivityOrdersQuery extends PaginationParams {
  order_status?: number
  payment_status?: number
  user_id?: Int64Like
}

export const adminSeckillApi = {
  createActivity(req: ActivityUpsertReq): Promise<{ activity: ActivityAdmin }> {
    return adminApiClient.post<{ activity: ActivityAdmin }, ActivityUpsertReq>("/seckill/activities", req)
  },

  updateActivity(activityId: Int64Like, req: ActivityUpsertReq): Promise<{ activity: ActivityAdmin }> {
    return adminApiClient.patch<{ activity: ActivityAdmin }, ActivityUpsertReq>(
      `/seckill/activities/${activityId}`,
      req
    )
  },

  deleteActivity(activityId: Int64Like): Promise<{ activity_id: Int64 }> {
    return adminApiClient.delete<{ activity_id: Int64 }>(`/seckill/activities/${activityId}`)
  },

  getActivity(activityId: Int64Like): Promise<{ activity: ActivityAdmin }> {
    return adminApiClient.get<{ activity: ActivityAdmin }>(`/seckill/activities/${activityId}`)
  },

  listActivities(query?: ListActivitiesQuery): Promise<PaginatedList<ActivityAdmin>> {
    return adminApiClient.get<PaginatedList<ActivityAdmin>>("/seckill/activities", {
      params: query,
    })
  },

  createActivityItem(activityId: Int64Like, req: UpsertActivityItemReq): Promise<{ item: ActivityItemAdmin }> {
    return adminApiClient.post<
      { item: ActivityItemAdmin },
      Omit<UpsertActivityItemReq, "product_id"> & { product_id: string }
    >(
      `/seckill/activities/${activityId}/items`,
      {
        ...req,
        product_id: toInt64String(req.product_id, "product_id"),
      }
    )
  },

  updateActivityItem(
    activityId: Int64Like,
    itemId: Int64Like,
    req: UpsertActivityItemReq
  ): Promise<{ item: ActivityItemAdmin }> {
    return adminApiClient.put<
      { item: ActivityItemAdmin },
      Omit<UpsertActivityItemReq, "product_id"> & { product_id: string }
    >(
      `/seckill/activities/${activityId}/items/${itemId}`,
      {
        ...req,
        product_id: toInt64String(req.product_id, "product_id"),
      }
    )
  },

  removeActivityItem(activityId: Int64Like, itemId: Int64Like): Promise<{ activity_id: Int64; item_id: Int64 }> {
    return adminApiClient.delete<{ activity_id: Int64; item_id: Int64 }>(
      `/seckill/activities/${activityId}/items/${itemId}`
    )
  },

  publishActivity(activityId: Int64Like): Promise<{ activity: ActivityAdmin }> {
    return adminApiClient.post<{ activity: ActivityAdmin }>(`/seckill/activities/${activityId}/publish`)
  },

  offlineActivity(activityId: Int64Like): Promise<{ activity: ActivityAdmin }> {
    return adminApiClient.post<{ activity: ActivityAdmin }>(`/seckill/activities/${activityId}/offline`)
  },

  getTraffic(activityId: Int64Like, query: GetActivityTrafficQuery): Promise<{ rows: TrafficBucket[] }> {
    return adminApiClient.get<{ rows: TrafficBucket[] }>(`/seckill/activities/${activityId}/traffic`, {
      params: query,
    })
  },

  listOrders(activityId: Int64Like, query?: ListActivityOrdersQuery): Promise<PaginatedList<ActivityOrderView>> {
    return adminApiClient.get<PaginatedList<ActivityOrderView>>(`/seckill/activities/${activityId}/orders`, {
      params: query,
    })
  },
}
