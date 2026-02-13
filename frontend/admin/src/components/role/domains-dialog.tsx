import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"

import { type RoleView } from "@/api/modules/admin"
import { useSetRoleDomainsMutation } from "@/hooks/admin/use-role-hooks"

const ALL_DOMAINS = [
  "admin_management", "user_management", "product_management",
  "order_management", "order_review_management", "seckill_management",
]

export function DomainsDialog({ role, selectedDomains, setSelectedDomains, onClose }: {
  role: RoleView; selectedDomains: string[]; setSelectedDomains: (d: string[]) => void; onClose: () => void
}) {
  const domainsMutation = useSetRoleDomainsMutation(role.role_id)
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader><DialogTitle>领域配置 - {role.role_name}</DialogTitle></DialogHeader>
        <ScrollArea className="max-h-64">
          <div className="space-y-2 p-1">
            {ALL_DOMAINS.map((d) => {
              const checked = selectedDomains.includes(d)
              return (
                <label key={d} className="flex items-center gap-2 rounded p-2 hover:bg-accent">
                  <Checkbox checked={checked} onCheckedChange={(c) => setSelectedDomains(c ? [...selectedDomains, d] : selectedDomains.filter((x) => x !== d))} />
                  <span className="text-sm">{d}</span>
                </label>
              )
            })}
          </div>
        </ScrollArea>
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="outline" onClick={onClose}>取消</Button>
          <Button onClick={() => domainsMutation.mutate({ domains: selectedDomains }, { onSuccess: onClose })} disabled={domainsMutation.isPending}>
            {domainsMutation.isPending ? "保存中…" : "确认保存"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
