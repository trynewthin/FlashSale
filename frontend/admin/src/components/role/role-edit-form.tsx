import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

import { type RoleView } from "@/api/modules/admin"
import { useUpdateRoleMutation } from "@/hooks/admin/use-role-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

export function RoleEditForm({ role, onClose }: { role: RoleView; onClose: () => void }) {
  const updateMutation = useUpdateRoleMutation(role.role_id)
  const { toUserMessage } = useApiError()
  const [roleName, setRoleName] = useState(role.role_name)
  const [status, setStatus] = useState(role.status)

  return (
    <form onSubmit={(e) => { e.preventDefault(); updateMutation.mutate({ role_name: roleName, status }, { onSuccess: onClose }) }} className="space-y-4">
      <div className="space-y-2"><Label>角色名称</Label><Input value={roleName} onChange={(e) => setRoleName(e.target.value)} required /></div>
      <div className="space-y-2">
        <Label>状态</Label>
        <Select value={String(status)} onValueChange={(v) => v && setStatus(Number(v))}
          items={[{ value: "1", label: "启用" }, { value: "2", label: "禁用" }]}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent><SelectItem value="1">启用</SelectItem><SelectItem value="2">禁用</SelectItem></SelectContent>
        </Select>
      </div>
      {updateMutation.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(updateMutation.error)}</AlertDescription></Alert>}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onClose}>取消</Button>
        <Button type="submit" disabled={updateMutation.isPending}>{updateMutation.isPending ? "保存中…" : "保存"}</Button>
      </div>
    </form>
  )
}
