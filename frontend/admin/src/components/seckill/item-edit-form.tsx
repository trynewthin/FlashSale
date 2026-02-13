import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"

import { type ActivityItemAdmin } from "@/api/modules/seckill"
import { useUpdateSeckillItemMutation } from "@/hooks/biz/use-seckill-mgmt-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { ItemFormFields, type ItemFormData } from "./item-form-fields"

export function ItemEditForm({ activityId, item, formData, setFormData, onClose }: {
  activityId: string; item: ActivityItemAdmin; formData: ItemFormData; setFormData: (d: ItemFormData) => void; onClose: () => void
}) {
  const updateMutation = useUpdateSeckillItemMutation(activityId, item.item_id)
  const { toUserMessage } = useApiError()
  return (
    <form onSubmit={(e) => { e.preventDefault(); updateMutation.mutate(formData, { onSuccess: onClose }) }} className="space-y-4">
      <ItemFormFields formData={formData} setFormData={setFormData} />
      {updateMutation.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(updateMutation.error)}</AlertDescription></Alert>}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onClose}>取消</Button>
        <Button type="submit" disabled={updateMutation.isPending}>{updateMutation.isPending ? "保存中…" : "保存"}</Button>
      </div>
    </form>
  )
}
