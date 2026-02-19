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
import { Plus, Search, X } from "lucide-react"

import { type ActivityAdmin } from "@/api/modules/seckill"
import { useSeckillActivityListQuery, useCreateSeckillActivityMutation } from "@/hooks/biz/use-seckill-mgmt-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { formatUnix } from "@/lib/format"

import {
  ActivityFormFields,
  ActivityEditForm, DeleteActivityDialog, ActivityRowActions,
} from "@/components/seckill"

const STATUS_MAP: Record<number, string> = { 1: "草稿", 2: "已发布", 3: "已下线", 4: "已删除" }


function unixToLocalDatetimeStr(unix: number) {
  if (!unix) return ""
  const d = new Date(unix * 1000)
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
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
    setStartStr("")
    setEndStr("")
    setFormOpen(true)
  }

  const openEdit = (a: ActivityAdmin) => {
    setEditTarget(a)
    setFormData({
      title: a.title, description: a.description, style_config_json: a.style_config_json,
      start_at_unix: a.start_at_unix, end_at_unix: a.end_at_unix,
    })
    setStartStr(unixToLocalDatetimeStr(a.start_at_unix))
    setEndStr(unixToLocalDatetimeStr(a.end_at_unix))
    setFormOpen(true)
  }

  const handleCreateSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const req = {
      ...formData,
      start_at_unix: startStr ? Math.floor(new Date(startStr).getTime() / 1000) : 0,
      end_at_unix: endStr ? Math.floor(new Date(endStr).getTime() / 1000) : 0,
    }
    createMutation.mutate(req, { onSuccess: () => setFormOpen(false) })
  }

  return (
    <div className="space-y-4 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold tracking-tight">秒杀活动</h1>
        <Button onClick={openCreate} size="sm"><Plus className="mr-1 size-4" />新建活动</Button>
      </div>

      <div className="flex items-end gap-3">
        <div className="space-y-1">
          <Label className="text-xs">关键字</Label>
          <Input placeholder="活动标题" value={keyword} onChange={(e) => setKeyword(e.target.value)} className="w-48"
            onKeyDown={(e) => { if (e.key === "Enter") { setSearchKeyword(keyword); setPage(1) } }} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">状态</Label>
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v ?? "")}>
            <SelectTrigger className="w-28"><SelectValue placeholder="全部" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="1">草稿</SelectItem>
              <SelectItem value="2">已发布</SelectItem>
              <SelectItem value="3">已下线</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button size="sm" onClick={() => { setSearchKeyword(keyword); setPage(1) }}><Search className="mr-1 size-4" />查询</Button>
        <Button size="sm" variant="outline" onClick={() => { setKeyword(""); setSearchKeyword(""); setStatusFilter(""); setPage(1) }}><X className="mr-1 size-4" />重置</Button>
      </div>

      {listQuery.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(listQuery.error)}</AlertDescription></Alert>}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>标题</TableHead>
              <TableHead>开始时间</TableHead>
              <TableHead>结束时间</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>商品数</TableHead>
              <TableHead className="w-16">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 && (
              <TableRow><TableCell colSpan={6} className="text-center text-muted-foreground">{listQuery.isLoading ? "加载中…" : "暂无数据"}</TableCell></TableRow>
            )}
            {items.map((a) => (
              <TableRow key={a.activity_id} className="cursor-pointer" onClick={() => navigate(`/seckill/${a.activity_id}`)}>
                <TableCell className="font-medium">{a.title}</TableCell>
                <TableCell className="text-sm text-muted-foreground">{formatUnix(a.start_at_unix)}</TableCell>
                <TableCell className="text-sm text-muted-foreground">{formatUnix(a.end_at_unix)}</TableCell>
                <TableCell><Badge variant={a.status === 2 ? "default" : "secondary"}>{STATUS_MAP[a.status] ?? a.status}</Badge></TableCell>
                <TableCell>{a.items?.length ?? 0}</TableCell>
                <TableCell onClick={(e) => e.stopPropagation()}>
                  <ActivityRowActions activity={a} onEdit={() => openEdit(a)} onDelete={() => setDeleteTarget(a)} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">共 {total} 条</span>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</Button>
          <span className="flex items-center px-2">{page} / {totalPages || 1}</span>
          <Button size="sm" variant="outline" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</Button>
        </div>
      </div>

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
              {createMutation.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(createMutation.error)}</AlertDescription></Alert>}
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

