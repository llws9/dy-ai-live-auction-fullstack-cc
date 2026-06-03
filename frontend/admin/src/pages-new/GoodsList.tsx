import React from "react"
import {
  Search,
  Plus,
  MoreHorizontal,
  Edit,
  Trash2,
  Eye,
  Filter,
  Upload,
  ArrowDown
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
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Card, CardContent } from "@/components/ui/card"
import { useNavigate } from "react-router-dom"
import { productApi, Product } from "@/shared/api"

type BadgeVariant = React.ComponentProps<typeof Badge>["variant"]

const statusMap: Record<number, { label: string; variant: BadgeVariant }> = {
  0: { label: "未发布", variant: "secondary" },
  1: { label: "已发布", variant: "success" },
  2: { label: "已下架", variant: "outline" },
}

export default function GoodsList() {
  const navigate = useNavigate()
  const [products, setProducts] = React.useState<Product[]>([])
  const [loading, setLoading] = React.useState(true)
  const [statusFilter, setStatusFilter] = React.useState<number | undefined>(undefined)
  const [page, setPage] = React.useState(1)
  const [total, setTotal] = React.useState(0)
  const [searchTerm, setSearchTerm] = React.useState("")
  const pageSize = 10

  // 获取商品列表
  const fetchProducts = React.useCallback(async () => {
    setLoading(true)
    try {
      const response = await productApi.list({
        status: statusFilter,
        page,
        page_size: pageSize,
      })
      setProducts(response.list || [])
      setTotal(response.total || 0)
    } catch (e) {
      console.error('获取商品列表失败:', e)
    } finally {
      setLoading(false)
    }
  }, [statusFilter, page])

  React.useEffect(() => {
    fetchProducts()
  }, [fetchProducts])

  // 删除商品
  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个商品吗？')) return
    try {
      await productApi.delete(id)
      fetchProducts()
    } catch (e) {
      console.error('删除失败:', e)
    }
  }

  // 发布商品
  const handlePublish = async (id: number) => {
    try {
      await productApi.publish(id)
      fetchProducts()
    } catch (e) {
      console.error('发布失败:', e)
    }
  }

  // 下架商品
  const handleUnpublish = async (id: number) => {
    if (!confirm('确定要下架这个商品吗？')) return
    try {
      await productApi.unpublish(id)
      fetchProducts()
    } catch (e) {
      console.error('下架失败:', e)
    }
  }

  // 状态筛选
  const handleStatusChange = (value: string) => {
    if (value === 'all') {
      setStatusFilter(undefined)
    } else {
      setStatusFilter(Number(value))
    }
    setPage(1)
  }

  // 过滤商品（本地搜索）
  const filteredProducts = React.useMemo(() => {
    if (!searchTerm) return products
    return products.filter(p =>
      p.name.toLowerCase().includes(searchTerm.toLowerCase())
    )
  }, [products, searchTerm])

  // 分页计算
  const totalPages = Math.ceil(total / pageSize)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">商品管理</h1>
          <p className="text-sm text-slate-500">管理您的所有竞拍商品</p>
        </div>
        <Button className="bg-amber-500 hover:bg-amber-600 text-[#0f172a]" onClick={() => navigate("/goods/create")}>
          <Plus className="mr-2 w-4 h-4" />
          新增商品
        </Button>
      </div>

      <Card className="border-slate-200">
        <CardContent className="p-0">
          <div className="p-4 border-b border-slate-100 flex flex-col md:flex-row md:items-center justify-between gap-4">
            <Tabs defaultValue="all" onValueChange={handleStatusChange} className="w-full md:w-auto">
              <TabsList className="bg-slate-100 border-none">
                <TabsTrigger value="all">全部</TabsTrigger>
                <TabsTrigger value="0">未发布</TabsTrigger>
                <TabsTrigger value="1">已发布</TabsTrigger>
                <TabsTrigger value="2">已下架</TabsTrigger>
              </TabsList>
            </Tabs>

            <div className="flex items-center gap-2">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <Input
                  placeholder="搜索商品名称..."
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
            <Table>
              <TableHeader className="bg-slate-50/50">
                <TableRow>
                  <TableHead className="w-[300px]">商品信息</TableHead>
                  <TableHead>类别</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredProducts.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-slate-500 py-8">
                      暂无商品数据
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredProducts.map((item) => (
                    <TableRow key={item.id} className="hover:bg-slate-50/80 transition-colors">
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <div className="w-12 h-12 rounded-lg bg-slate-100 overflow-hidden border border-slate-200 shrink-0">
                            {item.images?.[0] ? (
                              <img src={item.images[0]} alt={item.name} className="w-full h-full object-cover" />
                            ) : (
                              <div className="w-full h-full flex items-center justify-center text-slate-400">
                                <Eye className="w-4 h-4" />
                              </div>
                            )}
                          </div>
                          <div className="min-w-0">
                            <p className="text-sm font-semibold text-slate-900 truncate">{item.name}</p>
                            <p className="text-xs text-slate-500">ID: {item.id}</p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="font-normal border-slate-200">
                          {item.category || '未分类'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusMap[item.status]?.variant || 'secondary'}>
                          {statusMap[item.status]?.label || '未知'}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-slate-500 text-sm">
                        {new Date(item.created_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            variant="ghost"
                            size="icon"
                            className="text-slate-400 hover:text-blue-500"
                            onClick={() => navigate(`/goods/edit?id=${item.id}`)}
                          >
                            <Edit className="w-4 h-4" />
                          </Button>
                          {item.status === 0 && (
                            <Button
                              variant="ghost"
                              size="icon"
                              className="text-slate-400 hover:text-green-500"
                              onClick={() => handlePublish(item.id)}
                              title="发布"
                            >
                              <Upload className="w-4 h-4" />
                            </Button>
                          )}
                          {item.status === 1 && (
                            <Button
                              variant="ghost"
                              size="icon"
                              className="text-slate-400 hover:text-orange-500"
                              onClick={() => handleUnpublish(item.id)}
                              title="下架"
                            >
                              <ArrowDown className="w-4 h-4" />
                            </Button>
                          )}
                          <Button
                            variant="ghost"
                            size="icon"
                            className="text-slate-400 hover:text-red-500"
                            onClick={() => handleDelete(item.id)}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                          <Button variant="ghost" size="icon" className="text-slate-400">
                            <MoreHorizontal className="w-4 h-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          )}

          <div className="p-4 border-t border-slate-100 flex items-center justify-between">
            <p className="text-sm text-slate-500">
              显示 {((page - 1) * pageSize) + 1} 到 {Math.min(page * pageSize, total)}，共 {total} 条商品
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
              {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => (
                <Button
                  key={i + 1}
                  variant="outline"
                  size="sm"
                  className={`border-slate-200 ${page === i + 1 ? 'bg-amber-50 text-amber-600 border-amber-200' : ''}`}
                  onClick={() => setPage(i + 1)}
                >
                  {i + 1}
                </Button>
              ))}
              <Button
                variant="outline"
                size="sm"
                className="border-slate-200"
                disabled={page >= totalPages}
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