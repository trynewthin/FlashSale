import { Package, type LucideIcon } from "lucide-react"

import { Button } from "@/components/ui/button"

export interface EmptyStateProps {
    icon?: LucideIcon
    title?: string
    description?: string
    actionLabel?: string
    onAction?: () => void
}

export function EmptyState({
    icon: Icon = Package,
    title = "暂无数据",
    description = "目前还没有相关数据呢，去别处逛逛吧",
    actionLabel,
    onAction,
}: EmptyStateProps) {
    return (
        <div className="flex flex-col items-center justify-center py-16 text-center space-y-4 animate-in fade-in zoom-in duration-500">
            <div className="flex size-20 items-center justify-center rounded-full bg-muted/50 transition-transform hover:scale-105 duration-300">
                <Icon className="size-10 text-muted-foreground/40 stroke-[1.5]" />
            </div>
            <div className="space-y-1.5 pb-2">
                <h3 className="text-lg font-semibold tracking-tight">{title}</h3>
                <p className="text-sm text-muted-foreground max-w-xs mx-auto">
                    {description}
                </p>
            </div>
            {actionLabel && onAction && (
                <Button onClick={onAction} variant="outline" className="rounded-full px-6">
                    {actionLabel}
                </Button>
            )}
        </div>
    )
}
