import React from "react"
import { ArrowLeft, Save, Plus, X, Sparkles } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Category, productApi } from "@/shared/api"
import {
  buildCopywritingKeywords,
  formatAiDescription,
  getCopywritingErrorMessage,
  getValidCopywritingImages,
} from "./goodsEditAi"

export default function GoodsEdit() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const productId = searchParams.get('id')
  const isEditMode = !!productId

  const [loading, setLoading] = React.useState(false)
  const [saving, setSaving] = React.useState(false)
  const [aiGenerating, setAiGenerating] = React.useState(false)
  const [categories, setCategories] = React.useState<Category[]>([])
  const [categoryNameFallback, setCategoryNameFallback] = React.useState('')
  const [aiDraft, setAiDraft] = React.useState<{
    sellingPoints: string[]
    suggestedStartPrice: string
    appliedAt?: string
  } | null>(null)
  const [formData, setFormData] = React.useState({
    name: '',
    category_id: null as number | null,
    brand: '',
    description: '',
    images: [] as string[],
  })
  const [imageUrlInput, setImageUrlInput] = React.useState('')

  React.useEffect(() => {
    productApi.listCategories()
      .then((data) => {
        setCategories(data)
      })
      .catch((e) => {
        console.error('获取商品分类失败:', e)
        alert('获取商品分类失败')
      })
  }, [])

  // 编辑模式：获取商品详情
  React.useEffect(() => {
    if (isEditMode && productId) {
      setLoading(true)
      productApi.get(Number(productId))
        .then((data) => {
          setFormData({
            name: data.name || '',
            category_id: data.category_id ?? null,
            brand: '',
            description: data.description || '',
            images: data.images || [],
          })
          setCategoryNameFallback(data.category_name || '')
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
  const updateField = <K extends keyof typeof formData>(field: K, value: (typeof formData)[K]) => {
    setFormData(prev => ({ ...prev, [field]: value }))
  }

  const selectedCategoryName = React.useMemo(() => {
    const selectedCategory = categories.find((item) => item.id === formData.category_id)
    return selectedCategory?.name || categoryNameFallback
  }, [categories, categoryNameFallback, formData.category_id])

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

  // AI 一键文案只预填当前表单，不自动保存或发布。
  const handleGenerateCopywriting = async () => {
    const hasInvalidImage = formData.images.some((image) => {
      const value = image.trim()
      return value !== '' && !value.startsWith('http://') && !value.startsWith('https://')
    })
    if (hasInvalidImage) {
      alert('图片 URL 必须以 http:// 或 https:// 开头')
      return
    }

    const images = getValidCopywritingImages(formData.images)
    if (images.length === 0) {
      alert('请先添加至少一张商品图片')
      return
    }

    if (formData.images.length > images.length) {
      alert('最多使用前 6 张合法图片生成文案')
    }

    setAiGenerating(true)
    try {
      const draft = await productApi.generateCopywriting({
        images,
        keywords: buildCopywritingKeywords({
          ...formData,
          category: selectedCategoryName,
        }),
      })

      setFormData(prev => ({
        ...prev,
        name: draft.name,
        description: formatAiDescription(draft.description, draft.selling_points),
      }))
      setAiDraft({
        sellingPoints: draft.selling_points,
        suggestedStartPrice: draft.suggested_start_price,
        appliedAt: new Date().toISOString(),
      })
    } catch (e: any) {
      console.error('AI 文案生成失败:', e)
      alert(getCopywritingErrorMessage(e))
    } finally {
      setAiGenerating(false)
    }
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
    if (!formData.category_id) {
      alert('请选择商品分类')
      return
    }

    setSaving(true)
    try {
      const payload = {
        name: formData.name,
        description: formData.description,
        images: formData.images,
        category_id: formData.category_id,
      }

      if (isEditMode && productId) {
        await productApi.update(Number(productId), payload)
        if (publish) {
          await productApi.publish(Number(productId))
        }
        alert('商品更新成功')
      } else {
        const result = await productApi.create(payload)
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
              <div className="flex items-start justify-between gap-3">
                <div>
                  <CardTitle className="text-lg">基本信息</CardTitle>
                  <CardDescription>设置商品的名称、类别和描述</CardDescription>
                </div>
                <Button
                  type="button"
                  className="bg-amber-500 hover:bg-amber-600 text-[#0f172a] font-bold"
                  disabled={saving || aiGenerating}
                  onClick={handleGenerateCopywriting}
                >
                  <Sparkles className="mr-2 w-4 h-4" />
                  {aiGenerating ? 'AI 生成中...' : 'AI 一键文案'}
                </Button>
              </div>
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
                    value={formData.category_id?.toString() || ''}
                    onChange={(e) => {
                      const nextValue = e.target.value ? Number(e.target.value) : null
                      updateField('category_id', nextValue)
                    }}
                  >
                    <option value="">请选择分类</option>
                    {categories.map((category) => (
                      <option key={category.id} value={category.id}>
                        {category.name}
                      </option>
                    ))}
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
                    aria-label="添加图片 URL"
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

          {aiDraft && (
            <Card className="border-amber-200 bg-amber-50">
              <CardHeader>
                <CardTitle className="text-lg text-amber-900">AI 建议</CardTitle>
                <CardDescription className="text-amber-800">
                  AI 仅生成草稿，请确认后再保存或发布。
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                {aiDraft.sellingPoints.length > 0 && (
                  <div className="space-y-2">
                    <div className="text-sm font-medium text-amber-900">核心卖点</div>
                    <div className="flex flex-wrap gap-2">
                      {aiDraft.sellingPoints.map((point) => (
                        <Badge key={point} variant="secondary" className="bg-white text-amber-900 border border-amber-200">
                          {point}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
                {aiDraft.suggestedStartPrice && (
                  <div className="text-sm text-amber-900">
                    AI 建议起拍价：¥{aiDraft.suggestedStartPrice}
                  </div>
                )}
              </CardContent>
            </Card>
          )}

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
