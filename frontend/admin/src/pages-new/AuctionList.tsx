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
import { auctionApi, statisticsApi } from "@/shared/api"

const statusMap: Record<number, { label: string; variant: BadgeProps["variant"] }> = {
  0: { label: "待开始", variant: "blue" },
  1: { label: "进行中", variant: "success" },
  2: { label: "延时中", variant: "warning" },
  3: { label: "已结束", variant: "outline" },
  4: { label: "已取消", variant: "secondary" },
}

export default function AuctionList() {
  const navigate = useNavigate()
  const [auctions, setAuctions] = React.useState<any[]>([])
  const [loading, setLoading] = React.useState(true)
  const [statusFilter, setStatusFilter] = React.useState<number | undefined>(undefined)
  const [searchTerm, setSearchTerm] = React.useState("")
  const [page, setPage] = React.useState(1)
  const [total, setTotal] = React.useState(0)
  const pageSize = 20

  // 统计数据
  const [stats, setStats] = React.useState({
    totalAuctions: 0,
    totalParticipants: 0,
    avgPrice: 0,
  })

  // 获取竞拍列表
  const fetchAuctions = React.useCallback(async () => {
    setLoading(true)
    try {
      const response = await auctionApi.list({
        status: statusFilter,
        search: searchTerm || undefined,
        page,
        page_size: pageSize,
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
    if (value === 'all') {
      setStatusFilter(undefined)
    } else {
      const statusValue = { ongoing: 1, pending: 0, completed: 3 }[value]
      setStatusFilter(statusValue)
    }
    setPage(1)
  }

  // 本地搜索过滤
  const filteredAuctions = React.useMemo(() => {
    if (!searchTerm) return auctions
    return auctions.filter(a =>
      (a.product?.name || a.title || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      (a.live_stream_name || '').toLowerCase().includes(searchTerm.toLowerCase())
    )
  }, [auctions, searchTerm])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">竞拍管理</h1>
          <p className="text-sm text-slate-500">监控和管理所有竞拍场次</p>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="outline" className="border-slate-200" onClick={() => navigate("/auction/rules")}>
            规则模板
          </Button>
          {/* 创建竞拍场次 - 后端无接口，暂空置 */}
          <Button className="bg-amber-500 hover:bg-amber-600 text-[#0f172a]" disabled>
            创建竞拍场次
          </Button>
        </div>
      </div>

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
                <TabsTrigger value="ongoing">进行中</TabsTrigger>
                <TabsTrigger value="pending">待开始</TabsTrigger>
                <TabsTrigger value="completed">已结束</TabsTrigger>
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
              {filteredAuctions.length === 0 ? (
                <div className="p-8 text-center text-slate-500">暂无竞拍数据</div>
              ) : (
                filteredAuctions.map((auction) => (
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
                          <Badge variant={statusMap[auction.status]?.variant || 'secondary'}>
                            {statusMap[auction.status]?.label || '未知'}
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
