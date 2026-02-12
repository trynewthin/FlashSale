// 运营探针 Hooks：用于管理端登录后快速校验权限与链路状态。
import { useQuery } from "@tanstack/react-query"

import { adminOpsApi } from "@/api/modules/operations"
import { adminQueryKeys } from "@/hooks/query-keys"
import { useAdminAuthStore } from "@/stores/auth-store"

export function useOpsPingQuery() {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.opsPing(operatorAdminId),
    queryFn: () => adminOpsApi.ping(),
    retry: 1,
  })
}

