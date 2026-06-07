import {
  buildCopywritingKeywords,
  formatAiDescription,
  getCopywritingErrorMessage,
  getValidCopywritingImages,
} from '../goodsEditAi'

describe('goodsEditAi helpers', () => {
  it('keeps only http/https images and limits to six images', () => {
    const images = [
      'https://cdn.example.com/1.jpg',
      'http://cdn.example.com/2.jpg',
      'ftp://cdn.example.com/3.jpg',
      '',
      '   https://cdn.example.com/4.jpg   ',
      'https://cdn.example.com/5.jpg',
      'https://cdn.example.com/6.jpg',
      'https://cdn.example.com/7.jpg',
      'https://cdn.example.com/8.jpg',
    ]

    expect(getValidCopywritingImages(images)).toEqual([
      'https://cdn.example.com/1.jpg',
      'http://cdn.example.com/2.jpg',
      'https://cdn.example.com/4.jpg',
      'https://cdn.example.com/5.jpg',
      'https://cdn.example.com/6.jpg',
      'https://cdn.example.com/7.jpg',
    ])
  })

  it('builds keywords from category brand name and description within 100 chars', () => {
    const keywords = buildCopywritingKeywords({
      category: '艺术收藏',
      brand: 'Canon',
      name: '复古相机',
      description: '九成新，自用一年，镜头干净，适合收藏和直播竞拍展示',
    })

    expect(keywords).toContain('类目：艺术收藏')
    expect(keywords).toContain('品牌：Canon')
    expect(keywords).toContain('现有标题：复古相机')
    expect(keywords.length).toBeLessThanOrEqual(100)
  })

  it('formats AI description with selling points appended', () => {
    expect(
      formatAiDescription('这是一台适合收藏的复古相机。', ['复古外观', '成色良好', '适合收藏'])
    ).toBe('这是一台适合收藏的复古相机。\n\n核心卖点：\n- 复古外观\n- 成色良好\n- 适合收藏')
  })

  it('returns original description when selling points are empty', () => {
    expect(formatAiDescription('只有描述。', [])).toBe('只有描述。')
  })

  it('maps known API status codes to user-safe messages', () => {
    expect(getCopywritingErrorMessage({ status: 429 })).toBe('AI 使用过于频繁，请稍后再试')
    expect(getCopywritingErrorMessage({ status: 503 })).toBe('AI 服务未配置，请联系管理员注入 ARK_API_KEY 后重启服务')
    expect(getCopywritingErrorMessage({ status: 504 })).toBe('AI 生成超时，请换一张更稳定的公网图片或稍后重试')
    expect(getCopywritingErrorMessage({ status: 502 })).toBe('AI 服务暂时不可用，请稍后重试或手动填写')
  })
})
