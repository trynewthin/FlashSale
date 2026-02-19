import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Search, X, AlertCircle, CheckCircle2, XCircle, ChevronRight } from "lucide-react"

import { type AdminAuditLogView } from "@/api/modules/admin"
import { useAuditLogsQuery } from "@/hooks/admin/use-audit-hooks"
import { useAdminPermission } from "@/hooks/auth/use-admin-permission"
import { useApiError } from "@/hooks/common/use-api-error"
import { formatUnix } from "@/lib/format"

// ── 常量映射 ──────────────────────────────────────────────
const ACTION_MAP: Record<string, { label: string; color: string }> = {
  create: { label: "创建", color: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400" },
  update: { label: "更新", color: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-400" },
  delete: { label: "删除", color: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400" },
  login: { label: "登录", color: "bg-violet-100 text-violet-700 dark:bg-violet-950 dark:text-violet-400" },
  status_change: { label: "状态变更", color: "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-400" },
}

const TARGET_LABELS: Record<string, string> = {
  admin: "管理员",
  role: "角色",
  product: "商品",
  order: "订单",
  user: "用户",
  seckill_activity: "秒杀活动",
  seckill_item: "秒杀商品",
}

function ActionBadge({ action }: { action: string }) {
  const info = ACTION_MAP[action]
  if (!info) {
    return (
      <span className="inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] font-medium">
        {action}
      </span>
    )
  }
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${info.color}`}>
      {info.label}
    </span>
  )
}

function ResultBadge({ result }: { result: string }) {
  const isSuccess = result === "success"
  return (
    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${isSuccess
      ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400"
      : "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400"
      }`}>
      {isSuccess
        ? <CheckCircle2 className="size-2.5" />
        : <XCircle className="size-2.5" />
      }
      {isSuccess ? "成功" : result}
    </span>
  )
}

// ── 详情 Sheet 中的字段行 ─────────────────────────────────
function DetailRow({ label, value, mono = false }: { label: string; value?: string | number | null; mono?: boolean }) {
  if (!value && value !== 0) return null
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">{label}</span>
      <span className={`break-all text-sm ${mono ? "font-mono" : ""}`}>{String(value)}</span>
    </div>
  )
}

