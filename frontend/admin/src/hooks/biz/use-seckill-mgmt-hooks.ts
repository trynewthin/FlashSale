// 秒杀管理 Hooks：封装活动、活动商品、发布/下线、流量与订单追溯。
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { type Int64Like } from "@/api/core/types"
import {
  adminSeckillApi,
  type ActivityUpsertReq,
  type GetActivityTrafficQuery,
  type ListActivitiesQuery,
  type ListActivityOrdersQuery,
  type UpsertActivityItemReq,
} from "@/api/modules/seckill"
import { adminQueryKeys } from "@/hooks/query-keys"
import { useAdminAuthStore } from "@/stores/auth-store"

export function useSeckillActivityListQuery(params?: ListActivitiesQuery) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.seckillActivities(operatorAdminId, params),
    queryFn: () => adminSeckillApi.listActivities(params),
    retry: 1,
  })
}

export function useSeckillActivityDetailQuery(activityId: Int64Like | undefined) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.seckillActivityDetail(operatorAdminId, activityId ?? "0"),
    queryFn: () => adminSeckillApi.getActivity(activityId ?? "0"),
    enabled: activityId !== undefined && activityId !== null,
    retry: 1,
  })
}

export function useCreateSeckillActivityMutation() {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: (req: ActivityUpsertReq) => adminSeckillApi.createActivity(req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.seckillActivities(operatorAdminId),
      })
    },
  })
}

export function useUpdateSeckillActivityMutation(activityId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: (req: ActivityUpsertReq) => adminSeckillApi.updateActivity(activityId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.seckillActivityDetail(operatorAdminId, activityId),
      })
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.seckillActivities(operatorAdminId),
      })
    },
  })
}

export function useDeleteSeckillActivityMutation(activityId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: () => adminSeckillApi.deleteActivity(activityId),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.seckillActivities(operatorAdminId),
      })
      queryClient.removeQueries({
        queryKey: adminQueryKeys.seckillActivityDetail(operatorAdminId, activityId),
      })
    },
  })
}

export function useCreateSeckillItemMutation(activityId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: (req: UpsertActivityItemReq) => adminSeckillApi.createActivityItem(activityId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.seckillActivityDetail(operatorAdminId, activityId),
      })
    },
  })
}

export function useUpdateSeckillItemMutation(activityId: Int64Like, itemId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: (req: UpsertActivityItemReq) => adminSeckillApi.updateActivityItem(activityId, itemId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.seckillActivityDetail(operatorAdminId, activityId),
      })
    },
  })
}

export function useRemoveSeckillItemMutation(activityId: Int64Like, itemId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: () => adminSeckillApi.removeActivityItem(activityId, itemId),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.seckillActivityDetail(operatorAdminId, activityId),
      })
    },
  })
}

export function usePublishSeckillActivityMutation(activityId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: () => adminSeckillApi.publishActivity(activityId),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.seckillActivityDetail(operatorAdminId, activityId),
      })
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.seckillActivities(operatorAdminId),
      })
    },
  })
}

export function useOfflineSeckillActivityMutation(activityId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: () => adminSeckillApi.offlineActivity(activityId),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.seckillActivityDetail(operatorAdminId, activityId),
      })
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.seckillActivities(operatorAdminId),
      })
    },
  })
}

export function useSeckillTrafficQuery(activityId: Int64Like | undefined, params: GetActivityTrafficQuery | undefined) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.seckillTraffic(operatorAdminId, activityId ?? "0", params),
    queryFn: () => adminSeckillApi.getTraffic(activityId ?? "0", params ?? { from_minute_unix: 0, to_minute_unix: 0 }),
    enabled:
      activityId !== undefined &&
      activityId !== null &&
      params !== undefined &&
      params.from_minute_unix > 0 &&
      params.to_minute_unix > 0,
    retry: 1,
  })
}

export function useSeckillOrdersQuery(activityId: Int64Like | undefined, params?: ListActivityOrdersQuery) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.seckillOrders(operatorAdminId, activityId ?? "0", params),
    queryFn: () => adminSeckillApi.listOrders(activityId ?? "0", params),
    enabled: activityId !== undefined && activityId !== null,
    retry: 1,
  })
}

