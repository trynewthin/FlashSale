import { useState } from "react"
import { useNavigate } from "react-router-dom"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Zap } from "lucide-react"

import { useSeckillActivitiesQuery } from "@/hooks/user/use-seckill-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

function formatUnix(unix: number) {
  if (!unix) return "-"
  return new Date(unix * 1000).toLocaleString("zh-CN")
}

function getActivityStatus(startUnix: number, endUnix: number): { label: string; variant: "default" | "secondary" | "destructive" } {
  const now = Math.floor(Date.now() / 1000)
  if (now < startUnix) return { label: "未开始", variant: "secondary" }
  if (now > endUnix) return { label: "已结束", variant: "destructive" }
  return { label: "进行中", variant: "default" }
}

export function SeckillListPage() {
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()
  const [page, setPage] = useState(1)

  const listQuery = useSeckillActivitiesQuery({ page, page_size: 10 })

  const items = listQuery.data?.list ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / 10)

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Zap className="size-5 text-primary" />
        <h1 className="text-xl font-semibold">秒杀活动</h1>
      </div>

      {listQuery.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(listQuery.error)}</AlertDescription></Alert>}
      {listQuery.isLoading && <p className="text-center text-muted-foreground py-12">加载中…</p>}

      <div className="space-y-3">
        {items.map((activity) => {
          const status = getActivityStatus(activity.start_at_unix, activity.end_at_unix)
          return (
            <Card key={activity.activity_id} className="cursor-pointer transition-shadow hover:shadow-md"
              onClick={() => navigate(`/seckill/${activity.activity_id}`)}>
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between">
                  <CardTitle className="text-base">{activity.title}</CardTitle>
                  <Badge variant={status.variant}>{status.label}</Badge>
                </div>
              </CardHeader>
              <CardContent className="space-y-2">
                {activity.description && <p className="text-sm text-muted-foreground line-clamp-2">{activity.description}</p>}
                <div className="flex items-center gap-4 text-xs text-muted-foreground">
                  <span>开始: {formatUnix(activity.start_at_unix)}</span>
                  <span>结束: {formatUnix(activity.end_at_unix)}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground">商品数: {activity.items?.length ?? 0}</span>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>

      {items.length === 0 && !listQuery.isLoading && (
        <p className="text-center text-muted-foreground py-12">暂无秒杀活动</p>
      )}

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-4">
          <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</Button>
          <span className="text-sm text-muted-foreground">{page} / {totalPages}</span>
          <Button size="sm" variant="outline" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</Button>
        </div>
      )}
    </div>
  )
}
