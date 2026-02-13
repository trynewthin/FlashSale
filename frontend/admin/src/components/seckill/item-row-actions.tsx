import { Button } from "@/components/ui/button"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { MoreHorizontal } from "lucide-react"

import { type ActivityItemAdmin } from "@/api/modules/seckill"

export function ItemRowActions({ onEdit, onDelete }: {
  activityId: string; item: ActivityItemAdmin; onEdit: () => void; onDelete: () => void
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-7" />}>
        <MoreHorizontal className="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onEdit}>编辑</DropdownMenuItem>
        <DropdownMenuItem className="text-destructive" onClick={onDelete}>移除</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
