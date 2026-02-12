// 订单管理 Hooks：封装管理端订单审核、发货、详情与列表。
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { type Int64Like } from "@/api/core/types"
import {
  adminOrderApi,
  type ListAdminOrdersQuery,
  type ReviewOrderReq,
  type ShipOrderReq,
} from "@/api/modules/order"
import { adminQueryKeys } from "@/hooks/query-keys"
import { useAdminAuthStore } from "@/stores/auth-store"

export function useAdminOrderListQuery(params?: ListAdminOrdersQuery) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.ordersList(operatorAdminId, params),
    queryFn: () => adminOrderApi.listOrders(params),
    retry: 1,
  })
}

export function useAdminOrderDetailQuery(orderId: Int64Like | undefined) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.orderDetail(operatorAdminId, orderId ?? "0"),
    queryFn: () => adminOrderApi.getOrder(orderId ?? "0"),
    enabled: orderId !== undefined && orderId !== null,
    retry: 1,
  })
}

export function useReviewOrderMutation(orderId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: (req: ReviewOrderReq) => adminOrderApi.reviewOrder(orderId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.orderDetail(operatorAdminId, orderId),
      })
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.ordersList(operatorAdminId),
      })
    },
  })
}

export function useShipOrderMutation(orderId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: (req: ShipOrderReq) => adminOrderApi.shipOrder(orderId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.orderDetail(operatorAdminId, orderId),
      })
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.ordersList(operatorAdminId),
      })
    },
  })
}

