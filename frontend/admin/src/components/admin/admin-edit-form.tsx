import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

import { type AdminView } from "@/api/modules/auth"
import { useUpdateAdminMutation } from "@/hooks/admin/use-admin-account-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

export function AdminEditForm({ admin, onClose }: { admin: AdminView; onClose: () => void }) {
  const updateMutation = useUpdateAdminMutation(admin.admin_id)
  const { toUserMessage } = useApiError()
  const [displayName, setDisplayName] = useState(admin.display_name)
  const [dataScope, setDataScope] = useState<"all" | "self">(admin.data_scope as "all" | "self")

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        updateMutation.mutate({ display_name: displayName, data_scope: dataScope }, { onSuccess: onClose })
      }}
      className="space-y-4"
    >
      <div className="space-y-2">
        <Label>昵称</Label>
        <Input value={displayName} onChange={(e) => setDisplayName(e.target.value)} required />
      </div>
      <div className="space-y-2">
        <Label>数据范围</Label>
        <Select value={dataScope} onValueChange={(v) => v && setDataScope(v as "all" | "self")}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">all</SelectItem>
            <SelectItem value="self">self</SelectItem>
          </SelectContent>
        </Select>
      </div>
      {updateMutation.isError && (
        <Alert variant="destructive">
          <AlertDescription>{toUserMessage(updateMutation.error)}</AlertDescription>
        </Alert>
      )}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onClose}>取消</Button>
        <Button type="submit" disabled={updateMutation.isPending}>
          {updateMutation.isPending ? "保存中…" : "保存"}
        </Button>
      </div>
    </form>
  )
}
