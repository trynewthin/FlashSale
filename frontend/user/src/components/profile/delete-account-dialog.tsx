import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import { useDeleteUserMutation } from "@/hooks/user/use-auth-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

export function DeleteAccountDialog({ onClose, onDeleted }: { onClose: () => void; onDeleted: () => void }) {
  const mutation = useDeleteUserMutation()
  const { toUserMessage } = useApiError()
  return (
    <AlertDialog open onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>注销账号</AlertDialogTitle>
          <AlertDialogDescription>此操作不可逆，注销后所有数据将被删除。确定要继续吗？</AlertDialogDescription>
        </AlertDialogHeader>
        {mutation.isError && <Alert variant="destructive" className="mx-6"><AlertDescription>{toUserMessage(mutation.error)}</AlertDescription></Alert>}
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction onClick={() => mutation.mutate(undefined, { onSuccess: onDeleted })} disabled={mutation.isPending}>
            {mutation.isPending ? "注销中…" : "确认注销"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
