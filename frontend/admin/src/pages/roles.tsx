import { useState } from "react"

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

import { type RoleView } from "@/api/modules/admin"
import { useRoleListQuery, useCreateRoleMutation } from "@/hooks/admin/use-role-hooks"
import { useAdminPermission } from "@/hooks/auth/use-admin-permission"
import { useApiError } from "@/hooks/common/use-api-error"

import { RoleEditForm, DeleteRoleDialog, DomainsDialog, RoleRowActions } from "@/components/role"

const STATUS_MAP: Record<number, string> = { 1: "启用", 2: "禁用" }

function formatUnix(unix: number) {
  if (!unix) return "-"
  return new Date(unix * 1000).toLocaleString("zh-CN")
}

export function RoleListPage() {
  const { isDataScopeAll } = useAdminPermission()
  const { toUserMessage } = useApiError()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState("")
  const [searchKeyword, setSearchKeyword] = useState("")
  const [statusFilter, setStatusFilter] = useState("")

  const listQuery = useRoleListQuery({
    page, page_size: 20,
    keyword: searchKeyword || undefined,
    status: statusFilter ? Number(statusFilter) : undefined,
  })

  const createMutation = useCreateRoleMutation()

  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<RoleView | null>(null)
  const [formData, setFormData] = useState({ role_code: "", role_name: "", status: 1 })
  const [deleteTarget, setDeleteTarget] = useState<RoleView | null>(null)
  const [domainsTarget, setDomainsTarget] = useState<RoleView | null>(null)
  const [selectedDomains, setSelectedDomains] = useState<string[]>([])

  if (!isDataScopeAll()) {
    return (
      <div className="flex items-center justify-center p-12">
        <Alert variant="destructive" className="max-w-md">
          <AlertDescription>您的数据范围不是 all，无法访问角色管理页面。</AlertDescription>
        </Alert>
      </div>
    )
  }

  const items = listQuery.data?.items ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / 20)

  const openCreate = () => {
    setEditTarget(null)
    setFormData({ role_code: "", role_name: "", status: 1 })
    setFormOpen(true)
  }

  const openEdit = (role: RoleView) => {
    setEditTarget(role)
    setFormData({ role_code: role.role_code, role_name: role.role_name, status: role.status })
    setFormOpen(true)
  }

  const openDomains = (role: RoleView) => {
    setDomainsTarget(role)
    setSelectedDomains(role.domains ?? [])
  }

  const handleCreateSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    createMutation.mutate(formData, { onSuccess: () => setFormOpen(false) })
  }

  return (
    <div className="space-y-4 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold tracking-tight">角色管理</h1>
        <Button onClick={openCreate} size="sm"><Plus className="mr-1 size-4" />新建角色</Button>
      </div>

      <div className="flex items-end gap-3">
        <div className="space-y-1">
          <Label className="text-xs">关键字</Label>
          <Input placeholder="角色编码/名称" value={keyword} onChange={(e) => setKeyword(e.target.value)} className="w-48"
            onKeyDown={(e) => { if (e.key === "Enter") { setSearchKeyword(keyword); setPage(1) } }} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">状态</Label>
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v ?? "")}>
            <SelectTrigger className="w-28"><SelectValue placeholder="全部" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="1">启用</SelectItem>
              <SelectItem value="2">禁用</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button size="sm" onClick={() => { setSearchKeyword(keyword); setPage(1) }}>
          <Search className="mr-1 size-4" />查询
        </Button>
        <Button size="sm" variant="outline" onClick={() => { setKeyword(""); setSearchKeyword(""); setStatusFilter(""); setPage(1) }}>
          <X className="mr-1 size-4" />重置
        </Button>
      </div>

      {listQuery.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(listQuery.error)}</AlertDescription></Alert>}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>角色编码</TableHead>
              <TableHead>角色名称</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>领域数</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead className="w-16">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 && (
              <TableRow><TableCell colSpan={6} className="text-center text-muted-foreground">{listQuery.isLoading ? "加载中…" : "暂无数据"}</TableCell></TableRow>
            )}
            {items.map((role) => (
              <TableRow key={role.role_id}>
                <TableCell className="font-medium">{role.role_code}</TableCell>
                <TableCell>{role.role_name}</TableCell>
                <TableCell><Badge variant={role.status === 1 ? "default" : "secondary"}>{STATUS_MAP[role.status] ?? role.status}</Badge></TableCell>
                <TableCell>{role.domains?.length ?? 0}</TableCell>
                <TableCell className="text-sm text-muted-foreground">{formatUnix(role.created_at_unix)}</TableCell>
                <TableCell>
                  <RoleRowActions role={role} onEdit={() => openEdit(role)} onDelete={() => setDeleteTarget(role)} onDomains={() => openDomains(role)} />
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
        <DialogContent>
          <DialogHeader><DialogTitle>{editTarget ? "编辑角色" : "新建角色"}</DialogTitle></DialogHeader>
          {editTarget ? (
            <RoleEditForm role={editTarget} onClose={() => setFormOpen(false)} />
          ) : (
            <form onSubmit={handleCreateSubmit} className="space-y-4">
              <div className="space-y-2"><Label>角色编码</Label><Input value={formData.role_code} onChange={(e) => setFormData({ ...formData, role_code: e.target.value })} required /></div>
              <div className="space-y-2"><Label>角色名称</Label><Input value={formData.role_name} onChange={(e) => setFormData({ ...formData, role_name: e.target.value })} required /></div>
              {createMutation.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(createMutation.error)}</AlertDescription></Alert>}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setFormOpen(false)}>取消</Button>
                <Button type="submit" disabled={createMutation.isPending}>{createMutation.isPending ? "保存中…" : "保存"}</Button>
              </div>
            </form>
          )}
        </DialogContent>
      </Dialog>

      {deleteTarget && <DeleteRoleDialog role={deleteTarget} onClose={() => setDeleteTarget(null)} />}
      {domainsTarget && (
        <DomainsDialog role={domainsTarget} selectedDomains={selectedDomains} setSelectedDomains={setSelectedDomains} onClose={() => setDomainsTarget(null)} />
      )}
    </div>
  )
}

