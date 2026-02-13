import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"

import { type AdminView } from "@/api/modules/auth"
import { useBindAdminRolesMutation } from "@/hooks/admin/use-admin-account-hooks"

export function BindRolesDialog({
  admin,
  roles,
  selectedRoleIds,
  setSelectedRoleIds,
  onClose,
}: {
  admin: AdminView
  roles: { role_id: string; role_code: string; role_name: string }[]
  selectedRoleIds: string[]
  setSelectedRoleIds: (ids: string[]) => void
  onClose: () => void
}) {
  const bindMutation = useBindAdminRolesMutation(admin.admin_id)
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>绑定角色 - {admin.username}</DialogTitle>
        </DialogHeader>
        <ScrollArea className="max-h-64">
          <div className="space-y-2 p-1">
            {roles.map((role) => {
              const id = String(role.role_id)
              const checked = selectedRoleIds.includes(id)
              return (
                <label key={role.role_id} className="flex items-center gap-2 rounded p-2 hover:bg-accent">
                  <Checkbox
                    checked={checked}
                    onCheckedChange={(c) =>
                      setSelectedRoleIds(c ? [...selectedRoleIds, id] : selectedRoleIds.filter((r) => r !== id))
                    }
                  />
                  <span className="text-sm">{role.role_name}</span>
                  <Badge variant="outline" className="ml-auto text-xs">{role.role_code}</Badge>
                </label>
              )
            })}
          </div>
        </ScrollArea>
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="outline" onClick={onClose}>取消</Button>
          <Button
            onClick={() => bindMutation.mutate({ role_ids: selectedRoleIds }, { onSuccess: onClose })}
            disabled={bindMutation.isPending}
          >
            {bindMutation.isPending ? "保存中…" : "确认绑定"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
