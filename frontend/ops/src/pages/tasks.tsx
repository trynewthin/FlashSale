import { RefreshCcw } from "lucide-react"
import { useCallback, useEffect, useMemo, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import type { TaskDef } from "@/api/types"
import { useOpsApiError } from "@/hooks/use-ops-api-error"
import { useOpsUIStore } from "@/store/ops-ui-store"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

function parseArgs(raw: string): string[] {
  const trimmed = (raw || "").trim()
  if (!trimmed) {
    return []
  }
  return trimmed.split(/\s+/)
}

// TasksPage 提供白名单任务浏览与触发。
export function TasksPage() {
  const showApiError = useOpsApiError()
  const showNotice = useOpsUIStore((state) => state.showNotice)
  const setSelectedJobId = useOpsUIStore((state) => state.setSelectedJobId)

  const [tasks, setTasks] = useState<TaskDef[]>([])
  const [loading, setLoading] = useState(false)
  const [running, setRunning] = useState(false)
  const [selectedTask, setSelectedTask] = useState("")
  const [argsText, setArgsText] = useState("")

  const loadTasks = useCallback(async () => {
    try {
      setLoading(true)
      const data = await opsApi.listTasks()
      setTasks(data)
      if (data.length > 0 && !selectedTask) {
        setSelectedTask(data[0].id)
      }
    } catch (error) {
      showApiError(error, "任务列表加载失败")
    } finally {
      setLoading(false)
    }
  }, [selectedTask, showApiError])

  useEffect(() => {
    void loadTasks()
  }, [loadTasks])

  const selectedTaskDef = useMemo(
    () => tasks.find((item) => item.id === selectedTask) || null,
    [tasks, selectedTask]
  )

  const runTask = async () => {
    if (!selectedTask) {
      showNotice("error", "请选择任务")
      return
    }
    try {
      setRunning(true)
      const job = await opsApi.createJob({
        task: selectedTask,
        args: parseArgs(argsText),
      })
      setSelectedJobId(job.id)
      showNotice("success", `任务已创建：${job.id}`)
    } catch (error) {
      showApiError(error, "任务执行失败")
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>任务执行</CardTitle>
          <CardDescription>按白名单触发任务，并将输出写入任务日志流。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <Select
              value={selectedTask || undefined}
              onValueChange={(value) => setSelectedTask(value || "")}
            >
              <SelectTrigger className="w-[360px] max-w-full">
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
              onChange={(event) => setArgsText(event.target.value)}
              placeholder="额外参数，例如: -rate 120 -open-duration 30s"
              className="w-[440px] max-w-full"
            />
            <Button onClick={() => void runTask()} disabled={running || loading}>
              执行任务
            </Button>
            <Button variant="outline" onClick={() => void loadTasks()} disabled={loading}>
              <RefreshCcw className="size-4" />
              刷新
            </Button>
          </div>
          {selectedTaskDef ? (
            <div className="rounded-md border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
              {selectedTaskDef.description}
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>任务白名单</CardTitle>
          <CardDescription>只允许执行此列表中的任务。</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Task ID</TableHead>
                <TableHead>名称</TableHead>
                <TableHead>说明</TableHead>
                <TableHead>危险</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tasks.map((task) => (
                <TableRow key={task.id}>
                  <TableCell>{task.id}</TableCell>
                  <TableCell>{task.name}</TableCell>
                  <TableCell className="max-w-[460px] whitespace-normal">{task.description}</TableCell>
                  <TableCell>
                    <Badge variant={task.dangerous ? "destructive" : "secondary"}>
                      {task.dangerous ? "Y" : "N"}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
