import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Sheet, SheetContent, SheetHeader, SheetTitle,
} from "@/components/ui/sheet"
import { Search, X, Pencil, UserX, AlertCircle, Phone, Clock, Calendar } from "lucide-react"

import { type ManagedUserView } from "@/api/modules/user"
import { useUserListQuery } from "@/hooks/biz/use-user-mgmt-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { formatUnix } from "@/lib/format"

import { NicknameDialog, DeleteUserDialog } from "@/components/user"

const STATUS_MAP: Record<number, { label: string; color: string }> = {
  1: { label: "正常", color: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400" },
  2: { label: "禁用", color: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400" },
}

function UserAvatar({ user }: { user: ManagedUserView }) {
  const letter = (user.nickname || user.phone || "U").charAt(0).toUpperCase()
  return (
    <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-950 text-xs font-semibold text-blue-600 dark:text-blue-400">
      {letter}
    </span>
  )
}

export function UserManagementPage() {
  const { toUserMessage } = useApiError()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState("")
  const [searchKeyword, setSearchKeyword] = useState("")

  const listQuery = useUserListQuery({
    page, page_size: 20,
    keyword: searchKeyword || undefined,
  })

  const [detailUser, setDetailUser] = useState<ManagedUserView | null>(null)
  const [nicknameOpen, setNicknameOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const items = listQuery.data?.list ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / 20)

  const handleSearch = () => { setSearchKeyword(keyword); setPage(1) }
  const handleReset = () => { setKeyword(""); setSearchKeyword(""); setPage(1) }

  return (
    <div className="space-y-5 p-6">

      {/* 顶部 */}
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">用户管理</h1>
        <p className="text-sm text-muted-foreground mt-0.5">共 {total} 位用户</p>
      </div>

      {/* 筛选栏 */}
      <div className="flex flex-wrap items-end gap-3 rounded-lg border bg-card px-4 py-3">
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">关键字</Label>
          <Input
            placeholder="手机号 / 昵称 / 用户ID"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSearch()}
            className="h-8 w-56 text-sm"
          />
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
              <TableHead className="pl-4">用户</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>最近登录</TableHead>
              <TableHead>注册时间</TableHead>
              <TableHead className="w-24 pr-4">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="py-12 text-center text-sm text-muted-foreground">
                  {listQuery.isLoading
                    ? <span className="flex items-center justify-center gap-2"><span className="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />加载中…</span>
                    : "暂无数据"}
                </TableCell>
              </TableRow>
            )}
            {items.map((u) => {
              const statusInfo = STATUS_MAP[u.status]
              return (
                <TableRow
                  key={u.user_id}
                  className="cursor-pointer"
                  onClick={() => setDetailUser(u)}
                >
                  {/* 用户信息 */}
                  <TableCell className="pl-4">
                    <div className="flex items-center gap-3">
                      <UserAvatar user={u} />
                      <div className="min-w-0">
                        <p className="text-sm font-medium leading-none">{u.nickname || "未设置昵称"}</p>
                        <p className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground">
                          <Phone className="size-3" />{u.phone}
                        </p>
                      </div>
                    </div>
                  </TableCell>

                  {/* 状态 */}
                  <TableCell>
                    {statusInfo ? (
                      <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${statusInfo.color}`}>
                        {statusInfo.label}
                      </span>
                    ) : (
                      <span className="text-xs text-muted-foreground">{u.status}</span>
                    )}
                  </TableCell>

                  {/* 最近登录 */}
                  <TableCell className="text-xs text-muted-foreground">
                    {u.last_login_at_unix ? formatUnix(u.last_login_at_unix) : "—"}
                  </TableCell>

                  {/* 注册时间 */}
                  <TableCell className="text-xs text-muted-foreground">
                    {formatUnix(u.created_at_unix)}
                  </TableCell>

                  {/* 操作 */}
                  <TableCell className="pr-4" onClick={(e) => e.stopPropagation()}>
                    <div className="flex gap-1">
                      <Button size="icon" variant="ghost" className="size-7" title="改昵称"
                        onClick={() => { setDetailUser(u); setNicknameOpen(true) }}>
                        <Pencil className="size-3.5" />
                      </Button>
                      <Button size="icon" variant="ghost"
                        className="size-7 text-destructive hover:bg-destructive/10 hover:text-destructive"
                        title="删除用户"
                        onClick={() => { setDetailUser(u); setDeleteOpen(true) }}>
                        <UserX className="size-3.5" />
                      </Button>
                    </div>
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

      {/* 用户详情 Sheet */}
      <Sheet open={detailUser !== null && !nicknameOpen && !deleteOpen} onOpenChange={(open) => { if (!open) setDetailUser(null) }}>
        <SheetContent className="w-[360px] sm:max-w-[360px]">
          <SheetHeader>
            <SheetTitle className="flex items-center gap-3">
              {detailUser && <UserAvatar user={detailUser} />}
              <span>{detailUser?.nickname || detailUser?.phone || "用户详情"}</span>
            </SheetTitle>
          </SheetHeader>
          {detailUser && (
            <div className="space-y-5">
              <div className="rounded-lg border bg-muted/30 p-4 space-y-3">
                <div className="grid grid-cols-2 gap-3 text-sm">
                  <div><p className="text-xs text-muted-foreground mb-0.5">用户 ID</p><p className="font-mono text-xs">{detailUser.user_id}</p></div>
                  <div><p className="text-xs text-muted-foreground mb-0.5">手机号</p><p className="flex items-center gap-1"><Phone className="size-3 text-muted-foreground" />{detailUser.phone}</p></div>
                  <div><p className="text-xs text-muted-foreground mb-0.5">昵称</p><p>{detailUser.nickname || "—"}</p></div>
                  <div><p className="text-xs text-muted-foreground mb-0.5">状态</p>
                    {STATUS_MAP[detailUser.status] && (
                      <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${STATUS_MAP[detailUser.status].color}`}>
                        {STATUS_MAP[detailUser.status].label}
                      </span>
                    )}
                  </div>
                </div>
              </div>
              <div className="space-y-2 text-sm">
                <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">登录信息</p>
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Clock className="size-3.5" />
                  <span className="text-xs">最近登录：{detailUser.last_login_at_unix ? formatUnix(detailUser.last_login_at_unix) : "—"}</span>
                </div>
                <div className="flex items-center gap-2 text-muted-foreground">
                  <span className="size-3.5 text-center text-[10px]">IP</span>
                  <span className="font-mono text-xs">{detailUser.last_login_ip || "—"}</span>
                </div>
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Calendar className="size-3.5" />
                  <span className="text-xs">注册时间：{formatUnix(detailUser.created_at_unix)}</span>
                </div>
              </div>
              <div className="flex gap-2 pt-2">
                <Button size="sm" variant="outline" onClick={() => setNicknameOpen(true)}>
                  <Pencil className="mr-1 size-3" />改昵称
                </Button>
                <Button size="sm" variant="destructive" onClick={() => setDeleteOpen(true)}>
                  <UserX className="mr-1 size-3" />删除用户
                </Button>
              </div>
            </div>
          )}
        </SheetContent>
      </Sheet>

      {nicknameOpen && detailUser && (
        <NicknameDialog userId={detailUser.user_id} currentNickname={detailUser.nickname ?? ""} onClose={() => setNicknameOpen(false)} />
      )}
      {deleteOpen && detailUser && (
        <DeleteUserDialog userId={detailUser.user_id} onClose={() => setDeleteOpen(false)}
          onDeleted={() => { setDeleteOpen(false); setDetailUser(null) }} />
      )}
    </div>
  )
}
