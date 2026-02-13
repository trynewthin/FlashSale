import { Button } from "@/components/ui/button"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { MoreHorizontal } from "lucide-react"

import { type AdminView } from "@/api/modules/auth"
import { useSetAdminStatusMutation } from "@/hooks/admin/use-admin-account-hooks"

export function AdminRowActions({
  admin,
  onEdit,
  onDelete,
  onResetPwd,
  onBindRoles,
}: {
  admin: AdminView
  onEdit: () => void
  onDelete: () => void
  onResetPwd: () => void
  onBindRoles: () => void
}) {
  const statusMutation = useSetAdminStatusMutation(admin.admin_id)

  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-7" />}>
        <MoreHorizontal className="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onEdit}>编辑</DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => statusMutation.mutate({ status: admin.status === 1 ? 2 : 1 })}
        >
          {admin.status === 1 ? "禁用" : "启用"}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={onResetPwd}>重置密码</DropdownMenuItem>
        <DropdownMenuItem onClick={onBindRoles}>绑定角色</DropdownMenuItem>
        <DropdownMenuItem className="text-destructive" onClick={onDelete}>
          删除
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
