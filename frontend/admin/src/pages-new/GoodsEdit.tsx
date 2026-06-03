import React from "react"
import { ArrowLeft, Save, Plus, X } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { productApi } from "@/shared/api"

export default function GoodsEdit() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const productId = searchParams.get('id')
  const isEditMode = !!productId

  const [loading, setLoading] = React.useState(false)
  const [saving, setSaving] = React.useState(false)
  const [formData, setFormData] = React.useState({
    name: '',
    category: '艺术收藏',
    brand: '',
    description: '',
    images: [] as string[],
  })
  const [imageUrlInput, setImageUrlInput] = React.useState('')

  // 编辑模式：获取商品详情
  React.useEffect(() => {
    if (isEditMode && productId) {
      setLoading(true)
      productApi.get(Number(productId))
        .then((data) => {
          setFormData({
            name: data.name || '',
            category: data.category || '艺术收藏',
            brand: '',
            description: data.description || '',
            images: data.images || [],
          })
        })
        .catch((e) => {
          console.error('获取商品详情失败:', e)
          alert('获取商品详情失败')
          navigate('/goods/list')
        })
        .finally(() => setLoading(false))
    }
  }, [isEditMode, productId, navigate])

  // 更新表单字段
  const updateField = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }))
  }

  // 添加图片URL
  const addImage = () => {
    if (imageUrlInput.trim()) {
      setFormData(prev => ({
        ...prev,
        images: [...prev.images, imageUrlInput.trim()]
      }))
      setImageUrlInput('')
    }
  }

  // 删除图片
  const removeImage = (index: number) => {
    setFormData(prev => ({
      ...prev,
      images: prev.images.filter((_, i) => i !== index)
    }))
  }

  // 提交表单
  const handleSubmit = async (publish: boolean = false) => {
    if (!formData.name.trim()) {
      alert('请输入商品名称')
      return
    }
    if (!formData.description.trim()) {
      alert('请输入商品描述')
      return
    }

    setSaving(true)
    try {
      if (isEditMode && productId) {
        await productApi.update(Number(productId), formData)
        if (publish) {
          await productApi.publish(Number(productId))
        }
        alert('商品更新成功')
      } else {
        const result = await productApi.create(formData)
        if (publish && result.id) {
          await productApi.publish(result.id)
        }
        alert('商品创建成功')
      }
      navigate('/goods/list')
    } catch (e: any) {
      console.error('保存失败:', e)
      alert(e.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-slate-500">加载中...</div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="outline" size="icon" onClick={() => navigate("/goods/list")} className="border-slate-200">
          <ArrowLeft className="w-4 h-4" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold text-slate-900">
            {isEditMode ? '编辑商品' : '发布新商品'}
          </h1>
          <p className="text-sm text-slate-500">
            {isEditMode ? '修改商品详细信息' : '完善商品详细信息并提交审核'}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <Card className="border-slate-200">
            <CardHeader>
              <CardTitle className="text-lg">基本信息</CardTitle>
              <CardDescription>设置商品的名称、类别和描述</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">商品名称 *</label>
                <Input
                  placeholder="输入商品完整名称"
                  className="bg-slate-50 border-slate-200"
                  value={formData.name}
                  onChange={(e) => updateField('name', e.target.value)}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700">商品类别</label>
                  <select
                    className="flex h-10 w-full rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500"
                    value={formData.category}
                    onChange={(e) => updateField('category', e.target.value)}
                  >
                    <option>艺术收藏</option>
                    <option>珠宝名表</option>
                    <option>数码电子</option>
                    <option>奢侈品</option>
                    <option>其他</option>
                  </select>
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700">商品品牌</label>
                  <Input
                    placeholder="输入品牌名称"
                    className="bg-slate-50 border-slate-200"
                    value={formData.brand}
                    onChange={(e) => updateField('brand', e.target.value)}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">详细描述 *</label>
                <textarea
                  className="flex min-h-[120px] w-full rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500"
                  placeholder="详细介绍商品的来源、年代、成色等信息..."
                  value={formData.description}
                  onChange={(e) => updateField('description', e.target.value)}
                ></textarea>
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card className="border-slate-200">
            <CardHeader>
              <CardTitle className="text-lg">商品图片</CardTitle>
              <CardDescription>主图将作为搜索和列表封面</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* 图片列表 */}
              {formData.images.length > 0 && (
                <div className="grid grid-cols-2 gap-3 mb-4">
                  {formData.images.map((img, index) => (
                    <div key={index} className="relative aspect-square rounded-lg overflow-hidden border border-slate-200">
                      <img src={img} alt={`商品图片${index + 1}`} className="w-full h-full object-cover" />
                      <button
                        onClick={() => removeImage(index)}
                        className="absolute top-1 right-1 w-6 h-6 rounded-full bg-red-500 text-white flex items-center justify-center hover:bg-red-600"
                      >
                        <X className="w-3 h-3" />
                      </button>
                    </div>
                  ))}
                </div>
              )}

              {/* 添加图片URL */}
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">添加图片URL</label>
                <div className="flex gap-2">
                  <Input
                    placeholder="输入图片URL"
                    className="bg-slate-50 border-slate-200"
                    value={imageUrlInput}
                    onChange={(e) => setImageUrlInput(e.target.value)}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    className="border-slate-200"
                    onClick={addImage}
                  >
                    <Plus className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="border-slate-200">
            <CardHeader>
              <CardTitle className="text-lg">发布状态</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between p-3 rounded-lg bg-slate-50 border border-slate-100">
                <span className="text-sm text-slate-600">当前状态</span>
                <Badge variant="secondary">
                  {isEditMode ? '编辑中' : '未发布'}
                </Badge>
              </div>
              <div className="space-y-2">
                <Button
                  className="w-full bg-amber-500 hover:bg-amber-600 text-[#0f172a] font-bold"
                  disabled={saving}
                  onClick={() => handleSubmit(true)}
                >
                  <Save className="mr-2 w-4 h-4" />
                  {saving ? '保存中...' : '保存并发布'}
                </Button>
                <Button
                  variant="outline"
                  className="w-full border-slate-200"
                  disabled={saving}
                  onClick={() => handleSubmit(false)}
                >
                  保存为草稿
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}