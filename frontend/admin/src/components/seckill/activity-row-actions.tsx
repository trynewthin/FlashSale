import { Button } from "@/components/ui/button"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { MoreHorizontal } from "lucide-react"

import { type ActivityAdmin } from "@/api/modules/seckill"
import { usePublishSeckillActivityMutation, useOfflineSeckillActivityMutation } from "@/hooks/biz/use-seckill-mgmt-hooks"

export function ActivityRowActions({ activity, onEdit, onDelete }: { activity: ActivityAdmin; onEdit: () => void; onDelete: () => void }) {
  const publishMutation = usePublishSeckillActivityMutation(activity.activity_id)
  const offlineMutation = useOfflineSeckillActivityMutation(activity.activity_id)
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-7" />}>
        <MoreHorizontal className="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onEdit}>编辑</DropdownMenuItem>
        {activity.status === 1 && (
          <DropdownMenuItem onClick={() => publishMutation.mutate()}>发布</DropdownMenuItem>
        )}
        {activity.status === 2 && (
          <DropdownMenuItem onClick={() => offlineMutation.mutate()}>下线</DropdownMenuItem>
        )}
        <DropdownMenuItem className="text-destructive" onClick={onDelete}>删除</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
