# Product Category Contract Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 打通商品类别主数据链路，让 Admin 创建/编辑/列表与 H5 首页分类都基于同一套 `categories` + `products.category_id` 契约工作。

**Architecture:** 后端以 `category_id` 作为商品唯一分类字段；商品读接口补充 `category_name` 仅用于展示；Admin 不再维护硬编码 `category` 字符串，而是读取 `/categories` 并提交 `category_id`。H5 首页继续使用 `/categories` 作为 tab 来源，不改入口设计，只依赖修复后的主数据与商品分类关系。

**Tech Stack:** Go, Hertz, GORM, React, TypeScript, Vite

---

## File Map

**Backend**
- Modify: `backend/product/service/product.go`
- Modify: `backend/product/handler/product.go`
- Modify: `backend/product/dao/product.go`
- Modify: `backend/product/model/product.go`
- Add or Modify: `backend/product/service/product_test.go`
- Add or Modify: `backend/product/handler/product_test.go`

**Admin Frontend**
- Modify: `frontend/admin/src/shared/api/types.ts`
- Modify: `frontend/admin/src/shared/api/index.ts`
- Modify: `frontend/admin/src/shared/api/product.ts`
- Modify: `frontend/admin/src/pages-new/GoodsEdit.tsx`
- Modify: `frontend/admin/src/pages-new/GoodsList.tsx`

**H5 Frontend**
- No code change required for tab architecture
- Optional verification only: `frontend/h5/src/pages/Home/index.tsx`

## Task 1: 对齐后端商品分类契约

**Files:**
- Modify: `backend/product/service/product.go`
- Modify: `backend/product/handler/product.go`
- Modify: `backend/product/dao/product.go`
- Modify: `backend/product/model/product.go`

- [ ] 在 `CreateProductRequest` 增加 `CategoryID *int64 \`json:"category_id"\``。
- [ ] 在 `UpdateProductRequest` 增加 `CategoryID *int64 \`json:"category_id"\``，支持编辑时变更分类。
- [ ] 在 `CreateProduct` 中把 `req.CategoryID` 写入 `model.Product.CategoryID`。
- [ ] 在 `UpdateProduct` 中当请求显式包含 `category_id` 时更新 `product.CategoryID`。
- [ ] 在 `CreateProduct` / `UpdateProduct` 增加分类存在性校验：`category_id != nil` 时必须能在 `categories` 表查到对应记录，且 `status=active`。
- [ ] 商品详情与列表返回补充展示字段 `category_name`；不要把后端主字段改回字符串 `category`，只新增展示字段，避免再次漂移。
- [ ] 最短实现方式：在 DAO 层为商品列表/详情增加 `LEFT JOIN categories`，扫描到专用响应结构；若当前 DAO 不便改动，允许在 service 层二次查询分类并组装。

**Acceptance**
- `POST /api/v1/products` 接收并保存 `category_id`
- `PUT /api/v1/products/:id` 接收并更新 `category_id`
- `GET /api/v1/products/:id` 返回 `category_id` 与 `category_name`
- `GET /api/v1/products` 返回列表项 `category_id` 与 `category_name`
- 非法 `category_id` 返回 400，而不是静默丢弃

## Task 2: 对齐 Admin TS 类型与 API 封装

**Files:**
- Modify: `frontend/admin/src/shared/api/types.ts`
- Modify: `frontend/admin/src/shared/api/index.ts`
- Modify: `frontend/admin/src/shared/api/product.ts`

- [ ] 在 `Product` 类型中删除 `category?: string` 的写入语义，新增：
  - `category_id?: number | null`
  - `category_name?: string`
- [ ] 新增 `Category` 类型，至少包含：
  - `id: number`
  - `name: string`
  - `code: string`
  - `status?: number`
- [ ] `productApi.create` 请求体改为 `{ name; description; images; category_id? }`
- [ ] `productApi.update` 请求体改为 `Partial<{ name; description; images; category_id? }>`
- [ ] 在 Admin API 入口补一个 `listCategories()`，调用 `GET /categories`

