export interface GoodsEditAiKeywordInput {
  category?: string
  brand?: string
  name?: string
  description?: string
}

export interface CopywritingErrorLike {
  status?: number
  message?: string
}

const MAX_KEYWORDS_LENGTH = 100
const MAX_COPYWRITING_IMAGES = 6

export function getValidCopywritingImages(images: string[]): string[] {
  return images
    .map((image) => image.trim())
    .filter((image) => image.startsWith('http://') || image.startsWith('https://'))
    .slice(0, MAX_COPYWRITING_IMAGES)
}

export function buildCopywritingKeywords(input: GoodsEditAiKeywordInput): string {
  const parts: string[] = []

  if (input.category?.trim()) {
    parts.push(`类目：${input.category.trim()}`)
  }
  if (input.brand?.trim()) {
    parts.push(`品牌：${input.brand.trim()}`)
  }
  if (input.name?.trim()) {
    parts.push(`现有标题：${input.name.trim()}`)
  }
  if (input.description?.trim()) {
    parts.push(`补充描述：${input.description.trim()}`)
  }

  return parts.join('；').slice(0, MAX_KEYWORDS_LENGTH)
}

export function formatAiDescription(description: string, sellingPoints: string[]): string {
  const cleanDescription = description.trim()
  const points = sellingPoints.map((point) => point.trim()).filter(Boolean)

  if (points.length === 0) {
    return cleanDescription
  }

  return `${cleanDescription}\n\n核心卖点：\n${points.map((point) => `- ${point}`).join('\n')}`
}

export function getCopywritingErrorMessage(error: CopywritingErrorLike): string {
  switch (error.status) {
    case 400:
      return '图片或关键词不符合要求，请检查后重试'
    case 403:
      return '当前账号没有使用 AI 文案的权限'
    case 429:
      return 'AI 使用过于频繁，请稍后再试'
    case 503:
      return 'AI 服务未配置，请联系管理员注入 ARK_API_KEY 后重启服务'
    case 502:
      return 'AI 服务暂时不可用，请稍后重试或手动填写'
    case 504:
      return 'AI 生成超时，请换一张更稳定的公网图片或稍后重试'
    default:
      return error.message || 'AI 文案生成失败，请稍后重试或手动填写'
  }
}
