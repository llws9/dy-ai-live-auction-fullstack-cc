import React from "react"
import { Plus, Settings2, Trash2, Copy, FileText, X } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { auctionRuleTemplateApi, type AuctionRuleTemplate, type AuctionRuleTemplatePayload } from "@/shared/api"

const emptyForm: AuctionRuleTemplatePayload = {
  name: "",
  start_price: "0.00",
  increment: "10.00",
  cap_price: "",
  duration: 3600,
  delay_duration: 30,
  max_delay_time: 180,
  trigger_delay_before: 30,
  is_default: false,
}

function toPayload(template: AuctionRuleTemplate): AuctionRuleTemplatePayload {
  return {
    name: template.name,
    start_price: template.start_price,
    increment: template.increment,
    cap_price: template.cap_price || "",
    duration: template.duration,
    delay_duration: template.delay_duration,
    max_delay_time: template.max_delay_time,
    trigger_delay_before: template.trigger_delay_before,
    is_default: template.is_default,
  }
}

function describeTemplate(template: AuctionRuleTemplate) {
  const capPrice = template.cap_price ? `，封顶价 ${template.cap_price} 元` : ""
  return `起拍价 ${template.start_price} 元，加价幅度 ${template.increment} 元${capPrice}，竞拍 ${template.duration} 秒`
}

function normalizePayload(raw: AuctionRuleTemplatePayload): AuctionRuleTemplatePayload {
  return {
    ...raw,
    name: raw.name.trim(),
    start_price: raw.start_price.trim() || "0",
    increment: raw.increment.trim(),
    cap_price: raw.cap_price?.trim() || "",
    duration: Number(raw.duration),
    delay_duration: Number(raw.delay_duration),
    max_delay_time: Number(raw.max_delay_time),
    trigger_delay_before: Number(raw.trigger_delay_before),
  }
}

