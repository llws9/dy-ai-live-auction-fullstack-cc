# Admin AI Copywriting Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 Admin `GoodsEdit` 创建/编辑商品页接入「AI 一键文案」按钮，调用已完成的后端 C3 AI 文案接口，并把返回草稿预填到现有商品表单。

**Architecture:** 前端保持最小闭环：`GoodsEdit` 负责 UI 状态和表单预填，`shared/api/product.ts` 与 `shared/api/index.ts` 暴露 `generateCopywriting`，纯函数 helper 负责关键词、图片和描述格式化。所有流量继续走 Gateway 的 `/api/v1/products/ai/copywriting`，不直连后端服务，不接触 `ARK_API_KEY`。

**Tech Stack:** React 18, TypeScript, Vite, Jest, Testing Library, existing Admin UI components, existing `shared/api/request.ts`.

---

## Spec

- Design: `docs/superpowers/specs/2026-06-03-admin-ai-copywriting-integration-design.md`
- Backend contract: `docs/superpowers/specs/2026-06-01-ai-copywriting-mvp-design.md`

## Scope Check

本计划只覆盖 Admin 前端接入，不改后端、Gateway、Nacos、数据库模型或 H5 页面。Spec 中的「竞拍规则建议价应用」「图片上传」「类目 ID 选择器」均为后续扩展，不进入本计划。

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `frontend/admin/src/pages-new/goodsEditAi.ts` | Create | 放置可独立测试的纯函数：合法图片过滤、关键词构造、AI 描述格式化、错误文案映射 |
| `frontend/admin/src/pages-new/__tests__/goodsEditAi.test.ts` | Create | 覆盖 helper 的边界行为 |
| `frontend/admin/src/shared/api/product.ts` | Modify | 增加 AI 文案请求/响应类型与 `productApi.generateCopywriting` |
| `frontend/admin/src/shared/api/index.ts` | Modify | 同步给当前 `GoodsEdit` 使用的聚合 `productApi` 增加 `generateCopywriting` |
| `frontend/admin/src/shared/api/__tests__/product.test.ts` | Create | 验证 API 方法使用正确 path、payload 和 70s timeout |
| `frontend/admin/src/pages-new/GoodsEdit.tsx` | Modify | 新增按钮、状态、AI 建议卡片、调用和预填逻辑 |
| `frontend/admin/src/pages-new/__tests__/GoodsEdit.ai.test.tsx` | Create | 覆盖页面级交互：按钮、成功预填、失败不覆盖、loading 防重入 |

---

## Task 1: AI Helper Pure Functions

**Files:**
- Create: `frontend/admin/src/pages-new/goodsEditAi.ts`
- Create: `frontend/admin/src/pages-new/__tests__/goodsEditAi.test.ts`

- [ ] **Step 1: Write failing helper tests**

Create `frontend/admin/src/pages-new/__tests__/goodsEditAi.test.ts`:

```ts
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
    ).toBe('这是一台适合收藏的复古相机。\\n\\n核心卖点：\\n- 复古外观\\n- 成色良好\\n- 适合收藏')
  })

  it('returns original description when selling points are empty', () => {
    expect(formatAiDescription('只有描述。', [])).toBe('只有描述。')
  })

  it('maps known API status codes to user-safe messages', () => {
    expect(getCopywritingErrorMessage({ status: 429 })).toBe('AI 使用过于频繁，请稍后再试')
    expect(getCopywritingErrorMessage({ status: 504 })).toBe('AI 生成超时，请换一张更稳定的公网图片或稍后重试')
    expect(getCopywritingErrorMessage({ status: 502 })).toBe('AI 服务暂时不可用，请稍后重试或手动填写')
  })
})
```

- [ ] **Step 2: Run helper tests to verify RED**

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath src/pages-new/__tests__/goodsEditAi.test.ts
```

Expected: FAIL because `../goodsEditAi` does not exist.

- [ ] **Step 3: Implement helper functions**

Create `frontend/admin/src/pages-new/goodsEditAi.ts`:

```ts
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
    case 502:
      return 'AI 服务暂时不可用，请稍后重试或手动填写'
    case 504:
      return 'AI 生成超时，请换一张更稳定的公网图片或稍后重试'
    default:
      return error.message || 'AI 文案生成失败，请稍后重试或手动填写'
  }
}
```

- [ ] **Step 4: Run helper tests to verify GREEN**

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath src/pages-new/__tests__/goodsEditAi.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit helper functions**

```bash
git add frontend/admin/src/pages-new/goodsEditAi.ts frontend/admin/src/pages-new/__tests__/goodsEditAi.test.ts
git commit -m "test(admin): add AI copywriting form helpers"
```

---

## Task 2: Product API Wrapper

**Files:**
- Modify: `frontend/admin/src/shared/api/product.ts`
- Modify: `frontend/admin/src/shared/api/index.ts`
- Create: `frontend/admin/src/shared/api/__tests__/product.test.ts`

- [ ] **Step 1: Write failing API wrapper test**

Create `frontend/admin/src/shared/api/__tests__/product.test.ts`:

```ts
import { post } from '../request'
import { productApi } from '../product'

