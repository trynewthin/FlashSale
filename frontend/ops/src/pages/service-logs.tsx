import { Pause, Play, RefreshCcw } from "lucide-react"
import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useSearchParams } from "react-router-dom"

import { opsApi } from "@/api/modules/ops"
import type { SSELogLineFrame, ServiceLogFile } from "@/api/types"
import { LogStreamViewer } from "@/components/common/log-stream-viewer"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useEventSource } from "@/hooks/use-event-source"
import { useOpsApiError } from "@/hooks/use-ops-api-error"
import { useOpsUIStore } from "@/store/ops-ui-store"

// ServiceLogsPage 展示服务日志文件列表、tail 与 SSE。
export function ServiceLogsPage() {
  const showApiError = useOpsApiError()
  const showNotice = useOpsUIStore((state) => state.showNotice)
  const selectedFileId = useOpsUIStore((state) => state.selectedServiceLogFileId)
  const setSelectedFileId = useOpsUIStore((state) => state.setSelectedServiceLogFileId)
  const [searchParams, setSearchParams] = useSearchParams()

  const [files, setFiles] = useState<ServiceLogFile[]>([])
  const [loadingFiles, setLoadingFiles] = useState(false)
  const [loadingTail, setLoadingTail] = useState(false)
  const [streaming, setStreaming] = useState(false)
  const [tailLines, setTailLines] = useState("200")
  const [logText, setLogText] = useState("")
  const [hint, setHint] = useState("")
  const presetAppliedRef = useRef(false)

  const presetService = useMemo(
    () => (searchParams.get("service") || "").trim().toLowerCase(),
    [searchParams]
  )
  const presetContainer = useMemo(
    () => (searchParams.get("container") || "").trim().toLowerCase(),
    [searchParams]
  )

  const pickPresetLogFile = useCallback(
    (list: ServiceLogFile[]): ServiceLogFile | null => {
      if (!presetService && !presetContainer) {
        return null
      }
      if (presetContainer) {
        const matchedByContainer = list.find((item) => item.rel_path.toLowerCase().includes(presetContainer))
        if (matchedByContainer) {
          return matchedByContainer
        }
      }
      if (presetService) {
        const matchedByService = list.find((item) => item.rel_path.toLowerCase().includes(presetService))
        if (matchedByService) {
          return matchedByService
        }
      }
      return null
    },
    [presetContainer, presetService]
  )

  const selectedFile = useMemo(
    () => files.find((file) => file.id === selectedFileId) || null,
    [files, selectedFileId]
  )

  const loadFiles = useCallback(async () => {
    try {
      setLoadingFiles(true)
      const list = await opsApi.listServiceLogFiles()
      setFiles(list)
      if (!presetAppliedRef.current) {
        const preset = pickPresetLogFile(list)
        if (preset) {
          setSelectedFileId(preset.id)
          setHint(`已定位日志：${preset.rel_path}`)
          presetAppliedRef.current = true
          setSearchParams({}, { replace: true })
          return
        }
        if (presetService || presetContainer) {
          presetAppliedRef.current = true
          setSearchParams({}, { replace: true })
        }
      }
      if (!selectedFileId && list.length > 0) {
        setSelectedFileId(list[0].id)
      }
      setHint(list.length > 0 ? `共 ${list.length} 个日志文件` : "未找到日志文件")
    } catch (error) {
      showApiError(error, "日志文件列表加载失败")
    } finally {
      setLoadingFiles(false)
    }
  }, [
    pickPresetLogFile,
    presetContainer,
    presetService,
    selectedFileId,
    setSearchParams,
    setSelectedFileId,
    showApiError,
  ])

  const loadTail = async () => {
    if (!selectedFileId) {
      showNotice("error", "请选择日志文件")
      return
    }
    const parsedLines = Number.parseInt(tailLines, 10)
    const safeLines = Number.isFinite(parsedLines)
      ? Math.max(10, Math.min(5000, parsedLines))
      : 200

    try {
      setLoadingTail(true)
      const data = await opsApi.getServiceLogTail(selectedFileId, safeLines)
      setStreaming(false)
      setLogText(data.text || "")
      const metadata = [
        `文件：${data.rel_path || selectedFile?.name || selectedFileId}`,
        data.truncated ? "尾部快照已截断(1MB)" : "",
      ]
        .filter(Boolean)
        .join(" | ")
      setHint(metadata)
    } catch (error) {
      showApiError(error, "读取日志快照失败")
    } finally {
      setLoadingTail(false)
    }
  }

  useEffect(() => {
    void loadFiles()
  }, [loadFiles])

  const streamURL = useMemo(() => {
    if (!selectedFileId) {
      return ""
    }
    return opsApi.buildServiceLogStreamURL(selectedFileId, true)
  }, [selectedFileId])

  useEventSource({
    enabled: streaming && !!selectedFileId,
    url: streamURL,
    handlers: {
      onEvent: (eventType, payload) => {
        if (eventType === "meta") {
          setHint(`实时追踪：${selectedFile?.rel_path || selectedFile?.name || selectedFileId}`)
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
        showNotice("error", "服务日志流已断开，请重试")
      },
    },
  })

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>服务日志</CardTitle>
          <CardDescription>选择日志文件后可查看 tail 快照或实时流。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <Select
              value={selectedFileId || undefined}
              onValueChange={(value) => {
                setSelectedFileId(value || "")
                setStreaming(false)
                setLogText("")
              }}
            >
              <SelectTrigger className="w-[420px] max-w-full">
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
            <Button variant="outline" onClick={() => void loadTail()} disabled={loadingTail}>
              Tail
            </Button>
            <Button
              variant="outline"
              onClick={() => {
                if (!selectedFileId) {
                  showNotice("error", "请选择日志文件")
                  return
                }
                setLogText("")
                setStreaming(true)
              }}
              disabled={!selectedFileId}
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
          <LogStreamViewer value={logText} emptyText="日志内容为空" />
        </CardContent>
      </Card>
    </div>
  )
}
