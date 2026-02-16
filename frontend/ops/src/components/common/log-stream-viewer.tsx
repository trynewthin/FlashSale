import { ScrollArea } from "@/components/ui/scroll-area"

interface LogStreamViewerProps {
  value: string
  emptyText?: string
  className?: string
}

// LogStreamViewer 用统一样式渲染日志文本。
export function LogStreamViewer({
  value,
  emptyText = "暂无日志",
  className = "h-[420px] rounded-lg border bg-muted/20",
}: LogStreamViewerProps) {
  return (
    <ScrollArea className={className}>
      <pre className="min-h-full p-3 text-xs leading-5 whitespace-pre-wrap break-all">
        {value || emptyText}
      </pre>
    </ScrollArea>
  )
}
