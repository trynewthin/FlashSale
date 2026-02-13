import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import { type ActivityAdmin } from "@/api/modules/seckill"
import { useDeleteSeckillActivityMutation } from "@/hooks/biz/use-seckill-mgmt-hooks"

export function DeleteActivityDialog({ activity, onClose }: { activity: ActivityAdmin; onClose: () => void }) {
  const deleteMutation = useDeleteSeckillActivityMutation(activity.activity_id)
  return (
    <AlertDialog open onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除活动</AlertDialogTitle>
          <AlertDialogDescription>此操作不可逆，确定要删除活动 "{activity.title}" 吗？</AlertDialogDescription>
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
