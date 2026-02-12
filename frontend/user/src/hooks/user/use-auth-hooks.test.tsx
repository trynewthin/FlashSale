import { act, renderHook, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useLoginMutation, useUserProfileQuery } from "@/hooks/user/use-auth-hooks"
import { useUserAuthStore } from "@/stores/auth-store"
import { createQueryClientWrapper, createTestQueryClient } from "@/test/query-client"

const { loginMock, registerMock, getProfileMock, updateNicknameMock, deleteUserMock } = vi.hoisted(() => ({
  loginMock: vi.fn(),
  registerMock: vi.fn(),
  getProfileMock: vi.fn(),
  updateNicknameMock: vi.fn(),
  deleteUserMock: vi.fn(),
}))

vi.mock("@/api/modules/auth", () => ({
  userAuthApi: {
    login: loginMock,
    register: registerMock,
    getProfile: getProfileMock,
    updateNickname: updateNicknameMock,
    deleteUser: deleteUserMock,
  },
}))

describe("use-auth-hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUserAuthStore.setState({
      accessToken: "",
      profile: null,
    })
  })

  it("login success should clear previous cache and set new session", async () => {
    const queryClient = createTestQueryClient()
    const wrapper = createQueryClientWrapper(queryClient)
    const oldOrdersKey = ["orders", "old-user", "list", {}] as const
    queryClient.setQueryData(oldOrdersKey, { stale: true })

    loginMock.mockResolvedValue({
      user_id: "2021274864979042304",
      phone: "13800000000",
      nickname: "new-user",
      access_token: "new-access-token",
      expires_in_sec: 900,
    })

    const { result } = renderHook(() => useLoginMutation(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({
        phone: "13800000000",
        password: "abc12345",
      })
    })

    expect(queryClient.getQueryData(oldOrdersKey)).toBeUndefined()
    expect(queryClient.getQueryData(["user", "self", "profile"])).toEqual({
      user_id: "2021274864979042304",
      phone: "13800000000",
      nickname: "new-user",
    })
    expect(useUserAuthStore.getState().accessToken).toBe("new-access-token")
    expect(useUserAuthStore.getState().profile?.user_id).toBe("2021274864979042304")
  })

  it("profile query should use stable key and request once", async () => {
    const queryClient = createTestQueryClient()
    const wrapper = createQueryClientWrapper(queryClient)
    useUserAuthStore.setState({
      accessToken: "valid-token",
      profile: null,
    })

    getProfileMock.mockResolvedValue({
      user_id: "2021274864979042304",
      phone: "13800000000",
      nickname: "profile-user",
    })

    renderHook(() => useUserProfileQuery(), { wrapper })

    await waitFor(() => {
      expect(getProfileMock).toHaveBeenCalledTimes(1)
    })
    await waitFor(() => {
      expect(useUserAuthStore.getState().profile?.nickname).toBe("profile-user")
    })

    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(getProfileMock).toHaveBeenCalledTimes(1)
  })
})
