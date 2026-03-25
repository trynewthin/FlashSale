import { KeyRound, Loader2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

interface AuthOverlayProps {
  accessKey: string
  saving: boolean
  onChange: (value: string) => void
  onSave: () => void
}

export function AuthOverlay({ accessKey, saving, onChange, onSave }: AuthOverlayProps) {
  return (
    <div className="absolute inset-0 z-40 flex items-center justify-center bg-background/60 backdrop-blur-md">
      <div className="flex w-full max-w-sm flex-col gap-5 rounded-2xl border border-border/50 bg-background/90 p-6 shadow-xl backdrop-blur-xl">
        <div className="flex items-center gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10">
            <KeyRound className="size-5 text-primary" />
          </div>
          <h2 className="text-sm font-medium tracking-tight">请输入 Ops 访问密钥</h2>
        </div>

        <div className="flex w-full items-center gap-2">
          <Input
            value={accessKey}
            onChange={(e) => onChange(e.target.value)}
            placeholder="X-Ops-Key..."
            type="password"
            className="h-9 font-mono text-xs"
            disabled={saving}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !saving) onSave()
            }}
            autoFocus
          />
          <Button
            className="h-9 shrink-0 px-4"
            onClick={onSave}
            disabled={saving || !accessKey.trim()}
          >
            {saving ? <Loader2 className="size-4 animate-spin" /> : "确认"}
          </Button>
        </div>
      </div>
    </div>
  )
}
