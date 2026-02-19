import { useState } from "react"
import { useNavigate } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useLoginMutation, useRegisterMutation, useUserSessionState } from "@/hooks/user/use-auth-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

export function LoginPage() {
  const navigate = useNavigate()
  const { isAuthed } = useUserSessionState()
  const loginMutation = useLoginMutation()
  const registerMutation = useRegisterMutation()
  const { toUserMessage } = useApiError()

  const [loginPhone, setLoginPhone] = useState("")
  const [loginPassword, setLoginPassword] = useState("")
  const [regPhone, setRegPhone] = useState("")
  const [regPassword, setRegPassword] = useState("")
  const [regNickname, setRegNickname] = useState("")

  if (isAuthed) {
    navigate("/", { replace: true })
    return null
  }

  const handleLogin = (e: React.FormEvent) => {
    e.preventDefault()
    loginMutation.mutate(
      { phone: loginPhone, password: loginPassword },
      { onSuccess: () => navigate("/", { replace: true }) }
    )
  }

  const handleRegister = (e: React.FormEvent) => {
    e.preventDefault()
    registerMutation.mutate(
      { phone: regPhone, password: regPassword, nickname: regNickname },
      { onSuccess: () => navigate("/", { replace: true }) }
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-xl">FlashSale</CardTitle>
          <p className="text-sm text-muted-foreground">欢迎使用秒杀商城</p>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="login">
            <TabsList className="w-full">
              <TabsTrigger value="login" className="flex-1">登录</TabsTrigger>
              <TabsTrigger value="register" className="flex-1">注册</TabsTrigger>
            </TabsList>

            <TabsContent value="login">
              <form onSubmit={handleLogin} className="space-y-4 pt-2">
                <div className="space-y-2">
                  <Label htmlFor="login-phone">手机号</Label>
                  <Input id="login-phone" value={loginPhone} onChange={(e) => setLoginPhone(e.target.value)}
                    placeholder="13800138000" required autoFocus />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="login-pwd">密码</Label>
                  <Input id="login-pwd" type="password" value={loginPassword}
                    onChange={(e) => setLoginPassword(e.target.value)} placeholder="••••••••" required />
                </div>
                {loginMutation.isError && (
                  <Alert variant="destructive"><AlertDescription>{toUserMessage(loginMutation.error)}</AlertDescription></Alert>
                )}
                <Button type="submit" className="w-full" disabled={loginMutation.isPending}>
                  {loginMutation.isPending ? "登录中…" : "登录"}
                </Button>
              </form>
            </TabsContent>

            <TabsContent value="register">
              <form onSubmit={handleRegister} className="space-y-4 pt-2">
                <div className="space-y-2">
                  <Label htmlFor="reg-phone">手机号</Label>
                  <Input id="reg-phone" value={regPhone} onChange={(e) => setRegPhone(e.target.value)}
                    placeholder="13800138000" required />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="reg-pwd">密码</Label>
                  <Input id="reg-pwd" type="password" value={regPassword}
                    onChange={(e) => setRegPassword(e.target.value)} placeholder="••••••••" required />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="reg-nick">昵称</Label>
                  <Input id="reg-nick" value={regNickname} onChange={(e) => setRegNickname(e.target.value)}
                    placeholder="可选" />
                </div>
                {registerMutation.isError && (
                  <Alert variant="destructive"><AlertDescription>{toUserMessage(registerMutation.error)}</AlertDescription></Alert>
                )}
                <Button type="submit" className="w-full" disabled={registerMutation.isPending}>
                  {registerMutation.isPending ? "注册中…" : "注册"}
                </Button>
              </form>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  )
}