export default function AuctionRules() {
  const [templates, setTemplates] = React.useState<AuctionRuleTemplate[]>([])
  const [loading, setLoading] = React.useState(true)
  const [saving, setSaving] = React.useState(false)
  const [error, setError] = React.useState("")
  const [editing, setEditing] = React.useState<AuctionRuleTemplate | null>(null)
  const [formOpen, setFormOpen] = React.useState(false)
  const [form, setForm] = React.useState<AuctionRuleTemplatePayload>(emptyForm)

  const fetchTemplates = React.useCallback(async () => {
    setLoading(true)
    setError("")
    try {
      const response = await auctionRuleTemplateApi.list({ page: 1, page_size: 20 })
      setTemplates(response.list || [])
    } catch (e) {
      console.error("获取规则模板失败:", e)
      setError("获取规则模板失败，请稍后重试")
      setTemplates([])
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    fetchTemplates()
  }, [fetchTemplates])

  const openCreateForm = () => {
    setEditing(null)
    setForm(emptyForm)
    setFormOpen(true)
  }

  const openEditForm = (template: AuctionRuleTemplate) => {
    setEditing(template)
    setForm(toPayload(template))
    setFormOpen(true)
  }

  const handleChange = (field: keyof AuctionRuleTemplatePayload, value: string | boolean) => {
    setForm((current) => ({
      ...current,
      [field]: value,
    }))
  }

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    const payload = normalizePayload(form)
    if (!payload.name || !payload.increment || payload.duration <= 0) {
      setError("请填写模板名称、加价幅度和有效竞拍时长")
      return
    }

    setSaving(true)
    setError("")
    try {
      if (editing) {
        await auctionRuleTemplateApi.update(editing.id, payload)
      } else {
        await auctionRuleTemplateApi.create(payload)
      }
      setFormOpen(false)
      setEditing(null)
      await fetchTemplates()
    } catch (e) {
      console.error("保存规则模板失败:", e)
      setError("保存规则模板失败，请检查参数后重试")
    } finally {
      setSaving(false)
    }
  }

  const handleClone = async (template: AuctionRuleTemplate) => {
    setError("")
    try {
      await auctionRuleTemplateApi.create({
        ...toPayload(template),
        name: `${template.name} 副本`,
        is_default: false,
      })
      await fetchTemplates()
    } catch (e) {
      console.error("克隆规则模板失败:", e)
      setError("克隆规则模板失败，请稍后重试")
    }
  }

  const handleDelete = async (template: AuctionRuleTemplate) => {
    if (!confirm(`确定要删除「${template.name}」吗？`)) return
    setError("")
    try {
      await auctionRuleTemplateApi.delete(template.id)
      await fetchTemplates()
    } catch (e) {
      console.error("删除规则模板失败:", e)
      setError("删除规则模板失败，请稍后重试")
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">竞拍规则模板</h1>
          <p className="text-sm text-slate-500">商家维护可复用的竞拍参数，创建场次时可快速套用</p>
        </div>
        <Button className="bg-amber-500 hover:bg-amber-600 text-[#0f172a]" onClick={openCreateForm}>
          <Plus className="mr-2 w-4 h-4" />
          新建模板
        </Button>
      </div>

      {error && (
        <Card className="border-red-200 bg-red-50">
          <CardContent className="p-4 text-sm text-red-700">{error}</CardContent>
        </Card>
      )}

      {formOpen && (
        <Card className="border-amber-200">
          <CardContent className="p-6">
            <form className="space-y-4" onSubmit={handleSubmit}>
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-lg font-bold text-slate-900">{editing ? "配置规则模板" : "新建规则模板"}</h2>
                  <p className="text-sm text-slate-500">金额字段按字符串提交给后端，避免浮点精度误差</p>
                </div>
                <Button type="button" variant="ghost" size="icon" onClick={() => setFormOpen(false)} aria-label="关闭表单">
                  <X className="w-4 h-4" />
                </Button>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <label className="space-y-1 text-sm text-slate-600">
                  <span>模板名称</span>
                  <Input value={form.name} onChange={(e) => handleChange("name", e.target.value)} />
                </label>
                <label className="space-y-1 text-sm text-slate-600">
                  <span>起拍价</span>
                  <Input value={form.start_price} inputMode="decimal" onChange={(e) => handleChange("start_price", e.target.value)} />
                </label>
                <label className="space-y-1 text-sm text-slate-600">
                  <span>加价幅度</span>
                  <Input value={form.increment} inputMode="decimal" onChange={(e) => handleChange("increment", e.target.value)} />
                </label>
                <label className="space-y-1 text-sm text-slate-600">
                  <span>封顶价</span>
                  <Input value={form.cap_price || ""} inputMode="decimal" placeholder="可选" onChange={(e) => handleChange("cap_price", e.target.value)} />
                </label>
                <label className="space-y-1 text-sm text-slate-600">
                  <span>竞拍时长</span>
                  <Input value={form.duration} type="number" min={1} onChange={(e) => handleChange("duration", e.target.value)} />
                </label>
                <label className="space-y-1 text-sm text-slate-600">
                  <span>延时时长</span>
                  <Input value={form.delay_duration} type="number" min={1} onChange={(e) => handleChange("delay_duration", e.target.value)} />
                </label>
                <label className="space-y-1 text-sm text-slate-600">
                  <span>最大延时时长</span>
                  <Input value={form.max_delay_time} type="number" min={1} onChange={(e) => handleChange("max_delay_time", e.target.value)} />
                </label>
                <label className="space-y-1 text-sm text-slate-600">
                  <span>延时触发窗口</span>
                  <Input value={form.trigger_delay_before} type="number" min={1} onChange={(e) => handleChange("trigger_delay_before", e.target.value)} />
                </label>
              </div>

              <label className="flex items-center gap-2 text-sm text-slate-600">
                <input
                  type="checkbox"
                  checked={form.is_default}
                  onChange={(e) => handleChange("is_default", e.target.checked)}
                />
                设为默认模板
              </label>

              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setFormOpen(false)}>
                  取消
                </Button>
                <Button type="submit" disabled={saving} className="bg-amber-500 hover:bg-amber-600 text-[#0f172a]">
                  {saving ? "保存中..." : "保存模板"}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-4">
        {loading && <div className="p-8 text-center text-slate-500">加载中...</div>}
        {!loading && templates.length === 0 && (
          <Card className="border-slate-200">
            <CardContent className="p-8 text-center text-slate-500">暂无规则模板，请先新建模板</CardContent>
          </Card>
        )}
        {!loading && templates.map((template) => (
          <Card key={template.id} className="border-slate-200 hover:border-amber-400 transition-all group">
            <CardContent className="p-6">
              <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
                <div className="flex items-start gap-4">
                  <div className="w-12 h-12 rounded-xl bg-slate-100 flex items-center justify-center text-slate-400 group-hover:bg-amber-100 group-hover:text-amber-600 transition-colors">
                    <FileText className="w-6 h-6" />
                  </div>
                  <div>
                    <div className="flex items-center gap-3">
                      <h3 className="text-lg font-bold text-slate-900">{template.name}</h3>
                      {template.is_default && <Badge className="bg-amber-100 text-amber-700 border-amber-200">默认</Badge>}
                    </div>
                    <p className="text-sm text-slate-500 mt-1">{describeTemplate(template)}</p>
                    <p className="text-xs text-slate-400 mt-2">起拍价 {template.start_price} 元</p>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm" className="border-slate-200" onClick={() => openEditForm(template)}>
                    <Settings2 className="mr-2 w-4 h-4" />
                    配置规则
                  </Button>
                  <Button variant="outline" size="sm" className="border-slate-200" onClick={() => handleClone(template)}>
                    <Copy className="mr-2 w-4 h-4" />
                    克隆
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="text-slate-400 hover:text-red-500"
                    onClick={() => handleDelete(template)}
                    aria-label={`删除 ${template.name}`}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
