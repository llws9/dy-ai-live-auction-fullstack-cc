import React from "react"
import { ArrowLeft, Truck, Package, User, CreditCard, Loader2 } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { orderApi, productApi } from "@/shared/api"

const statusMap: Record<number, { label: string; badgeClass: string }> = {
  0: { label: "待支付", badgeClass: "bg-blue-500 text-white" },
  1: { label: "待发货", badgeClass: "bg-amber-500 text-[#0f172a]" },
  2: { label: "已发货", badgeClass: "bg-emerald-500 text-white" },
  3: { label: "已完成", badgeClass: "bg-slate-500 text-white" },
  4: { label: "已取消", badgeClass: "bg-slate-200 text-slate-600" },
}

export default function OrderDetail() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const orderId = searchParams.get('id')

  const [loading, setLoading] = React.useState(true)
  const [order, setOrder] = React.useState<any>(null)
  const [shipping, setShipping] = React.useState(false)

  // 获取订单详情
  React.useEffect(() => {
    if (!orderId) {
      navigate('/order/list')
      return
    }

    const fetchOrder = async () => {
      setLoading(true)
      try {
        const data = await orderApi.get(Number(orderId))
        setOrder(data)
      } catch (e) {
        console.error('获取订单详情失败:', e)
        alert('获取订单详情失败')
        navigate('/order/list')
      } finally {
        setLoading(false)
      }
    }
    fetchOrder()
  }, [orderId, navigate])

  // 标记发货
  const handleShip = async () => {
    if (!confirm('确认标记该订单为已发货？')) return

    setShipping(true)
    try {
      await orderApi.ship(Number(orderId))
      alert('发货成功')
      // 重新获取订单数据
      const data = await orderApi.get(Number(orderId))
      setOrder(data)
    } catch (e: any) {
      console.error('发货失败:', e)
      alert(e.message || '发货失败')
    } finally {
      setShipping(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-400" />
      </div>
    )
  }

  if (!order) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <p className="text-slate-500">订单不存在</p>
      </div>
    )
  }

  const statusInfo = statusMap[order.status] || statusMap[0]
  const canShip = order.status === 1

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="outline" size="icon" onClick={() => navigate("/order/list")} className="border-slate-200">
          <ArrowLeft className="w-4 h-4" />
        </Button>
        <h1 className="text-2xl font-bold text-slate-900">订单详情</h1>
        <Badge className={statusInfo.badgeClass}>{statusInfo.label}</Badge>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <Card className="border-slate-200">
            <CardHeader><CardTitle className="text-lg">商品信息</CardTitle></CardHeader>
            <CardContent className="flex gap-6">
              <div className="w-32 h-32 rounded-lg bg-slate-100 overflow-hidden border">
                {order.product_image ? (
                  <img src={order.product_image} alt={order.product_name} className="w-full h-full object-cover" />
                ) : (
                  <div className="w-full h-full flex items-center justify-center text-slate-400">
                    <Package className="w-10 h-10" />
                  </div>
                )}
              </div>
              <div className="space-y-2">
                <h3 className="text-lg font-bold">{order.product_name || `商品 #${order.product_id}`}</h3>
                <p className="text-slate-500">订单编号: #{order.id}</p>
                <p className="text-slate-500">竞拍场次: #{order.auction_id}</p>
                <p className="text-xl font-bold text-amber-600">
                  成交价: ¥{(order.final_price || 0).toLocaleString()}
                </p>
              </div>
            </CardContent>
          </Card>

          <Card className="border-slate-200">
            <CardHeader><CardTitle className="text-lg">买家信息</CardTitle></CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-start gap-4">
                <div className="p-2 rounded-full bg-slate-100">
                  <User className="w-5 h-5 text-slate-500" />
                </div>
                <div>
                  <p className="font-bold">买家：{order.user_name || `用户 #${order.user_id}`}</p>
                  <p className="text-sm text-slate-500">用户ID: {order.user_id}</p>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="border-slate-200">
            <CardHeader><CardTitle className="text-lg">订单时间</CardTitle></CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between border-b pb-2">
                <span className="text-slate-500">创建时间</span>
                <span>{new Date(order.created_at).toLocaleString()}</span>
              </div>
              {order.paid_at && (
                <div className="flex justify-between border-b pb-2">
                  <span className="text-slate-500">支付时间</span>
                  <span>{new Date(order.paid_at).toLocaleString()}</span>
                </div>
              )}
              {order.shipped_at && (
                <div className="flex justify-between border-b pb-2">
                  <span className="text-slate-500">发货时间</span>
                  <span>{new Date(order.shipped_at).toLocaleString()}</span>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card className="border-slate-200">
            <CardHeader><CardTitle className="text-lg">订单摘要</CardTitle></CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between">
                <span className="text-slate-500">成交金额</span>
                <span>¥{(order.final_price || 0).toLocaleString()}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">佣金 (5%)</span>
                <span>¥{Math.round((order.final_price || 0) * 0.05).toLocaleString()}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">运费</span>
                <span>¥0 (包邮)</span>
              </div>
              <div className="border-t pt-4 flex justify-between font-bold text-lg">
                <span>应收总计</span>
                <span className="text-amber-600">
                  ¥{Math.round((order.final_price || 0) * 1.05).toLocaleString()}
                </span>
              </div>
              {canShip && (
                <Button
                  className="w-full bg-amber-500 text-[#0f172a] mt-4"
                  disabled={shipping}
                  onClick={handleShip}
                >
                  {shipping ? <Loader2 className="w-4 h-4 animate-spin mr-2" /> : null}
                  标记发货
                </Button>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}