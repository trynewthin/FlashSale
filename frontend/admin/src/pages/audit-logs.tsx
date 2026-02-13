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
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { Search, X } from "lucide-react"

import { type AdminAuditLogView } from "@/api/modules/admin"
import { useAuditLogsQuery } from "@/hooks/admin/use-audit-hooks"
import { useAdminPermission } from "@/hooks/auth/use-admin-permission"
import { useApiError } from "@/hooks/common/use-api-error"

function formatUnix(unix: number) {
  if (!unix) return "-"
  return new Date(unix * 1000).toLocaleString("zh-CN")
}

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

  const handleSearch = () => {
    setSearchAdminId(adminIdFilter)
    setSearchAction(action)
    setPage(1)
  }

  const handleReset = () => {
    setAdminIdFilter("")
    setAction("")
    setSearchAdminId("")
    setSearchAction("")
    setPage(1)
  }

  return (
    <div className="space-y-4 p-6">
      <h1 className="text-2xl font-semibold tracking-tight">审计日志</h1>

      <div className="flex items-end gap-3">
        <div className="space-y-1">
          <Label className="text-xs">操作者 ID</Label>
          <Input placeholder="admin_id" value={adminIdFilter} onChange={(e) => setAdminIdFilter(e.target.value)} className="w-36"
            onKeyDown={(e) => e.key === "Enter" && handleSearch()} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">动作</Label>
          <Select value={action} onValueChange={(v) => setAction(v ?? "")}>
            <SelectTrigger className="w-36"><SelectValue placeholder="全部" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="create">create</SelectItem>
              <SelectItem value="update">update</SelectItem>
              <SelectItem value="delete">delete</SelectItem>
              <SelectItem value="login">login</SelectItem>
              <SelectItem value="status_change">status_change</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button size="sm" onClick={handleSearch}><Search className="mr-1 size-4" />查询</Button>
        <Button size="sm" variant="outline" onClick={handleReset}><X className="mr-1 size-4" />重置</Button>
      </div>

      {listQuery.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(listQuery.error)}</AlertDescription></Alert>}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>操作者</TableHead>
              <TableHead>动作</TableHead>
              <TableHead>目标类型</TableHead>
              <TableHead>目标 ID</TableHead>
              <TableHead>结果</TableHead>
              <TableHead>request_id</TableHead>
              <TableHead>时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 && (
              <TableRow><TableCell colSpan={7} className="text-center text-muted-foreground">{listQuery.isLoading ? "加载中…" : "暂无数据"}</TableCell></TableRow>
            )}
            {items.map((log) => (
              <TableRow key={log.log_id} className="cursor-pointer hover:bg-accent/50" onClick={() => setDetailLog(log)}>
                <TableCell>{log.admin_id}</TableCell>
                <TableCell><Badge variant="outline">{log.action}</Badge></TableCell>
                <TableCell>{log.target_type}</TableCell>
                <TableCell className="font-mono text-xs">{log.target_id}</TableCell>
                <TableCell>
                  <Badge variant={log.result === "success" ? "default" : "destructive"}>{log.result}</Badge>
                </TableCell>
                <TableCell className="max-w-32 truncate font-mono text-xs">{log.request_id}</TableCell>
                <TableCell className="text-sm text-muted-foreground">{formatUnix(log.created_at_unix)}</TableCell>
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

      <Sheet open={!!detailLog} onOpenChange={(open) => !open && setDetailLog(null)}>
        <SheetContent>
          <SheetHeader><SheetTitle>审计详情</SheetTitle></SheetHeader>
          {detailLog && (
            <ScrollArea className="mt-4 h-[calc(100vh-8rem)]">
              <div className="space-y-3 pr-4">
                <InfoRow label="日志 ID" value={detailLog.log_id} />
                <InfoRow label="操作者 ID" value={detailLog.admin_id} />
                <InfoRow label="动作" value={detailLog.action} />
                <InfoRow label="目标类型" value={detailLog.target_type} />
                <InfoRow label="目标 ID" value={detailLog.target_id} />
                <InfoRow label="结果" value={detailLog.result} />
                <InfoRow label="Request ID" value={detailLog.request_id} />
                <InfoRow label="IP" value={detailLog.ip} />
                <InfoRow label="User Agent" value={detailLog.user_agent} />
                <InfoRow label="时间" value={formatUnix(detailLog.created_at_unix)} />
                <Separator />
                <div>
                  <p className="mb-1 text-xs font-medium text-muted-foreground">详情 JSON</p>
                  <pre className="whitespace-pre-wrap rounded bg-muted p-3 text-xs">
                    {detailLog.detail_json ? (() => { try { return JSON.stringify(JSON.parse(detailLog.detail_json), null, 2) } catch { return detailLog.detail_json } })() : "-"}
                  </pre>
                </div>
              </div>
            </ScrollArea>
          )}
        </SheetContent>
      </Sheet>
    </div>
  )
}

function InfoRow({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-sm break-all">{value ?? "-"}</p>
    </div>
  )
}
