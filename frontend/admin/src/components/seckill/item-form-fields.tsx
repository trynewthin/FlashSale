import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

export interface ItemFormData {
  product_id: string; seckill_price_cent: number; reserved_stock_total: number
  user_limit_mode: number; user_limit_window_sec: number; user_limit_qty: number; max_qty_per_order: number; status: number
}

export function ItemFormFields({ formData, setFormData, isCreate }: { formData: ItemFormData; setFormData: (d: ItemFormData) => void; isCreate?: boolean }) {
  return (
    <>
      {isCreate && <div className="space-y-2"><Label>商品 ID</Label><Input value={formData.product_id} onChange={(e) => setFormData({ ...formData, product_id: e.target.value })} required /></div>}
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2"><Label>秒杀价（分）</Label><Input type="number" value={formData.seckill_price_cent} onChange={(e) => setFormData({ ...formData, seckill_price_cent: Number(e.target.value) })} required /></div>
        <div className="space-y-2"><Label>预占库存</Label><Input type="number" value={formData.reserved_stock_total} onChange={(e) => setFormData({ ...formData, reserved_stock_total: Number(e.target.value) })} required /></div>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2">
          <Label>限购模式</Label>
          <Select value={String(formData.user_limit_mode)} onValueChange={(v) => v && setFormData({ ...formData, user_limit_mode: Number(v) })}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="0">不限</SelectItem>
              <SelectItem value="1">按窗口</SelectItem>
              <SelectItem value="2">按活动</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2"><Label>窗口秒数</Label><Input type="number" value={formData.user_limit_window_sec} onChange={(e) => setFormData({ ...formData, user_limit_window_sec: Number(e.target.value) })} /></div>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2"><Label>限购件数</Label><Input type="number" value={formData.user_limit_qty} onChange={(e) => setFormData({ ...formData, user_limit_qty: Number(e.target.value) })} /></div>
        <div className="space-y-2"><Label>单次上限</Label><Input type="number" value={formData.max_qty_per_order} onChange={(e) => setFormData({ ...formData, max_qty_per_order: Number(e.target.value) })} /></div>
      </div>
      <div className="space-y-2">
        <Label>状态</Label>
        <Select value={String(formData.status)} onValueChange={(v) => v && setFormData({ ...formData, status: Number(v) })}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent><SelectItem value="1">启用</SelectItem><SelectItem value="2">禁用</SelectItem></SelectContent>
        </Select>
      </div>
    </>
  )
}
