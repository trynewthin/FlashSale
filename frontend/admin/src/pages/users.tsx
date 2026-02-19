import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Search, UserX, Pencil } from "lucide-react"

import { useManagedUserProfileQuery } from "@/hooks/biz/use-user-mgmt-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { formatUnix } from "@/lib/format"

import { NicknameDialog, DeleteUserDialog } from "@/components/user"


export function UserManagementPage() {
  const { toUserMessage } = useApiError()
  const [userId, setUserId] = useState("")
  const [queryUserId, setQueryUserId] = useState<string | undefined>(undefined)
  const [nicknameOpen, setNicknameOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const profileQuery = useManagedUserProfileQuery(queryUserId)
  const user = profileQuery.data?.user

  const handleSearch = () => {
    if (userId.trim()) setQueryUserId(userId.trim())
  }

  return (
    <div className="space-y-4 p-6">
      <h1 className="text-2xl font-semibold tracking-tight">用户管理</h1>

      <Card>
        <CardHeader><CardTitle className="text-base">用户查询</CardTitle></CardHeader>
        <CardContent>
          <div className="flex items-end gap-3">
            <div className="space-y-1">
              <Label className="text-xs">用户 ID</Label>
              <Input placeholder="输入 user_id" value={userId} onChange={(e) => setUserId(e.target.value)} className="w-48"
                onKeyDown={(e) => e.key === "Enter" && handleSearch()} />
            </div>
            <Button size="sm" onClick={handleSearch}><Search className="mr-1 size-4" />查询</Button>
          </div>
        </CardContent>
      </Card>

      {profileQuery.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(profileQuery.error)}</AlertDescription></Alert>}

      {user && (
        <Card>
          <CardHeader><CardTitle className="text-base">用户信息</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <div className="grid grid-cols-2 gap-4">
              <div><p className="text-xs text-muted-foreground">用户 ID</p><p className="text-sm font-mono">{user.user_id}</p></div>
              <div><p className="text-xs text-muted-foreground">手机号</p><p className="text-sm">{user.phone}</p></div>
              <div><p className="text-xs text-muted-foreground">昵称</p><p className="text-sm">{user.nickname || "-"}</p></div>
              <div><p className="text-xs text-muted-foreground">创建时间</p><p className="text-sm">{formatUnix(user.created_at_unix)}</p></div>
            </div>
            <div className="flex gap-2 pt-2">
              <Button size="sm" variant="outline" onClick={() => setNicknameOpen(true)}>
                <Pencil className="mr-1 size-3" />改昵称
              </Button>
              <Button size="sm" variant="destructive" onClick={() => setDeleteOpen(true)}>
                <UserX className="mr-1 size-3" />删除用户
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {nicknameOpen && queryUserId && (
        <NicknameDialog userId={queryUserId} currentNickname={user?.nickname ?? ""} onClose={() => setNicknameOpen(false)} />
      )}

      {deleteOpen && queryUserId && (
        <DeleteUserDialog userId={queryUserId} onClose={() => setDeleteOpen(false)} onDeleted={() => { setDeleteOpen(false); setQueryUserId(undefined) }} />
      )}
    </div>
  )
}

