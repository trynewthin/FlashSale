import { useState } from "react"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

import { type AdminProduct } from "@/api/modules/product"
import { ProductPicker } from "./product-picker"

export interface ItemFormData {
  product_id: string; seckill_price_cent: number; reserved_stock_total: number
  user_limit_mode: number; user_limit_window_sec: number; user_limit_qty: number; max_qty_per_order: number; status: number
}

// 价格辅助：分 ↔ 元
function centToYuan(cent: number) {
  if (!cent) return ""
  return (cent / 100).toFixed(2).replace(/\.?0+$/, "")
}
function yuanToCent(yuan: string): number {
  const n = parseFloat(yuan)
  if (isNaN(n)) return 0
  return Math.round(n * 100)
}

export function ItemFormFields({
  formData,
  setFormData,
  isCreate,
}: {
  formData: ItemFormData
  setFormData: (d: ItemFormData) => void
  isCreate?: boolean
}) {
  const [priceYuan, setPriceYuan] = useState(centToYuan(formData.seckill_price_cent))

  const handleProductSelect = (_id: string, product: AdminProduct | null) => {
    setFormData({ ...formData, product_id: _id })
    // 若未设置过秒杀价，自动填充商品原价 * 0.8
    if (product && !formData.seckill_price_cent) {
      const suggested = Math.round(product.price_cent * 0.8)
      setFormData({ ...formData, product_id: _id, seckill_price_cent: suggested })
      setPriceYuan(centToYuan(suggested))
    }
  }

  return (
    <>
      {/* 商品选择（仅新建时） */}
      {isCreate && (
        <div className="space-y-1.5">
          <Label>选择商品</Label>
          <ProductPicker
            value={formData.product_id}
            onChange={handleProductSelect}
          />
          {formData.product_id && (
            <p className="text-[10px] text-muted-foreground pl-0.5">
              ID: <span className="font-mono">{formData.product_id}</span>
            </p>
          )}
        </div>
      )}

      {/* 秒杀价 + 预占库存 */}
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label>
            秒杀价
            <span className="ml-1 text-[10px] text-muted-foreground font-normal">（元）</span>
          </Label>
          <div className="relative">
            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-sm text-muted-foreground pointer-events-none">¥</span>
            <Input
              type="number"
              min="0"
              step="0.01"
              placeholder="0.00"
              className="pl-7"
              value={priceYuan}
              onChange={(e) => {
                setPriceYuan(e.target.value)
                setFormData({ ...formData, seckill_price_cent: yuanToCent(e.target.value) })
              }}
              required
            />
          </div>
        </div>
        <div className="space-y-1.5">
          <Label>预占库存</Label>
          <Input
            type="number"
            min="0"
            placeholder="例：100"
            value={formData.reserved_stock_total || ""}
            onChange={(e) => setFormData({ ...formData, reserved_stock_total: Number(e.target.value) })}
            required
          />
        </div>
      </div>

      {/* 限购模式 + 窗口秒数 */}
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label>限购模式</Label>
          <Select
            value={String(formData.user_limit_mode)}
            onValueChange={(v) => v && setFormData({ ...formData, user_limit_mode: Number(v) })}
            items={[{ value: "0", label: "不限" }, { value: "1", label: "按时间窗口" }, { value: "2", label: "按活动总量" }]}
          >
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="0">不限</SelectItem>
              <SelectItem value="1">按时间窗口</SelectItem>
              <SelectItem value="2">按活动总量</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label>
            窗口时长
            <span className="ml-1 text-[10px] text-muted-foreground font-normal">（秒）</span>
          </Label>
          <Input
            type="number"
            min="0"
            placeholder={formData.user_limit_mode === 1 ? "例：3600" : "—"}
            disabled={formData.user_limit_mode !== 1}
            value={formData.user_limit_mode === 1 ? (formData.user_limit_window_sec || "") : ""}
            onChange={(e) => setFormData({ ...formData, user_limit_window_sec: Number(e.target.value) })}
          />
        </div>
      </div>

      {/* 限购件数 + 每单上限 */}
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label>
            限购件数
            <span className="ml-1 text-[10px] text-muted-foreground font-normal">（0 = 不限）</span>
          </Label>
          <Input
            type="number"
            min="0"
            placeholder="0"
            value={formData.user_limit_qty || ""}
            onChange={(e) => setFormData({ ...formData, user_limit_qty: Number(e.target.value) })}
          />
        </div>
        <div className="space-y-1.5">
          <Label>单次上限（件/单）</Label>
          <Input
            type="number"
            min="1"
            placeholder="1"
            value={formData.max_qty_per_order || ""}
            onChange={(e) => setFormData({ ...formData, max_qty_per_order: Number(e.target.value) })}
          />
        </div>
      </div>

      {/* 状态 */}
      <div className="space-y-1.5">
        <Label>状态</Label>
        <Select
          value={String(formData.status)}
          onValueChange={(v) => v && setFormData({ ...formData, status: Number(v) })}
          items={[{ value: "0", label: "停用" }, { value: "1", label: "启用" }]}
        >
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="1">启用</SelectItem>
            <SelectItem value="0">停用</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </>
  )
}
