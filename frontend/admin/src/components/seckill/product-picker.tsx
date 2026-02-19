import { useEffect, useRef, useState } from "react"

import { Input } from "@/components/ui/input"
import { Popover } from "@base-ui/react/popover"
import { Package, Search, X, Check, ChevronDown, Loader2 } from "lucide-react"

import { type AdminProduct } from "@/api/modules/product"
import { useAdminProductListQuery } from "@/hooks/biz/use-product-mgmt-hooks"
import { cdnUrl } from "@/lib/cdn"
import { formatCent } from "@/lib/format"
import { cn } from "@/lib/utils"

interface ProductPickerProps {
    value: string          // product_id (string bigint)
    onChange: (id: string, product: AdminProduct) => void
    placeholder?: string
    className?: string
}

export function ProductPicker({ value, onChange, placeholder = "搜索并选择商品…", className }: ProductPickerProps) {
    const [open, setOpen] = useState(false)
    const [keyword, setKeyword] = useState("")
    const [debouncedKw, setDebouncedKw] = useState("")
    const [selectedProduct, setSelectedProduct] = useState<AdminProduct | null>(null)
    const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

    // 关键字防抖
    useEffect(() => {
        if (debounceRef.current) clearTimeout(debounceRef.current)
        debounceRef.current = setTimeout(() => setDebouncedKw(keyword), 300)
        return () => { if (debounceRef.current) clearTimeout(debounceRef.current) }
    }, [keyword])

    const listQuery = useAdminProductListQuery(
        open ? { keyword: debouncedKw || undefined, page: 1, page_size: 20 } : undefined
    )
    const products = listQuery.data?.list ?? []

    const handleSelect = (p: AdminProduct) => {
        setSelectedProduct(p)
        onChange(String(p.product_id), p)
        setOpen(false)
        setKeyword("")
    }

    const handleClear = (e: React.MouseEvent) => {
        e.stopPropagation()
        setSelectedProduct(null)
        onChange("", null as unknown as AdminProduct)
    }

    return (
        <Popover.Root open={open} onOpenChange={setOpen}>
            <Popover.Trigger
                className={cn(
                    "flex h-9 w-full items-center gap-2 rounded-md border bg-transparent px-3 py-2 text-sm ring-offset-background",
                    "hover:bg-accent/40 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
                    "transition-colors",
                    className
                )}
            >
                {selectedProduct ? (
                    <>
                        {selectedProduct.main_image
                            ? <img src={cdnUrl(selectedProduct.main_image)} alt="" className="size-5 rounded object-cover shrink-0" />
                            : <Package className="size-4 shrink-0 text-muted-foreground" />
                        }
                        <span className="flex-1 truncate text-left">{selectedProduct.name}</span>
                        <span className="text-xs text-muted-foreground shrink-0">{formatCent(selectedProduct.price_cent)}</span>
                        <button
                            type="button"
                            onClick={handleClear}
                            className="ml-1 rounded-sm opacity-50 hover:opacity-100 focus:outline-none"
                        >
                            <X className="size-3.5" />
                        </button>
                    </>
                ) : (
                    <>
                        <Package className="size-4 shrink-0 text-muted-foreground" />
                        <span className="flex-1 text-left text-muted-foreground">{placeholder}</span>
                        <ChevronDown className="size-3.5 shrink-0 text-muted-foreground" />
                    </>
                )}
            </Popover.Trigger>

            <Popover.Portal>
                <Popover.Positioner sideOffset={6} align="start" className="z-50">
                    <Popover.Popup className="w-80 rounded-lg border bg-popover p-0 shadow-md">
                        {/* 搜索框 */}
                        <div className="flex items-center gap-2 border-b px-3 py-2">
                            <Search className="size-3.5 shrink-0 text-muted-foreground" />
                            <Input
                                autoFocus
                                placeholder="商品名称 / SKU"
                                value={keyword}
                                onChange={(e) => setKeyword(e.target.value)}
                                className="h-7 border-0 bg-transparent p-0 text-sm shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
                            />
                            {keyword && (
                                <button onClick={() => setKeyword("")} className="text-muted-foreground hover:text-foreground">
                                    <X className="size-3.5" />
                                </button>
                            )}
                        </div>

                        {/* 列表 */}
                        <div className="max-h-64 overflow-y-auto p-1">
                            {listQuery.isLoading && (
                                <div className="flex items-center justify-center py-6 text-xs text-muted-foreground">
                                    <Loader2 className="mr-1.5 size-3.5 animate-spin" />加载中…
                                </div>
                            )}
                            {!listQuery.isLoading && products.length === 0 && (
                                <div className="py-6 text-center text-xs text-muted-foreground">暂无商品</div>
                            )}
                            {products.map((p) => {
                                const isSelected = String(p.product_id) === value
                                return (
                                    <button
                                        key={String(p.product_id)}
                                        type="button"
                                        onClick={() => handleSelect(p)}
                                        className={cn(
                                            "flex w-full items-center gap-3 rounded-md px-2 py-2 text-left text-sm transition-colors hover:bg-accent",
                                            isSelected && "bg-accent"
                                        )}
                                    >
                                        {/* 缩略图 */}
                                        {p.main_image
                                            ? <img src={cdnUrl(p.main_image)} alt="" className="size-8 rounded object-cover shrink-0" />
                                            : <span className="flex size-8 items-center justify-center rounded bg-muted shrink-0"><Package className="size-4 text-muted-foreground" /></span>
                                        }
                                        {/* 信息 */}
                                        <div className="min-w-0 flex-1">
                                            <p className="truncate text-xs font-medium">{p.name}</p>
                                            <p className="truncate text-[10px] text-muted-foreground">
                                                {p.sku_code && <span className="mr-2">SKU: {p.sku_code}</span>}
                                                <span className="text-rose-600 dark:text-rose-400 font-medium">{formatCent(p.price_cent)}</span>
                                                {" · "}库存 {p.stock}
                                            </p>
                                        </div>
                                        {isSelected && <Check className="size-3.5 shrink-0 text-primary" />}
                                    </button>
                                )
                            })}
                        </div>
                    </Popover.Popup>
                </Popover.Positioner>
            </Popover.Portal>
        </Popover.Root>
    )
}
