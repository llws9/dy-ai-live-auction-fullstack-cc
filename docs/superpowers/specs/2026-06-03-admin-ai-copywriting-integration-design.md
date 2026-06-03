# Admin AI 一键文案接入设计（C3 前端闭环）

> 创建日期：2026-06-03
> 关联后端 Spec：`docs/superpowers/specs/2026-06-01-ai-copywriting-mvp-design.md`
> 方案选择：A「最小闭环接入」
> 目标页面：`frontend/admin/src/pages-new/GoodsEdit.tsx`

## 1. 背景

C3 AI 一键文案后端能力已经完成并验证：

- Gateway 已暴露 `POST /api/v1/products/ai/copywriting`。
- Gateway 会鉴权并透传 `X-User-ID` / `X-User-Role`。
- `product-service` 会调用 `shared/llm` 的 DoubaoProvider。
- 正式 Nacos 使用 `model=doubao-seed-1-6-vision-250815`，`timeout_ms=60000`。
- ECS demo 已验证真实 Ark 请求成功。

当前缺口在 Admin 前端：商家在创建商品时还不能通过 UI 触发 AI 文案，也不能把 AI 草稿预填到 `GoodsEdit` 表单。

## 2. 目标

本期目标是打通从「商品图片」到「AI 草稿预填」的最短前端闭环：

1. 在 `GoodsEdit` 的创建商品表单中新增「AI 一键文案」按钮。
2. 使用当前表单的图片 URL 和关键词信息调用 `POST /api/v1/products/ai/copywriting`。
3. 将返回的 `name` / `description` 预填到现有表单字段。
4. 将 `selling_points` 和 `suggested_start_price` 作为 AI 建议展示给商家。
5. 保留商家人工确认，AI 不自动创建商品、不自动发布、不自动配置竞拍规则。

## 3. 非目标

本期不做以下内容：

- 不新增后端接口。
- 不修改 AI Prompt 或 Ark 模型配置。
- 不新增商品数据库字段。
- 不把 `selling_points` 独立保存为商品字段。
- 不自动创建 `auction_rules`。
- 不自动把 `suggested_start_price` 写入竞拍规则。
- 不引入图片上传能力；继续使用现有图片 URL 输入方式。
- 不改造 Admin 类目体系为后端 `category_id` 选择器。

## 4. 用户故事

作为商家，我希望在发布新商品时：

1. 先添加至少一张公网可访问的商品图片 URL。
2. 点击「AI 一键文案」。
3. 等待 AI 生成标题、描述、卖点和建议起拍价。
4. 页面自动把标题和描述填入表单。
5. 我可以继续编辑 AI 草稿。
6. 最后由我决定保存草稿或保存并发布。

## 5. 页面设计

### 5.1 按钮位置

在 `GoodsEdit` 的「基本信息」卡片标题区右侧新增按钮：

- 文案：`AI 一键文案`
- 样式：沿用 Admin 现有 Button 体系，优先使用 amber 主色，避免引入新视觉体系。
- 图标：可使用现有 `lucide-react` 图标，例如 `Sparkles`；如不新增图标也可只显示文字。

推荐结构：

```text
基本信息                               [AI 一键文案]
设置商品的名称、类别和描述
```

### 5.2 AI 建议展示区

在右侧栏增加轻量「AI 建议」卡片，或复用发布状态上方空间展示：

- `selling_points`：以短标签或逗号列表展示。
- `suggested_start_price`：展示为「AI 建议起拍价：¥xxx」。
- 说明文案：`AI 仅生成草稿，请确认后再保存或发布。`

如果实现阶段为了最小改动不新增卡片，也可以在基本信息卡片下方显示提示条，但建议保留 `suggested_start_price` 可见，否则后端返回值无法被用户感知。

### 5.3 表单预填规则

AI 返回后按以下规则写入页面状态：

| AI 字段 | 前端处理 |
|---|---|
| `name` | 覆盖 `formData.name` |
| `description` | 覆盖 `formData.description`，并可在末尾追加卖点列表 |
| `selling_points` | 存入本地 `aiDraft.sellingPoints`，展示在 AI 建议区 |
| `suggested_start_price` | 存入本地 `aiDraft.suggestedStartPrice`，展示在 AI 建议区 |

