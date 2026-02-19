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
import { Plus, Search, X, ShieldCheck, UserCog, AlertCircle } from "lucide-react"

import { type AdminView } from "@/api/modules/auth"
import { useAdminListQuery, useCreateAdminMutation } from "@/hooks/admin/use-admin-account-hooks"
import { useRoleListQuery } from "@/hooks/admin/use-role-hooks"
import { useAdminPermission } from "@/hooks/auth/use-admin-permission"
import { useApiError } from "@/hooks/common/use-api-error"
import { formatUnix } from "@/lib/format"

import {
  AdminEditForm, DeleteAdminDialog, ResetPwdDialog, BindRolesDialog, AdminRowActions,
} from "@/components/admin"

const STATUS_MAP: Record<number, { label: string; variant: "default" | "secondary" }> = {
  1: { label: "启用", variant: "default" },
  2: { label: "禁用", variant: "secondary" },
}

const SCOPE_MAP: Record<string, { label: string; variant: "default" | "secondary" | "outline" }> = {
  all: { label: "全部数据", variant: "default" },
  self: { label: "仅本人", variant: "outline" },
}

// ── 用户头像字母 ──────────────────────────────────────────
function AdminAvatar({ admin }: { admin: AdminView }) {
  const letter = (admin.display_name || admin.username).charAt(0).toUpperCase()
  return (
    <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
      {letter}
    </span>
  )
}

