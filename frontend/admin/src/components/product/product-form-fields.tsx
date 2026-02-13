import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"

export interface ProductFormData {
  name: string; main_image: string; description: string; price_cent: number; stock: number; status: number
}

export function ProductFormFields({ formData, setFormData }: { formData: ProductFormData; setFormData: (d: ProductFormData) => void }) {
  return (
    <>
      <div className="space-y-2"><Label>名称</Label><Input value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} required /></div>
      <div className="space-y-2"><Label>主图 URL</Label><Input value={formData.main_image} onChange={(e) => setFormData({ ...formData, main_image: e.target.value })} /></div>
      <div className="space-y-2"><Label>描述</Label><Textarea value={formData.description} onChange={(e) => setFormData({ ...formData, description: e.target.value })} rows={3} /></div>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2"><Label>价格（分）</Label><Input type="number" value={formData.price_cent} onChange={(e) => setFormData({ ...formData, price_cent: Number(e.target.value) })} required /></div>
        <div className="space-y-2"><Label>库存</Label><Input type="number" value={formData.stock} onChange={(e) => setFormData({ ...formData, stock: Number(e.target.value) })} required /></div>
      </div>
      <div className="space-y-2">
        <Label>状态</Label>
        <Select value={String(formData.status)} onValueChange={(v) => v && setFormData({ ...formData, status: Number(v) })}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent><SelectItem value="1">上架</SelectItem><SelectItem value="2">下架</SelectItem></SelectContent>
        </Select>
      </div>
    </>
  )
}