jest.mock('../request', () => ({
  get: jest.fn(),
  post: jest.fn(),
  put: jest.fn(),
  del: jest.fn(),
  buildQuery: jest.fn(() => ''),
}))

describe('productApi.generateCopywriting', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('posts to the Gateway AI copywriting route with a 70s timeout', async () => {
    ;(post as jest.Mock).mockResolvedValue({
      name: 'AI 标题',
      description: 'AI 描述',
      selling_points: ['卖点一'],
      suggested_start_price: '199.00',
    })

    const payload = {
      images: ['https://cdn.example.com/product.jpg'],
      keywords: '类目：艺术收藏',
    }

    const result = await productApi.generateCopywriting(payload)

    expect(post).toHaveBeenCalledWith('/products/ai/copywriting', payload, { timeout: 70000 })
    expect(result.name).toBe('AI 标题')
  })
})
```

- [ ] **Step 2: Run API test to verify RED**

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath src/shared/api/__tests__/product.test.ts
```

Expected: FAIL because `productApi.generateCopywriting` is not defined.

- [ ] **Step 3: Add types and method to `shared/api/product.ts`**

Modify `frontend/admin/src/shared/api/product.ts` by adding these interfaces after `RuleCreateData`:

```ts
export interface CopywritingGenerateData {
  images: string[];
  category_id?: number;
  keywords?: string;
}

export interface CopywritingDraft {
  name: string;
  description: string;
  selling_points: string[];
  suggested_start_price: string;
}
```

Add this method inside `productApi` after `create`:

```ts
  // AI 一键文案
  generateCopywriting: (data: CopywritingGenerateData) =>
    post<CopywritingDraft>('/products/ai/copywriting', data, { timeout: 70000 }),
```

- [ ] **Step 4: Add matching method to `shared/api/index.ts`**

Modify `frontend/admin/src/shared/api/index.ts` by adding these types near the existing product API section:

```ts
export interface CopywritingGenerateData {
  images: string[];
  category_id?: number;
  keywords?: string;
}

export interface CopywritingDraft {
  name: string;
  description: string;
  selling_points: string[];
  suggested_start_price: string;
}
```

Add this method inside the inline `productApi` after `create`:

```ts
  generateCopywriting: (data: CopywritingGenerateData) =>
    post<CopywritingDraft>('/products/ai/copywriting', data, { timeout: 70000 }),
```

Rationale: `GoodsEdit.tsx` currently imports `productApi` from `@/shared/api`, so the aggregator must expose the new method. Do not refactor the whole API aggregation layer in this task.

- [ ] **Step 5: Run API test to verify GREEN**

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath src/shared/api/__tests__/product.test.ts
```

Expected: PASS.

- [ ] **Step 6: Commit API wrapper**

```bash
git add frontend/admin/src/shared/api/product.ts frontend/admin/src/shared/api/index.ts frontend/admin/src/shared/api/__tests__/product.test.ts
git commit -m "feat(admin): add AI copywriting product API"
```

---

## Task 3: GoodsEdit UI Integration

**Files:**
- Modify: `frontend/admin/src/pages-new/GoodsEdit.tsx`
- Create: `frontend/admin/src/pages-new/__tests__/GoodsEdit.ai.test.tsx`

- [ ] **Step 1: Write failing page interaction tests**

Create `frontend/admin/src/pages-new/__tests__/GoodsEdit.ai.test.tsx`:

```tsx
import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import GoodsEdit from '../GoodsEdit'
import { productApi } from '@/shared/api'

jest.mock('@/shared/api', () => ({
  productApi: {
    get: jest.fn(),
    create: jest.fn(),
    update: jest.fn(),
    publish: jest.fn(),
    generateCopywriting: jest.fn(),
  },
}))

const mockNavigate = jest.fn()

