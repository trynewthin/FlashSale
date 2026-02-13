import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"

export interface ActivityFormData {
  title: string; description: string; style_config_json: string; start_at_unix: number; end_at_unix: number
}

export function ActivityFormFields({ formData, setFormData, startStr, setStartStr, endStr, setEndStr }: {
  formData: ActivityFormData; setFormData: (d: ActivityFormData) => void
  startStr: string; setStartStr: (s: string) => void; endStr: string; setEndStr: (s: string) => void
}) {
  return (
    <>
      <div className="space-y-2"><Label>标题</Label><Input value={formData.title} onChange={(e) => setFormData({ ...formData, title: e.target.value })} required /></div>
      <div className="space-y-2"><Label>描述</Label><Textarea value={formData.description} onChange={(e) => setFormData({ ...formData, description: e.target.value })} rows={2} /></div>
      <div className="space-y-2"><Label>样式配置 JSON</Label><Textarea value={formData.style_config_json} onChange={(e) => setFormData({ ...formData, style_config_json: e.target.value })} rows={2} /></div>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2"><Label>开始时间</Label><Input type="datetime-local" value={startStr} onChange={(e) => setStartStr(e.target.value)} required /></div>
        <div className="space-y-2"><Label>结束时间</Label><Input type="datetime-local" value={endStr} onChange={(e) => setEndStr(e.target.value)} required /></div>
      </div>
    </>
  )
}
