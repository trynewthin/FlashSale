import { act, renderHook, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useAdminRefreshMutation, useMyAdminProfileQuery } from "@/hooks/auth/use-admin-session"
import { useAdminAuthStore } from "@/stores/auth-store"
import { createQueryClientWrapper, createTestQueryClient } from "@/test/query-client"

const { loginMock, refreshMock, logoutMock, getMyProfileMock, changeMyPasswordMock, persistTokensMock } =
  vi.hoisted(() => ({
    loginMock: vi.fn(),
    refreshMock: vi.fn(),
    logoutMock: vi.fn(),
    getMyProfileMock: vi.fn(),
    changeMyPasswordMock: vi.fn(),
    persistTokensMock: vi.fn(),
  }))

vi.mock("@/api/modules/auth", () => ({
  adminAuthApi: {
    login: loginMock,
    refresh: refreshMock,
    logout: logoutMock,
    getMyProfile: getMyProfileMock,
    changeMyPassword: changeMyPasswordMock,
  },
  persistAdminTokens: persistTokensMock,
}))

describe("use-admin-session", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useAdminAuthStore.setState({
      accessToken: "",
      refreshToken: "",
      profile: null,
    })
  })

  it("refresh failure should clear session and query cache", async () => {
    const queryClient = createTestQueryClient()
    const wrapper = createQueryClientWrapper(queryClient)
    const protectedKey = ["mgmt", "2021274864979042304", "orders", "list", {}] as const
    queryClient.setQueryData(protectedKey, { total: 99 })

    useAdminAuthStore.getState().setSession({
      accessToken: "expired-access",
      refreshToken: "expired-refresh",
      profile: {
        admin_id: "2021274864979042304",
        username: "root",
        display_name: "Root",
        status: 1,
        data_scope: "all",
        is_super_admin: true,
        last_login_at_unix: 0,
        last_login_ip: "",
        role_ids: [],
        domains: ["admin_management"],
        created_at_unix: 0,
        updated_at_unix: 0,
      },
    })

    refreshMock.mockRejectedValue(new Error("refresh failed"))

    const { result } = renderHook(() => useAdminRefreshMutation(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync(undefined).catch(() => undefined)
    })

    await waitFor(() => {
      expect(useAdminAuthStore.getState().accessToken).toBe("")
    })
    expect(useAdminAuthStore.getState().refreshToken).toBe("")
    expect(useAdminAuthStore.getState().profile).toBeNull()
    expect(queryClient.getQueryData(protectedKey)).toBeUndefined()
  })

  it("profile query should use stable key and request once", async () => {
    const queryClient = createTestQueryClient()
    const wrapper = createQueryClientWrapper(queryClient)
    useAdminAuthStore.setState({
      accessToken: "valid-access",
      refreshToken: "valid-refresh",
      profile: null,
    })

    getMyProfileMock.mockResolvedValue({
      admin: {
        admin_id: "2021274864979042304",
        username: "root",
        display_name: "Root",
        status: 1,
        data_scope: "all",
        is_super_admin: true,
        last_login_at_unix: 0,
        last_login_ip: "",
        role_ids: [],
        domains: ["admin_management"],
        created_at_unix: 0,
        updated_at_unix: 0,
      },
    })

    renderHook(() => useMyAdminProfileQuery(), { wrapper })

    await waitFor(() => {
      expect(getMyProfileMock).toHaveBeenCalledTimes(1)
    })
    await waitFor(() => {
      expect(useAdminAuthStore.getState().profile?.admin_id).toBe("2021274864979042304")
    })

    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(getMyProfileMock).toHaveBeenCalledTimes(1)
  })
})
