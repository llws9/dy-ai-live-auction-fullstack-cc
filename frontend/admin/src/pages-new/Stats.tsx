import React from "react"
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  LineChart,
  Line,
  AreaChart,
  Area,
  Legend
} from "recharts"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { TrendingUp, Users, DollarSign, Gavel, Loader2 } from "lucide-react"
import { statisticsApi } from "@/shared/api"
import { useAuth } from "@/shared/auth"
import { ADMIN_ROLE } from "@/shared/auth/roles"
import { useLocation, useNavigate } from "react-router-dom"

type StatsTab = "auction" | "revenue" | "user"

function tabFromPath(pathname: string): StatsTab {
  if (pathname.endsWith("/stats/revenue")) return "revenue"
  if (pathname.endsWith("/stats/user")) return "user"
  return "auction"
}

function formatLocalDate(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, "0")
  const day = String(date.getDate()).padStart(2, "0")
  return `${year}-${month}-${day}`
}

function getDefaultAuctionRange(now = new Date()) {
  const start = new Date(now)
  start.setDate(now.getDate() - 6)
  return {
    start_date: formatLocalDate(start),
    end_date: formatLocalDate(now),
    group_by: "day",
  }
}

export default function Stats() {
  const location = useLocation()
  const navigate = useNavigate()
  const activeTab = tabFromPath(location.pathname)
  const { user } = useAuth()
  const canViewUserStats = user?.role === ADMIN_ROLE
  const visibleActiveTab = activeTab === "user" && !canViewUserStats ? "auction" : activeTab
  const [loading, setLoading] = React.useState(true)
  const [auctionData, setAuctionData] = React.useState<any[]>([])
  const [revenueData, setRevenueData] = React.useState<any[]>([])
  const [userData, setUserData] = React.useState<any[]>([])
  const [indicators, setIndicators] = React.useState({
    auction: { total: 0, rate: 0, avgBids: 0 },
    revenue: { total: 0, avgPrice: 0, commission: 0 },
    user: { total: 0, active: 0, rate: 0 },
  })

  React.useEffect(() => {
    if (activeTab === "user" && !canViewUserStats) {
      navigate("/stats/auction", { replace: true })
    }
  }, [activeTab, canViewUserStats, navigate])

  // 获取统计数据
  React.useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        // 获取竞拍统计
        const auctionStats = await statisticsApi.getAuctionStats(getDefaultAuctionRange())
        const normalizedAuctionStats = Array.isArray(auctionStats) ? auctionStats : []
        setAuctionData(normalizedAuctionStats.map((item: any) => ({
          name: new Date(item.date).toLocaleDateString('zh-CN', { weekday: 'short' }),
          count: item.auction_count || 0,
          rate: item.success_rate || 0,
        })))
        const totalAuctionCount = normalizedAuctionStats.reduce((sum: number, item: any) => sum + (item.auction_count || 0), 0)
        const totalBidCount = normalizedAuctionStats.reduce((sum: number, item: any) => sum + (item.bid_count || 0), 0)
        const avgSuccessRate = normalizedAuctionStats.length > 0
          ? normalizedAuctionStats.reduce((sum: number, item: any) => sum + (item.success_rate || 0), 0) / normalizedAuctionStats.length
          : 0
        const avgBids = totalAuctionCount > 0 ? totalBidCount / totalAuctionCount : 0
        setIndicators(prev => ({
          ...prev,
          auction: { total: totalAuctionCount, rate: avgSuccessRate, avgBids },
        }))

        // 获取收入统计
        const revenueStats = await statisticsApi.getRevenueStats({ group_by: 'month' })
        const normalizedRevenueStats = Array.isArray(revenueStats) ? revenueStats : []
        setRevenueData(normalizedRevenueStats.map((item: any) => ({
          month: item.date,
          value: item.revenue || 0,
        })))
        const totalRevenue = normalizedRevenueStats.reduce((sum: number, item: any) => sum + (item.revenue || 0), 0)
        const revenueIndicators = {
          total: totalRevenue,
          avgPrice: normalizedRevenueStats.length > 0 ? totalRevenue / normalizedRevenueStats.length : 0,
          commission: totalRevenue * 0.05,
        }
        setIndicators(prev => ({ ...prev, revenue: revenueIndicators }))

        if (canViewUserStats) {
          // 获取用户统计
          const userStats = await statisticsApi.getUserStats()
          const normalizedUserStats = Array.isArray(userStats) ? userStats : []
          setUserData(normalizedUserStats.slice(-7).map((item: any) => ({
            name: new Date(item.date).toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' }),
            new: item.new_users || 0,
            active: item.active_users || 0,
          })))
          const userIndicators = normalizedUserStats.reduce((acc: any, item: any) => ({
            total: acc.total + (item.new_users || 0),
            active: item.active_users || acc.active,
            rate: 8.4,
          }), { total: 0, active: 0, rate: 0 })
          setIndicators(prev => ({ ...prev, user: userIndicators }))
        } else {
          setUserData([])
          setIndicators(prev => ({ ...prev, user: { total: 0, active: 0, rate: 0 } }))
        }
      } catch (e) {
        console.error('获取统计数据失败:', e)
        setAuctionData([])
        setRevenueData([])
        setUserData([])
        setIndicators({
          auction: { total: 0, rate: 0, avgBids: 0 },
          revenue: { total: 0, avgPrice: 0, commission: 0 },
          user: { total: 0, active: 0, rate: 0 },
        })
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [canViewUserStats])

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">数据统计</h1>
        <p className="text-sm text-slate-500">多维度分析平台经营状况与用户行为</p>
      </div>

      <Tabs
        value={visibleActiveTab}
        onValueChange={(value) => navigate(`/stats/${value}`)}
        className="space-y-6"
      >
        <TabsList className="bg-white border border-slate-200 p-1 h-12">
          <TabsTrigger value="auction" className="px-8 h-10 data-[state=active]:bg-amber-500 data-[state=active]:text-[#0f172a]">竞拍统计</TabsTrigger>
          <TabsTrigger value="revenue" className="px-8 h-10 data-[state=active]:bg-amber-500 data-[state=active]:text-[#0f172a]">收入统计</TabsTrigger>
          {canViewUserStats ? (
            <TabsTrigger value="user" className="px-8 h-10 data-[state=active]:bg-amber-500 data-[state=active]:text-[#0f172a]">用户统计</TabsTrigger>
          ) : null}
        </TabsList>

        <TabsContent value="auction" className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <StatsIndicator title="总竞拍场次" value={indicators.auction.total.toString()} trend="+12%" icon={Gavel} />
            <StatsIndicator title="竞拍成功率" value={`${indicators.auction.rate.toFixed(1)}%`} trend="+2%" icon={TrendingUp} />
            <StatsIndicator title="平均出价次数" value={indicators.auction.avgBids.toFixed(1)} trend="+5%" icon={Users} />
          </div>
          <Card className="border-slate-200">
            <CardHeader>
              <CardTitle className="text-lg">竞拍热度分析</CardTitle>
              <CardDescription>本周每日竞拍场次与成功率走势</CardDescription>
            </CardHeader>
            <CardContent className="h-[400px]">
              {loading ? (
                <div className="flex items-center justify-center h-full">
                  <Loader2 className="w-8 h-8 animate-spin text-slate-400" />
                </div>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={auctionData}>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                    <XAxis dataKey="name" axisLine={false} tickLine={false} />
                    <YAxis axisLine={false} tickLine={false} />
                    <Tooltip />
                    <Legend />
                    <Bar dataKey="count" name="场次" fill="#f59e0b" radius={[4, 4, 0, 0]} />
                    <Bar dataKey="rate" name="成功率 (%)" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="revenue" className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <StatsIndicator title="年度总营收" value={`¥${(indicators.revenue.total / 10000).toFixed(1)}万`} trend="+28%" icon={DollarSign} />
            <StatsIndicator title="客单价" value={`¥${indicators.revenue.avgPrice.toLocaleString()}`} trend="+8%" icon={TrendingUp} />
            <StatsIndicator title="佣金收入" value={`¥${(indicators.revenue.commission / 10000).toFixed(1)}万`} trend="+15%" icon={DollarSign} />
          </div>
          <Card className="border-slate-200">
            <CardHeader>
              <CardTitle className="text-lg">营收趋势</CardTitle>
              <CardDescription>年度月度营收增长曲线</CardDescription>
            </CardHeader>
            <CardContent className="h-[400px]">
              {loading ? (
                <div className="flex items-center justify-center h-full">
                  <Loader2 className="w-8 h-8 animate-spin text-slate-400" />
                </div>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={revenueData}>
                    <defs>
                      <linearGradient id="colorValue" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.1} />
                        <stop offset="95%" stopColor="#f59e0b" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                    <XAxis dataKey="month" axisLine={false} tickLine={false} />
                    <YAxis axisLine={false} tickLine={false} />
                    <Tooltip formatter={(value: unknown) => formatCurrencyValue(value)} />
                    <Area type="monotone" dataKey="value" name="营收金额" stroke="#f59e0b" strokeWidth={3} fillOpacity={1} fill="url(#colorValue)" />
                  </AreaChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {canViewUserStats ? (
          <TabsContent value="user" className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <StatsIndicator title="累计注册用户" value={indicators.user.total.toString()} trend="+15%" icon={Users} />
              <StatsIndicator title="活跃用户 (MAU)" value={indicators.user.active.toString()} trend="+22%" icon={TrendingUp} />
              <StatsIndicator title="付费转化率" value={`${indicators.user.rate}%`} trend="+1%" icon={TrendingUp} />
            </div>
            <Card className="border-slate-200">
              <CardHeader>
                <CardTitle className="text-lg">用户增长与活跃</CardTitle>
                <CardDescription>近期每日新增用户与活跃用户趋势</CardDescription>
              </CardHeader>
              <CardContent className="h-[400px]">
                {loading ? (
                  <div className="flex items-center justify-center h-full">
                    <Loader2 className="w-8 h-8 animate-spin text-slate-400" />
                  </div>
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={userData}>
                      <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                      <XAxis dataKey="name" axisLine={false} tickLine={false} />
                      <YAxis axisLine={false} tickLine={false} />
                      <Tooltip />
                      <Legend />
                      <Line type="monotone" dataKey="active" name="活跃用户" stroke="#3b82f6" strokeWidth={3} dot={{ r: 4 }} />
                      <Line type="monotone" dataKey="new" name="新增用户" stroke="#f59e0b" strokeWidth={2} dot={{ r: 4 }} />
                    </LineChart>
                  </ResponsiveContainer>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        ) : null}
      </Tabs>
    </div>
  )
}

function formatCurrencyValue(value: unknown) {
  return typeof value === "number" ? `¥${value.toLocaleString()}` : `¥${value ?? ""}`
}

function StatsIndicator({ title, value, trend, icon: Icon }: any) {
  return (
    <Card className="border-slate-200">
      <CardContent className="p-6">
        <div className="flex items-center justify-between">
          <div className="p-2 rounded-lg bg-slate-50 text-slate-500">
            <Icon className="w-5 h-5" />
          </div>
          <Badge className="bg-emerald-50 text-emerald-700 border-emerald-100">{trend}</Badge>
        </div>
        <div className="mt-4">
          <p className="text-sm text-slate-500 font-medium">{title}</p>
          <h3 className="text-2xl font-bold text-slate-900 mt-1">{value}</h3>
        </div>
      </CardContent>
    </Card>
  )
}