jest.mock('react-router-dom', () => {
  const actual = jest.requireActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

function renderGoodsEdit() {
  return render(
    <MemoryRouter initialEntries={['/goods/edit']}>
      <GoodsEdit />
    </MemoryRouter>
  )
}

describe('GoodsEdit AI copywriting integration', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    window.alert = jest.fn()
  })

  it('does not call AI copywriting without a valid image URL', () => {
    renderGoodsEdit()

    fireEvent.click(screen.getByRole('button', { name: /AI 一键文案/ }))

    expect(productApi.generateCopywriting).not.toHaveBeenCalled()
    expect(window.alert).toHaveBeenCalledWith('请先添加至少一张商品图片')
  })

  it('generates AI copywriting and applies the draft to the form', async () => {
    ;(productApi.generateCopywriting as jest.Mock).mockResolvedValue({
      name: 'AI 复古相机',
      description: '这是一台适合直播竞拍的复古相机。',
      selling_points: ['复古外观', '成色良好', '适合收藏'],
      suggested_start_price: '199.00',
    })

    renderGoodsEdit()

    fireEvent.change(screen.getByPlaceholderText('输入图片URL'), {
      target: { value: 'https://cdn.example.com/camera.jpg' },
    })
    fireEvent.click(screen.getByRole('button', { name: '' }))
    fireEvent.click(screen.getByRole('button', { name: /AI 一键文案/ }))

    await waitFor(() => {
      expect(productApi.generateCopywriting).toHaveBeenCalledWith({
        images: ['https://cdn.example.com/camera.jpg'],
        keywords: expect.stringContaining('类目：艺术收藏'),
      })
    })

    expect(screen.getByDisplayValue('AI 复古相机')).toBeInTheDocument()
    expect(screen.getByDisplayValue(/这是一台适合直播竞拍的复古相机/)).toBeInTheDocument()
    expect(screen.getByText('复古外观')).toBeInTheDocument()
    expect(screen.getByText(/AI 建议起拍价：¥199.00/)).toBeInTheDocument()
    expect(screen.getByText('AI 仅生成草稿，请确认后再保存或发布。')).toBeInTheDocument()
  })

  it('does not overwrite current form values when AI generation fails', async () => {
    ;(productApi.generateCopywriting as jest.Mock).mockRejectedValue({ status: 504 })

    renderGoodsEdit()

    fireEvent.change(screen.getByPlaceholderText('输入商品完整名称'), {
      target: { value: '用户手写标题' },
    })
    fireEvent.change(screen.getByPlaceholderText('详细介绍商品的来源、年代、成色等信息...'), {
      target: { value: '用户手写描述' },
    })
    fireEvent.change(screen.getByPlaceholderText('输入图片URL'), {
      target: { value: 'https://cdn.example.com/item.jpg' },
    })
    fireEvent.click(screen.getByRole('button', { name: '' }))
    fireEvent.click(screen.getByRole('button', { name: /AI 一键文案/ }))

    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith('AI 生成超时，请换一张更稳定的公网图片或稍后重试')
    })

    expect(screen.getByDisplayValue('用户手写标题')).toBeInTheDocument()
    expect(screen.getByDisplayValue('用户手写描述')).toBeInTheDocument()
  })
})
```

Note: If `screen.getByRole('button', { name: '' })` becomes ambiguous because icon-only buttons share empty names, update the implementation in Step 3 to give the add-image button `aria-label="添加图片 URL"` and update these tests to use `screen.getByRole('button', { name: '添加图片 URL' })`.

- [ ] **Step 2: Run page tests to verify RED**

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath src/pages-new/__tests__/GoodsEdit.ai.test.tsx
```

Expected: FAIL because the AI button and integration do not exist.

- [ ] **Step 3: Update `GoodsEdit.tsx` imports**

Modify the import line in `frontend/admin/src/pages-new/GoodsEdit.tsx`:

```tsx
import { ArrowLeft, Save, Plus, X, Sparkles } from "lucide-react"
```

Add helper imports below the `productApi` import:

```tsx
import {
  buildCopywritingKeywords,
  formatAiDescription,
  getCopywritingErrorMessage,
  getValidCopywritingImages,
} from "./goodsEditAi"
```

- [ ] **Step 4: Add AI state to `GoodsEdit.tsx`**

Add this state after `saving`:

```tsx
  const [aiGenerating, setAiGenerating] = React.useState(false)
  const [aiDraft, setAiDraft] = React.useState<{
    sellingPoints: string[]
    suggestedStartPrice: string
    appliedAt?: string
  } | null>(null)
```

- [ ] **Step 5: Add AI generation handler**

Add this function before `handleSubmit`:

```tsx
  // AI 一键文案：只生成草稿并预填表单，不自动保存或发布
  const handleGenerateCopywriting = async () => {
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
        keywords: buildCopywritingKeywords(formData),
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
```

- [ ] **Step 6: Add the AI button in the Basic Info card header**

Replace the current basic card header:

```tsx
            <CardHeader>
              <CardTitle className="text-lg">基本信息</CardTitle>
              <CardDescription>设置商品的名称、类别和描述</CardDescription>
            </CardHeader>
```

with:

```tsx
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
```

- [ ] **Step 7: Make the add-image button accessible**

Update the add-image button in `GoodsEdit.tsx` by adding an aria-label:

```tsx
                  <Button
                    type="button"
                    aria-label="添加图片 URL"
                    variant="outline"
                    className="border-slate-200"
                    onClick={addImage}
                  >
                    <Plus className="w-4 h-4" />
                  </Button>
```

