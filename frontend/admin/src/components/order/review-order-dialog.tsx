import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Textarea } from "@/components/ui/textarea"

import { type OrderView } from "@/api/modules/order"
import { useReviewOrderMutation } from "@/hooks/biz/use-order-mgmt-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

export function ReviewOrderDialog({ order, onClose }: { order: OrderView; onClose: () => void }) {
  const mutation = useReviewOrderMutation(order.order_id)
  const { toUserMessage } = useApiError()
  const [approved, setApproved] = useState<string>("true")
  const [reason, setReason] = useState("")

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader><DialogTitle>审核订单 - {order.order_no}</DialogTitle></DialogHeader>
        <form onSubmit={(e) => {
          e.preventDefault()
          mutation.mutate({ approved: approved === "true", reason: reason || undefined }, { onSuccess: onClose })
        }} className="space-y-4">
          <div className="space-y-2">
            <Label>审核结果</Label>
            <RadioGroup value={approved} onValueChange={setApproved}>
              <div className="flex items-center gap-2"><RadioGroupItem value="true" /><Label>通过</Label></div>
              <div className="flex items-center gap-2"><RadioGroupItem value="false" /><Label>拒绝</Label></div>
            </RadioGroup>
          </div>
          <div className="space-y-2"><Label>备注</Label><Textarea value={reason} onChange={(e) => setReason(e.target.value)} rows={2} /></div>
          {mutation.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(mutation.error)}</AlertDescription></Alert>}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose}>取消</Button>
            <Button type="submit" disabled={mutation.isPending}>{mutation.isPending ? "提交中…" : "确认"}</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
