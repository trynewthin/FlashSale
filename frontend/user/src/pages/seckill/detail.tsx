import { useParams, useNavigate } from "react-router-dom"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { ArrowLeft } from "lucide-react"

import { useSeckillActivityDetailQuery } from "@/hooks/user/use-seckill-hooks"
import { useApiError } from "@/hooks/common/use-api-error"

import { SeckillItemCard } from "@/components/seckill"

function formatUnix(unix: number) { if (!unix) return "-"; return new Date(unix * 1000).toLocaleString("zh-CN") }

function getActivityStatus(startUnix: number, endUnix: number): { label: string; variant: "default" | "secondary" | "destructive"; active: boolean } {
  const now = Math.floor(Date.now() / 1000)
  if (now < startUnix) return { label: "未开始", variant: "secondary", active: false }
  if (now > endUnix) return { label: "已结束", variant: "destructive", active: false }
  return { label: "进行中", variant: "default", active: true }
}

export function SeckillDetailPage() {
  const { activityId } = useParams<{ activityId: string }>()
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()

  const detailQuery = useSeckillActivityDetailQuery(activityId)
  const activity = detailQuery.data?.activity

  if (detailQuery.isLoading) {
    return <div className="flex items-center justify-center py-12 text-muted-foreground">加载中…</div>
  }

  if (detailQuery.isError) {
    return <Alert variant="destructive"><AlertDescription>{toUserMessage(detailQuery.error)}</AlertDescription></Alert>
  }

  if (!activity) return null

  const status = getActivityStatus(activity.start_at_unix, activity.end_at_unix)

  return (
    <div className="space-y-4">
      <Button variant="ghost" size="sm" onClick={() => navigate("/seckill")}>
        <ArrowLeft className="mr-1 size-4" />返回活动列表
      </Button>

      <Card>
        <CardHeader>
          <div className="flex items-start justify-between">
            <div className="space-y-1">
              <CardTitle className="text-lg">{activity.title}</CardTitle>
              {activity.description && <p className="text-sm text-muted-foreground">{activity.description}</p>}
            </div>
            <Badge variant={status.variant}>{status.label}</Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <div className="flex gap-4 text-muted-foreground">
            <span>开始: {formatUnix(activity.start_at_unix)}</span>
            <span>结束: {formatUnix(activity.end_at_unix)}</span>
          </div>
        </CardContent>
      </Card>

      <Separator />

      <h2 className="text-base font-medium">秒杀商品</h2>

      {(!activity.items || activity.items.length === 0) && (
        <p className="text-center text-muted-foreground py-8">暂无秒杀商品</p>
      )}

      <div className="space-y-3">
        {activity.items?.map((item) => (
          <SeckillItemCard
            key={item.item_id}
            activityId={activityId!}
            item={item}
            active={status.active}
          />
        ))}
      </div>
    </div>
  )
}

