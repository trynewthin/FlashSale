import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"

import { useConfirmPaymentAndInfoMutation } from "@/hooks/user/use-order-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

export function PayDialog({ orderId, onClose }: { orderId: string; onClose: () => void }) {
  const mutation = useConfirmPaymentAndInfoMutation(orderId)
  const { toUserMessage } = useApiError()
  const [form, setForm] = useState({
    pay_channel: "mock_pay",
    receiver_name: "",
    receiver_phone: "",
    receiver_address: "",
    buyer_remark: "",
  })

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader><DialogTitle>支付 & 填写收货信息</DialogTitle></DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault()
            mutation.mutate(
              {
                ...form,
                pay_reference: `PAY_${Date.now()}`,
              },
              { onSuccess: onClose },
            )
          }}
          className="space-y-3"
        >
          <div className="space-y-2"><Label>收件人</Label><Input value={form.receiver_name} onChange={(e) => setForm({ ...form, receiver_name: e.target.value })} required /></div>
          <div className="space-y-2"><Label>手机号</Label><Input value={form.receiver_phone} onChange={(e) => setForm({ ...form, receiver_phone: e.target.value })} required /></div>
          <div className="space-y-2"><Label>地址</Label><Input value={form.receiver_address} onChange={(e) => setForm({ ...form, receiver_address: e.target.value })} required /></div>
          <div className="space-y-2"><Label>备注</Label><Textarea value={form.buyer_remark} onChange={(e) => setForm({ ...form, buyer_remark: e.target.value })} rows={2} /></div>
          {mutation.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(mutation.error)}</AlertDescription></Alert>}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose}>取消</Button>
            <Button type="submit" disabled={mutation.isPending}>{mutation.isPending ? "提交中…" : "确认支付"}</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
