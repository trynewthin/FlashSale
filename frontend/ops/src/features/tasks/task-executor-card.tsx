import { ShieldAlert } from "lucide-react"

import type { TaskDef } from "@/api/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

export function TaskExecutorCard({
  tasks,
  selectedTask,
  selectedTaskDef,
  argsText,
  runningTask,
  loadingTasks,
  onSelectTask,
  onArgsChange,
  onRunTask,
}: {
  tasks: TaskDef[]
  selectedTask: string
  selectedTaskDef: TaskDef | null
  argsText: string
  runningTask: boolean
  loadingTasks: boolean
  onSelectTask: (value: string) => void
  onArgsChange: (value: string) => void
  onRunTask: () => void
}) {
  const selectedArgsPreview = selectedTaskDef?.default_args?.join(" ") || "无默认参数"

  return (
    <Card className="border-border/50 bg-background/70 shadow-sm">
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center justify-between gap-3 text-base">
          <span>任务执行器</span>
          <Badge variant="outline">{tasks.length} 个可执行任务</Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 xl:grid-cols-[minmax(320px,420px)_minmax(0,1fr)_auto]">
          <Select value={selectedTask || undefined} onValueChange={(value) => onSelectTask(value || "")}>
            <SelectTrigger className="h-10 w-full">
              <SelectValue placeholder="选择任务" />
            </SelectTrigger>
            <SelectContent>
              {tasks.map((task) => (
                <SelectItem key={task.id} value={task.id}>
                  {task.id} - {task.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Input
            value={argsText}
            onChange={(event) => onArgsChange(event.target.value)}
            placeholder="额外参数，例如: -rate 120 -open-duration 30s"
            className="h-10"
          />

          <Button className="h-10 min-w-28" onClick={onRunTask} disabled={runningTask || loadingTasks}>
            执行任务
          </Button>
        </div>

        {selectedTaskDef ? (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_minmax(0,0.8fr)]">
            <div className="rounded-xl border border-border/50 bg-muted/20 px-3 py-3">
              <div className="text-[11px] uppercase tracking-wider text-muted-foreground">任务说明</div>
              <div className="mt-1 text-sm text-foreground">{selectedTaskDef.description}</div>
            </div>
            <div className="rounded-xl border border-border/50 bg-muted/20 px-3 py-3">
              <div className="text-[11px] uppercase tracking-wider text-muted-foreground">默认参数</div>
              <div className="mt-1 font-mono text-sm text-foreground">{selectedArgsPreview}</div>
            </div>
            <div className="rounded-xl border border-border/50 bg-muted/20 px-3 py-3">
              <div className="text-[11px] uppercase tracking-wider text-muted-foreground">风险等级</div>
              <div className="mt-1 flex items-center gap-2 text-sm text-foreground">
                {selectedTaskDef.dangerous ? <ShieldAlert className="size-4 text-destructive" /> : null}
                <Badge variant={selectedTaskDef.dangerous ? "destructive" : "secondary"}>
                  {selectedTaskDef.dangerous ? "危险任务" : "常规任务"}
                </Badge>
              </div>
            </div>
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}
