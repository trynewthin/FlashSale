import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import { type AdminView } from "@/api/modules/auth"
import { useDeleteAdminMutation } from "@/hooks/admin/use-admin-account-hooks"

export function DeleteAdminDialog({ admin, onClose }: { admin: AdminView; onClose: () => void }) {
  const deleteMutation = useDeleteAdminMutation(admin.admin_id)
  return (
    <AlertDialog open onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除管理员</AlertDialogTitle>
          <AlertDialogDescription>
            此操作不可逆，确定要删除管理员 "{admin.username}" 吗？
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction
            onClick={() => deleteMutation.mutate(undefined, { onSuccess: onClose })}
            disabled={deleteMutation.isPending}
          >
            {deleteMutation.isPending ? "删除中…" : "确认删除"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
