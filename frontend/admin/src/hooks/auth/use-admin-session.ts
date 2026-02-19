// 管理端会话 Hooks：统一登录、续期、退出、资料同步。
// 注：AUTH_UNAUTHORIZED / ADMIN_NOT_FOUND 由 http.ts interceptor 全局处理，hooks 无需重复。
import { useEffect } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { adminTokenStore } from "@/api/core/token-store"
import {
  adminAuthApi,
  type AdminLoginReq,
  type AdminLogoutReq,
  type AdminRefreshReq,
  type ChangeMyPasswordReq,
  persistAdminTokens,
} from "@/api/modules/auth"
import { adminQueryKeys } from "@/hooks/query-keys"
import { useAdminAuthStore } from "@/stores/auth-store"

export function useAdminSessionState() {
  const accessToken = useAdminAuthStore((state) => state.accessToken)
  const refreshToken = useAdminAuthStore((state) => state.refreshToken)
  const profile = useAdminAuthStore((state) => state.profile)
  const domains = profile?.domains ?? []
  const dataScope = profile?.data_scope ?? "self"
  const isAuthed = accessToken.trim() !== ""
  return { accessToken, refreshToken, profile, domains, dataScope, isAuthed }
}

export function useAdminLoginMutation() {
  const queryClient = useQueryClient()
  const setSession = useAdminAuthStore((state) => state.setSession)

  return useMutation({
    mutationFn: (req: AdminLoginReq) => adminAuthApi.login(req),
    retry: 0,
    onSuccess: (resp) => {
      persistAdminTokens(resp)
      setSession({
        accessToken: resp.access_token,
        refreshToken: resp.refresh_token,
        profile: resp.admin,
      })
      queryClient.clear()
      queryClient.setQueryData(adminQueryKeys.profile("self"), { admin: resp.admin })
    },
  })
}

export function useAdminRefreshMutation() {
  const queryClient = useQueryClient()
  const setSession = useAdminAuthStore((state) => state.setSession)
  const clearSession = useAdminAuthStore((state) => state.clearSession)

  return useMutation({
    mutationFn: (req?: AdminRefreshReq) =>
      adminAuthApi.refresh({
        refresh_token: req?.refresh_token ?? adminTokenStore.getRefreshToken(),
      }),
    retry: 0,
    onSuccess: (resp) => {
      persistAdminTokens(resp)
      setSession({
        accessToken: resp.access_token,
        refreshToken: resp.refresh_token,
        profile: resp.admin,
      })
    },
    onError: () => {
      clearSession()
      queryClient.clear()
    },
  })
}

export function useAdminLogoutMutation() {
  const queryClient = useQueryClient()
  const clearSession = useAdminAuthStore((state) => state.clearSession)

  return useMutation({
    mutationFn: (req?: Partial<AdminLogoutReq>) =>
      adminAuthApi.logout({
        refresh_token: req?.refresh_token ?? adminTokenStore.getRefreshToken(),
      }),
    retry: 0,
    onSettled: () => {
      clearSession()
      queryClient.clear()
    },
  })
}

export function useMyAdminProfileQuery() {
  const accessToken = useAdminAuthStore((state) => state.accessToken)
  const setProfile = useAdminAuthStore((state) => state.setProfile)

  const profileQuery = useQuery({
    queryKey: adminQueryKeys.profile("self"),
    queryFn: () => adminAuthApi.getMyProfile(),
    enabled: accessToken.trim() !== "",
    retry: 1,
  })

  useEffect(() => {
    if (profileQuery.data?.admin) {
      setProfile(profileQuery.data.admin)
    }
  }, [profileQuery.data, setProfile])

  return profileQuery
}

export function useChangeMyPasswordMutation() {
  return useMutation({
    mutationFn: (req: ChangeMyPasswordReq) => adminAuthApi.changeMyPassword(req),
    retry: 0,
  })
}
