// 管理员账号管理 Hooks：覆盖管理员列表、详情与写操作失效策略。
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { type Int64Like } from "@/api/core/types"
import {
  adminManagementApi,
  type BindAdminRolesReq,
  type CreateAdminReq,
  type ListAdminsQuery,
  type ResetAdminPasswordReq,
  type SetAdminStatusReq,
  type UpdateAdminReq,
} from "@/api/modules/admin"
import { adminQueryKeys } from "@/hooks/query-keys"
import { useAdminAuthStore } from "@/stores/auth-store"

export function useAdminListQuery(params?: ListAdminsQuery) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.admins(operatorAdminId, params),
    queryFn: () => adminManagementApi.listAdmins(params),
    retry: 1,
  })
}

export function useAdminDetailQuery(targetAdminId: Int64Like | undefined) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.adminDetail(operatorAdminId, targetAdminId ?? "0"),
    queryFn: () => adminManagementApi.getAdmin(targetAdminId ?? "0"),
    enabled: targetAdminId !== undefined && targetAdminId !== null,
    retry: 1,
  })
}

export function useCreateAdminMutation() {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)

  return useMutation({
    mutationFn: (req: CreateAdminReq) => adminManagementApi.createAdmin(req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.admins(operatorAdminId),
      })
    },
  })
}

export function useUpdateAdminMutation(targetAdminId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)

  return useMutation({
    mutationFn: (req: UpdateAdminReq) => adminManagementApi.updateAdmin(targetAdminId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.adminDetail(operatorAdminId, targetAdminId),
      })
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.admins(operatorAdminId),
      })
    },
  })
}

export function useSetAdminStatusMutation(targetAdminId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)

  return useMutation({
    mutationFn: (req: SetAdminStatusReq) => adminManagementApi.setAdminStatus(targetAdminId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.adminDetail(operatorAdminId, targetAdminId),
      })
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.admins(operatorAdminId),
      })
    },
  })
}

export function useResetAdminPasswordMutation(targetAdminId: Int64Like) {
  return useMutation({
    mutationFn: (req: ResetAdminPasswordReq) => adminManagementApi.resetAdminPassword(targetAdminId, req),
    retry: 0,
  })
}

export function useDeleteAdminMutation(targetAdminId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)

  return useMutation({
    mutationFn: () => adminManagementApi.deleteAdmin(targetAdminId),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.admins(operatorAdminId),
      })
      queryClient.removeQueries({
        queryKey: adminQueryKeys.adminDetail(operatorAdminId, targetAdminId),
      })
    },
  })
}

export function useBindAdminRolesMutation(targetAdminId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)

  return useMutation({
    mutationFn: (req: BindAdminRolesReq) => adminManagementApi.bindAdminRoles(targetAdminId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.adminDetail(operatorAdminId, targetAdminId),
      })
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.admins(operatorAdminId),
      })
    },
  })
}

