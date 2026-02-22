import { RefreshCcw } from "lucide-react"
import { useCallback, useEffect, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import { LogStreamViewer } from "@/components/common/log-stream-viewer"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { useOpsApiError } from "@/hooks/use-ops-api-error"

interface ReplicaLogSheetProps {
  open: boolean
  serviceName: string
  containerName: string
  onOpenChange: (open: boolean) => void
}

// ReplicaLogSheet 在容器编排页内以 Sheet 形式查看副本日志。
// 直接使用 docker logs 获取指定容器的日志，而非文件匹配。
export function ReplicaLogSheet({
  open,
  serviceName,
  containerName,
  onOpenChange,
}: ReplicaLogSheetProps) {
  const showApiError = useOpsApiError()

  const [tailLines, setTailLines] = useState("200")
  const [logText, setLogText] = useState("")
  const [loading, setLoading] = useState(false)
  const [hint, setHint] = useState("")

  const loadLogs = useCallback(async () => {
    if (!open || !containerName) {
      return
    }
    const parsedLines = Number.parseInt(tailLines, 10)
    const safeLines = Number.isFinite(parsedLines)
      ? Math.max(10, Math.min(10000, parsedLines))
      : 200

    try {
      setLoading(true)
      const text = await opsApi.getContainerLogs(containerName, safeLines)
      setLogText(text)
      setHint(`docker logs --tail ${safeLines} ${containerName}`)
    } catch (error) {
      showApiError(error, `获取容器日志失败：${containerName}`)
    } finally {
      setLoading(false)
    }
  }, [containerName, open, showApiError, tailLines])

  // 打开时自动加载
  useEffect(() => {
    if (!open || !containerName) {
      return
    }
    void loadLogs()
  }, [open, containerName, loadLogs])

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      setLogText("")
      setHint("")
    }
    onOpenChange(nextOpen)
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent side="right" className="w-[95vw] max-w-none sm:max-w-4xl">
        <SheetHeader>
          <SheetTitle>副本运行日志</SheetTitle>
          <SheetDescription>
            服务：{serviceName || "-"} · 容器：{containerName || "-"}
          </SheetDescription>
        </SheetHeader>

        <div className="flex h-full min-h-0 flex-col gap-3 px-4 pb-4">
          <div className="flex flex-wrap items-center gap-2">
            <Input
              type="number"
              min={10}
              max={10000}
              value={tailLines}
              onChange={(event) => setTailLines(event.target.value)}
              className="w-[120px]"
              placeholder="tail 行数"
            />
            <Button variant="outline" onClick={() => void loadLogs()} disabled={loading || !containerName}>
              <RefreshCcw className="size-4" />
              刷新日志
            </Button>
          </div>
          <div className="text-xs text-muted-foreground">{hint || "加载中..."}</div>
          <LogStreamViewer
            value={logText}
            emptyText="日志内容为空"
            className="h-[68vh] rounded-lg border bg-muted/20"
          />
        </div>
      </SheetContent>
    </Sheet>
  )
}