**Acceptance**
- Admin 侧不再提交 `category` 字符串
- Admin 侧能读取后端返回的 `category_name`
- 类别下拉数据源从硬编码切到 `/categories`

## Task 3: 修复 Admin 商品创建/编辑页

**Files:**
- Modify: `frontend/admin/src/pages-new/GoodsEdit.tsx`

- [ ] 将本地表单从 `category: string` 改为 `category_id?: number | null`
- [ ] 页面初始化时请求 `productApi.listCategories()`，保存分类列表状态
- [ ] 编辑态读取详情时，使用 `data.category_id` 回填，而不是 `data.category`
- [ ] 下拉框 `<option>` 来源改为真实分类列表，不再硬编码“艺术收藏/珠宝名表/...”
- [ ] 提交 create/update 时发送 `category_id`
- [ ] 如果业务要求“创建商品必须选分类”，则在提交前对 `category_id` 做前端非空校验；如果允许未分类，则 UI 必须提供“请选择分类”占位并与后端规则一致。当前最短路径建议：保持必选，并在前后端都校验。

**Acceptance**
- 新建商品后数据库能写入 `products.category_id`
- 编辑商品后类别可持久化更新
- 下拉选项与 `/categories` 返回一致

## Task 4: 修复 Admin 商品列表页

**Files:**
- Modify: `frontend/admin/src/pages-new/GoodsList.tsx`

- [ ] 列表展示改为 `item.category_name || '未分类'`
- [ ] 若后端仍未返回 `category_name`，不要继续依赖 `item.category`
- [ ] 保持“未分类”仅作为历史脏数据或空分类兜底，而不是常态路径

**Acceptance**
- 已绑定分类的商品显示真实类别名
- 仅 `category_id` 为空的历史商品显示“未分类”

## Task 5: 验证 H5 首页分类闭环

**Files:**
- Verify: `frontend/h5/src/pages/Home/index.tsx`

- [ ] 不改 `SPECIAL_TABS = ['全部', '收藏']` 设计
- [ ] 验证 `/api/v1/categories` 返回的分类能继续进入首页动态 tab
- [ ] 验证点击动态 tab 时，`auctionApi.list` 继续传 `category_id`
- [ ] 明确 H5 首页无需读取商品字符串类别，因此本任务只做联调验证，不做代码改造

**Acceptance**
- 首页始终显示“全部”“收藏”
- `/categories` 非空时能显示动态分类 tab
- 某商品绑定分类后，相关竞拍能在对应 tab 下被筛到

## Task 6: 历史数据修复

**Files:**
- Add: `scripts/backfill_product_category.sql` 或一次性运维脚本

- [ ] 查询历史空分类商品：`SELECT id, name, created_at FROM products WHERE category_id IS NULL;`
- [ ] 按业务规则回填历史商品分类；若无法自动判断，至少先产出待人工处理清单
- [ ] 对于确定无法映射的商品，允许继续保留空分类，但必须接受它们在 Admin 中显示“未分类”，且不会出现在 H5 动态分类过滤结果中

**Acceptance**
- 新增脏数据不再继续产生
- 历史脏数据规模可量化、可治理

## Test Plan

- Backend 单测
  - 创建商品时传合法 `category_id`，断言写入成功
  - 创建商品时传不存在 `category_id`，断言返回 400/错误
  - 更新商品时修改 `category_id`，断言变更成功
  - 商品详情/列表返回 `category_name`
- Admin 联调
  - 创建页下拉加载真实分类
  - 新建后列表展示正确类别名
  - 编辑类别后刷新列表仍正确
- H5 联调
  - 首页分类 tab 显示动态分类
  - 点击某分类 tab 后只出现对应分类竞拍

## Minimal Rollout Order

1. 后端契约与校验
2. Admin TS 类型与 API 封装
3. Admin 创建/编辑页
4. Admin 列表页
5. H5 联调验证
6. 历史数据回填

## Not In Scope

- 不改 H5 首页 tab 交互样式
- 不新增“未分类”首页 tab
- 不做类别管理页重构
- 不做商品搜索/筛选能力扩展
