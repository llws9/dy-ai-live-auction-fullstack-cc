import React from "react"
import { Plus, Settings2, Trash2, Copy, FileText } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"

const ruleTemplates = [
  {
    id: "T-001",
    name: "高价值艺术品模板",
    description: "适用于 5 万以上拍品，加价幅度 1000 元，最后 5 分钟延时规则",
    items: 42,
    isDefault: true,
  },
  {
    id: "T-002",
    name: "大众数码模板",
    description: "适用于 1 万以下拍品，加价幅度 100 元，无延时规则",
    items: 128,
    isDefault: false,
  },
  {
    id: "T-003",
    name: "珠宝名表专场模板",
    description: "加价幅度 500 元，保证金 5000 元，含专家鉴定服务费",
    items: 15,
    isDefault: false,
  },
]

export default function AuctionRules() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">竞拍规则模板</h1>
          <p className="text-sm text-slate-500">预设常见的竞拍规则，快速应用于不同场次</p>
        </div>
        <Button className="bg-amber-500 hover:bg-amber-600 text-[#0f172a]">
          <Plus className="mr-2 w-4 h-4" />
          新建模板
        </Button>
      </div>

      <div className="grid grid-cols-1 gap-4">
        {ruleTemplates.map((template) => (
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
                      {template.isDefault && <Badge className="bg-amber-100 text-amber-700 border-amber-200">默认</Badge>}
                    </div>
                    <p className="text-sm text-slate-500 mt-1">{template.description}</p>
                    <p className="text-xs text-slate-400 mt-2">已应用于 {template.items} 个竞拍项目</p>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm" className="border-slate-200">
                    <Settings2 className="mr-2 w-4 h-4" />
                    配置规则
                  </Button>
                  <Button variant="outline" size="sm" className="border-slate-200">
                    <Copy className="mr-2 w-4 h-4" />
                    克隆
                  </Button>
                  <Button variant="ghost" size="icon" className="text-slate-400 hover:text-red-500">
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
