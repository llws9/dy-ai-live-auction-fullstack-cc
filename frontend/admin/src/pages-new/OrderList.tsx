import React from "react"
import {
  Search,
  Download,
  Truck,
  CreditCard,
  CheckCircle,
  XCircle,
  Clock,
  ShoppingBag,
  Loader2
} from "lucide-react"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge, type BadgeProps } from "@/components/ui/badge"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Card, CardContent } from "@/components/ui/card"
import { useNavigate } from "react-router-dom"
import { cn } from "@/lib/utils"
import { orderApi, statisticsApi } from "@/shared/api"

const statusMap: Record<number, { label: string; variant: BadgeProps["variant"]; icon: React.ElementType }> = {
  0: { label: "待支付", variant: "warning", icon: CreditCard },
  1: { label: "已支付", variant: "secondary", icon: ShoppingBag },
  2: { label: "已发货", variant: "success", icon: Truck },
  3: { label: "已完成", variant: "success", icon: CheckCircle },
  4: { label: "已取消", variant: "outline", icon: XCircle },
}

export default function OrderList() {
  const navigate = useNavigate()
  const [orders, setOrders] = React.useState<any[]>([])
  const [loading, setLoading] = React.useState(true)
  const [statusFilter, setStatusFilter] = React.useState<number | undefined>(undefined)
  const [searchTerm, setSearchTerm] = React.useState("")
  const [searchQuery, setSearchQuery] = React.useState("")
  const [page, setPage] = React.useState(1)
  const [total, setTotal] = React.useState(0)
  const [shippingOrderId, setShippingOrderId] = React.useState<number | null>(null)
  const pageSize = 20

  // 统计数据
  const [stats, setStats] = React.useState({
    todayRevenue: 0,
    pendingPayment: 0,
    pendingShipment: 0,
    monthlyRevenue: 0,
  })

  // 获取订单列表
  const fetchOrders = React.useCallback(async () => {
    setLoading(true)
    try {
      const response = await orderApi.list({
        status: statusFilter,
        ...(searchQuery ? { search: searchQuery } : {}),
        page,
        page_size: pageSize,
      })
      setOrders(response.list || [])
      setTotal(response.total || 0)
      setStats((prev) => ({
        ...prev,
        pendingPayment: response.summary?.pending_payment_count || 0,
        pendingShipment: response.summary?.paid_count || 0,
      }))
    } catch (e) {
      console.error('获取订单列表失败:', e)
    } finally {
      setLoading(false)
    }
  }, [statusFilter, searchQuery, page])

  // 获取统计数据
  const fetchStats = React.useCallback(async () => {
    try {
      const overview = await statisticsApi.getOverview()
      setStats((prev) => ({
        ...prev,
        todayRevenue: overview.today_revenue || 0,
        monthlyRevenue: overview.total_revenue || 0,
      }))
    } catch (e) {
      console.error('获取统计数据失败:', e)
    }
  }, [])

  React.useEffect(() => {
    fetchOrders()
    fetchStats()
  }, [fetchOrders, fetchStats])

  // 状态筛选
  const handleStatusChange = (value: string) => {
    if (value === 'all') {
      setStatusFilter(undefined)
    } else {
      const statusValue = {
        pending_payment: 0,
        paid: 1,
        shipped: 2,
        completed: 3,
      }[value]
      setStatusFilter(statusValue)
    }
    setPage(1)
  }

  const handleSearch = () => {
    setSearchQuery(searchTerm.trim())
    setPage(1)
  }

  // 标记发货
  const handleShip = async (orderId: number) => {
    if (!confirm('确认标记该订单为已发货？')) return

    setShippingOrderId(orderId)
    try {
      await orderApi.ship(orderId)
      alert('发货成功')
      fetchOrders()
    } catch (e: any) {
      console.error('发货失败:', e)
      alert(e.message || '发货失败')
    } finally {
      setShippingOrderId(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">订单管理</h1>
          <p className="text-sm text-slate-500">处理竞拍成交后的订单流程</p>
        </div>
        {/* 导出订单 - 后端无接口，暂空置 */}
        <Button variant="outline" className="border-slate-200" disabled>
          <Download className="mr-2 w-4 h-4" />
          导出订单
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <OrderStatCard
          title="今日成交额"
          value={`¥${stats.todayRevenue.toLocaleString()}`}
          color="amber"
        />
        <OrderStatCard
          title="待支付订单"
          value={`${stats.pendingPayment}`}
          color="blue"
        />
        <OrderStatCard
          title="待发货订单"
          value={`${stats.pendingShipment}`}
          color="emerald"
        />
        <OrderStatCard
          title="本月累计"
          value={`¥${stats.monthlyRevenue.toLocaleString()}`}
          color="slate"
        />
      </div>

      <Card className="border-slate-200">
        <CardContent className="p-0">
          <div className="p-4 border-b border-slate-100 flex flex-col md:flex-row md:items-center justify-between gap-4">
            <Tabs defaultValue="all" onValueChange={handleStatusChange} className="w-full md:w-auto">
              <TabsList className="bg-slate-100 border-none">
                <TabsTrigger value="all">全部</TabsTrigger>
                <TabsTrigger value="pending_payment">待支付</TabsTrigger>
                <TabsTrigger value="paid">待发货</TabsTrigger>
                <TabsTrigger value="shipped">已发货</TabsTrigger>
                <TabsTrigger value="completed">已完成</TabsTrigger>
              </TabsList>
            </Tabs>

            <form
              className="flex items-center gap-2"
              onSubmit={(e) => {
                e.preventDefault()
                handleSearch()
              }}
            >
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <Input
                  placeholder="搜索订单号/商品/买家ID"
                  className="pl-9 w-64 bg-slate-50 border-slate-200"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
              <Button type="submit" variant="outline" className="border-slate-200">
                搜索
              </Button>
            </form>
          </div>

          {loading ? (
            <div className="p-8 text-center text-slate-500">
              <Loader2 className="w-6 h-6 animate-spin inline-block" />
              加载中...
            </div>
          ) : (
            <Table>
              <TableHeader className="bg-slate-50/50">
                <TableRow>
                  <TableHead>订单编号</TableHead>
                  <TableHead>商品名称</TableHead>
                  <TableHead>买家</TableHead>
                  <TableHead>成交价</TableHead>
                  <TableHead>订单状态</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {orders.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center text-slate-500 py-8">
                      暂无订单数据
                    </TableCell>
                  </TableRow>
                ) : (
                  orders.map((order) => {
                    const StatusIcon = statusMap[order.status]?.icon || Clock
                    return (
                      <TableRow
                        key={order.id}
                        className="hover:bg-slate-50/80 transition-colors cursor-pointer"
                        onClick={() => navigate(`/order/detail?id=${order.id}`)}
                      >
                        <TableCell className="font-medium text-slate-900">
                          #{order.id}
                        </TableCell>
                        <TableCell className="max-w-[200px] truncate">
                          {order.product_name || `商品 #${order.product_id}`}
                        </TableCell>
                        <TableCell>{order.user_name || `用户 #${order.user_id}`}</TableCell>
                        <TableCell className="font-bold text-slate-900">
                          ¥{(order.final_price || 0).toLocaleString()}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={statusMap[order.status]?.variant || 'secondary'}
                            className="flex items-center w-fit gap-1.5 font-medium"
                          >
                            <StatusIcon className="w-3 h-3" />
                            {statusMap[order.status]?.label || '未知'}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-slate-500 text-sm">
                          {new Date(order.created_at).toLocaleString()}
                        </TableCell>
                        <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                          <div className="flex items-center justify-end gap-2">
                            {order.status === 1 && (
                              <Button
                                size="sm"
                                className="bg-amber-500 hover:bg-amber-600 text-[#0f172a] h-8"
                                disabled={shippingOrderId === order.id}
                                onClick={() => handleShip(order.id)}
                              >
                                {shippingOrderId === order.id ? (
                                  <Loader2 className="w-3 h-3 animate-spin" />
                                ) : null}
                                标记发货
                              </Button>
                            )}
                            {order.status === 0 && (
                              <Button
                                size="sm"
                                variant="outline"
                                className="border-amber-200 text-amber-600 hover:bg-amber-50 h-8"
                                disabled
                              >
                                催付
                              </Button>
                            )}
                            <Button
                              variant="ghost"
                              size="sm"
                              className="text-slate-600 h-8"
                              onClick={() => navigate(`/order/detail?id=${order.id}`)}
                            >
                              查看详情
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          )}

          <div className="p-4 border-t border-slate-100 flex items-center justify-between">
            <p className="text-sm text-slate-500">
              显示 {((page - 1) * pageSize) + 1} 到 {Math.min(page * pageSize, total)}，共 {total} 条订单
            </p>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                className="border-slate-200"
                disabled={page <= 1}
                onClick={() => setPage(page - 1)}
              >
                上一页
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="border-slate-200"
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

function OrderStatCard({ title, value, color }: { title: string; value: string; color: string }) {
  const colors: Record<string, string> = {
    amber: "border-amber-100 bg-amber-50 text-amber-900",
    blue: "border-blue-100 bg-blue-50 text-blue-900",
    emerald: "border-emerald-100 bg-emerald-50 text-emerald-900",
    slate: "border-slate-100 bg-slate-50 text-slate-900",
  }

  return (
    <Card className={cn("border", colors[color])}>
      <CardContent className="p-4">
        <p className="text-xs font-medium opacity-70">{title}</p>
        <div className="flex items-baseline mt-2">
          <h3 className="text-xl font-bold">{value}</h3>
        </div>
      </CardContent>
    </Card>
  )
}
