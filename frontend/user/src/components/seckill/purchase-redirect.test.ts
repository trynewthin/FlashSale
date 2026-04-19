import { describe, expect, it } from "vitest"

import { resolveSeckillPurchaseRedirect } from "@/components/seckill/purchase-redirect"

describe("resolveSeckillPurchaseRedirect", () => {
  it("navigates to order detail when order_id is present", () => {
    expect(
      resolveSeckillPurchaseRedirect({
        order_id: "90001",
        order_no: "SCKORDER90001",
      })
    ).toBe("/orders/90001")
  })

  it("falls back to pending order list when async path omits order_id", () => {
    expect(
      resolveSeckillPurchaseRedirect({
        order_no: "SCKORDERPENDING01",
      })
    ).toBe("/orders?order_no=SCKORDERPENDING01&pending=1")
  })

  it("falls back to pending order list when order_id is zero", () => {
    expect(
      resolveSeckillPurchaseRedirect({
        order_id: "0",
        order_no: "SCKORDERPENDING02",
      })
    ).toBe("/orders?order_no=SCKORDERPENDING02&pending=1")
  })
})
