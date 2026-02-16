import { Pause, Play, RefreshCcw, Trash2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

interface RealtimeToolbarProps {
  running: boolean
  loading: boolean
  windowSeconds: number
  onWindowSecondsChange: (value: number) => void
  onToggleRunning: () => void
  onRefresh: () => void
  onClear: () => void
}

const WINDOW_OPTIONS = [
  { value: 60, label: "最近 1 分钟" },
  { value: 180, label: "最近 3 分钟" },
  { value: 300, label: "最近 5 分钟" },
]

// RealtimeToolbar 提供监测页的控制按钮与窗口设置。
export function RealtimeToolbar({
  running,
  loading,
  windowSeconds,
  onWindowSecondsChange,
  onToggleRunning,
  onRefresh,
  onClear,
}: RealtimeToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button variant={running ? "outline" : "default"} onClick={onToggleRunning}>
        {running ? <Pause className="size-4" /> : <Play className="size-4" />}
        {running ? "暂停采样" : "继续采样"}
      </Button>
      <Button variant="outline" onClick={onRefresh} disabled={loading}>
        <RefreshCcw className="size-4" />
        立即采样
      </Button>
      <Button variant="outline" onClick={onClear}>
        <Trash2 className="size-4" />
        清空曲线
      </Button>
      <Select value={String(windowSeconds)} onValueChange={(value) => onWindowSecondsChange(Number(value))}>
        <SelectTrigger className="w-36">
          <SelectValue placeholder="时间窗口" />
        </SelectTrigger>
        <SelectContent>
          {WINDOW_OPTIONS.map((item) => (
            <SelectItem key={item.value} value={String(item.value)}>
              {item.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
