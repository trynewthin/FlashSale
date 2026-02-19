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
import { Plus, Search, X, Lock, AlertCircle } from "lucide-react"

import { type RoleView } from "@/api/modules/admin"
import { useRoleListQuery, useCreateRoleMutation } from "@/hooks/admin/use-role-hooks"
import { useAdminPermission } from "@/hooks/auth/use-admin-permission"
import { useApiError } from "@/hooks/common/use-api-error"
import { formatUnix } from "@/lib/format"

import { RoleEditForm, DeleteRoleDialog, DomainsDialog, RoleRowActions } from "@/components/role"

const DOMAIN_LABELS: Record<string, string> = {
  admin_management: "管理员",
  user_management: "用户",
  product_management: "商品",
  order_management: "订单",
  seckill_management: "秒杀",
}

// ── 角色图标 ──────────────────────────────────────────────
function RoleIcon({ role }: { role: RoleView }) {
  const letter = (role.role_name || role.role_code).charAt(0).toUpperCase()
  return (
    <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-950 text-xs font-semibold text-violet-600 dark:text-violet-400">
      {letter}
    </span>
  )
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

  const handleSearch = () => { setSearchKeyword(keyword); setPage(1) }
  const handleReset = () => { setKeyword(""); setSearchKeyword(""); setStatusFilter(""); setPage(1) }

  return (
    <div className="space-y-5 p-6">

      {/* ── 顶部 ── */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">角色管理</h1>
          <p className="text-sm text-muted-foreground mt-0.5">共 {total} 个角色</p>
        </div>
        <Button onClick={openCreate} size="sm">
          <Plus className="mr-1.5 size-3.5" />新建角色
        </Button>
      </div>

      {/* ── 筛选栏 ── */}
      <div className="flex flex-wrap items-end gap-3 rounded-lg border bg-card px-4 py-3">
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">关键字</Label>
          <Input
            placeholder="角色编码 / 名称"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSearch()}
            className="h-8 w-44 text-sm"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">状态</Label>
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v ?? "")}
            items={[{ value: "1", label: "启用" }, { value: "2", label: "禁用" }]}>
            <SelectTrigger className="h-8 w-28 text-sm"><SelectValue placeholder="全部" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="1">启用</SelectItem>
              <SelectItem value="2">禁用</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="flex gap-2">
          <Button size="sm" className="h-8" onClick={handleSearch}>
            <Search className="mr-1 size-3.5" />查询
          </Button>
          <Button size="sm" variant="outline" className="h-8" onClick={handleReset}>
            <X className="mr-1 size-3.5" />重置
          </Button>
        </div>
      </div>

      {/* ── 错误 ── */}
      {listQuery.isError && (
        <div className="flex items-center gap-2 rounded-lg border border-destructive/40 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          <AlertCircle className="size-4 shrink-0" />
          {toUserMessage(listQuery.error)}
        </div>
      )}

      {/* ── 表格 ── */}
      <div className="rounded-lg border overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-muted/40 hover:bg-muted/40">
              <TableHead className="pl-4">角色</TableHead>
              <TableHead>编码</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>权限域</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead className="w-14 pr-4">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="py-12 text-center text-sm text-muted-foreground">
                  {listQuery.isLoading
                    ? <span className="flex items-center justify-center gap-2"><span className="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />加载中…</span>
                    : "暂无数据"}
                </TableCell>
              </TableRow>
            )}
            {items.map((role) => (
              <TableRow key={role.role_id}>
                {/* 角色（图标 + 名称 + 系统角色标记） */}
                <TableCell className="pl-4">
                  <div className="flex items-center gap-3">
                    <RoleIcon role={role} />
                    <div className="min-w-0">
                      <div className="flex items-center gap-1.5">
                        <span className="text-sm font-medium">{role.role_name}</span>
                        {role.is_system && (
                          <span title="系统内置角色，不可删除">
                            <Lock className="size-3 text-muted-foreground" />
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </TableCell>

                {/* 编码 */}
                <TableCell>
                  <code className="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                    {role.role_code}
                  </code>
                </TableCell>

                {/* 状态 */}
                <TableCell>
                  <Badge variant={role.status === 1 ? "default" : "secondary"} className="text-xs">
                    {role.status === 1 ? "启用" : "禁用"}
                  </Badge>
                </TableCell>

                {/* 权限域 */}
                <TableCell>
                  {role.domains?.length > 0 ? (
                    <div className="flex flex-wrap gap-1">
                      {role.domains.slice(0, 3).map((d) => (
                        <Badge key={d} variant="outline" className="text-[10px] px-1.5 py-0">
                          {DOMAIN_LABELS[d] ?? d.replace("_management", "")}
                        </Badge>
                      ))}
                      {role.domains.length > 3 && (
                        <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                          +{role.domains.length - 3}
                        </Badge>
                      )}
                    </div>
                  ) : (
                    <span className="text-xs text-muted-foreground">—</span>
                  )}
                </TableCell>

                {/* 创建时间 */}
                <TableCell className="text-xs text-muted-foreground">
                  {formatUnix(role.created_at_unix)}
                </TableCell>

                {/* 操作 */}
                <TableCell className="pr-4">
                  <RoleRowActions
                    role={role}
                    onEdit={() => openEdit(role)}
                    onDelete={() => setDeleteTarget(role)}
                    onDomains={() => openDomains(role)}
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {/* ── 分页 ── */}
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">第 {page} 页 · 共 {total} 条</span>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</Button>
          <span className="flex items-center px-2 text-muted-foreground">{page} / {totalPages || 1}</span>
          <Button size="sm" variant="outline" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</Button>
        </div>
      </div>

      {/* ── 新建/编辑 Dialog ── */}
      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editTarget ? "编辑角色" : "新建角色"}</DialogTitle>
          </DialogHeader>
          {editTarget ? (
            <RoleEditForm role={editTarget} onClose={() => setFormOpen(false)} />
          ) : (
            <form onSubmit={handleCreateSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label>角色编码</Label>
                <Input
                  value={formData.role_code}
                  onChange={(e) => setFormData({ ...formData, role_code: e.target.value })}
                  placeholder="如：editor、viewer"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label>角色名称</Label>
                <Input
                  value={formData.role_name}
                  onChange={(e) => setFormData({ ...formData, role_name: e.target.value })}
                  placeholder="如：编辑员、查看者"
                  required
                />
              </div>
              {createMutation.isError && (
                <Alert variant="destructive">
                  <AlertDescription>{toUserMessage(createMutation.error)}</AlertDescription>
                </Alert>
              )}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setFormOpen(false)}>取消</Button>
                <Button type="submit" disabled={createMutation.isPending}>
                  {createMutation.isPending ? "保存中…" : "保存"}
                </Button>
              </div>
            </form>
          )}
        </DialogContent>
      </Dialog>

      {deleteTarget && <DeleteRoleDialog role={deleteTarget} onClose={() => setDeleteTarget(null)} />}
      {domainsTarget && (
        <DomainsDialog
          role={domainsTarget}
          selectedDomains={selectedDomains}
          setSelectedDomains={setSelectedDomains}
          onClose={() => setDomainsTarget(null)}
        />
      )}
    </div>
  )
}
