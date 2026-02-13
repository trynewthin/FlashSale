import { Button } from "@/components/ui/button"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { MoreHorizontal } from "lucide-react"

import { type RoleView } from "@/api/modules/admin"

export function RoleRowActions({ onEdit, onDelete, onDomains }: {
  role: RoleView; onEdit: () => void; onDelete: () => void; onDomains: () => void
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-7" />}>
        <MoreHorizontal className="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onEdit}>编辑</DropdownMenuItem>
        <DropdownMenuItem onClick={onDomains}>领域配置</DropdownMenuItem>
        <DropdownMenuItem className="text-destructive" onClick={onDelete}>删除</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
