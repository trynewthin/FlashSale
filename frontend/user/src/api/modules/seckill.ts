import { type Int64, type Int64Like, type PaginatedList, type PaginationParams } from "@/api/core/types"
import { userApiClient } from "@/api/core/http"
import { toInt64String } from "@/api/core/int64"

export interface ActivityItemPublic {
  item_id: Int64
  product_id: Int64
  sku_code: string
  snapshot_name: string
  snapshot_main_image: string
  origin_price_cent: number
  seckill_price_cent: number
  in_stock: boolean
  max_qty_per_order: number
}

export interface ActivityPublic {
  activity_id: Int64
  title: string
  description: string
  style_config_json: string
  start_at_unix: number
  end_at_unix: number
  status: number
  items: ActivityItemPublic[]
}

export type ListActivitiesResp = PaginatedList<ActivityPublic>

export interface GetActivityResp {
  activity: ActivityPublic
}

export interface PurchaseReq {
  activity_item_id: Int64Like
  quantity: number
  idempotency_key: string
}

export interface PurchaseResp {
  activity_id: Int64
  activity_item_id: Int64
  // Kafka 异步建单路径会返回 order_id=0；后端 JSON `omitempty` 会把该字段省略。
  order_id?: Int64
  order_no: string
}

export interface TrackEventReq {
  activity_item_id: Int64Like
  event_type: string
  client_id?: string
  idempotency_key: string
  occurred_at_unix?: number
}

export interface TrackEventResp {
  accepted: boolean
}

export const userSeckillApi = {
  listActivities(query?: PaginationParams): Promise<ListActivitiesResp> {
    return userApiClient.get<ListActivitiesResp>("/seckill/activities", {
      params: query,
    })
  },

  getActivity(activityId: Int64Like): Promise<GetActivityResp> {
    return userApiClient.get<GetActivityResp>(`/seckill/activities/${activityId}`)
  },

  purchase(activityId: Int64Like, req: PurchaseReq): Promise<PurchaseResp> {
    return userApiClient.post<PurchaseResp, Omit<PurchaseReq, "activity_item_id"> & { activity_item_id: string }>(
      `/seckill/activities/${activityId}/purchase`,
      {
        ...req,
        activity_item_id: toInt64String(req.activity_item_id, "activity_item_id"),
      }
    )
  },

  trackEvent(activityId: Int64Like, req: TrackEventReq): Promise<TrackEventResp> {
    return userApiClient.post<
      TrackEventResp,
      Omit<TrackEventReq, "activity_item_id"> & { activity_item_id: string }
    >(`/seckill/activities/${activityId}/track`, {
      ...req,
      activity_item_id: toInt64String(req.activity_item_id, "activity_item_id"),
    })
  },
}