Then update the page tests from:

```tsx
fireEvent.click(screen.getByRole('button', { name: '' }))
```

to:

```tsx
fireEvent.click(screen.getByRole('button', { name: '添加图片 URL' }))
```

- [ ] **Step 8: Add AI suggestion card**

Insert this card above the existing「发布状态」card:

```tsx
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
```

- [ ] **Step 9: Run page tests to verify GREEN**

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath src/pages-new/__tests__/GoodsEdit.ai.test.tsx
```

Expected: PASS.

- [ ] **Step 10: Commit GoodsEdit UI integration**

```bash
git add frontend/admin/src/pages-new/GoodsEdit.tsx frontend/admin/src/pages-new/__tests__/GoodsEdit.ai.test.tsx
git commit -m "feat(admin): wire AI copywriting into GoodsEdit"
```

---

## Task 4: Regression, Build, and Manual Verification Notes

**Files:**
- Modify: `frontend/admin/src/mocks/handlers.ts`

- [ ] **Step 1: Run focused Admin tests**

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath \
  src/pages-new/__tests__/goodsEditAi.test.ts \
  src/shared/api/__tests__/product.test.ts \
  src/pages-new/__tests__/GoodsEdit.ai.test.tsx
```

Expected: PASS.

- [ ] **Step 2: Run Admin build**

Run:

```bash
cd frontend/admin
npm run build
```

Expected: PASS. If TypeScript fails because `productApi.generateCopywriting` is missing from the aggregator type, confirm Task 2 Step 4 was applied to `frontend/admin/src/shared/api/index.ts`.

- [ ] **Step 3: Run broader Admin tests if runtime is acceptable**

Run:

```bash
cd frontend/admin
npm test -- --runInBand
```

Expected: PASS. If existing unrelated tests fail, record the failing test names and confirm the focused tests from Step 1 still pass.

- [ ] **Step 4: Add MSW mock route for local dev**

Add this handler near product handlers in `frontend/admin/src/mocks/handlers.ts`:

```ts
  http.post('/api/v1/products/ai/copywriting', async () => {
    await delay(300)
    return HttpResponse.json({
      code: 0,
      message: 'success',
      data: {
        name: 'AI 复古相机',
        description: '这是一台适合直播竞拍的复古相机，外观经典，成色良好，适合收藏与日常拍摄使用。',
        selling_points: ['复古外观', '成色良好', '适合收藏'],
        suggested_start_price: '199.00',
      },
    })
  }),
```

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath src/pages-new/__tests__/GoodsEdit.ai.test.tsx
```

Expected: PASS.

- [ ] **Step 5: Manual browser verification**

Run:

```bash
cd frontend/admin
npm run dev
```

Open the Admin dev URL, log in with a merchant/admin account, and verify:

```text
1. Open /goods/edit.
2. Add https://ark-project.tos-cn-beijing.volces.com/images/view.jpeg as image URL.
3. Click 添加图片 URL.
4. Click AI 一键文案.
5. Confirm 商品名称 and 详细描述 are filled.
6. Confirm AI 建议 card shows selling points and suggested start price.
7. Edit the generated text manually.
8. Click 保存为草稿 and confirm it still uses productApi.create.
```

- [ ] **Step 6: Commit MSW mock route**

```bash
git add frontend/admin/src/mocks/handlers.ts
git commit -m "test(admin): mock AI copywriting endpoint"
```

---

## Final Verification

Before declaring implementation complete, run:

```bash
cd frontend/admin
npm test -- --runTestsByPath \
  src/pages-new/__tests__/goodsEditAi.test.ts \
  src/shared/api/__tests__/product.test.ts \
  src/pages-new/__tests__/GoodsEdit.ai.test.tsx
npm run build
```

Expected:

```text
PASS src/pages-new/__tests__/goodsEditAi.test.ts
PASS src/shared/api/__tests__/product.test.ts
PASS src/pages-new/__tests__/GoodsEdit.ai.test.tsx
build completed without TypeScript errors
```

## Spec Coverage Checklist

- 「AI 一键文案」按钮：Task 3 Steps 6 and 9.
- Gateway API call: Task 2 Steps 3-5.
- `name` / `description` prefill: Task 3 Steps 5 and 9.
- `selling_points` / `suggested_start_price` visible: Task 3 Step 8.
- Failure does not overwrite user input: Task 3 Step 1 test.
- No backend/Gateway/Nacos changes: File Structure and Task scope.
- 70s frontend timeout: Task 2 tests and implementation.
- At least one valid http/https image and max six images: Task 1 helper tests and Task 3 handler.
- Manual save/publish semantics unchanged: Task 3 only calls AI handler from new button; `handleSubmit` is not changed.
