import React from "react"
import { ArrowLeft, Loader2, Plus, RefreshCw } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router-dom"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  auctionApi,
  fixedPriceAdminApi,
  liveStreamApi,
  productApi,
  type FixedPriceAdminItem,
  type FixedPriceAdminStatus,
  type Product,
} from "@/shared/api"

interface LiveStreamFixedPriceProps {
  liveStreamId?: number
}

const pageSize = 20

const statusMeta: Record<FixedPriceAdminStatus, { label: string; variant: "success" | "warning" | "outline" }> = {
  on_sale: { label: "在售", variant: "success" },
  sold_out: { label: "已售罄", variant: "warning" },
  offline: { label: "已下架", variant: "outline" },
}

function getProductTitle(item: FixedPriceAdminItem) {
  return item.product_title || item.product?.title || `商品 #${item.product_id}`
}

function normalizeItem(item: FixedPriceAdminItem): FixedPriceAdminItem {
  return {
    ...item,
    product_title: getProductTitle(item),
  }
}

export default function LiveStreamFixedPrice({ liveStreamId: propLiveStreamId }: LiveStreamFixedPriceProps) {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const routeLiveStreamId = Number(searchParams.get("id") || 0)
  const [resolvedLiveStreamId, setResolvedLiveStreamId] = React.useState(0)
  const liveStreamId = propLiveStreamId || routeLiveStreamId || resolvedLiveStreamId

  const [items, setItems] = React.useState<FixedPriceAdminItem[]>([])
  const [total, setTotal] = React.useState(0)
  const [loading, setLoading] = React.useState(true)
  const [submitting, setSubmitting] = React.useState(false)
  const [auctionOptions, setAuctionOptions] = React.useState<any[]>([])
  const [productOptions, setProductOptions] = React.useState<Product[]>([])
  const [auctionId, setAuctionId] = React.useState("")
  const [productId, setProductId] = React.useState("")
  const [price, setPrice] = React.useState("")
  const [stock, setStock] = React.useState("")

  React.useEffect(() => {
    if (propLiveStreamId || routeLiveStreamId) {
      setResolvedLiveStreamId(0)
      return
    }

    let cancelled = false
    setLoading(true)
    liveStreamApi.adminList({ page: 1, page_size: 1 })
      .then((response) => {
        if (cancelled) return
        const firstLiveStreamId = Number(response.list?.[0]?.id || 0)
        if (firstLiveStreamId > 0) {
          setResolvedLiveStreamId(firstLiveStreamId)
          navigate(`/live/fixed-price?id=${firstLiveStreamId}`, { replace: true })
          return
        }
        setLoading(false)
      })
      .catch((error) => {
        if (cancelled) return
        console.error("获取商家直播间失败:", error)
        alert("请先创建直播间")
        setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [navigate, propLiveStreamId, routeLiveStreamId])

  const fetchItems = React.useCallback(async () => {
    if (!liveStreamId) {
      return
    }

    setLoading(true)
    try {
      const [response, auctionResponse, productResponse] = await Promise.all([
        fixedPriceAdminApi.list(liveStreamId, { page: 1, page_size: pageSize }),
        auctionApi.list({ live_stream_id: liveStreamId, page: 1, page_size: 100 }),
        productApi.list({ display_status: "schedulable", page: 1, page_size: 100 }),
      ])
      const nextItems = (response.items || []).map(normalizeItem)
      const nextAuctions = (auctionResponse.list || []).filter((auction: any) => [0, 1, 2].includes(Number(auction.status)))
      const nextProducts = productResponse.list || []
      setItems(nextItems)
      setTotal(response.total ?? nextItems.length)
      setAuctionOptions(nextAuctions)
      setProductOptions(nextProducts)
      setAuctionId((current) => current || (nextAuctions[0]?.id ? String(nextAuctions[0].id) : ""))
      setProductId((current) => current || (nextProducts[0]?.id ? String(nextProducts[0].id) : ""))
    } catch (error) {
      console.error("获取一口价列表失败:", error)
      alert("获取一口价列表失败")
    } finally {
      setLoading(false)
    }
  }, [liveStreamId])

  React.useEffect(() => {
    fetchItems()
  }, [fetchItems])

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!liveStreamId || submitting) return
    if (!auctionId) {
      alert("请先创建竞拍场次")
      return
    }

    setSubmitting(true)
    try {
      const created = await fixedPriceAdminApi.listItem(liveStreamId, {
        auction_id: Number(auctionId),
        product_id: Number(productId),
        price,
        stock: Number(stock),
      })
      const completedItem: FixedPriceAdminItem = {
        ...created,
        auction_id: created.auction_id ?? Number(auctionId),
        live_stream_id: created.live_stream_id ?? liveStreamId,
        product_id: created.product_id ?? Number(productId),
        price: created.price ?? price,
        total_stock: created.total_stock ?? Number(stock),
        remaining_stock: created.remaining_stock ?? Number(stock),
        status: created.status || "on_sale",
      }
      const selectedProductName = productOptions.find((p) => String(p.id) === productId)?.name
      if (selectedProductName) {
        completedItem.product_title = selectedProductName
      }
      setItems((prev) => [normalizeItem(completedItem), ...prev])
      setTotal((prev) => prev + 1)
      setPrice("")
      setStock("")
    } catch (error) {
      console.error("上架一口价商品失败:", error)
      alert("上架一口价商品失败")
    } finally {
      setSubmitting(false)
    }
  }

  const handleOffline = async (itemId: number) => {
    if (!window.confirm("确认下架该一口价商品？")) return

    try {
      const result = await fixedPriceAdminApi.offline(itemId)
      setItems((prev) => prev.map((item) => (
        item.id === itemId ? { ...item, status: result.status || "offline" } : item
      )))
    } catch (error) {
      console.error("下架一口价商品失败:", error)
      alert("下架一口价商品失败")
    }
  }

  if (!liveStreamId) {
    return (
      <div className="flex flex-col items-center justify-center gap-4 min-h-[400px] text-slate-500">
        <p>请先创建直播间，再管理一口价商品</p>
        <Button className="bg-amber-500 text-[#0f172a] hover:bg-amber-600" onClick={() => navigate("/live/my")}>
          前往我的直播间
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="icon" onClick={() => navigate("/live/list")} className="border-slate-200">
            <ArrowLeft className="w-4 h-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-slate-900">一口价上下架</h1>
            <p className="text-sm text-slate-500">直播间 #{liveStreamId} 的固定价商品列表、上架与下架</p>
          </div>
        </div>
        <Button variant="outline" onClick={fetchItems} className="border-slate-200">
          <RefreshCw className="mr-2 w-4 h-4" />
          刷新
        </Button>
      </div>

      <Card className="border-slate-200">
        <CardHeader>
          <CardTitle className="text-lg">新增上架</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="grid grid-cols-1 gap-4 md:grid-cols-[1fr_1fr_1fr_1fr_auto]" onSubmit={handleSubmit}>
            <label className="space-y-2 text-sm font-medium text-slate-700">
              竞拍场次
              <select
                aria-label="竞拍场次"
                className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm"
                value={auctionId}
                onChange={(event) => setAuctionId(event.target.value)}
                required
              >
                {auctionOptions.length === 0 ? (
                  <option value="">请先创建竞拍场次</option>
                ) : (
                  auctionOptions.map((auction) => (
                    <option key={auction.id} value={auction.id}>
                      {auction.product?.name || auction.title || `竞拍 #${auction.id}`}
                    </option>
                  ))
                )}
              </select>
            </label>
            <label className="space-y-2 text-sm font-medium text-slate-700">
              搭售商品
              <select
                aria-label="搭售商品"
                className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm"
                value={productId}
                onChange={(event) => setProductId(event.target.value)}
                required
                disabled={productOptions.length === 0}
              >
                {productOptions.length === 0 ? (
                  <option value="">暂无可搭售商品，请先创建并发布商品</option>
                ) : (
                  productOptions.map((product) => (
                    <option key={product.id} value={product.id}>
                      {product.name}（#{product.id}）
                    </option>
                  ))
                )}
              </select>
            </label>
            <label className="space-y-2 text-sm font-medium text-slate-700">
              一口价
              <Input
                value={price}
                onChange={(event) => setPrice(event.target.value)}
                required
                placeholder="99.00"
              />
            </label>
            <label className="space-y-2 text-sm font-medium text-slate-700">
              库存
              <Input
                type="number"
                min="1"
                value={stock}
                onChange={(event) => setStock(event.target.value)}
                required
                placeholder="20"
              />
            </label>
            <Button type="submit" disabled={submitting || !auctionId || productOptions.length === 0} className="self-end bg-amber-500 text-[#0f172a] hover:bg-amber-600">
              <Plus className="mr-2 w-4 h-4" />
              新增上架
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card className="border-slate-200">
        <CardContent className="p-0">
          {loading ? (
            <div className="flex items-center justify-center gap-2 p-8 text-slate-500">
              <Loader2 className="w-5 h-5 animate-spin" />
              加载中...
            </div>
          ) : (
            <Table>
              <TableHeader className="bg-slate-50/50">
                <TableRow>
                  <TableHead>商品</TableHead>
                  <TableHead>价格</TableHead>
                  <TableHead>库存</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="py-8 text-center text-slate-500">
                      暂无一口价商品
                    </TableCell>
                  </TableRow>
                ) : (
                  items.map((item) => {
                    const meta = statusMeta[item.status] || statusMeta.offline
                    return (
                      <TableRow key={item.id} className="hover:bg-slate-50/80">
                        <TableCell>
                          <div className="font-medium text-slate-900">{getProductTitle(item)}</div>
                          <div className="text-xs text-slate-500">商品 ID: {item.product_id} | 条目 ID: {item.id}</div>
                        </TableCell>
                        <TableCell className="font-semibold text-slate-900">¥{item.price}</TableCell>
                        <TableCell>{item.remaining_stock} / {item.total_stock}</TableCell>
                        <TableCell>
                          <Badge variant={meta.variant}>{meta.label}</Badge>
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={item.status === "offline"}
                            onClick={() => handleOffline(item.id)}
                            className="border-slate-200"
                          >
                            下架
                          </Button>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          )}
          <div className="border-t border-slate-100 p-4 text-sm text-slate-500">
            共 {total} 条一口价商品
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
