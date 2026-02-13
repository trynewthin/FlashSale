import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import { useConfirmReceiptMutation } from "@/hooks/user/use-order-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

export function ConfirmReceiptDialog({ orderId, onClose }: { orderId: string; onClose: () => void }) {
  const mutation = useConfirmReceiptMutation(orderId)
  const { toUserMessage } = useApiError()
  return (
    <AlertDialog open onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认收货</AlertDialogTitle>
          <AlertDialogDescription>确认已收到商品？</AlertDialogDescription>
        </AlertDialogHeader>
        {mutation.isError && <Alert variant="destructive" className="mx-6"><AlertDescription>{toUserMessage(mutation.error)}</AlertDescription></Alert>}
        <AlertDialogFooter>
          <AlertDialogCancel>返回</AlertDialogCancel>
          <AlertDialogAction onClick={() => mutation.mutate(undefined, { onSuccess: onClose })} disabled={mutation.isPending}>
            {mutation.isPending ? "确认中…" : "确认收货"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
