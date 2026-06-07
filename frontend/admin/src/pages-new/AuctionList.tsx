import React from "react"
import {
  Search,
  Filter,
  Calendar,
  Clock,
  Users,
  ArrowUpRight
} from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge, type BadgeProps } from "@/components/ui/badge"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useNavigate } from "react-router-dom"
import { auctionApi, auctionRuleTemplateApi, productApi, statisticsApi, type AuctionRuleTemplate } from "@/shared/api"
import { useAuth } from "@/shared/auth"
import { MERCHANT_ROLE } from "@/shared/auth/roles"

function getAuctionStatus(auction: any): { label: string; variant: BadgeProps["variant"] } {
  if (auction.status === 0) return { label: "待开始", variant: "blue" }
  if (auction.status === 1) return { label: "竞拍中", variant: "success" }
  if (auction.status === 2) return { label: "竞拍中（延时）", variant: "warning" }
  if (auction.status === 3 && auction.winner_id) return { label: "已拍卖", variant: "outline" }
  if (auction.status === 3 && !auction.winner_id) return { label: "流拍", variant: "secondary" }
  if (auction.status === 4) return { label: "已取消", variant: "secondary" }
  return { label: "未知", variant: "secondary" }
}

export default function AuctionList() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const [auctions, setAuctions] = React.useState<any[]>([])
  const [loading, setLoading] = React.useState(true)
  const [statusFilter, setStatusFilter] = React.useState<number | number[] | undefined>(undefined)
  const [activeTab, setActiveTab] = React.useState("all")
  const [searchTerm, setSearchTerm] = React.useState("")
  const [page, setPage] = React.useState(1)
  const [total, setTotal] = React.useState(0)
  const [createOpen, setCreateOpen] = React.useState(false)
  const [createLoading, setCreateLoading] = React.useState(false)
  const [createSubmitting, setCreateSubmitting] = React.useState(false)
  const [createError, setCreateError] = React.useState("")
  const [products, setProducts] = React.useState<any[]>([])
  const [templates, setTemplates] = React.useState<AuctionRuleTemplate[]>([])
  const [createForm, setCreateForm] = React.useState({
    product_id: "",
    template_id: "",
    duration: 3600,
  })
  const pageSize = 20

  // 统计数据
  const [stats, setStats] = React.useState({
    totalAuctions: 0,
    totalParticipants: 0,
    avgPrice: 0,
  })
  const isMerchant = user?.role === MERCHANT_ROLE

  // 获取竞拍列表
  const fetchAuctions = React.useCallback(async () => {
    setLoading(true)
    try {
      const baseParams = {
        search: searchTerm || undefined,
        page,
        page_size: pageSize,
      }
      const response = Array.isArray(statusFilter)
        ? await Promise.all(statusFilter.map((status) => auctionApi.list({ ...baseParams, status }))).then((items) => ({
          list: items.flatMap((item) => item.list || []),
          total: items.reduce((sum, item) => sum + (item.total || 0), 0),
        }))
        : await auctionApi.list({
          ...baseParams,
          status: statusFilter,
        })
      setAuctions(response.list || [])
      setTotal(response.total || 0)
    } catch (e) {
      console.error('获取竞拍列表失败:', e)
    } finally {
      setLoading(false)
    }
  }, [statusFilter, searchTerm, page])

  // 获取统计数据
  const fetchStats = React.useCallback(async () => {
    try {
      const overview = await statisticsApi.getOverview()
      setStats({
        totalAuctions: overview.total_auctions || 0,
        totalParticipants: overview.total_users || 0,
        avgPrice: overview.today_revenue / Math.max(overview.total_auctions, 1) || 0,
      })
    } catch (e) {
      console.error('获取统计数据失败:', e)
    }
  }, [])

  React.useEffect(() => {
    fetchAuctions()
    fetchStats()
  }, [fetchAuctions, fetchStats])

  // 状态筛选
  const handleStatusChange = (value: string) => {
    setActiveTab(value)
    if (value === 'all') {
      setStatusFilter(undefined)
    } else {
      const statusValue = { active: [1, 2], sold: 3, unsold: 3, cancelled: 4 }[value as "active" | "sold" | "unsold" | "cancelled"]
      setStatusFilter(statusValue)
    }
    setPage(1)
  }

  const openCreateAuction = async () => {
    setCreateOpen(true)
    setCreateLoading(true)
    setCreateError("")
    try {
      const [productResponse, templateResponse] = await Promise.all([
        productApi.list({ status: 1, page: 1, page_size: 100 }),
        auctionRuleTemplateApi.list({ page: 1, page_size: 100 }),
      ])
      const nextProducts = (productResponse.list || []).filter((product) => product.display_status === "schedulable")
      const nextTemplates = templateResponse.list || []
      setProducts(nextProducts)
      setTemplates(nextTemplates)
      const defaultTemplate = nextTemplates.find((item) => item.is_default) || nextTemplates[0]
      setCreateForm({
        product_id: nextProducts[0]?.id ? String(nextProducts[0].id) : "",
        template_id: defaultTemplate?.id ? String(defaultTemplate.id) : "",
        duration: defaultTemplate?.duration || 3600,
      })
    } catch (e) {
      console.error('加载创建竞拍依赖失败:', e)
      setCreateError("加载商品或规则模板失败，请稍后重试")
    } finally {
      setCreateLoading(false)
    }
  }

  const handleTemplateChange = (templateID: string) => {
    const selected = templates.find((template) => String(template.id) === templateID)
    setCreateForm((current) => ({
      ...current,
      template_id: templateID,
      duration: selected?.duration || current.duration,
    }))
  }

  const submitCreateAuction = async (event: React.FormEvent) => {
    event.preventDefault()
    const productID = Number(createForm.product_id)
    const templateID = Number(createForm.template_id)
    const duration = Number(createForm.duration)
    if (products.length === 0) {
      setCreateError("暂无可排期商品")
      return
    }
    if (!productID || !templateID || duration <= 0) {
      setCreateError("请选择商品、规则模板并填写有效竞拍时长")
      return
    }

    setCreateSubmitting(true)
    setCreateError("")
    try {
      await productApi.applyRuleTemplate(productID, templateID)
      await auctionApi.create({ product_id: productID, duration })
      setCreateOpen(false)
      await fetchAuctions()
      await fetchStats()
    } catch (e) {
      console.error('创建竞拍场次失败:', e)
      setCreateError("创建竞拍场次失败，请检查商品和规则模板后重试")
    } finally {
      setCreateSubmitting(false)
    }
  }

  // 本地搜索过滤
  const filteredAuctions = React.useMemo(() => {
    if (!searchTerm) return auctions
    return auctions.filter(a =>
      (a.product?.name || a.title || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      (a.live_stream_name || '').toLowerCase().includes(searchTerm.toLowerCase())
    )
  }, [auctions, searchTerm])

  const visibleAuctions = React.useMemo(() => {
    if (activeTab === "sold") return filteredAuctions.filter((auction) => auction.status === 3 && !!auction.winner_id)
    if (activeTab === "unsold") return filteredAuctions.filter((auction) => auction.status === 3 && !auction.winner_id)
    if (activeTab === "active") return filteredAuctions.filter((auction) => auction.status === 1 || auction.status === 2)
    return filteredAuctions
  }, [activeTab, filteredAuctions])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">竞拍管理</h1>
          <p className="text-sm text-slate-500">监控和管理所有竞拍场次</p>
        </div>
        {isMerchant && (
          <div className="flex items-center gap-3">
            <Button variant="outline" className="border-slate-200" onClick={() => navigate("/auction/rules")}>
              规则模板
            </Button>
            <Button className="bg-amber-500 hover:bg-amber-600 text-[#0f172a]" onClick={openCreateAuction}>
              创建竞拍场次
            </Button>
          </div>
        )}
      </div>

      {createOpen && (
        <Card className="border-amber-200">
          <CardContent className="p-6">
            <form className="space-y-4" onSubmit={submitCreateAuction}>
              <div>
                <h2 className="text-lg font-bold text-slate-900">创建竞拍场次</h2>
                <p className="text-sm text-slate-500">先将规则模板应用到商品，再创建真实竞拍场次</p>
              </div>
              {createError && <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">{createError}</div>}
              {createLoading ? (
                <div className="text-sm text-slate-500">加载商品和模板中...</div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  {products.length === 0 && (
                    <div className="md:col-span-3 rounded-md bg-amber-50 p-3 text-sm text-amber-700">
                      暂无可排期商品
                    </div>
                  )}
                  <label className="space-y-1 text-sm text-slate-600">
                    <span>竞拍商品</span>
                    <select
                      aria-label="竞拍商品"
                      className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm"
                      value={createForm.product_id}
                      onChange={(e) => setCreateForm((current) => ({ ...current, product_id: e.target.value }))}
                    >
                      {products.map((product) => (
                        <option key={product.id} value={product.id}>{product.name}</option>
                      ))}
                    </select>
                  </label>
                  <label className="space-y-1 text-sm text-slate-600">
                    <span>规则模板</span>
                    <select
                      aria-label="规则模板"
                      className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm"
                      value={createForm.template_id}
                      onChange={(e) => handleTemplateChange(e.target.value)}
                    >
                      {templates.map((template) => (
                        <option key={template.id} value={template.id}>{template.name}</option>
                      ))}
                    </select>
                  </label>
                  <label className="space-y-1 text-sm text-slate-600">
                    <span>竞拍时长</span>
                    <Input
                      aria-label="竞拍时长"
                      type="number"
                      min={1}
                      value={createForm.duration}
                      onChange={(e) => setCreateForm((current) => ({ ...current, duration: Number(e.target.value) }))}
                    />
                  </label>
                </div>
              )}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setCreateOpen(false)}>
                  取消
                </Button>
                <Button type="submit" disabled={createLoading || createSubmitting || products.length === 0} className="bg-amber-500 hover:bg-amber-600 text-[#0f172a]">
                  {createSubmitting ? "创建中..." : "确认创建竞拍"}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatCard title="总竞拍场次" value={stats.totalAuctions.toString()} subValue="累计" />
        <StatCard title="累计参与人次" value={stats.totalParticipants.toString()} subValue="人次" />
        <StatCard title="平均成交价" value={`¥${Math.round(stats.avgPrice).toLocaleString()}`} subValue="平均" />
      </div>

      <Card className="border-slate-200">
        <CardContent className="p-0">
          <div className="p-4 border-b border-slate-100 flex flex-col md:flex-row md:items-center justify-between gap-4">
            <Tabs defaultValue="all" onValueChange={handleStatusChange} className="w-full md:w-auto">
              <TabsList className="bg-slate-100 border-none">
                <TabsTrigger value="all">全部场次</TabsTrigger>
                <TabsTrigger value="active">竞拍中</TabsTrigger>
                <TabsTrigger value="sold">已拍卖</TabsTrigger>
                <TabsTrigger value="unsold">流拍</TabsTrigger>
                <TabsTrigger value="cancelled">已取消</TabsTrigger>
              </TabsList>
            </Tabs>

            <div className="flex items-center gap-2">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <Input
                  placeholder="搜索场次名称..."
                  className="pl-9 w-64 bg-slate-50 border-slate-200"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
              <Button variant="outline" size="icon" className="border-slate-200">
                <Filter className="w-4 h-4" />
              </Button>
            </div>
          </div>

          {loading ? (
            <div className="p-8 text-center text-slate-500">加载中...</div>
          ) : (
            <div className="divide-y divide-slate-100">
              {visibleAuctions.length === 0 ? (
                <div className="p-8 text-center text-slate-500">暂无竞拍数据</div>
              ) : (
                visibleAuctions.map((auction) => (
                  <div
                    key={auction.id}
                    className="p-6 hover:bg-slate-50 transition-all cursor-pointer group"
                    onClick={() => navigate(`/auction/detail?id=${auction.id}`)}
                  >
                    <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-6">
                      <div className="flex-1 space-y-3">
                        <div className="flex items-center gap-3">
                          <h3 className="text-lg font-bold text-slate-900 group-hover:text-amber-600 transition-colors">
                            {auction.product?.name || `竞拍场次 #${auction.id}`}
                          </h3>
                          <Badge variant={getAuctionStatus(auction).variant}>
                            {getAuctionStatus(auction).label}
                          </Badge>
                        </div>
                        <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-slate-500">
                          <div className="flex items-center gap-1.5">
                            <Calendar className="w-4 h-4" />
                            <span>{new Date(auction.start_time).toLocaleString()}</span>
                          </div>
                          <div className="flex items-center gap-1.5">
                            <Users className="w-4 h-4" />
                            <span>{auction.bid_count || 0} 次出价</span>
                          </div>
                          <div className="flex items-center gap-1.5">
                            <Clock className="w-4 h-4" />
                            <span>竞拍ID: {auction.id}</span>
                          </div>
                        </div>
                      </div>

                      <div className="flex items-center gap-8 px-8 border-l border-slate-100">
                        <div className="text-center">
                          <p className="text-xs text-slate-400 mb-1">当前最高价</p>
                          <p className="text-lg font-bold text-amber-600">
                            ¥{auction.current_price?.toLocaleString() || '0'}
                          </p>
                        </div>
                        <div className="text-center">
                          <p className="text-xs text-slate-400 mb-1">关联直播间</p>
                          <p className="text-sm font-medium text-slate-700">
                            {auction.live_stream_name || `直播间 #${auction.live_stream_id}`}
                          </p>
                        </div>
                        <Button variant="ghost" size="icon" className="text-slate-300 group-hover:text-slate-600">
                          <ArrowUpRight className="w-5 h-5" />
                        </Button>
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          )}

          <div className="p-4 border-t border-slate-100 flex items-center justify-between">
            <p className="text-sm text-slate-500">
              显示 {((page - 1) * pageSize) + 1} 到 {Math.min(page * pageSize, total)}，共 {total} 条
            </p>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage(page - 1)}
              >
                上一页
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={page * pageSize >= total}
                onClick={() => setPage(page + 1)}
              >
                下一页
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function StatCard({ title, value, subValue }: { title: string; value: string; subValue: string }) {
  return (
    <Card className="border-slate-200">
      <CardContent className="p-6">
        <p className="text-sm text-slate-500 font-medium">{title}</p>
        <div className="flex items-baseline justify-between mt-2">
          <h3 className="text-2xl font-bold text-slate-900">{value}</h3>
          <span className="text-xs font-bold text-emerald-500">{subValue}</span>
        </div>
      </CardContent>
    </Card>
  )
}
