// 用户管理 Hooks：封装管理端用户资料查询、改昵称、删除。
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { type Int64Like } from "@/api/core/types"
import { adminUserApi, type ListUsersQuery, type UpdateManagedUserNicknameReq } from "@/api/modules/user"
import { adminQueryKeys } from "@/hooks/query-keys"
import { useAdminAuthStore } from "@/stores/auth-store"

export function useUserListQuery(params: ListUsersQuery) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.usersList(operatorAdminId, params),
    queryFn: () => adminUserApi.listUsers(params),
    retry: 1,
  })
}

export function useManagedUserProfileQuery(userId: Int64Like | undefined) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.managedUserProfile(operatorAdminId, userId ?? "0"),
    queryFn: () => adminUserApi.getUserProfile(userId ?? "0"),
    enabled: userId !== undefined && userId !== null,
    retry: 1,
  })
}

export function useManagedUserNicknameMutation(userId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: (req: UpdateManagedUserNicknameReq) => adminUserApi.updateUserNickname(userId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.managedUserProfile(operatorAdminId, userId),
      })
    },
  })
}

export function useManagedUserDeleteMutation(userId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: () => adminUserApi.deleteUser(userId),
    retry: 0,
    onSuccess: () => {
      queryClient.removeQueries({
        queryKey: adminQueryKeys.managedUserProfile(operatorAdminId, userId),
      })
    },
  })
}

