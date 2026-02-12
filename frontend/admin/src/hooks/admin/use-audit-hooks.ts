// 审计日志 Hooks：提供管理员审计日志列表查询。
import { useQuery } from "@tanstack/react-query"

import { adminManagementApi, type ListAuditLogsQuery } from "@/api/modules/admin"
import { adminQueryKeys } from "@/hooks/query-keys"
import { useAdminAuthStore } from "@/stores/auth-store"

export function useAuditLogsQuery(params?: ListAuditLogsQuery) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.auditLogs(operatorAdminId, params),
    queryFn: () => adminManagementApi.listAuditLogs(params),
    retry: 1,
  })
}