// ── 主页面 ────────────────────────────────────────────────
export function AuditLogPage() {
  const { isDataScopeAll } = useAdminPermission()
  const { toUserMessage } = useApiError()
  const [page, setPage] = useState(1)
  const [action, setAction] = useState("")
  const [adminIdFilter, setAdminIdFilter] = useState("")
  const [searchAdminId, setSearchAdminId] = useState("")
  const [searchAction, setSearchAction] = useState("")
  const [detailLog, setDetailLog] = useState<AdminAuditLogView | null>(null)

  const listQuery = useAuditLogsQuery({
    page, page_size: 20,
    admin_id: searchAdminId || undefined,
    action: searchAction || undefined,
  })

  if (!isDataScopeAll()) {
    return (
      <div className="flex items-center justify-center p-12">
        <Alert variant="destructive" className="max-w-md">
          <AlertDescription>您的数据范围不是 all，无法访问审计日志页面。</AlertDescription>
        </Alert>
      </div>
    )
  }

  const items = listQuery.data?.items ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / 20)

  const handleSearch = () => { setSearchAdminId(adminIdFilter); setSearchAction(action); setPage(1) }
  const handleReset = () => { setAdminIdFilter(""); setAction(""); setSearchAdminId(""); setSearchAction(""); setPage(1) }

  // 格式化 detail_json
  const formatJson = (raw: string) => {
    try { return JSON.stringify(JSON.parse(raw), null, 2) } catch { return raw }
  }

  return (
    <div className="space-y-5 p-6">

      {/* ── 顶部 ── */}
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">审计日志</h1>
        <p className="text-sm text-muted-foreground mt-0.5">共 {total} 条操作记录</p>
      </div>

      {/* ── 筛选栏 ── */}
      <div className="flex flex-wrap items-end gap-3 rounded-lg border bg-card px-4 py-3">
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">操作者 ID</Label>
          <Input
            placeholder="admin_id"
            value={adminIdFilter}
            onChange={(e) => setAdminIdFilter(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSearch()}
            className="h-8 w-36 text-sm font-mono"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">动作类型</Label>
          <Select value={action} onValueChange={(v) => setAction(v ?? "")}
            items={[
              { value: "create", label: "创建" },
              { value: "update", label: "更新" },
              { value: "delete", label: "删除" },
              { value: "login", label: "登录" },
              { value: "status_change", label: "状态变更" },
            ]}>
            <SelectTrigger className="h-8 w-36 text-sm"><SelectValue placeholder="全部" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="create">创建</SelectItem>
              <SelectItem value="update">更新</SelectItem>
              <SelectItem value="delete">删除</SelectItem>
              <SelectItem value="login">登录</SelectItem>
              <SelectItem value="status_change">状态变更</SelectItem>
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
              <TableHead className="pl-4">时间</TableHead>
              <TableHead>操作者</TableHead>
              <TableHead>动作</TableHead>
              <TableHead>目标</TableHead>
              <TableHead>IP</TableHead>
              <TableHead>结果</TableHead>
              <TableHead className="w-10 pr-4" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} className="py-12 text-center text-sm text-muted-foreground">
                  {listQuery.isLoading
                    ? <span className="flex items-center justify-center gap-2"><span className="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />加载中…</span>
                    : "暂无审计记录"}
                </TableCell>
              </TableRow>
            )}
            {items.map((log) => (
              <TableRow
                key={log.log_id}
                className="cursor-pointer"
                onClick={() => setDetailLog(log)}
              >
                {/* 时间 */}
                <TableCell className="pl-4 text-xs text-muted-foreground whitespace-nowrap">
                  {formatUnix(log.created_at_unix)}
                </TableCell>

                {/* 操作者 */}
                <TableCell>
                  <code className="rounded bg-muted px-1.5 py-0.5 text-xs">{log.admin_id}</code>
                </TableCell>

                {/* 动作 */}
                <TableCell><ActionBadge action={log.action} /></TableCell>

                {/* 目标 */}
                <TableCell>
                  <div className="flex flex-col gap-0.5">
                    <span className="text-xs font-medium">
                      {TARGET_LABELS[log.target_type] ?? log.target_type}
                    </span>
                    {log.target_id && (
                      <code className="text-[10px] text-muted-foreground">{log.target_id}</code>
                    )}
                  </div>
                </TableCell>

                {/* IP */}
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {log.ip || "—"}
                </TableCell>

                {/* 结果 */}
                <TableCell><ResultBadge result={log.result} /></TableCell>

                {/* 展开箭头 */}
                <TableCell className="pr-4">
                  <ChevronRight className="size-4 text-muted-foreground/50" />
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

      {/* ── 详情 Sheet ── */}
      <Sheet open={!!detailLog} onOpenChange={(open) => !open && setDetailLog(null)}>
        <SheetContent className="w-[440px] sm:max-w-[520px] flex flex-col">
          <SheetHeader>
            <SheetTitle className="flex items-center gap-2">
              审计详情
              {detailLog && <ResultBadge result={detailLog.result} />}
            </SheetTitle>
          </SheetHeader>

          {detailLog && (
            <ScrollArea className="flex-1">
              <div className="space-y-5 pr-1">

                {/* 基本信息区 */}
                <div className="rounded-lg border bg-muted/30 p-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <ActionBadge action={detailLog.action} />
                    <span className="text-xs text-muted-foreground">{formatUnix(detailLog.created_at_unix)}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <DetailRow label="日志 ID" value={detailLog.log_id} mono />
                    <DetailRow label="操作者 ID" value={detailLog.admin_id} mono />
                    <DetailRow label="目标类型" value={TARGET_LABELS[detailLog.target_type] ?? detailLog.target_type} />
                    <DetailRow label="目标 ID" value={detailLog.target_id} mono />
                  </div>
                </div>

                {/* 请求信息 */}
                <div className="space-y-3">
                  <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">请求信息</p>
                  <DetailRow label="Request ID" value={detailLog.request_id} mono />
                  <DetailRow label="IP 地址" value={detailLog.ip} mono />
                  <DetailRow label="User Agent" value={detailLog.user_agent} />
                </div>

                {/* 详情 JSON */}
                {detailLog.detail_json && (
                  <div className="space-y-2">
                    <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">变更详情</p>
                    <pre className="overflow-x-auto rounded-lg bg-muted p-3 text-xs leading-relaxed text-muted-foreground whitespace-pre-wrap break-all">
                      {formatJson(detailLog.detail_json)}
                    </pre>
                  </div>
                )}
              </div>
            </ScrollArea>
          )}
        </SheetContent>
      </Sheet>
    </div>
  )
}
