import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

interface PageShellProps {
    /** 页面标题，显示在 Header 左侧 */
    title?: ReactNode
    /** Header 右侧操作按钮区域 */
    actions?: ReactNode
    /** 主内容 */
    children: ReactNode
    /** 内容区是否使用 padding（默认 true） */
    padded?: boolean
    /** 额外附加到根容器的 className */
    className?: string
    /** 额外附加到内容区的 className */
    contentClassName?: string
}

/**
 * PageShell — Ops 各页面的统一布局外壳。
 *
 * 结构：
 *   ┌─────────────────────────────────────────────┐
 *   │  Header: [title]          [actions...]      │
 *   ├─────────────────────────────────────────────┤
 *   │  Content (flex-1, overflow-y-auto)          │
 *   └─────────────────────────────────────────────┘
 *
 * 用法示例：
 *   <PageShell title="任务执行" actions={<Button>刷新</Button>}>
 *     <Card>...</Card>
 *   </PageShell>
 */
export function PageShell({
    title,
    actions,
    children,
    padded = true,
    className,
    contentClassName,
}: PageShellProps) {
    return (
        <div className={cn("flex h-full flex-col overflow-hidden", className)}>
            {/* Header 行：仅当 title 或 actions 存在时才渲染 */}
            {(title ?? actions) ? (
                <div className="flex shrink-0 items-center justify-between gap-4 bg-background/60 px-6 py-3 backdrop-blur-md">
                    {title ? (
                        <h1 className="text-base font-semibold tracking-tight text-foreground/90 truncate">
                            {title}
                        </h1>
                    ) : (
                        <span />
                    )}
                    {actions ? (
                        <div className="flex shrink-0 items-center gap-2">{actions}</div>
                    ) : null}
                </div>
            ) : null}

            {/* 主内容区：撑满剩余高度，可滚动 */}
            <div
                className={cn(
                    "flex-1 overflow-y-auto",
                    padded && "p-6",
                    contentClassName
                )}
            >
                {children}
            </div>
        </div>
    )
}