描述字段推荐格式：

```text
{description}

核心卖点：
- {selling_points[0]}
- {selling_points[1]}
- {selling_points[2]}
```

原因：当前商品模型没有独立 `selling_points` 字段；把卖点合入描述能让 H5 用户在商品详情或直播商品卡中直接看到 AI 产物。

## 6. 数据流

### 6.1 前端调用链

```text
GoodsEdit
  -> productApi.generateCopywriting()
  -> shared/api/request.post()
  -> /api/v1/products/ai/copywriting
  -> Gateway
  -> product-service
  -> Ark
```

前端必须继续走 Gateway 的 `/api/v1` 入口，不允许直连 `product-service`。

### 6.2 请求体

后端契约：

```ts
interface CopywritingRequest {
  images: string[]
  category_id?: number
  keywords?: string
}
```

本期 Admin 当前没有后端类目 ID，因此不传 `category_id`。

本期请求体：

```ts
{
  images: formData.images.slice(0, 6),
  keywords: buildKeywords(formData)
}
```

`buildKeywords(formData)` 规则：

- 包含 `category`，例如 `类目：艺术收藏`。
- 包含 `brand`，例如 `品牌：Canon`。
- 如果 `name` 已有输入，包含 `现有标题：...`。
- 如果 `description` 已有输入，截断后包含 `补充描述：...`。
- 总长度控制在 100 个字符以内，避免触发后端 `keywords` 限制。

### 6.3 响应体

```ts
interface CopywritingResponse {
  name: string
  description: string
  selling_points: string[]
  suggested_start_price: string
}
```

### 6.4 API 封装

在 `frontend/admin/src/shared/api/product.ts` 新增：

```ts
export interface CopywritingGenerateData {
  images: string[]
  category_id?: number
  keywords?: string
}

export interface CopywritingDraft {
  name: string
  description: string
  selling_points: string[]
  suggested_start_price: string
}

generateCopywriting: (data: CopywritingGenerateData) =>
  post<CopywritingDraft>('/products/ai/copywriting', data, { timeout: 70000 })
```

超时设置为 `70000ms`，原因是线上 Ark vision 请求实测可能超过 30 秒，后端超时为 60 秒，前端需要略大于后端。

## 7. 状态设计

`GoodsEdit` 新增本地状态：

```ts
const [aiGenerating, setAiGenerating] = React.useState(false)
const [aiDraft, setAiDraft] = React.useState<{
  sellingPoints: string[]
  suggestedStartPrice: string
  appliedAt?: string
} | null>(null)
```

按钮可用条件：

- `saving === false`
- `aiGenerating === false`
- `formData.images.length > 0`
- `formData.images` 至少包含一个 `http://` 或 `https://` URL

按钮禁用提示：

- 没有图片：`请先添加至少一张图片 URL`
- 正在生成：`AI 生成中...`

## 8. 错误处理

### 8.1 输入错误

前端在调用前做最小校验：

- 图片为空：提示 `请先添加至少一张商品图片`
- 图片 URL 非 `http/https`：提示 `图片 URL 必须以 http:// 或 https:// 开头`
- 图片超过 6 张：只发送前 6 张，并提示 `最多使用前 6 张图片生成文案`

### 8.2 API 错误

使用现有 `ApiError` 和全局 toast 机制；页面内不吞掉错误。

错误文案建议：

| HTTP | 后端 code | 前端提示 |
|---|---|---|
| 400 | `invalid_request` | `图片或关键词不符合要求，请检查后重试` |
| 401 | `unauthorized` | 复用现有登录过期逻辑 |
| 403 | `forbidden_role` | `当前账号没有使用 AI 文案的权限` |
| 429 | `rate_limited` | `AI 使用过于频繁，请稍后再试` |
| 502 | `upstream_failed` / `upstream_invalid_output` | `AI 服务暂时不可用，请稍后重试或手动填写` |
| 504 | `upstream_timeout` | `AI 生成超时，请换一张更稳定的公网图片或稍后重试` |

