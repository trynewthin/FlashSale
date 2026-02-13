import { useState } from "react"
import { useNavigate } from "react-router-dom"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Pencil, Trash2 } from "lucide-react"

import { useUserSessionState, useUserProfileQuery, useLogoutAction } from "@/hooks/user/use-auth-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

import { NicknameDialog, DeleteAccountDialog } from "@/components/profile"

export function ProfilePage() {
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()
  const { profile: sessionProfile } = useUserSessionState()
  const logout = useLogoutAction()

  const profileQuery = useUserProfileQuery()
  const user = profileQuery.data

  const [nicknameOpen, setNicknameOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">个人中心</h1>

      {profileQuery.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(profileQuery.error)}</AlertDescription></Alert>}

      <Card>
        <CardHeader><CardTitle className="text-base">账号信息</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          {profileQuery.isLoading ? (
            <p className="text-muted-foreground">加载中…</p>
          ) : user ? (
            <div className="grid grid-cols-2 gap-4">
              <div><p className="text-xs text-muted-foreground">用户 ID</p><p className="text-sm font-mono">{user.user_id}</p></div>
              <div><p className="text-xs text-muted-foreground">手机号</p><p className="text-sm">{user.phone}</p></div>
              <div><p className="text-xs text-muted-foreground">昵称</p><p className="text-sm">{user.nickname || "-"}</p></div>
            </div>
          ) : (
            <p className="text-muted-foreground">无法加载用户信息</p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle className="text-base">账号操作</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <Button size="sm" variant="outline" onClick={() => setNicknameOpen(true)}>
            <Pencil className="mr-1 size-3" />修改昵称
          </Button>
          <Separator />
          <Button size="sm" variant="destructive" onClick={() => setDeleteOpen(true)}>
            <Trash2 className="mr-1 size-3" />注销账号
          </Button>
        </CardContent>
      </Card>

      {nicknameOpen && (
        <NicknameDialog currentNickname={user?.nickname ?? sessionProfile?.nickname ?? ""} onClose={() => setNicknameOpen(false)} />
      )}

      {deleteOpen && (
        <DeleteAccountDialog onClose={() => setDeleteOpen(false)} onDeleted={() => { logout(); navigate("/login") }} />
      )}
    </div>
  )
}

