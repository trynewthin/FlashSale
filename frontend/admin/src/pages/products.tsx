import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { Plus, Search, X } from "lucide-react"

import { type AdminProduct } from "@/api/modules/product"
import { useAdminProductListQuery, useCreateProductMutation } from "@/hooks/biz/use-product-mgmt-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { formatCent, formatUnix } from "@/lib/format"

import {
  ProductFormFields,
  ProductEditForm, DeleteProductDialog, ProductRowActions,
} from "@/components/product"

const STATUS_MAP: Record<number, string> = { 1: "上架", 2: "下架" }

export function ProductManagementPage() {
  const { toUserMessage } = useApiError()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState("")
  const [searchKeyword, setSearchKeyword] = useState("")

  const listQuery = useAdminProductListQuery({
    page, page_size: 20,
    keyword: searchKeyword || undefined,
  })

  const createMutation = useCreateProductMutation()

  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<AdminProduct | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<AdminProduct | null>(null)
  const [formData, setFormData] = useState({
    name: "", main_image: "", description: "", price_cent: 0, stock: 0, status: 1,
  })

  const items = listQuery.data?.list ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / 20)

  const openCreate = () => {
    setEditTarget(null)
    setFormData({ name: "", main_image: "", description: "", price_cent: 0, stock: 0, status: 1 })
    setDrawerOpen(true)
  }

  const openEdit = (p: AdminProduct) => {
    setEditTarget(p)
    setFormData({
      name: p.name, main_image: p.main_image, description: p.description,
      price_cent: p.price_cent, stock: p.stock, status: p.status,
    })
    setDrawerOpen(true)
  }

  const handleCreateSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    createMutation.mutate(formData, { onSuccess: () => setDrawerOpen(false) })
  }

  return (
    <div className="space-y-4 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold tracking-tight">商品管理</h1>
        <Button onClick={openCreate} size="sm"><Plus className="mr-1 size-4" />新建商品</Button>
      </div>

      <div className="flex items-end gap-3">
        <div className="space-y-1">
          <Label className="text-xs">关键字</Label>
          <Input placeholder="商品名称/SKU" value={keyword} onChange={(e) => setKeyword(e.target.value)} className="w-48"
            onKeyDown={(e) => { if (e.key === "Enter") { setSearchKeyword(keyword); setPage(1) } }} />
        </div>
        <Button size="sm" onClick={() => { setSearchKeyword(keyword); setPage(1) }}><Search className="mr-1 size-4" />查询</Button>
        <Button size="sm" variant="outline" onClick={() => { setKeyword(""); setSearchKeyword(""); setPage(1) }}><X className="mr-1 size-4" />重置</Button>
      </div>

      {listQuery.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(listQuery.error)}</AlertDescription></Alert>}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>SKU</TableHead>
              <TableHead>名称</TableHead>
              <TableHead>价格</TableHead>
              <TableHead>库存</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="w-16">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 && (
              <TableRow><TableCell colSpan={7} className="text-center text-muted-foreground">{listQuery.isLoading ? "加载中…" : "暂无数据"}</TableCell></TableRow>
            )}
            {items.map((p) => (
              <TableRow key={p.product_id}>
                <TableCell className="font-mono text-xs">{p.sku_code}</TableCell>
                <TableCell className="font-medium">{p.name}</TableCell>
                <TableCell>{formatCent(p.price_cent)}</TableCell>
                <TableCell>{p.stock}</TableCell>
                <TableCell><Badge variant={p.status === 1 ? "default" : "secondary"}>{STATUS_MAP[p.status] ?? p.status}</Badge></TableCell>
                <TableCell className="text-sm text-muted-foreground">{formatUnix(p.updated_at_unix)}</TableCell>
                <TableCell>
                  <ProductRowActions product={p} onEdit={() => openEdit(p)} onDelete={() => setDeleteTarget(p)} />
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

      <Sheet open={drawerOpen} onOpenChange={setDrawerOpen}>
        <SheetContent>
          <SheetHeader><SheetTitle>{editTarget ? "编辑商品" : "新建商品"}</SheetTitle></SheetHeader>
          {editTarget ? (
            <ProductEditForm product={editTarget} formData={formData} setFormData={setFormData} onClose={() => setDrawerOpen(false)} />
          ) : (
            <form onSubmit={handleCreateSubmit} className="mt-4 space-y-4">
              <ProductFormFields formData={formData} setFormData={setFormData} />
              {createMutation.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(createMutation.error)}</AlertDescription></Alert>}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setDrawerOpen(false)}>取消</Button>
                <Button type="submit" disabled={createMutation.isPending}>{createMutation.isPending ? "保存中…" : "保存"}</Button>
              </div>
            </form>
          )}
        </SheetContent>
      </Sheet>

      {deleteTarget && <DeleteProductDialog product={deleteTarget} onClose={() => setDeleteTarget(null)} />}
    </div>
  )
}

