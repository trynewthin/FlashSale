// 秒杀 Hooks：封装活动查询、购买与埋点，统一幂等键注入与缓存更新。
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { type Int64Like, type PaginationParams } from "@/api/core/types"
import { userSeckillApi, type PurchaseReq, type TrackEventReq } from "@/api/modules/seckill"
import { useIdempotencyKey } from "@/hooks/common/use-idempotency-key"
import { userQueryKeys } from "@/hooks/query-keys"
import { useUserAuthStore } from "@/stores/auth-store"

interface PurchaseInput extends Omit<PurchaseReq, "idempotency_key"> {
  idempotency_key?: string
}

interface TrackInput extends Omit<TrackEventReq, "idempotency_key"> {
  idempotency_key?: string
}

export function useSeckillActivitiesQuery(params?: PaginationParams) {
  return useQuery({
    queryKey: userQueryKeys.seckillActivities(params),
    queryFn: () => userSeckillApi.listActivities(params),
    retry: 1,
  })
}

export function useSeckillActivityDetailQuery(activityId: Int64Like | undefined) {
  return useQuery({
    queryKey: userQueryKeys.seckillActivity(activityId ?? "0"),
    queryFn: () => userSeckillApi.getActivity(activityId ?? "0"),
    enabled: activityId !== undefined && activityId !== null,
    retry: 1,
  })
}

export function useSeckillPurchaseMutation(activityId: Int64Like) {
  const queryClient = useQueryClient()
  const userId = useUserAuthStore((state) => state.profile?.user_id)
  const { renewIdempotencyKey, clearIdempotencyKey } = useIdempotencyKey()

  return useMutation({
    mutationFn: (req: PurchaseInput) =>
      userSeckillApi.purchase(activityId, {
        ...req,
        idempotency_key: req.idempotency_key ?? renewIdempotencyKey(),
      }),
    retry: 0,
    onSuccess: () => {
      clearIdempotencyKey()
      queryClient.invalidateQueries({
        queryKey: userQueryKeys.ordersList(userId),
      })
      queryClient.invalidateQueries({
        queryKey: userQueryKeys.seckillActivity(activityId),
      })
    },
  })
}

export function useSeckillTrackMutation(activityId: Int64Like) {
  const { renewIdempotencyKey } = useIdempotencyKey()

  return useMutation({
    mutationFn: (req: TrackInput) =>
      userSeckillApi.trackEvent(activityId, {
        ...req,
        idempotency_key: req.idempotency_key ?? renewIdempotencyKey(),
      }),
    retry: 0,
  })
}
