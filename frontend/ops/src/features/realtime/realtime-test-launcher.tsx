import { FlaskConical, Play, RefreshCcw } from "lucide-react"

import type { TaskDef } from "@/api/types"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Textarea } from "@/components/ui/textarea"
import {
  getRealtimeTaskDisplayName,
  getRealtimeTestPreset,
  type RealtimeTestPreset,
} from "@/features/realtime/realtime-test-presets"
import { cn } from "@/lib/utils"

interface RealtimeTestLauncherProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  tasks: TaskDef[]
  selectedTaskID: string
  selectedPreset: RealtimeTestPreset | null
  formValues: Record<string, string>
  extraArgsText: string
  creating: boolean
  onSelectTask: (taskID: string) => void
  onSetFormValue: (flag: string, value: string) => void
  onChangeExtraArgs: (value: string) => void
  onRefreshTasks: () => void
  onStartTest: () => void
  showFloatingTrigger?: boolean
}

// RealtimeTestLauncher 提供“测试按钮 -> 配置 -> 启动”交互。
export function RealtimeTestLauncher({
  open,
  onOpenChange,
  tasks,
  selectedTaskID,
  selectedPreset,
  formValues,
  extraArgsText,
  creating,
  onSelectTask,
  onSetFormValue,
  onChangeExtraArgs,
  onRefreshTasks,
  onStartTest,
  showFloatingTrigger = true,
}: RealtimeTestLauncherProps) {
  const selectedTask = tasks.find((task) => task.id === selectedTaskID) || null

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      {showFloatingTrigger ? (
        <div className="fixed right-6 bottom-6 z-50">
          <Button variant="default" size="lg" className="shadow-lg" onClick={() => onOpenChange(true)}>
            <FlaskConical className="size-4" />
            开始测试
          </Button>
        </div>
      ) : null}

      <SheetContent side="right" className="w-[98vw] max-w-none gap-0 p-0 sm:max-w-2xl">
        <SheetHeader className="border-b bg-background">
          <SheetTitle>启动测试任务</SheetTitle>
          <SheetDescription>默认临时数据模式：自动创建测试数据，完成后自动清理。</SheetDescription>
        </SheetHeader>

        <div className="flex-1 space-y-4 overflow-y-auto px-4 py-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">测试场景</CardTitle>
              <CardDescription>优先从推荐场景开始，之后再做细调。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              <div className="flex items-center gap-2">
                <Select value={selectedTaskID || undefined} onValueChange={(value) => onSelectTask(value || "")}>
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="选择测试场景" />
                  </SelectTrigger>
                  <SelectContent>
                    {tasks.map((task) => {
                      const preset = getRealtimeTestPreset(task.id)
                      return (
                        <SelectItem key={task.id} value={task.id}>
                          {preset?.recommended ? "推荐 · " : ""}
                          {getRealtimeTaskDisplayName(task)}
                        </SelectItem>
                      )
                    })}
                  </SelectContent>
                </Select>
                <Button variant="outline" size="icon-sm" onClick={onRefreshTasks} title="刷新测试场景">
                  <RefreshCcw className="size-4" />
                </Button>
              </div>

              <div className="grid gap-2 sm:grid-cols-2">
                {tasks.map((task) => {
                  const preset = getRealtimeTestPreset(task.id)
                  const selected = selectedTaskID === task.id
                  return (
                    <button
                      key={task.id}
                      type="button"
                      onClick={() => onSelectTask(task.id)}
                      className={cn(
                        "rounded-lg border px-3 py-2 text-left transition-colors",
                        selected ? "border-primary bg-primary/5" : "hover:border-primary/40"
                      )}
                    >
                      <div className="text-sm font-medium">{getRealtimeTaskDisplayName(task)}</div>
                      <div className="text-xs text-muted-foreground">{preset?.subtitle || task.description || task.id}</div>
                    </button>
                  )
                })}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">参数配置</CardTitle>
              <CardDescription>{selectedPreset?.description || "根据测试目标调整参数。"}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {selectedPreset ? (
                <div className="grid gap-3 sm:grid-cols-2">
                  {selectedPreset.fields.map((field) => (
                    <div key={field.flag} className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">{field.label}</Label>
                      {field.type === "select" ? (
                        <Select
                          value={formValues[field.flag] || ""}
                          onValueChange={(value) => onSetFormValue(field.flag, value || "")}
                        >
                          <SelectTrigger>
                            <SelectValue placeholder={field.placeholder || "请选择"} />
                          </SelectTrigger>
                          <SelectContent>
                            {(field.options || []).map((option) => (
                              <SelectItem key={option.value} value={option.value}>
                                {option.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      ) : (
                        <Input
                          type={field.type === "number" ? "number" : "text"}
                          value={formValues[field.flag] || ""}
                          min={field.min}
                          max={field.max}
                          step={field.step}
                          placeholder={field.placeholder}
                          onChange={(event) => onSetFormValue(field.flag, event.target.value)}
                        />
                      )}
                      {field.helper ? <div className="text-[11px] text-muted-foreground">{field.helper}</div> : null}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="rounded-md border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
                  当前任务暂无结构化参数模板，将按默认参数执行。
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">高级参数（可选）</CardTitle>
              <CardDescription>仅在需要覆盖默认行为时填写，支持原始命令行参数。</CardDescription>
            </CardHeader>
            <CardContent>
              <Textarea
                value={extraArgsText}
                onChange={(event) => onChangeExtraArgs(event.target.value)}
                className="min-h-28 font-mono text-xs"
                placeholder="例如：--max-network-errors 200 --keep-temp-data"
              />
            </CardContent>
          </Card>
        </div>

        <SheetFooter className="border-t bg-background">
          <div className="flex w-full flex-wrap items-center justify-between gap-3">
            <div className="text-xs text-muted-foreground">
              当前场景：
              <span className="ml-1 font-medium text-foreground">
                {selectedTask ? getRealtimeTaskDisplayName(selectedTask) : "未选择"}
              </span>
            </div>
            <Button className="min-w-32" onClick={onStartTest} disabled={creating || !selectedTaskID}>
              <Play className="size-4" />
              {creating ? "启动中..." : "开始测试"}
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
