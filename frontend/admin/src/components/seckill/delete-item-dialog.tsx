import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import { type ActivityItemAdmin } from "@/api/modules/seckill"
import { useRemoveSeckillItemMutation } from "@/hooks/biz/use-seckill-mgmt-hooks"

export function DeleteItemDialog({ activityId, item, onClose }: { activityId: string; item: ActivityItemAdmin; onClose: () => void }) {
  const removeMutation = useRemoveSeckillItemMutation(activityId, item.item_id)
  return (
    <AlertDialog open onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认移除商品</AlertDialogTitle>
          <AlertDialogDescription>确定要从活动中移除商品 "{item.snapshot_name}" 吗？</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction onClick={() => removeMutation.mutate(undefined, { onSuccess: onClose })} disabled={removeMutation.isPending}>
            {removeMutation.isPending ? "移除中…" : "确认移除"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
