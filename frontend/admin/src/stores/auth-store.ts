// 管理端会话状态：维护 access/refresh token 与当前管理员资料。
import { create } from "zustand"

import { adminTokenStore } from "@/api/core/token-store"
import { type AdminView } from "@/api/modules/auth"

interface AdminAuthState {
  accessToken: string
  refreshToken: string
  profile: AdminView | null
  setSession: (input: { accessToken: string; refreshToken: string; profile: AdminView }) => void
  setProfile: (profile: AdminView) => void
  clearSession: () => void
}

export const useAdminAuthStore = create<AdminAuthState>((set) => ({
  accessToken: adminTokenStore.getAccessToken(),
  refreshToken: adminTokenStore.getRefreshToken(),
  profile: null,
  setSession: ({ accessToken, refreshToken, profile }) => {
    adminTokenStore.setAccessToken(accessToken)
    adminTokenStore.setRefreshToken(refreshToken)
    set({ accessToken, refreshToken, profile })
  },
  setProfile: (profile) => set({ profile }),
  clearSession: () => {
    adminTokenStore.clear()
    set({ accessToken: "", refreshToken: "", profile: null })
  },
}))

