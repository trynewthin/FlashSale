import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

import { type OrderView } from "@/api/modules/order"
import { useShipOrderMutation } from "@/hooks/biz/use-order-mgmt-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

export function ShipOrderDialog({ order, onClose }: { order: OrderView; onClose: () => void }) {
  const mutation = useShipOrderMutation(order.order_id)
  const { toUserMessage } = useApiError()
  const [trackingNo, setTrackingNo] = useState("")

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader><DialogTitle>发货 - {order.order_no}</DialogTitle></DialogHeader>
        <form onSubmit={(e) => {
          e.preventDefault()
          mutation.mutate({ tracking_no: trackingNo }, { onSuccess: onClose })
        }} className="space-y-4">
          <div className="space-y-2"><Label>物流单号</Label><Input value={trackingNo} onChange={(e) => setTrackingNo(e.target.value)} required /></div>
          {mutation.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(mutation.error)}</AlertDescription></Alert>}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose}>取消</Button>
            <Button type="submit" disabled={mutation.isPending}>{mutation.isPending ? "提交中…" : "确认发货"}</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
