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

import { type AdminView } from "@/api/modules/auth"
import { useAdminListQuery, useCreateAdminMutation } from "@/hooks/admin/use-admin-account-hooks"
import { useRoleListQuery } from "@/hooks/admin/use-role-hooks"
import { useAdminPermission } from "@/hooks/auth/use-admin-permission"
import { useApiError } from "@/hooks/common/use-api-error"

import {
  AdminEditForm, DeleteAdminDialog, ResetPwdDialog, BindRolesDialog, AdminRowActions,
} from "@/components/admin"

const STATUS_MAP: Record<number, string> = { 1: "启用", 2: "禁用" }

function formatUnix(unix: number) {
  if (!unix) return "-"
  return new Date(unix * 1000).toLocaleString("zh-CN")
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
    if (editTarget) {
      // update handled via AdminFormDialog
    } else {
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

  const handleSearch = () => {
    setSearchKeyword(keyword)
    setPage(1)
  }

  const handleReset = () => {
    setKeyword("")
    setSearchKeyword("")
    setStatusFilter("")
    setPage(1)
  }

  const openBindRoles = (admin: AdminView) => {
    setBindRolesTarget(admin)
    setSelectedRoleIds(admin.role_ids?.map(String) ?? [])
  }

  return (
    <div className="space-y-4 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold tracking-tight">管理员列表</h1>
        <Button onClick={openCreate} size="sm">
          <Plus className="mr-1 size-4" />
          新建管理员
        </Button>
      </div>

      <div className="flex items-end gap-3">
        <div className="space-y-1">
          <Label className="text-xs">关键字</Label>
          <Input
            placeholder="用户名/昵称"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            className="w-48"
            onKeyDown={(e) => e.key === "Enter" && handleSearch()}
          />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">状态</Label>
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v ?? "")}>
            <SelectTrigger className="w-28">
              <SelectValue placeholder="全部" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="1">启用</SelectItem>
              <SelectItem value="2">禁用</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button size="sm" onClick={handleSearch}>
          <Search className="mr-1 size-4" />
          查询
        </Button>
        <Button size="sm" variant="outline" onClick={handleReset}>
          <X className="mr-1 size-4" />
          重置
        </Button>
      </div>

      {listQuery.isError && (
        <Alert variant="destructive">
          <AlertDescription>{toUserMessage(listQuery.error)}</AlertDescription>
        </Alert>
      )}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>用户名</TableHead>
              <TableHead>昵称</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>数据范围</TableHead>
              <TableHead>角色数</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead className="w-16">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} className="text-center text-muted-foreground">
                  {listQuery.isLoading ? "加载中…" : "暂无数据"}
                </TableCell>
              </TableRow>
            )}
            {items.map((admin) => (
              <TableRow key={admin.admin_id}>
                <TableCell className="font-medium">{admin.username}</TableCell>
                <TableCell>{admin.display_name || "-"}</TableCell>
                <TableCell>
                  <Badge variant={admin.status === 1 ? "default" : "secondary"}>
                    {STATUS_MAP[admin.status] ?? admin.status}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Badge variant="outline">{admin.data_scope}</Badge>
                </TableCell>
                <TableCell>{admin.role_ids?.length ?? 0}</TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {formatUnix(admin.created_at_unix)}
                </TableCell>
                <TableCell>
                  <AdminRowActions
                    admin={admin}
                    onEdit={() => openEdit(admin)}
                    onDelete={() => setDeleteTarget(admin)}
                    onResetPwd={() => { setResetPwdTarget(admin); setNewPassword("") }}
                    onBindRoles={() => openBindRoles(admin)}
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">共 {total} 条</span>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            上一页
          </Button>
          <span className="flex items-center px-2">
            {page} / {totalPages || 1}
          </span>
          <Button size="sm" variant="outline" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>
            下一页
          </Button>
        </div>
      </div>

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
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">all</SelectItem>
                    <SelectItem value="self">self</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {createMutation.isError && (
                <Alert variant="destructive">
                  <AlertDescription>{toUserMessage(createMutation.error)}</AlertDescription>
                </Alert>
              )}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setFormOpen(false)}>
                  取消
                </Button>
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

