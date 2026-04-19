import type { PurchaseResp } from "@/api/modules/seckill"

export function resolveSeckillPurchaseRedirect(data: Pick<PurchaseResp, "order_id" | "order_no">): string {
  const orderId = data.order_id === undefined || data.order_id === null ? "" : String(data.order_id).trim()
  if (orderId !== "" && orderId !== "0") {
    return `/orders/${orderId}`
  }

  const params = new URLSearchParams()
  const orderNo = data.order_no.trim()
  if (orderNo !== "") {
    params.set("order_no", orderNo)
  }
  params.set("pending", "1")
  return `/orders?${params.toString()}`
}
