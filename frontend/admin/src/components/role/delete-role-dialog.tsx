import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import { type RoleView } from "@/api/modules/admin"
import { useDeleteRoleMutation } from "@/hooks/admin/use-role-hooks"

export function DeleteRoleDialog({ role, onClose }: { role: RoleView; onClose: () => void }) {
  const deleteMutation = useDeleteRoleMutation(role.role_id)
  return (
    <AlertDialog open onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除角色</AlertDialogTitle>
          <AlertDialogDescription>此操作不可逆，确定要删除角色 "{role.role_name}" 吗？</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction onClick={() => deleteMutation.mutate(undefined, { onSuccess: onClose })} disabled={deleteMutation.isPending}>
            {deleteMutation.isPending ? "删除中…" : "确认删除"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
