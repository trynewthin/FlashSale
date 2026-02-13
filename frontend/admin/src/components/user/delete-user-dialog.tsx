import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import { useManagedUserDeleteMutation } from "@/hooks/biz/use-user-mgmt-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

export function DeleteUserDialog({ userId, onClose, onDeleted }: { userId: string; onClose: () => void; onDeleted: () => void }) {
  const mutation = useManagedUserDeleteMutation(userId)
  const { toUserMessage } = useApiError()
  return (
    <AlertDialog open onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除用户</AlertDialogTitle>
          <AlertDialogDescription>此操作不可逆，确定要删除用户 {userId} 吗？</AlertDialogDescription>
        </AlertDialogHeader>
        {mutation.isError && <Alert variant="destructive" className="mx-6"><AlertDescription>{toUserMessage(mutation.error)}</AlertDescription></Alert>}
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction onClick={() => mutation.mutate(undefined, { onSuccess: onDeleted })} disabled={mutation.isPending}>
            {mutation.isPending ? "删除中…" : "确认删除"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
