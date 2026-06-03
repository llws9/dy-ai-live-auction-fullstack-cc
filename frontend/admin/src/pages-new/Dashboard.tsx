import React from "react"
import {
  TrendingUp,
  TrendingDown,
  Users,
  ShoppingBag,
  Gavel,
  DollarSign,
  ArrowRight,
  PlusCircle,
  Clock,
  Video,
  BarChart3,
  Loader2
} from "lucide-react"
import {
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
  PieChart,
  Pie,
  Cell,
  Legend
} from "recharts"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { useNavigate } from "react-router-dom"
import { cn } from "@/lib/utils"
import { liveStreamApi, statisticsApi } from "@/shared/api"
import { useAuth } from "@/shared/auth"

const COLORS = ["#f59e0b", "#3b82f6", "#10b981", "#6366f1"]

export default function Dashboard() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const currentTime = new Date().toLocaleString("zh-CN", {
    year: "numeric", month: "long", day: "numeric", hour: "2-digit", minute: "2-digit"
  })

  const [loading, setLoading] = React.useState(true)
  const [overview, setOverview] = React.useState<any>(null)
  const [trendData, setTrendData] = React.useState<any[]>([])
  const [revenueComposition, setRevenueComposition] = React.useState<any[]>([])

  const handleStartLive = async () => {
    const id = window.prompt("请输入要开启的直播间 ID")
    if (!id) return
    const liveStreamId = Number(id)
    if (!Number.isFinite(liveStreamId) || liveStreamId <= 0) {
      alert("直播间 ID 无效")
      return
    }
    try {
      await liveStreamApi.start(liveStreamId)
      alert("直播已开启")
      navigate(`/live/detail?id=${liveStreamId}`)
    } catch (e) {
      console.error("开启直播失败:", e)
      alert("开启直播失败")
    }
  }

  // 获取数据
  React.useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        // 获取概览数据
        const overviewData = await statisticsApi.getOverview()
        setOverview(overviewData)

        // 获取趋势数据（最近7天）
        const revenueData = await statisticsApi.getRevenueStats({
          group_by: 'day'
        })
        setTrendData(revenueData.slice(-7).map((item: any) => ({
          name: new Date(item.date).toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' }),
          revenue: item.revenue || 0,
          auctions: item.order_count || 0,
        })))

        // 获取收入构成（按类别）
        const categoryData = await statisticsApi.getRevenueStats({
          group_by: 'category'
        })
        setRevenueComposition(categoryData.map((item: any) => ({
          name: item.category || '其他',
          value: item.revenue || 0,
        })))
      } catch (e) {
        console.error('获取统计数据失败:', e)
        // 使用默认数据
        setTrendData([
          { name: "05-21", revenue: 45000, auctions: 12 },
          { name: "05-22", revenue: 52000, auctions: 15 },
          { name: "05-23", revenue: 48000, auctions: 10 },
          { name: "05-24", revenue: 61000, auctions: 18 },
          { name: "05-25", revenue: 55000, auctions: 14 },
          { name: "05-26", revenue: 67000, auctions: 22 },
          { name: "05-27", revenue: 72000, auctions: 25 },
        ])
        setRevenueComposition([
          { name: "艺术收藏", value: 45 },
          { name: "珠宝名表", value: 30 },
          { name: "数码电子", value: 15 },
          { name: "其他", value: 10 },
        ])
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [])

  return (
    <div className="space-y-6">
      {/* Welcome Section */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 tracking-tight">
            欢迎，{user?.name || '管理员'}
          </h1>
          <div className="flex items-center gap-2 text-slate-500 mt-1">
            <Clock className="w-4 h-4" />
            <span className="text-sm">当前时间：{currentTime}</span>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="outline" className="border-slate-200" onClick={() => navigate("/goods/create")}>
            <PlusCircle className="mr-2 w-4 h-4" />
            发布商品
          </Button>
          <Button className="bg-amber-500 hover:bg-amber-600 text-[#0f172a]" onClick={handleStartLive}>
            <Video className="mr-2 w-4 h-4" />
            开启直播
          </Button>
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
        <KPICard
          title="总竞拍数"
          value={overview?.total_auctions?.toString() || '0'}
          trend="+12%"
          isUp={true}
          icon={Gavel}
          color="blue"
          onClick={() => navigate("/auction/list")}
        />
        <KPICard
          title="进行中竞拍"
          value={overview?.ongoing_auctions?.toString() || '0'}
          trend="+5%"
          isUp={true}
          icon={Clock}
          color="amber"
          onClick={() => navigate("/auction/list")}
        />
        <KPICard
          title="总收入"
          value={`¥${(overview?.total_revenue || 0).toLocaleString()}`}
          trend="+18%"
          isUp={true}
          icon={DollarSign}
          color="emerald"
          onClick={() => navigate("/stats/revenue")}
        />
        <KPICard
          title="参与用户"
          value={(overview?.total_users || 0).toLocaleString()}
          trend="+8%"
          isUp={true}
          icon={Users}
          color="indigo"
          onClick={() => navigate("/stats/user")}
        />
        <KPICard
          title="今日成交"
          value={`¥${(overview?.today_revenue || 0).toLocaleString()}`}
          trend="+4%"
          isUp={true}
          icon={TrendingUp}
          color="violet"
          onClick={() => navigate("/stats/revenue")}
        />
        <KPICard
          title="总订单数"
          value={(overview?.total_orders || 0).toLocaleString()}
          trend="+3%"
          isUp={true}
          icon={ShoppingBag}
          color="rose"
          onClick={() => navigate("/order/list")}
        />
      </div>

      {/* Charts Section */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        <Card className="lg:col-span-8 border-slate-200">
          <CardHeader className="flex flex-row items-center justify-between">
            <div>
              <CardTitle className="text-lg font-bold">近期趋势</CardTitle>
              <CardDescription>最近7天的收入与竞拍场次趋势</CardDescription>
            </div>
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-amber-500"></div>
                <span className="text-xs text-slate-500">收入</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-blue-500"></div>
                <span className="text-xs text-slate-500">场次</span>
              </div>
            </div>
          </CardHeader>
          <CardContent className="h-[350px]">
            {loading ? (
              <div className="flex items-center justify-center h-full">
                <Loader2 className="w-8 h-8 animate-spin text-slate-400" />
              </div>
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={trendData}>
                  <defs>
                    <linearGradient id="colorRevenue" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.1} />
                      <stop offset="95%" stopColor="#f59e0b" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                  <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fontSize: 12, fill: '#64748b' }} dy={10} />
                  <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 12, fill: '#64748b' }} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#fff', borderRadius: '8px', border: '1px solid #e2e8f0', boxShadow: '0 4px 12px rgba(0,0,0,0.05)' }}
                  />
                  <Area type="monotone" dataKey="revenue" stroke="#f59e0b" strokeWidth={3} fillOpacity={1} fill="url(#colorRevenue)" />
                  <Line type="monotone" dataKey="auctions" stroke="#3b82f6" strokeWidth={2} dot={{ r: 4, fill: '#3b82f6' }} />
                </AreaChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>

        <Card className="lg:col-span-4 border-slate-200">
          <CardHeader>
            <CardTitle className="text-lg font-bold">收入构成</CardTitle>
            <CardDescription>按商品类别的收入分布</CardDescription>
          </CardHeader>
          <CardContent className="h-[300px] flex flex-col items-center justify-center">
            {loading ? (
              <div className="flex items-center justify-center h-full">
                <Loader2 className="w-6 h-6 animate-spin text-slate-400" />
              </div>
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={revenueComposition}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={80}
                    paddingAngle={5}
                    dataKey="value"
                  >
                    {revenueComposition.map((_, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip />
                  <Legend layout="horizontal" verticalAlign="bottom" align="center" />
                </PieChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Quick Actions & Tasks */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card className="border-slate-200">
          <CardHeader>
            <CardTitle className="text-lg font-bold">待办事项</CardTitle>
            <CardDescription>需要立即处理的业务</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <TaskItem
              title="待处理订单"
              count={overview?.total_orders || 0}
              description="用户已支付，等待发货"
              color="amber"
              onClick={() => navigate("/order/list")}
            />
            <TaskItem
              title="进行中竞拍"
              count={overview?.ongoing_auctions || 0}
              description="当前正在进行的竞拍场次"
              color="blue"
              onClick={() => navigate("/auction/list")}
            />
            <TaskItem
              title="查看数据报表"
              count={0}
              description="今日数据统计"
              color="rose"
              onClick={() => navigate("/stats/revenue")}
            />
          </CardContent>
        </Card>

        <Card className="border-slate-200 bg-[#0f172a] text-white">
          <CardHeader>
            <CardTitle className="text-lg font-bold text-white">快捷入口</CardTitle>
            <CardDescription className="text-slate-400">常用功能一键直达</CardDescription>
          </CardHeader>
          <CardContent className="grid grid-cols-2 gap-4">
            <QuickActionBtn
              title="发布商品"
              icon={PlusCircle}
              onClick={() => navigate("/goods/create")}
            />
            <QuickActionBtn
              title="竞拍管理"
              icon={Gavel}
              onClick={() => navigate("/auction/list")}
            />
            <QuickActionBtn
              title="直播间管理"
              icon={Video}
              onClick={() => navigate("/live/list")}
            />
            <QuickActionBtn
              title="生成报表"
              icon={BarChart3}
              onClick={() => navigate("/stats/revenue")}
            />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function KPICard({ title, value, trend, isUp, icon: Icon, color, onClick }: any) {
  const colors: Record<string, string> = {
    blue: "bg-blue-50 text-blue-600 border-blue-100",
    amber: "bg-amber-50 text-amber-600 border-amber-100",
    emerald: "bg-emerald-50 text-emerald-600 border-emerald-100",
    indigo: "bg-indigo-50 text-indigo-600 border-indigo-100",
    violet: "bg-violet-50 text-violet-600 border-violet-100",
    rose: "bg-rose-50 text-rose-600 border-rose-100",
  }

  return (
    <Card className="border-slate-200 hover:shadow-lg transition-all cursor-pointer group" onClick={onClick}>
      <CardContent className="p-4">
        <div className="flex items-center justify-between mb-3">
          <div className={cn("p-2 rounded-lg", colors[color])}>
            <Icon className="w-4 h-4" />
          </div>
          <div className={cn("flex items-center text-xs font-bold", isUp ? "text-emerald-500" : "text-rose-500")}>
            {isUp ? <TrendingUp className="w-3 h-3 mr-1" /> : <TrendingDown className="w-3 h-3 mr-1" />}
            {trend}
          </div>
        </div>
        <p className="text-xs text-slate-500 font-medium">{title}</p>
        <p className="text-xl font-bold text-slate-900 tabular-nums mt-1">{value}</p>
      </CardContent>
    </Card>
  )
}

function TaskItem({ title, count, description, color, onClick }: any) {
  const colors: Record<string, string> = {
    amber: "bg-amber-500",
    blue: "bg-blue-500",
    rose: "bg-rose-500",
  }

  return (
    <div
      className="flex items-center justify-between p-3 rounded-xl border border-slate-100 hover:bg-slate-50 transition-all cursor-pointer group"
      onClick={onClick}
    >
      <div className="flex items-center gap-4">
        <div className={cn("w-10 h-10 rounded-full flex items-center justify-center text-white font-bold", colors[color])}>
          {count}
        </div>
        <div>
          <p className="text-sm font-semibold text-slate-900">{title}</p>
          <p className="text-xs text-slate-500">{description}</p>
        </div>
      </div>
      <ArrowRight className="w-4 h-4 text-slate-300 group-hover:text-slate-600 transition-all" />
    </div>
  )
}

function QuickActionBtn({ title, icon: Icon, onClick }: any) {
  return (
    <button
      onClick={onClick}
      className="flex flex-col items-center justify-center p-4 rounded-xl bg-slate-800 hover:bg-amber-500 hover:text-[#0f172a] transition-all gap-2 group border border-slate-700 hover:border-amber-400"
    >
      <Icon className="w-6 h-6 text-amber-500 group-hover:text-[#0f172a]" />
      <span className="text-xs font-medium">{title}</span>
    </button>
  )
}