### 8.3 失败不覆盖原则

AI 调用失败时：

- 不清空 `formData`。
- 不覆盖用户已输入内容。
- 不阻塞「保存为草稿」或「保存并发布」。
- `aiDraft` 保持上一次成功结果，或在 UI 上标注本次失败。

## 9. 权限与安全

- 前端不处理 `X-User-ID` / `X-User-Role`，只携带现有 JWT。
- Gateway 负责鉴权、角色校验和身份透传。
- 前端不接触 `ARK_API_KEY`。
- 错误提示不展示上游 Ark 原始错误全文，避免泄露内部细节。
- 图片 URL 必须是用户主动提供或上传后得到的公网 URL。

## 10. 与创建/发布链路的关系

AI 一键文案只影响表单草稿，不改变保存语义：

- 点击 AI 按钮不会创建商品。
- 点击 AI 按钮不会发布商品。
- 保存仍走 `productApi.create()` 或 `productApi.update()`。
- 发布仍走 `productApi.publish()`。

当前 `suggested_start_price` 只展示，不自动创建 `auction_rules`。后续如要完整闭环到竞拍规则，可单独设计「AI 建议价应用到竞拍规则」功能。

## 11. 测试策略

### 11.1 单元测试

建议为 `GoodsEdit` 增加或扩展测试：

- 无图片时点击 AI 按钮不发请求，并提示用户先添加图片。
- 有图片时点击 AI 按钮调用 `productApi.generateCopywriting`。
- 成功返回后，`name` 和 `description` 被预填。
- `selling_points` 和 `suggested_start_price` 在页面中可见。
- API 失败时不覆盖已有表单内容。
- `aiGenerating` 期间按钮禁用，避免重复请求。

### 11.2 API 封装测试

如当前 Admin 测试体系支持 API mock，覆盖：

- `generateCopywriting` 请求路径为 `/products/ai/copywriting`。
- 请求使用 `POST`。
- timeout 为 `70000ms`。

### 11.3 手动验收

本地或 demo 环境：

1. 使用商家或管理员账号登录 Admin。
2. 进入发布新商品页。
3. 添加公网图片 URL。
4. 点击「AI 一键文案」。
5. 验证名称、描述被预填。
6. 验证卖点和建议起拍价可见。
7. 修改文案后保存草稿。
8. 保存并发布后，在 H5 商品/直播入口确认用户可见文案。

## 12. 实施边界

本 spec 是前端接入 spec，建议实施只修改以下范围：

- `frontend/admin/src/pages-new/GoodsEdit.tsx`
- `frontend/admin/src/shared/api/product.ts`
- `frontend/admin/src/shared/api/types.ts`（如选择集中放类型）
- `frontend/admin/src/pages-new/__tests__/GoodsEdit.test.tsx` 或现有对应测试文件
- 必要时更新 mock handler

不应修改：

- 后端 AI copywriting service。
- Gateway 路由。
- Nacos LLM 配置。
- 商品数据库模型。
- H5 展示逻辑。

## 13. 验收标准

功能完成后必须满足：

- 商家在创建商品页能看到「AI 一键文案」按钮。
- 至少一张合法图片 URL 时按钮可用。
- 点击按钮后通过 Gateway 调用 `POST /api/v1/products/ai/copywriting`。
- 成功响应会预填商品名称和描述。
- 卖点和建议起拍价能被商家看到。
- AI 失败不影响手动填写、保存草稿、保存并发布。
- 前端没有任何 Ark key 或内部服务直连配置。
- 相关 Admin 测试通过。

## 14. 后续扩展

后续可以在独立 spec 中继续扩展：

- 后端类目 ID 与 Admin 类目选择器对齐。
- 将 `suggested_start_price` 一键应用到竞拍规则。
- 支持图片上传到 TOS 后自动使用返回 URL。
- AI 草稿对比视图：保留当前输入与 AI 建议，用户选择性应用字段。
- AI 文案生成历史与重试记录。
