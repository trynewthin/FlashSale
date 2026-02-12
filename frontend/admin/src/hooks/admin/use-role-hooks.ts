// 角色管理 Hooks：封装角色 CRUD 与角色域绑定的缓存策略。
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { type Int64Like } from "@/api/core/types"
import {
  adminManagementApi,
  type CreateRoleReq,
  type ListRolesQuery,
  type SetRoleDomainsReq,
  type UpdateRoleReq,
} from "@/api/modules/admin"
import { adminQueryKeys } from "@/hooks/query-keys"
import { useAdminAuthStore } from "@/stores/auth-store"

export function useRoleListQuery(params?: ListRolesQuery) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.roles(operatorAdminId, params),
    queryFn: () => adminManagementApi.listRoles(params),
    retry: 1,
  })
}

export function useRoleDetailQuery(roleId: Int64Like | undefined) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.roleDetail(operatorAdminId, roleId ?? "0"),
    queryFn: () => adminManagementApi.getRole(roleId ?? "0"),
    enabled: roleId !== undefined && roleId !== null,
    retry: 1,
  })
}

export function useCreateRoleMutation() {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)

  return useMutation({
    mutationFn: (req: CreateRoleReq) => adminManagementApi.createRole(req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.roles(operatorAdminId),
      })
    },
  })
}

export function useUpdateRoleMutation(roleId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)

  return useMutation({
    mutationFn: (req: UpdateRoleReq) => adminManagementApi.updateRole(roleId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.roleDetail(operatorAdminId, roleId),
      })
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.roles(operatorAdminId),
      })
    },
  })
}

export function useDeleteRoleMutation(roleId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)

  return useMutation({
    mutationFn: () => adminManagementApi.deleteRole(roleId),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.roles(operatorAdminId),
      })
      queryClient.removeQueries({
        queryKey: adminQueryKeys.roleDetail(operatorAdminId, roleId),
      })
    },
  })
}

export function useSetRoleDomainsMutation(roleId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)

  return useMutation({
    mutationFn: (req: SetRoleDomainsReq) => adminManagementApi.setRoleDomains(roleId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.roleDetail(operatorAdminId, roleId),
      })
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.roles(operatorAdminId),
      })
    },
  })
}

