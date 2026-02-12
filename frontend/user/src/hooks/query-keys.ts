import { type Int64 } from "@/api/core/types"
import { type ListOrdersQuery } from "@/api/modules/order"
import { type ListProductsQuery } from "@/api/modules/product"

export const userQueryKeys = {
  session: () => ["user", "session"] as const,
  profile: (userId?: Int64) => ["user", userId ?? "anonymous", "profile"] as const,
  productsList: (params?: ListProductsQuery) => ["products", "list", params ?? {}] as const,
  productDetail: (productId: string | number) => ["products", "detail", String(productId)] as const,
  ordersList: (userId?: Int64, params?: ListOrdersQuery) =>
    ["orders", userId ?? "anonymous", "list", params ?? {}] as const,
  orderDetail: (userId: Int64 | undefined, orderId: string | number) =>
    ["orders", userId ?? "anonymous", "detail", String(orderId)] as const,
  seckillActivities: (params?: { page?: number; page_size?: number }) =>
    ["seckill", "activities", params ?? {}] as const,
  seckillActivity: (activityId: string | number) => ["seckill", "activity", String(activityId)] as const,
}
