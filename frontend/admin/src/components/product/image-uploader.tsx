import { useState, useRef, useCallback } from "react"
import { Upload, X, Loader2, ImageIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { mediaApi } from "@/api/modules/media"
import { cdnUrl } from "@/lib/cdn"

interface ImageUploaderProps {
    value: string
    onChange: (url: string) => void
    category?: string
    label?: string
}

export function ImageUploader({
    value,
    onChange,
    category = "products",
    label = "主图",
}: ImageUploaderProps) {
    const [uploading, setUploading] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [dragOver, setDragOver] = useState(false)
    const fileInputRef = useRef<HTMLInputElement>(null)

    const handleUpload = useCallback(async (file: File) => {
        if (!file.type.startsWith("image/") && !file.name.endsWith(".svg")) {
            setError("请选择图片文件")
            return
        }
        if (file.size > 10 * 1024 * 1024) {
            setError("文件大小不能超过 10MB")
            return
        }

        setUploading(true)
        setError(null)
        try {
            const info = await mediaApi.upload(file, category)
            onChange(info.url)
        } catch (err) {
            setError(err instanceof Error ? err.message : "上传失败")
        } finally {
            setUploading(false)
        }
    }, [category, onChange])

    const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0]
        if (file) handleUpload(file)
        // 重置 input 以允许重复选择同一文件
        e.target.value = ""
    }

    const handleDrop = (e: React.DragEvent) => {
        e.preventDefault()
        setDragOver(false)
        const file = e.dataTransfer.files[0]
        if (file) handleUpload(file)
    }

    const handleClear = () => {
        onChange("")
        setError(null)
    }

    const hasImage = value && value.trim() !== ""

    return (
        <div className="space-y-2">
            <Label>{label}</Label>

            {hasImage ? (
                // ─── 已有图片：预览 ───
                <div className="relative group">
                    <div className="relative overflow-hidden rounded-lg border bg-muted/30">
                        <img
                            src={cdnUrl(value)}
                            alt="商品主图"
                            className="h-40 w-full object-contain"
                            onError={(e) => {
                                (e.target as HTMLImageElement).style.display = "none"
                            }}
                        />
                    </div>
                    <div className="mt-2 flex items-center gap-2">
                        <Input
                            value={value}
                            onChange={(e) => onChange(e.target.value)}
                            placeholder="图片 URL"
                            className="flex-1 text-xs"
                        />
                        <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={handleClear}
                            className="shrink-0 text-destructive hover:text-destructive"
                        >
                            <X className="size-4" />
                        </Button>
                        <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            onClick={() => fileInputRef.current?.click()}
                            disabled={uploading}
                            className="shrink-0"
                        >
                            {uploading ? <Loader2 className="size-4 animate-spin" /> : <Upload className="size-4" />}
                        </Button>
                    </div>
                </div>
            ) : (
                // ─── 无图片：上传区域 ───
                <div
                    className={`
            flex flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed p-6
            cursor-pointer transition-colors
            ${dragOver
                            ? "border-primary bg-primary/5"
                            : "border-muted-foreground/25 hover:border-muted-foreground/50 hover:bg-muted/30"
                        }
          `}
                    onClick={() => fileInputRef.current?.click()}
                    onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
                    onDragLeave={() => setDragOver(false)}
                    onDrop={handleDrop}
                >
                    {uploading ? (
                        <Loader2 className="size-8 animate-spin text-muted-foreground" />
                    ) : (
                        <ImageIcon className="size-8 text-muted-foreground/60" />
                    )}
                    <p className="text-sm text-muted-foreground">
                        {uploading ? "上传中…" : "点击或拖拽图片上传"}
                    </p>
                    <p className="text-xs text-muted-foreground/60">
                        支持 JPG、PNG、SVG，最大 10MB
                    </p>
                </div>
            )}

            {/* 手动输入 URL（无图片时的备选） */}
            {!hasImage && (
                <div className="flex items-center gap-2">
                    <Input
                        value={value}
                        onChange={(e) => onChange(e.target.value)}
                        placeholder="或直接粘贴图片 URL"
                        className="text-xs"
                    />
                </div>
            )}

            {error && (
                <p className="text-xs text-destructive">{error}</p>
            )}

            <input
                ref={fileInputRef}
                type="file"
                accept="image/*,.svg"
                onChange={handleFileSelect}
                className="hidden"
            />
        </div>
    )
}
