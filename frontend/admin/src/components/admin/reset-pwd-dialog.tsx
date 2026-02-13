import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

import { type AdminView } from "@/api/modules/auth"
import { useResetAdminPasswordMutation } from "@/hooks/admin/use-admin-account-hooks"

export function ResetPwdDialog({
  admin,
  newPassword,
  setNewPassword,
  onClose,
}: {
  admin: AdminView
  newPassword: string
  setNewPassword: (v: string) => void
  onClose: () => void
}) {
  const resetMutation = useResetAdminPasswordMutation(admin.admin_id)
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>重置密码 - {admin.username}</DialogTitle>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault()
            resetMutation.mutate({ new_password: newPassword }, { onSuccess: onClose })
          }}
          className="space-y-4"
        >
          <div className="space-y-2">
            <Label>新密码</Label>
            <Input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} required />
          </div>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose}>取消</Button>
            <Button type="submit" disabled={resetMutation.isPending}>
              {resetMutation.isPending ? "重置中…" : "确认重置"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
