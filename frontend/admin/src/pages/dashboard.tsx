import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useAdminPermission } from "@/hooks/auth/use-admin-permission"
import { useAdminSessionState } from "@/hooks/auth/use-admin-session"
import { useOpsPingQuery } from "@/hooks/biz/use-ops-hooks"
import { RefreshCw } from "lucide-react"

export function DashboardPage() {
  const { profile } = useAdminSessionState()
  const { domains, dataScope } = useAdminPermission()
  const pingQuery = useOpsPingQuery()

  return (
    <div className="space-y-6 p-6">
      <h1 className="text-2xl font-semibold tracking-tight">运营工作台</h1>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">权限概览</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <p className="mb-1 text-sm text-muted-foreground">当前管理员</p>
              <p className="font-medium">{profile?.display_name || profile?.username || "-"}</p>
            </div>
            <div>
              <p className="mb-1 text-sm text-muted-foreground">数据范围</p>
              <Badge variant={dataScope === "all" ? "default" : "secondary"}>{dataScope}</Badge>
            </div>
            <div>
              <p className="mb-1 text-sm text-muted-foreground">权限域</p>
              <div className="flex flex-wrap gap-1">
                {domains.length === 0 && (
                  <span className="text-sm text-muted-foreground">无权限域</span>
                )}
                {domains.map((d) => (
                  <Badge key={d} variant="outline">
                    {d}
                  </Badge>
                ))}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">环境探针</CardTitle>
            <Button
              variant="ghost"
              size="icon"
              className="size-7"
              onClick={() => pingQuery.refetch()}
              disabled={pingQuery.isFetching}
            >
              <RefreshCw className={pingQuery.isFetching ? "size-4 animate-spin" : "size-4"} />
            </Button>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">状态</span>
              {pingQuery.isLoading ? (
                <Badge variant="secondary">检测中…</Badge>
              ) : pingQuery.isError ? (
                <Badge variant="destructive">异常</Badge>
              ) : (
                <Badge variant="default">正常</Badge>
              )}
            </div>
            {pingQuery.data && (
              <p className="text-sm text-muted-foreground">{pingQuery.data.message}</p>
            )}
            {pingQuery.isError && (
              <p className="text-sm text-destructive">
                探针请求失败，请检查网络或后端服务状态。
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
