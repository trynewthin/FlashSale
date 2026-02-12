import { create } from "zustand"

import { userTokenStore } from "@/api/core/token-store"
import { type Int64 } from "@/api/core/types"

export interface UserProfileState {
  user_id: Int64
  phone: string
  nickname: string
}

interface UserAuthState {
  accessToken: string
  profile: UserProfileState | null
  setSession: (input: { accessToken: string; profile: UserProfileState }) => void
  setProfile: (profile: UserProfileState) => void
  clearSession: () => void
}

export const useUserAuthStore = create<UserAuthState>((set) => ({
  accessToken: userTokenStore.getAccessToken(),
  profile: null,
  setSession: ({ accessToken, profile }) => {
    userTokenStore.setAccessToken(accessToken)
    set({ accessToken, profile })
  },
  setProfile: (profile) => set({ profile }),
  clearSession: () => {
    userTokenStore.clear()
    set({ accessToken: "", profile: null })
  },
}))
