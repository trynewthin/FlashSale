import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

import { useUpdateNicknameMutation } from "@/hooks/user/use-auth-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

export function NicknameDialog({ currentNickname, onClose }: { currentNickname: string; onClose: () => void }) {
  const mutation = useUpdateNicknameMutation()
  const { toUserMessage } = useApiError()
  const [nickname, setNickname] = useState(currentNickname)

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader><DialogTitle>修改昵称</DialogTitle></DialogHeader>
        <form onSubmit={(e) => { e.preventDefault(); mutation.mutate({ nickname }, { onSuccess: onClose }) }} className="space-y-4">
          <div className="space-y-2"><Label>新昵称</Label><Input value={nickname} onChange={(e) => setNickname(e.target.value)} required /></div>
          {mutation.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(mutation.error)}</AlertDescription></Alert>}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose}>取消</Button>
            <Button type="submit" disabled={mutation.isPending}>{mutation.isPending ? "保存中…" : "保存"}</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
