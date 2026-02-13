import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"

import { type ActivityAdmin } from "@/api/modules/seckill"
import { useUpdateSeckillActivityMutation } from "@/hooks/biz/use-seckill-mgmt-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { ActivityFormFields, type ActivityFormData } from "./activity-form-fields"

export function ActivityEditForm({ activity, formData, setFormData, startStr, setStartStr, endStr, setEndStr, onClose }: {
  activity: ActivityAdmin; formData: ActivityFormData; setFormData: (d: ActivityFormData) => void
  startStr: string; setStartStr: (s: string) => void; endStr: string; setEndStr: (s: string) => void; onClose: () => void
}) {
  const updateMutation = useUpdateSeckillActivityMutation(activity.activity_id)
  const { toUserMessage } = useApiError()
  return (
    <form onSubmit={(e) => {
      e.preventDefault()
      updateMutation.mutate({
        ...formData,
        start_at_unix: startStr ? Math.floor(new Date(startStr).getTime() / 1000) : 0,
        end_at_unix: endStr ? Math.floor(new Date(endStr).getTime() / 1000) : 0,
      }, { onSuccess: onClose })
    }} className="space-y-4">
      <ActivityFormFields formData={formData} setFormData={setFormData}
        startStr={startStr} setStartStr={setStartStr} endStr={endStr} setEndStr={setEndStr} />
      {updateMutation.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(updateMutation.error)}</AlertDescription></Alert>}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onClose}>取消</Button>
        <Button type="submit" disabled={updateMutation.isPending}>{updateMutation.isPending ? "保存中…" : "保存"}</Button>
      </div>
    </form>
  )
}
