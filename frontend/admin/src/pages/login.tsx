import { useState } from "react"
import { useNavigate } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { useAdminLoginMutation, useAdminSessionState } from "@/hooks/auth/use-admin-session"
import { useApiError } from "@/hooks/common/use-api-error"

export function LoginPage() {
  const navigate = useNavigate()
  const { isAuthed } = useAdminSessionState()
  const loginMutation = useAdminLoginMutation()
  const { toUserMessage } = useApiError()

  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")

  if (isAuthed) {
    navigate("/", { replace: true })
    return null
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    loginMutation.mutate(
      { username, password },
      { onSuccess: () => navigate("/", { replace: true }) }
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-xl">FlashSale 管理端</CardTitle>
          <p className="text-sm text-muted-foreground">请输入管理员账号登录</p>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">用户名</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="admin"
                required
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">密码</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                required
              />
            </div>
            {loginMutation.isError && (
              <Alert variant="destructive">
                <AlertDescription>{toUserMessage(loginMutation.error)}</AlertDescription>
              </Alert>
            )}
            <Button type="submit" className="w-full" disabled={loginMutation.isPending}>
              {loginMutation.isPending ? "登录中…" : "登录"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
