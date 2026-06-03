import React from "react"
import {
  ArrowLeft,
  History,
  AlertCircle,
  ShieldCheck,
  Loader2
} from "lucide-react"
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer
} from "recharts"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge, type BadgeProps } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useNavigate, useSearchParams } from "react-router-dom"
import { auctionApi, productApi } from "@/shared/api"

const statusMap: Record<number, { label: string; variant: BadgeProps["variant"] }> = {
  0: { label: "待开始", variant: "blue" },
  1: { label: "进行中", variant: "success" },
  2: { label: "延时中", variant: "warning" },
  3: { label: "已结束", variant: "outline" },
  4: { label: "已取消", variant: "secondary" },
}

export default function AuctionDetail() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const auctionId = searchParams.get('id')

  const [loading, setLoading] = React.useState(true)
  const [auction, setAuction] = React.useState<any>(null)
  const [bids, setBids] = React.useState<any[]>([])
  const [product, setProduct] = React.useState<any>(null)
  const [canceling, setCanceling] = React.useState(false)

  // 获取竞拍详情和出价记录
  React.useEffect(() => {
    if (!auctionId) {
      navigate('/auction/list')
      return
    }

    const fetchData = async () => {
      setLoading(true)
      try {
        // 获取竞拍详情
        const auctionData = await auctionApi.get(Number(auctionId))
        setAuction(auctionData)

        // 获取出价记录
        const bidsData = await auctionApi.getBids(Number(auctionId))
        setBids(bidsData || [])

        // 获取商品详情
        if (auctionData.product_id) {
          const productData = await productApi.get(auctionData.product_id)
          setProduct(productData)
        }
      } catch (e) {
        console.error('获取竞拍详情失败:', e)
        alert('获取竞拍详情失败')
        navigate('/auction/list')
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [auctionId, navigate])

  // 终止竞拍
  const handleCancel = async () => {
    if (!confirm('确定要终止这场竞拍吗？此操作不可撤销。')) return

    setCanceling(true)
    try {
      await auctionApi.cancel(Number(auctionId))
      alert('竞拍已终止')
      navigate('/auction/list')
    } catch (e: any) {
      console.error('终止竞拍失败:', e)
      alert(e.message || '终止竞拍失败')
    } finally {
      setCanceling(false)
    }
  }

  // 计算剩余时间
  const getRemainingTime = () => {
    if (!auction?.end_time) return '未知'
    const end = new Date(auction.end_time)
    const now = new Date()
    const diff = Math.max(0, Math.floor((end.getTime() - now.getTime()) / 1000))
    const minutes = Math.floor(diff / 60)
    const seconds = diff % 60
    return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
  }

  // 价格走势数据（基于出价记录）
  const priceData = React.useMemo(() => {
    if (!bids.length) return [{ time: '开始', price: auction?.current_price || 0 }]
    return bids.slice(0, 20).reverse().map((bid) => ({
      time: new Date(bid.created_at).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
      price: bid.price,
    }))
  }, [bids, auction])

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-400" />
      </div>
    )
  }

  if (!auction) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <p className="text-slate-500">竞拍不存在</p>
      </div>
    )
  }

  const canCancel = auction.status === 0 || auction.status === 1 || auction.status === 2

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="icon" onClick={() => navigate("/auction/list")} className="border-slate-200">
            <ArrowLeft className="w-4 h-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-slate-900">竞拍场次详情</h1>
            <p className="text-sm text-slate-500">
              场次 ID: {auction.id} | {product?.name || `商品 #${auction.product_id}`}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          {canCancel && (
            <Button
              variant="destructive"
              className="bg-rose-500 hover:bg-rose-600"
              disabled={canceling}
              onClick={handleCancel}
            >
              {canceling ? <Loader2 className="mr-2 w-4 h-4 animate-spin" /> : <AlertCircle className="mr-2 w-4 h-4" />}
              终止竞拍
            </Button>
          )}
          {/* 进入直播间 - 后端无控制接口，暂空置跳转 */}
          <Button className="bg-amber-500 hover:bg-amber-600 text-[#0f172a]" disabled>
            进入直播间
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column: Stats & Chart */}
        <div className="lg:col-span-8 space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Card className="border-slate-200 bg-amber-50 border-amber-100">
              <CardContent className="p-4">
                <p className="text-xs text-amber-700 font-medium">当前最高出价</p>
                <h3 className="text-2xl font-bold text-amber-600 mt-1">
                  ¥{(auction.current_price || 0).toLocaleString()}
                </h3>
                <p className="text-xs text-amber-700 mt-1">
                  领先者: {auction.winner_name || bids[0]?.user_name || '暂无'}
                </p>
              </CardContent>
            </Card>
            <Card className="border-slate-200">
              <CardContent className="p-4">
                <p className="text-xs text-slate-500 font-medium">出价次数</p>
                <h3 className="text-2xl font-bold text-slate-900 mt-1">
                  {auction.bid_count || bids.length || 0}
                </h3>
                <p className="text-xs text-emerald-500 mt-1">累计参与</p>
              </CardContent>
            </Card>
            <Card className="border-slate-200">
              <CardContent className="p-4">
                <p className="text-xs text-slate-500 font-medium">竞拍状态</p>
                <div className="mt-1">
                  <Badge variant={statusMap[auction.status]?.variant || 'secondary'}>
                    {statusMap[auction.status]?.label || '未知'}
                  </Badge>
                </div>
                <p className="text-xs text-slate-500 mt-2">
                  {auction.status === 1 || auction.status === 2
                    ? `剩余: ${getRemainingTime()}`
                    : auction.end_time
                      ? `结束于: ${new Date(auction.end_time).toLocaleString()}`
                      : ''
                  }
                </p>
              </CardContent>
            </Card>
          </div>

          <Card className="border-slate-200">
            <CardHeader>
              <CardTitle className="text-lg">价格走势图</CardTitle>
              <CardDescription>出价变化趋势</CardDescription>
            </CardHeader>
            <CardContent className="h-[300px]">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={priceData}>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                  <XAxis dataKey="time" axisLine={false} tickLine={false} tick={{ fontSize: 12, fill: '#64748b' }} />
                  <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 12, fill: '#64748b' }} />
                  <Tooltip formatter={(value) => `¥${Number(value).toLocaleString()}`} />
                  <Line type="monotone" dataKey="price" stroke="#f59e0b" strokeWidth={3} dot={{ r: 6, fill: '#f59e0b' }} />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <Card className="border-slate-200">
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle className="text-lg">出价记录</CardTitle>
                <CardDescription>最近出价明细</CardDescription>
              </div>
              <Button variant="outline" size="sm" className="border-slate-200">
                <History className="mr-2 w-4 h-4" />
                完整记录
              </Button>
            </CardHeader>
            <CardContent className="p-0">
              <Table>
                <TableHeader className="bg-slate-50">
                  <TableRow>
                    <TableHead>出价人</TableHead>
                    <TableHead>出价金额</TableHead>
                    <TableHead>出价时间</TableHead>
                    <TableHead>状态</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {bids.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center text-slate-500 py-4">
                        暂无出价记录
                      </TableCell>
                    </TableRow>
                  ) : (
                    bids.slice(0, 10).map((bid, index) => (
                      <TableRow key={bid.id || index}>
                        <TableCell className="font-medium">{bid.user_name || `用户#${bid.user_id}`}</TableCell>
                        <TableCell className="font-bold text-slate-900">
                          ¥{(bid.price || 0).toLocaleString()}
                        </TableCell>
                        <TableCell className="text-slate-500">
                          {new Date(bid.created_at).toLocaleTimeString()}
                        </TableCell>
                        <TableCell>
                          {index === 0 ? (
                            <Badge className="bg-amber-500 text-[#0f172a]">当前领先</Badge>
                          ) : (
                            <Badge variant="secondary">已出局</Badge>
                          )}
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>

        {/* Right Column: Product & Rules */}
        <div className="lg:col-span-4 space-y-6">
          <Card className="border-slate-200">
            <CardHeader>
              <CardTitle className="text-lg">正在竞拍商品</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="aspect-video rounded-lg bg-slate-100 overflow-hidden border border-slate-200">
                {product?.images?.[0] ? (
                  <img src={product.images[0]} alt={product.name} className="w-full h-full object-cover" />
                ) : (
                  <div className="w-full h-full flex items-center justify-center text-slate-400">
                    无图片
                  </div>
                )}
              </div>
              <div>
                <h4 className="font-bold text-slate-900">{product?.name || `商品 #${auction.product_id}`}</h4>
                <p className="text-sm text-slate-500 mt-1">
                  起拍价: ¥{(product?.rules?.start_price || 0).toLocaleString()} |
                  当前价: ¥{(auction.current_price || 0).toLocaleString()}
                </p>
              </div>
              <div className="pt-4 border-t border-slate-100 flex items-center justify-between">
                <div className="flex items-center gap-2 text-emerald-600">
                  <ShieldCheck className="w-4 h-4" />
                  <span className="text-xs font-medium">正品保证</span>
                </div>
                <Button variant="link" className="text-amber-600 p-0 h-auto text-xs">
                  查看详情
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card className="border-slate-200">
            <CardHeader>
              <CardTitle className="text-lg">竞拍规则摘要</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {product?.rules ? (
                <>
                  <RuleItem label="起拍价" value={`¥${product.rules.start_price.toLocaleString()}`} />
                  <RuleItem label="加价幅度" value={`¥${product.rules.increment.toLocaleString()}`} />
                  <RuleItem label="封顶价" value={`¥${product.rules.cap_price.toLocaleString()}`} />
                  <RuleItem label="竞拍时长" value={`${product.rules.duration} 秒`} />
                  <RuleItem label="延时规则" value={`最后 ${product.rules.trigger_delay_before} 秒自动延时 ${product.rules.delay_duration} 秒`} />
                </>
              ) : (
                <p className="text-sm text-slate-500">暂无规则配置</p>
              )}
              <div className="pt-4">
                <Button variant="outline" className="w-full border-slate-200 text-slate-600">
                  查看完整规则模板
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}

function RuleItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between py-2 border-b border-slate-50 last:border-0">
      <span className="text-sm text-slate-500">{label}</span>
      <span className="text-sm font-semibold text-slate-900">{value}</span>
    </div>
  )
}
