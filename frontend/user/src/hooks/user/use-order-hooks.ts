// 用户订单 Hooks：封装创建、支付确认、取消、收货与列表查询的一致缓存策略。
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { type Int64Like } from "@/api/core/types"
import {
  userOrderApi,
  type CancelOrderReq,
  type ConfirmPaymentAndInfoReq,
  type CreateOrderReq,
  type ListOrdersQuery,
} from "@/api/modules/order"
import { useApiError } from "@/hooks/common/use-api-error"
import { userQueryKeys } from "@/hooks/query-keys"
import { useUserAuthStore } from "@/stores/auth-store"

export function useOrderListQuery(params?: ListOrdersQuery) {
  const accessToken = useUserAuthStore((state) => state.accessToken)
  const userId = useUserAuthStore((state) => state.profile?.user_id)

  return useQuery({
    queryKey: userQueryKeys.ordersList(userId, params),
    queryFn: () => userOrderApi.listOrders(params),
    enabled: accessToken.trim() !== "",
    retry: 1,
  })
}

export function useOrderDetailQuery(orderId: Int64Like | undefined) {
  const accessToken = useUserAuthStore((state) => state.accessToken)
  const userId = useUserAuthStore((state) => state.profile?.user_id)

  return useQuery({
    queryKey: userQueryKeys.orderDetail(userId, orderId ?? "0"),
    queryFn: () => userOrderApi.getOrder(orderId ?? "0"),
    enabled: accessToken.trim() !== "" && orderId !== undefined && orderId !== null,
    retry: 1,
  })
}

export function useCreateOrderMutation() {
  const queryClient = useQueryClient()
  const userId = useUserAuthStore((state) => state.profile?.user_id)

  return useMutation({
    mutationFn: (req: CreateOrderReq) => userOrderApi.createOrder(req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: userQueryKeys.ordersList(userId),
      })
    },
  })
}

export function useConfirmPaymentAndInfoMutation(orderId: Int64Like) {
  const queryClient = useQueryClient()
  const userId = useUserAuthStore((state) => state.profile?.user_id)
  const { isCode } = useApiError()
  const clearSession = useUserAuthStore((state) => state.clearSession)

  return useMutation({
    mutationFn: (req: ConfirmPaymentAndInfoReq) => userOrderApi.confirmPaymentAndInfo(orderId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: userQueryKeys.orderDetail(userId, orderId),
      })
      queryClient.invalidateQueries({
        queryKey: userQueryKeys.ordersList(userId),
      })
    },
    onError: (error) => {
      if (isCode(error, "AUTH_UNAUTHORIZED")) {
        clearSession()
        queryClient.clear()
      }
    },
  })
}

export function useCancelOrderMutation(orderId: Int64Like) {
  const queryClient = useQueryClient()
  const userId = useUserAuthStore((state) => state.profile?.user_id)

  return useMutation({
    mutationFn: (req: CancelOrderReq) => userOrderApi.cancelOrder(orderId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: userQueryKeys.orderDetail(userId, orderId),
      })
      queryClient.invalidateQueries({
        queryKey: userQueryKeys.ordersList(userId),
      })
    },
  })
}

export function useConfirmReceiptMutation(orderId: Int64Like) {
  const queryClient = useQueryClient()
  const userId = useUserAuthStore((state) => state.profile?.user_id)

  return useMutation({
    mutationFn: () => userOrderApi.confirmReceipt(orderId),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: userQueryKeys.orderDetail(userId, orderId),
      })
      queryClient.invalidateQueries({
        queryKey: userQueryKeys.ordersList(userId),
      })
    },
  })
}

