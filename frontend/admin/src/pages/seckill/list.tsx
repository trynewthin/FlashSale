import { useState } from "react"
import { useNavigate } from "react-router-dom"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Plus, Search, X, AlertCircle, Zap, Clock } from "lucide-react"

import { type ActivityAdmin } from "@/api/modules/seckill"
import { useSeckillActivityListQuery, useCreateSeckillActivityMutation } from "@/hooks/biz/use-seckill-mgmt-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { formatUnix } from "@/lib/format"

import {
  ActivityFormFields,
  ActivityEditForm, DeleteActivityDialog, ActivityRowActions,
} from "@/components/seckill"

const STATUS_MAP: Record<number, { label: string; color: string }> = {
  0: { label: "草稿", color: "bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400" },
  1: { label: "已发布", color: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400" },
  2: { label: "已下线", color: "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400" },
}

function unixToLocalDatetimeStr(unix: number) {
  if (!unix) return ""
  const d = new Date(unix * 1000)
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// 判断当前是否在活动时间段内
function getTimeStatus(startUnix: number, endUnix: number): "upcoming" | "ongoing" | "ended" {
  const now = Math.floor(Date.now() / 1000)
  if (now < startUnix) return "upcoming"
  if (now > endUnix) return "ended"
  return "ongoing"
}

export function SeckillActivityListPage() {
  const navigate = useNavigate()
  const { toUserMessage } = useApiError()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState("")
  const [searchKeyword, setSearchKeyword] = useState("")
  const [statusFilter, setStatusFilter] = useState("")

  const listQuery = useSeckillActivityListQuery({
    page, page_size: 20,
    keyword: searchKeyword || undefined,
    status: statusFilter ? Number(statusFilter) : undefined,
  })

  const createMutation = useCreateSeckillActivityMutation()

  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<ActivityAdmin | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<ActivityAdmin | null>(null)
  const [formData, setFormData] = useState({
    title: "", description: "", style_config_json: "",
    start_at_unix: 0, end_at_unix: 0,
  })
  const [startStr, setStartStr] = useState("")
  const [endStr, setEndStr] = useState("")

  const items = listQuery.data?.list ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / 20)

  const openCreate = () => {
    setEditTarget(null)
    setFormData({ title: "", description: "", style_config_json: "", start_at_unix: 0, end_at_unix: 0 })
    setStartStr(""); setEndStr("")
    setFormOpen(true)
  }

  const openEdit = (a: ActivityAdmin) => {
    setEditTarget(a)
    setFormData({ title: a.title, description: a.description, style_config_json: a.style_config_json, start_at_unix: a.start_at_unix, end_at_unix: a.end_at_unix })
    setStartStr(unixToLocalDatetimeStr(a.start_at_unix))
    setEndStr(unixToLocalDatetimeStr(a.end_at_unix))
    setFormOpen(true)
  }

  const handleCreateSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    createMutation.mutate(
      { ...formData, start_at_unix: startStr ? Math.floor(new Date(startStr).getTime() / 1000) : 0, end_at_unix: endStr ? Math.floor(new Date(endStr).getTime() / 1000) : 0 },
      { onSuccess: () => setFormOpen(false) }
    )
  }

  return (
    <div className="space-y-5 p-6">

      {/* 顶部 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">秒杀活动</h1>
          <p className="text-sm text-muted-foreground mt-0.5">共 {total} 个活动</p>
        </div>
        <Button onClick={openCreate} size="sm">
          <Plus className="mr-1.5 size-3.5" />新建活动
        </Button>
      </div>

      {/* 筛选栏 */}
      <div className="flex flex-wrap items-end gap-3 rounded-lg border bg-card px-4 py-3">
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">关键字</Label>
          <Input placeholder="活动标题" value={keyword} onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter") { setSearchKeyword(keyword); setPage(1) } }}
            className="h-8 w-48 text-sm" />
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">状态</Label>
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v ?? "")}
            items={[{ value: "0", label: "草稿" }, { value: "1", label: "已发布" }, { value: "2", label: "已下线" }]}>
            <SelectTrigger className="h-8 w-28 text-sm"><SelectValue placeholder="全部" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="0">草稿</SelectItem>
              <SelectItem value="1">已发布</SelectItem>
              <SelectItem value="2">已下线</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="flex gap-2">
          <Button size="sm" className="h-8" onClick={() => { setSearchKeyword(keyword); setPage(1) }}>
            <Search className="mr-1 size-3.5" />查询
          </Button>
          <Button size="sm" variant="outline" className="h-8" onClick={() => { setKeyword(""); setSearchKeyword(""); setStatusFilter(""); setPage(1) }}>
            <X className="mr-1 size-3.5" />重置
          </Button>
        </div>
      </div>

      {/* 错误 */}
      {listQuery.isError && (
        <div className="flex items-center gap-2 rounded-lg border border-destructive/40 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          <AlertCircle className="size-4 shrink-0" />{toUserMessage(listQuery.error)}
        </div>
      )}

      {/* 表格 */}
      <div className="rounded-lg border overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-muted/40 hover:bg-muted/40">
              <TableHead className="pl-4">活动</TableHead>
              <TableHead>时间段</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>商品数</TableHead>
              <TableHead className="w-14 pr-4">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="py-12 text-center text-sm text-muted-foreground">
                  {listQuery.isLoading
                    ? <span className="flex items-center justify-center gap-2"><span className="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />加载中…</span>
                    : "暂无活动"}
                </TableCell>
              </TableRow>
            )}
            {items.map((a) => {
              const status = Number(a.status ?? 0)
              const statusInfo = STATUS_MAP[status]
              const timeStatus = getTimeStatus(a.start_at_unix, a.end_at_unix)
              return (
                <TableRow key={a.activity_id} className="cursor-pointer" onClick={() => navigate(`/seckill/${a.activity_id}`)}>
                  {/* 活动标题 + 图标 */}
                  <TableCell className="pl-4">
                    <div className="flex items-center gap-2.5">
                      <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-amber-100 dark:bg-amber-950">
                        <Zap className="size-4 text-amber-600 dark:text-amber-400" />
                      </span>
                      <span className="text-sm font-medium">{a.title}</span>
                    </div>
                  </TableCell>

                  {/* 时间段 */}
                  <TableCell>
                    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                      <Clock className="size-3 shrink-0" />
                      <div>
                        <p>{formatUnix(a.start_at_unix)}</p>
                        <p>{formatUnix(a.end_at_unix)}</p>
                      </div>
                      {timeStatus === "ongoing" && (
                        <span className="ml-1 inline-flex items-center gap-1 rounded-full bg-emerald-100 px-1.5 py-0.5 text-[9px] font-medium text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400">
                          <span className="size-1.5 rounded-full bg-emerald-500 animate-pulse" />进行中
                        </span>
                      )}
                    </div>
                  </TableCell>

                  {/* 状态 */}
                  <TableCell>
                    {statusInfo
                      ? <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${statusInfo.color}`}>{statusInfo.label}</span>
                      : <span className="text-xs text-muted-foreground">{status}</span>
                    }
                  </TableCell>

                  {/* 商品数 */}
                  <TableCell>
                    <Badge variant="outline" className="text-xs">{a.items?.length ?? 0} 件</Badge>
                  </TableCell>

                  {/* 操作 */}
                  <TableCell className="pr-4" onClick={(e) => e.stopPropagation()}>
                    <ActivityRowActions activity={a} onEdit={() => openEdit(a)} onDelete={() => setDeleteTarget(a)} />
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>

      {/* 分页 */}
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">第 {page} 页 · 共 {total} 条</span>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</Button>
          <span className="flex items-center px-2 text-muted-foreground">{page} / {totalPages || 1}</span>
          <Button size="sm" variant="outline" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</Button>
        </div>
      </div>

      {/* 新建/编辑 Dialog */}
      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader><DialogTitle>{editTarget ? "编辑活动" : "新建活动"}</DialogTitle></DialogHeader>
          {editTarget ? (
            <ActivityEditForm activity={editTarget} formData={formData} setFormData={setFormData}
              startStr={startStr} setStartStr={setStartStr} endStr={endStr} setEndStr={setEndStr}
              onClose={() => setFormOpen(false)} />
          ) : (
            <form onSubmit={handleCreateSubmit} className="space-y-4">
              <ActivityFormFields formData={formData} setFormData={setFormData}
                startStr={startStr} setStartStr={setStartStr} endStr={endStr} setEndStr={setEndStr} />
              {createMutation.isError && (
                <Alert variant="destructive"><AlertDescription>{toUserMessage(createMutation.error)}</AlertDescription></Alert>
              )}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setFormOpen(false)}>取消</Button>
                <Button type="submit" disabled={createMutation.isPending}>{createMutation.isPending ? "保存中…" : "保存"}</Button>
              </div>
            </form>
          )}
        </DialogContent>
      </Dialog>

      {deleteTarget && <DeleteActivityDialog activity={deleteTarget} onClose={() => setDeleteTarget(null)} />}
    </div>
  )
}
