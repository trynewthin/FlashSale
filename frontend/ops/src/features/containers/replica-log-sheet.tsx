import { Pause, Play, RefreshCcw } from "lucide-react"
import { useCallback, useEffect, useMemo, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import type { SSELogLineFrame, ServiceLogFile } from "@/api/types"
import { LogStreamViewer } from "@/components/common/log-stream-viewer"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { useEventSource } from "@/hooks/use-event-source"
import { useOpsApiError } from "@/hooks/use-ops-api-error"
import { useOpsUIStore } from "@/store/ops-ui-store"

interface ReplicaLogSheetProps {
  open: boolean
  serviceName: string
  containerName: string
  onOpenChange: (open: boolean) => void
}

function pickBestLogFile(
  files: ServiceLogFile[],
  serviceName: string,
  containerName: string
): ServiceLogFile | null {
  const safeService = serviceName.trim().toLowerCase()
  const safeContainer = containerName.trim().toLowerCase()
  if (safeContainer) {
    const byContainer = files.find((item) => item.rel_path.toLowerCase().includes(safeContainer))
    if (byContainer) {
      return byContainer
    }
  }
  if (safeService) {
    const byService = files.find((item) => item.rel_path.toLowerCase().includes(safeService))
    if (byService) {
      return byService
    }
  }
  return files[0] ?? null
}

// ReplicaLogSheet 在容器编排页内以 Sheet 形式查看副本日志。
export function ReplicaLogSheet({
  open,
  serviceName,
  containerName,
  onOpenChange,
}: ReplicaLogSheetProps) {
  const showApiError = useOpsApiError()
  const showNotice = useOpsUIStore((state) => state.showNotice)

  const [files, setFiles] = useState<ServiceLogFile[]>([])
  const [selectedFileID, setSelectedFileID] = useState("")
  const [loadingFiles, setLoadingFiles] = useState(false)
  const [loadingTail, setLoadingTail] = useState(false)
  const [tailLines, setTailLines] = useState("200")
  const [streaming, setStreaming] = useState(false)
  const [logText, setLogText] = useState("")
  const [hint, setHint] = useState("")

  const selectedFile = useMemo(
    () => files.find((item) => item.id === selectedFileID) ?? null,
    [files, selectedFileID]
  )

  const streamURL = useMemo(() => {
    if (!selectedFileID) {
      return ""
    }
    return opsApi.buildServiceLogStreamURL(selectedFileID, true)
  }, [selectedFileID])

  const loadFiles = useCallback(async () => {
    if (!open) {
      return
    }
    try {
      setLoadingFiles(true)
      const list = await opsApi.listServiceLogFiles()
      setFiles(list)
      const matched = pickBestLogFile(list, serviceName, containerName)
      if (!matched) {
        setSelectedFileID("")
        setHint("未找到可用日志文件")
        return
      }
      setSelectedFileID(matched.id)
      setHint(`已定位日志：${matched.rel_path}`)
    } catch (error) {
      showApiError(error, "加载日志文件失败")
    } finally {
      setLoadingFiles(false)
    }
  }, [containerName, open, serviceName, showApiError])

  const loadTail = useCallback(async () => {
    if (!selectedFileID) {
      showNotice("error", "请选择日志文件")
      return
    }
    const parsedLines = Number.parseInt(tailLines, 10)
    const safeLines = Number.isFinite(parsedLines)
      ? Math.max(10, Math.min(5000, parsedLines))
      : 200

    try {
      setLoadingTail(true)
      const data = await opsApi.getServiceLogTail(selectedFileID, safeLines)
      setStreaming(false)
      setLogText(data.text || "")
      setHint(data.truncated ? "尾部快照已截断(1MB)" : `文件：${data.rel_path}`)
    } catch (error) {
      showApiError(error, "读取日志快照失败")
    } finally {
      setLoadingTail(false)
    }
  }, [selectedFileID, showApiError, showNotice, tailLines])

  useEffect(() => {
    if (!open) {
      return
    }
    void loadFiles()
  }, [loadFiles, open])

  useEffect(() => {
    if (!open || !selectedFileID) {
      return
    }
    void loadTail()
  }, [loadTail, open, selectedFileID])

  useEventSource({
    enabled: open && streaming && !!selectedFileID,
    url: streamURL,
    handlers: {
      onEvent: (eventType, payload) => {
        if (eventType === "meta") {
          setHint(`实时追踪：${selectedFile?.rel_path || selectedFileID}`)
          return
        }
        if (eventType !== "log") {
          return
        }
        const frame = payload as SSELogLineFrame
        if (typeof frame.chunk === "string") {
          setLogText((prev) => prev + frame.chunk)
          return
        }
        if (typeof frame.line === "string") {
          setLogText((prev) => `${prev}${prev.endsWith("\n") || prev.length === 0 ? "" : "\n"}${frame.line}\n`)
        }
      },
      onError: () => {
        setStreaming(false)
        showNotice("error", "日志流已断开，请重试")
      },
    },
  })

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      setStreaming(false)
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
            <Select
              value={selectedFileID || undefined}
              onValueChange={(value) => {
                setSelectedFileID(value || "")
                setStreaming(false)
                setLogText("")
              }}
            >
              <SelectTrigger className="w-[460px] max-w-full">
                <SelectValue placeholder="选择日志文件" />
              </SelectTrigger>
              <SelectContent>
                {files.map((file) => (
                  <SelectItem key={file.id} value={file.id}>
                    {file.rel_path}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Input
              type="number"
              min={10}
              max={5000}
              value={tailLines}
              onChange={(event) => setTailLines(event.target.value)}
              className="w-[120px]"
            />
            <Button variant="outline" onClick={() => void loadTail()} disabled={loadingTail || !selectedFileID}>
              Tail
            </Button>
            <Button
              variant="outline"
              onClick={() => {
                if (!selectedFileID) {
                  showNotice("error", "请选择日志文件")
                  return
                }
                setLogText("")
                setStreaming(true)
              }}
              disabled={!selectedFileID}
            >
              <Play className="size-4" />
              实时流
            </Button>
            <Button variant="outline" onClick={() => setStreaming(false)} disabled={!streaming}>
              <Pause className="size-4" />
              停止流
            </Button>
            <Button variant="outline" onClick={() => void loadFiles()} disabled={loadingFiles}>
              <RefreshCcw className="size-4" />
              刷新列表
            </Button>
          </div>
          <div className="text-xs text-muted-foreground">{hint || "请选择日志文件"}</div>
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
