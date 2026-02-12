// 用户认证与资料 Hooks：统一会话落库、缓存失效与鉴权失效回收行为。
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect } from "react"

import { userAuthApi, type LoginReq, type RegisterReq, type UpdateNicknameReq } from "@/api/modules/auth"
import { useApiError } from "@/hooks/common/use-api-error"
import { userQueryKeys } from "@/hooks/query-keys"
import { useUserAuthStore, type UserProfileState } from "@/stores/auth-store"

function toUserProfile(input: { user_id: string; phone: string; nickname: string }): UserProfileState {
  return {
    user_id: input.user_id,
    phone: input.phone,
    nickname: input.nickname,
  }
}

export function useUserSessionState() {
  const accessToken = useUserAuthStore((state) => state.accessToken)
  const profile = useUserAuthStore((state) => state.profile)
  const isAuthed = accessToken.trim() !== ""
  return { accessToken, profile, isAuthed }
}

export function useLoginMutation() {
  const queryClient = useQueryClient()
  const setSession = useUserAuthStore((state) => state.setSession)

  return useMutation({
    mutationFn: (req: LoginReq) => userAuthApi.login(req),
    retry: 0,
    onSuccess: (resp) => {
      queryClient.clear()
      const profile = toUserProfile(resp)
      setSession({ accessToken: resp.access_token, profile })
      queryClient.setQueryData(userQueryKeys.profile("self"), profile)
    },
  })
}

export function useRegisterMutation() {
  const queryClient = useQueryClient()
  const setSession = useUserAuthStore((state) => state.setSession)

  return useMutation({
    mutationFn: (req: RegisterReq) => userAuthApi.register(req),
    retry: 0,
    onSuccess: (resp) => {
      queryClient.clear()
      const profile = toUserProfile(resp)
      setSession({ accessToken: resp.access_token, profile })
      queryClient.setQueryData(userQueryKeys.profile("self"), profile)
    },
  })
}

export function useLogoutAction() {
  const queryClient = useQueryClient()
  const clearSession = useUserAuthStore((state) => state.clearSession)

  return () => {
    clearSession()
    queryClient.clear()
  }
}

export function useUserProfileQuery() {
  const queryClient = useQueryClient()
  const { isCode } = useApiError()
  const accessToken = useUserAuthStore((state) => state.accessToken)
  const setProfile = useUserAuthStore((state) => state.setProfile)
  const clearSession = useUserAuthStore((state) => state.clearSession)

  const profileQuery = useQuery({
    queryKey: userQueryKeys.profile("self"),
    queryFn: () => userAuthApi.getProfile(),
    enabled: accessToken.trim() !== "",
    retry: 1,
  })

  useEffect(() => {
    if (profileQuery.data) {
      setProfile(toUserProfile(profileQuery.data))
    }
  }, [profileQuery.data, setProfile])

  useEffect(() => {
    if (profileQuery.error && isCode(profileQuery.error, "AUTH_UNAUTHORIZED")) {
      clearSession()
      queryClient.clear()
    }
  }, [clearSession, isCode, profileQuery.error, queryClient])

  return profileQuery
}

export function useUpdateNicknameMutation() {
  const queryClient = useQueryClient()
  const { isCode } = useApiError()
  const setProfile = useUserAuthStore((state) => state.setProfile)
  const clearSession = useUserAuthStore((state) => state.clearSession)

  return useMutation({
    mutationFn: (req: UpdateNicknameReq) => userAuthApi.updateNickname(req),
    retry: 0,
    onSuccess: (resp) => {
      const next = toUserProfile(resp)
      setProfile(next)
      queryClient.setQueryData(userQueryKeys.profile("self"), next)
      queryClient.invalidateQueries({
        queryKey: userQueryKeys.profile("self"),
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

export function useDeleteUserMutation() {
  const queryClient = useQueryClient()
  const clearSession = useUserAuthStore((state) => state.clearSession)

  return useMutation({
    mutationFn: () => userAuthApi.deleteUser(),
    retry: 0,
    onSuccess: () => {
      clearSession()
      queryClient.clear()
    },
  })
}
