import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useSeckillTrackMutation } from "@/hooks/user/use-seckill-hooks"
import { createQueryClientWrapper, createTestQueryClient } from "@/test/query-client"

const { listActivitiesMock, getActivityMock, purchaseMock, trackEventMock } = vi.hoisted(() => ({
  listActivitiesMock: vi.fn(),
  getActivityMock: vi.fn(),
  purchaseMock: vi.fn(),
  trackEventMock: vi.fn(),
}))

vi.mock("@/api/modules/seckill", () => ({
  userSeckillApi: {
    listActivities: listActivitiesMock,
    getActivity: getActivityMock,
    purchase: purchaseMock,
    trackEvent: trackEventMock,
  },
}))

describe("use-seckill-hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("track mutation should propagate API error", async () => {
    const queryClient = createTestQueryClient()
    const wrapper = createQueryClientWrapper(queryClient)
    trackEventMock.mockRejectedValue(new Error("network down"))

    const { result } = renderHook(() => useSeckillTrackMutation("1001"), { wrapper })

    await expect(
      act(async () => {
        await result.current.mutateAsync({
          activity_item_id: "2001",
          event_type: "click",
          client_id: "cid-1",
        })
      })
    ).rejects.toThrow("network down")
  })
})
