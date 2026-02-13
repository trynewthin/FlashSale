import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Textarea } from "@/components/ui/textarea"

import { useCancelOrderMutation } from "@/hooks/user/use-order-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

export function CancelDialog({ orderId, onClose }: { orderId: string; onClose: () => void }) {
  const mutation = useCancelOrderMutation(orderId)
  const { toUserMessage } = useApiError()
  const [reason, setReason] = useState("")
  return (
    <AlertDialog open onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>取消订单</AlertDialogTitle>
          <AlertDialogDescription>确定要取消此订单吗？</AlertDialogDescription>
        </AlertDialogHeader>
        <div className="px-4"><Textarea placeholder="取消原因（可选）" value={reason} onChange={(e) => setReason(e.target.value)} rows={2} /></div>
        {mutation.isError && <Alert variant="destructive" className="mx-6"><AlertDescription>{toUserMessage(mutation.error)}</AlertDescription></Alert>}
        <AlertDialogFooter>
          <AlertDialogCancel>返回</AlertDialogCancel>
          <AlertDialogAction onClick={() => mutation.mutate({ reason }, { onSuccess: onClose })} disabled={mutation.isPending}>
            {mutation.isPending ? "取消中…" : "确认取消"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
