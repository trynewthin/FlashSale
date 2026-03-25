import { cn } from "@/lib/utils"

import type { ServiceViewModel } from "./overview-constants"

export function ServiceStatusTile({ item }: { item: ServiceViewModel }) {
  const toneBg = 
    item.tone === "healthy" ? "bg-emerald-500" :
    item.tone === "partial" ? "bg-amber-500" :
    item.tone === "down" ? "bg-destructive" : "bg-muted-foreground"

  return (
    <div className="group relative flex flex-col gap-3 rounded-xl border border-transparent bg-muted/40 p-4 transition-colors hover:bg-muted/60">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2.5">
          <span className="relative flex size-2.5 shrink-0">
            {item.tone === "healthy" && (
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-20" />
            )}
            <span className={cn("relative inline-flex size-2.5 rounded-full", toneBg)} />
          </span>
          <span className="truncate text-sm font-medium text-foreground">{item.label}</span>
        </div>
        <span className="shrink-0 text-xs font-medium text-muted-foreground">{item.detailText}</span>
      </div>
    </div>
  )
}