export function AdminListPage() {
  const { isDataScopeAll } = useAdminPermission()
  const { toUserMessage } = useApiError()

  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState("")
  const [statusFilter, setStatusFilter] = useState<string>("")
  const [searchKeyword, setSearchKeyword] = useState("")

  const listQuery = useAdminListQuery({
    page,
    page_size: 20,
    keyword: searchKeyword || undefined,
    status: statusFilter ? Number(statusFilter) : undefined,
  })

  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<AdminView | null>(null)
  const [formData, setFormData] = useState({
    username: "",
    display_name: "",
    password: "",
    data_scope: "self" as "all" | "self",
    status: 1,
  })

  const [deleteTarget, setDeleteTarget] = useState<AdminView | null>(null)
  const [resetPwdTarget, setResetPwdTarget] = useState<AdminView | null>(null)
  const [newPassword, setNewPassword] = useState("")
  const [bindRolesTarget, setBindRolesTarget] = useState<AdminView | null>(null)
  const [selectedRoleIds, setSelectedRoleIds] = useState<string[]>([])

  const createMutation = useCreateAdminMutation()
  const rolesQuery = useRoleListQuery({ page: 1, page_size: 100 })

  if (!isDataScopeAll()) {
    return (
      <div className="flex items-center justify-center p-12">
        <Alert variant="destructive" className="max-w-md">
          <AlertDescription>您的数据范围不是 all，无法访问管理员管理页面。</AlertDescription>
        </Alert>
      </div>
    )
  }

  const items = listQuery.data?.items ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / 20)

  const openCreate = () => {
    setEditTarget(null)
    setFormData({ username: "", display_name: "", password: "", data_scope: "self", status: 1 })
    setFormOpen(true)
  }

  const openEdit = (admin: AdminView) => {
    setEditTarget(admin)
    setFormData({
      username: admin.username,
      display_name: admin.display_name,
      password: "",
      data_scope: admin.data_scope as "all" | "self",
      status: admin.status,
    })
    setFormOpen(true)
  }

  const handleFormSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!editTarget) {
      createMutation.mutate(
        {
          username: formData.username,
          display_name: formData.display_name,
          password: formData.password,
          data_scope: formData.data_scope,
          status: formData.status,
        },
        { onSuccess: () => setFormOpen(false) }
      )
    }
  }

  const handleSearch = () => { setSearchKeyword(keyword); setPage(1) }
  const handleReset = () => { setKeyword(""); setSearchKeyword(""); setStatusFilter(""); setPage(1) }
  const openBindRoles = (admin: AdminView) => {
    setBindRolesTarget(admin)
    setSelectedRoleIds(admin.role_ids?.map(String) ?? [])
  }

  return (
    <div className="space-y-5 p-6">

      {/* ── 顶部 ── */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">管理员</h1>
          <p className="text-sm text-muted-foreground mt-0.5">共 {total} 位管理员</p>
        </div>
        <Button onClick={openCreate} size="sm">
          <Plus className="mr-1.5 size-3.5" />新建管理员
        </Button>
      </div>

      {/* ── 筛选栏 ── */}
      <div className="flex flex-wrap items-end gap-3 rounded-lg border bg-card px-4 py-3">
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">关键字</Label>
          <Input
            placeholder="用户名 / 昵称"
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
              <TableHead className="pl-4">管理员</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>数据范围</TableHead>
              <TableHead>权限域</TableHead>
              <TableHead>最近登录</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead className="w-14 pr-4">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} className="py-12 text-center text-sm text-muted-foreground">
                  {listQuery.isLoading
                    ? <span className="flex items-center justify-center gap-2"><span className="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />加载中…</span>
                    : "暂无数据"}
                </TableCell>
              </TableRow>
            )}
            {items.map((admin) => {
              const statusInfo = STATUS_MAP[admin.status] ?? { label: String(admin.status), variant: "outline" as const }
              const scopeInfo = SCOPE_MAP[admin.data_scope] ?? { label: admin.data_scope, variant: "outline" as const }
              return (
                <TableRow key={admin.admin_id} className="group">
                  {/* 管理员 */}
                  <TableCell className="pl-4">
                    <div className="flex items-center gap-3">
                      <AdminAvatar admin={admin} />
                      <div className="min-w-0">
                        <div className="flex items-center gap-1.5">
                          <span className="text-sm font-medium leading-none">{admin.display_name || admin.username}</span>
                          {admin.is_super_admin && (
                            <span title="超级管理员">
                              <ShieldCheck className="size-3.5 text-amber-500" />
                            </span>
                          )}
                        </div>
                        <span className="text-xs text-muted-foreground">{admin.username}</span>
                      </div>
                    </div>
                  </TableCell>

                  {/* 状态 */}
                  <TableCell>
                    <Badge variant={statusInfo.variant} className="text-xs">{statusInfo.label}</Badge>
                  </TableCell>

                  {/* 数据范围 */}
                  <TableCell>
                    <Badge variant={scopeInfo.variant} className="text-xs">{scopeInfo.label}</Badge>
                  </TableCell>

                  {/* 权限域 */}
                  <TableCell>
                    {admin.domains?.length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {admin.domains.slice(0, 2).map((d) => (
                          <Badge key={d} variant="outline" className="text-[10px] px-1.5 py-0">
                            <UserCog className="mr-0.5 size-2.5" />
                            {d.replace("_management", "")}
                          </Badge>
                        ))}
                        {admin.domains.length > 2 && (
                          <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                            +{admin.domains.length - 2}
                          </Badge>
                        )}
                      </div>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>

                  {/* 最近登录 */}
                  <TableCell className="text-xs text-muted-foreground">
                    {admin.last_login_at_unix ? formatUnix(admin.last_login_at_unix) : "—"}
                  </TableCell>

                  {/* 创建时间 */}
                  <TableCell className="text-xs text-muted-foreground">
                    {formatUnix(admin.created_at_unix)}
                  </TableCell>

                  {/* 操作 */}
                  <TableCell className="pr-4">
                    <AdminRowActions
                      admin={admin}
                      onEdit={() => openEdit(admin)}
                      onDelete={() => setDeleteTarget(admin)}
                      onResetPwd={() => { setResetPwdTarget(admin); setNewPassword("") }}
                      onBindRoles={() => openBindRoles(admin)}
                    />
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>

      {/* ── 分页 ── */}
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">
          第 {page} 页 · 共 {total} 条
        </span>
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
            <DialogTitle>{editTarget ? "编辑管理员" : "新建管理员"}</DialogTitle>
          </DialogHeader>
          {editTarget ? (
            <AdminEditForm admin={editTarget} onClose={() => setFormOpen(false)} />
          ) : (
            <form onSubmit={handleFormSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label>用户名</Label>
                <Input
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label>昵称</Label>
                <Input
                  value={formData.display_name}
                  onChange={(e) => setFormData({ ...formData, display_name: e.target.value })}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label>密码</Label>
                <Input
                  type="password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label>数据范围</Label>
                <Select
                  value={formData.data_scope}
                  onValueChange={(v) => v && setFormData({ ...formData, data_scope: v as "all" | "self" })}
                  items={[{ value: "all", label: "全部数据" }, { value: "self", label: "仅本人" }]}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">全部数据</SelectItem>
                    <SelectItem value="self">仅本人</SelectItem>
                  </SelectContent>
                </Select>
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

      {deleteTarget && (
        <DeleteAdminDialog admin={deleteTarget} onClose={() => setDeleteTarget(null)} />
      )}
      {resetPwdTarget && (
        <ResetPwdDialog
          admin={resetPwdTarget}
          newPassword={newPassword}
          setNewPassword={setNewPassword}
          onClose={() => { setResetPwdTarget(null); setNewPassword("") }}
        />
      )}
      {bindRolesTarget && (
        <BindRolesDialog
          admin={bindRolesTarget}
          roles={rolesQuery.data?.items ?? []}
          selectedRoleIds={selectedRoleIds}
          setSelectedRoleIds={setSelectedRoleIds}
          onClose={() => setBindRolesTarget(null)}
        />
      )}
    </div>
  )
}
