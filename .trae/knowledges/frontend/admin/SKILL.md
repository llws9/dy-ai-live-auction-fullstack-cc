---
name: knowledge-frontend-admin
description: >
  Covers Admin 管理后台的权限、API 封装、页面组织、实验配置、编码修复和测试约束。
  Navigate when: modifying frontend/admin pages, adding merchant/admin features, changing admin API calls, debugging role access, GrowthBook, orders, auctions, statistics, or live stream management.
  Excludes: H5 用户端和 Test Dashboard；Test Dashboard context is in ../test-dashboard/SKILL.md.
  Keywords: frontend/admin, Admin, RequireRole, RoleRoute, isAllowedRole, request.ts, /admin/orders, GrowthBook, decodePossibleMojibake, normalizeAuctionText, userEncoding
---

## Module Structure

Admin 是管理后台前端，面向商家和管理员角色；核心风险集中在角色权限、管理端 API 路径、响应归一化、实验配置隔离和生产构建路径。

### Directory Layout
- `frontend/admin/src/App.tsx` — Admin 路由和角色路由入口。
- `frontend/admin/src/components/Layout.tsx` — 后台布局与动态菜单。
- `frontend/admin/src/shared/auth/` — 登录态、角色判断和鉴权上下文。
- `frontend/admin/src/shared/api/` — 管理后台 API 封装、请求基础设施、类型定义和编码归一化。
- `frontend/admin/src/pages-new/` — 主要新版页面目录。
- `frontend/admin/src/pages/` — 存量页面目录，部分页面仍在这里维护。
- `frontend/admin/e2e/` — Playwright 管理后台 E2E 测试。
- `frontend/admin/nginx/`、`frontend/admin/Dockerfile` — 管理后台容器和静态服务配置。

### Key Entry Points
- `frontend/admin/src/shared/auth/roles.ts` — 角色权限判定工具。
- `frontend/admin/src/shared/api/request.ts` — Admin 统一请求封装。
- `frontend/admin/src/shared/api/index.ts` — 业务 API 聚合入口和响应归一化。
- `frontend/admin/src/shared/growthbook/GrowthBookContextProvider.tsx` — GrowthBook Provider。
- `frontend/admin/src/pages/LiveStreamFixedPrice/index.tsx` — 直播间一口价上下架和冲突处理。

## Gotchas
- 商家和管理员菜单与页面权限由 `RequireRole`、`RoleRoute`、`isAllowedRole` 多处共同约束，新增页面时必须同步路由守卫和菜单可见性，否则会出现可见但不可访问或可访问但不可见的错位（`frontend/admin/src/App.tsx`, `frontend/admin/src/components/Layout.tsx`, `frontend/admin/src/shared/auth/roles.ts`）
- Admin 端订单列表必须使用 `/admin/orders`，不能复用用户端 `/orders`，否则会被 `X-User-ID` 语义过滤成当前用户订单而不是管理视角订单（`frontend/admin/src/shared/api/index.ts`）
- `request.ts` 会在 401 时清理 token 并跳转登录页，新增静默探测类接口时需要显式控制错误展示策略，避免后台页面被非关键请求打断（`frontend/admin/src/shared/api/request.ts`）
- GrowthBook 必须在组件级用 `useMemo` 创建实例，模块级单例会导致属性在用户/环境之间泄漏，影响实验判断（`frontend/admin/src/shared/growthbook/GrowthBookContextProvider.tsx`）
- 后端返回的用户名、竞拍和出价文本可能存在 UTF-8 被误解析为 Windows-1252 的乱码，渲染前需走既有修复函数而不是在页面局部手写替换（`frontend/admin/src/shared/auth/AuthContext.tsx`, `frontend/admin/src/shared/api/auctionEncoding.ts`, `frontend/admin/src/shared/api/bidEncoding.ts`, `frontend/admin/src/shared/api/userEncoding.ts`）
- 一口价商品若已被其他竞拍绑定，后端返回 409；前端需捕获后刷新可售商品列表，不能只弹错误后保留旧选项（`frontend/admin/src/pages/LiveStreamFixedPrice/index.tsx`）
- 封禁状态直播间必须在 UI 层禁用开启直播动作，不能只依赖后端拒绝，否则演示时会暴露可点击但失败的操作路径（`frontend/admin/src/pages-new/LiveDetail.tsx`）
- **直播间开播入口位置**：一期将开播入口从 Dashboard 的 `window.prompt` 手输 ID 迁移到直播间详情页，商家在详情页执行开播操作（`frontend/admin/src/pages-new/LiveDetail.tsx`）
- **开播按钮产品语义**：按钮文案保留为 `开始直播`，但需配合弱提示说明当前为演示开播状态，用于跑通 H5 观看/竞拍/一口价链路；二期接入移动端主播页与真实推流后，开播状态将由推流成功回调触发（`frontend/admin/src/pages-new/LiveDetail.tsx`）
- **商家关闭直播权限**：一期商家**不可关闭**自己直播间，关闭直播接口 `/admin/live-streams/:id/end` 为 `RequireAdmin()` 仅管理员可调用；如需商家自关闭需二期新增路由（`backend/gateway/router/router.go:166`）
- **直播间状态枚举补全**：后端 `live_stream.go` 定义 `3=已封禁`，但前端 `types.ts:118` 注释漏写状态 3，需补全注释避免类型漂移（`backend/product/model/live_stream.go:9-12`, `frontend/admin/src/shared/api/types.ts:118`）
- **管理端详情接口必须使用 scoped 版本**：商家访问直播间详情应走 `/admin/live-streams/:id` 而非 `/live-streams/:id`，该接口带 owner scope 校验，非 owner 会在读取阶段 403，避免前端展示后再操作失败（`frontend/admin/src/pages-new/LiveDetail.tsx`, `backend/gateway/router/router.go`）
- **Admin 登录状态同步陷阱**：`Login.tsx` 登录成功后若只写 `localStorage` 而不调用 `AuthContext` 的 `login()` 方法，会导致内存状态未更新，`RequireAuth` 路由守卫仍认为未登录，用户会被重定向回登录页。正确做法是使用 `useAuth().login(response.token, response.user)` 同步更新内存状态
- **Admin Token Key 一致性**：`Login.tsx` 写入的 token key（如 `admin_auth_token`）必须与 `request.ts` 读取的 key 一致，否则后续 API 请求会 401。建议做兼容读取：`localStorage.getItem('admin_auth_token') || localStorage.getItem('token')`
- **角色权限未生效排查**：若登录后商家/管理员看到相同界面，检查三点：1) `Layout.tsx` 的 `navItems` 是否按角色动态过滤；2) `App.tsx` 路由是否使用 `RequireRole` 或 `RoleRoute` 进行角色拦截；3) 顶部身份文案是否硬编码为"管理员"而非根据 `role` 动态显示
- **统计页 Tab 与路由同步**：`Stats` 组件若使用 `defaultValue` 而非受控 `value`，路由切换后 Tab 激活状态不会同步更新。应使用受控 Tabs：当前路径决定激活 Tab，点击 Tab 时同步导航到对应路由
- **侧边栏菜单与路由权限一致性**：侧边栏菜单项的 `allowedRoles` 必须与对应页面的路由守卫（`RequireRole`/`RoleRoute`）保持一致，否则会出现菜单可见但页面 403，或菜单隐藏但可通过直接 URL 访问的错位
- **商品发布状态同步**：Admin 端编辑商品时保存的 `status` 字段必须与后端商品列表查询条件保持一致。若管理端保存为"未发布"状态但 H5 列表未过滤该字段，会导致未发布商品对用户可见。修复时需同时检查：1) Admin 保存接口字段值；2) H5 列表查询过滤条件；3) 两端状态枚举定义是否一致

## Architecture
- Admin API 层集中在 `shared/api/`，页面应通过封装函数访问后端，避免散落 axios/fetch 导致鉴权、错误提示和编码归一化不一致（`frontend/admin/src/shared/api/request.ts`, `frontend/admin/src/shared/api/index.ts`）
- 页面迁移处于 `pages-new/` 与 `pages/` 并存状态，修改导航或路由时必须确认目标页面实际位于哪个目录，不能假设所有页面都已迁移到新版目录（`frontend/admin/src/App.tsx`）
- 角色模型至少区分商家 `role=1` 与管理员 `role=2`，商家页面和管理员页面是同一 Admin 应用内的分支，而不是两个独立应用（`frontend/admin/src/App.tsx`, `frontend/admin/src/shared/auth/roles.ts`）

## Patterns
- 列表查询参数通过 `buildQuery` 过滤空值和 `undefined`，新增筛选器时应复用它，避免把空字符串或未定义参数发给后端造成过滤语义漂移（`frontend/admin/src/shared/api/request.ts`）
- 收入统计响应格式不稳定时通过 `normalizeRevenueStatsResponse` 兜底，页面不应直接假设单一响应结构（`frontend/admin/src/shared/api/index.ts`）
- 竞拍和出价列表进入 UI 前先归一化编码，后续组件应消费归一化后的对象而不是重复处理原始响应（`frontend/admin/src/shared/api/auctionEncoding.ts`, `frontend/admin/src/shared/api/bidEncoding.ts`, `frontend/admin/src/shared/api/userEncoding.ts`）

## Conventions
- Admin 登录页 `/admin-login` 是独立入口，不走后台 Layout；新增登录相关跳转时不要把它放入普通菜单体系（`frontend/admin/src/App.tsx`）
- 共享基础组件放在 `components/shared/`，Radix 封装 UI 组件放在 `components/ui/`；新增组件时按复用范围放置，避免页面私有组件污染共享目录（`frontend/admin/src/components/shared/`, `frontend/admin/src/components/ui/`）
- Admin 生产构建和部署路径独立于 H5，demo 发布时继续使用 `/admin/` 子路径构建（`deploy/demo/MAIN_DEPLOY_QUICKSTART.md`, `frontend/admin/package.json`）

## Testing Strategy
- API 模块测试使用 MSW 模拟服务端响应，新增 API 归一化逻辑时优先补充 `shared/api` 附近的单元测试而不是只测页面渲染（`frontend/admin/src/shared/api/__tests__/`, `frontend/admin/src/mocks/handlers.ts`）
- E2E 覆盖商品管理、竞拍管理、统计报表等核心后台流程，改动这些路径时需要评估是否更新 Playwright 用例（`frontend/admin/e2e/`）

## Feature Knowledge

### GrowthBook A/B 实验配置 (GrowthBook Experiment Configuration)

**功能概述**：Admin 和 H5 共享同一套 GrowthBook 实验平台，用于运行 A/B 实验（如直播开播弹窗可见性实验）。

**实验定义规范**：
- **实验 Key**：使用 kebab-case 命名，如 `live-start-popup-visibility`
- **变体设计**：至少包含 `control`（对照组）和 `treatment`（实验组）两个变体
- **分流比例**：本地开发环境通常配置 50/50 均分，便于快速验证

**本地初始化流程**：
1. 启动 GrowthBook 服务（`docker-compose` 包含）
2. 运行初始化脚本：`bash scripts/init-growthbook.sh`
3. 脚本会幂等地创建：管理员账号、SDK Connection (`dev-client-key`)、实验规则
4. 验证：`curl http://localhost:3200/api/features/dev-client-key | jq '.features["live-start-popup-visibility"]'`

**前端接入模式**：
```tsx
// GrowthBookContextProvider.tsx
const gb = new GrowthBook({
  clientKey: 'dev-client-key',  // 本地开发使用固定 key
  apiHost: 'http://localhost:3200',
});
await gb.loadFeatures();

// 组件内使用
const isVisible = gb.isOn('live-start-popup-visibility');
```

**常见陷阱**：
- **ERR_ABORTED 错误**：开发模式下 `React.StrictMode` 会触发组件重复挂载/清理，导致 GrowthBook SDK 的 `loadFeatures()` 请求被浏览器取消。这是正常行为，只要后续请求返回 200 即可忽略
- **SDK Payload 缓存**：GrowthBook 会缓存 public payload，新增实验后可能需要清除浏览器缓存才能看到新 feature
- **Client Key 一致性**：前端配置的 `clientKey` 必须与 GrowthBook 中创建的 SDK Connection key 完全匹配（区分大小写）

**对照组行为定义**：
- 对照组用户「看不到弹窗提醒」，但「消息中心仍有未读提醒」
- 弹窗和消息中心是两条独立链路，实验只控制弹窗展示，不影响消息生成

**来源**：session:6a229aad2ec60aa1a73a04da

### Admin 角色权限设计 (Admin Role-Based Access Control)

**设计背景**：Admin 管理后台需要同时服务「平台管理员」和「商家」两种角色，但平台管理员不具备代运营能力。

**角色边界**：
- **平台管理员**：看全局、做治理、看统计、管权限；不创建、不编辑、不上下架商家经营资产
- **商家**：看自己的经营数据，管理自己的商品、直播、竞拍、规则模板、一口价和订单

**页面可见性矩阵**（关键决策）：

| 页面 | 路由 | 平台管理员 | 商家 | 说明 |
|---|---|:---:|:---:|:---|
| 经营总览 | `/dashboard` | 可见 | 可见 | 数据 scope 不同 |
| 商品列表 | `/goods/list` | 可见 | 可见 | 管理员只读/治理视角 |
| 创建/编辑商品 | `/goods/create`, `/goods/edit` | 不可见 | 可见 | 平台不代运营 |
| 竞拍列表 | `/auction/list` | 可见 | 可见 | 商家仅限自己的竞拍 |
| 规则模板 | `/auction/rules` | 不可见 | 可见 | 商家自定义竞拍参数模板 |
| 直播间列表 | `/live/list` | 可见 | 可见 | 商家仅限自己的直播间 |
| 创建直播间 | `/live/create` | 不可见 | 可见 | 平台不代运营 |
| 一口价上下架 | `/live/fixed-price` | 不可见 | 可见 | 商家经营动作 |
| 订单列表/详情 | `/order/list`, `/order/detail` | 可见 | 可见 | 管理员看全局异常 |
| 用户统计 | `/stats/user` | 可见 | 不可见 | 平台用户洞察 |
| 角色/用户管理 | `/system/permission/*` | 可见 | 不可见 | 平台治理能力 |

**权限实现原则**：
1. **前端**：菜单动态过滤 + 路由 `RequireRole` 拦截 + 未授权访问显示 403 或重定向
2. **后端**：所有列表和详情接口必须按身份限制数据范围，商家只能访问 `owner_id == current_user_id` 的数据
3. **纵深防御**：即使前端隐藏菜单，后端仍需校验操作权限

**关键约束**：
- `/auction/rules` 定义为「商家自定义竞拍参数模板」，仅商家可见
- 平台管理员对商家经营资产默认只读，不写操作
- 详情页/编辑页/创建页不一定出现在菜单中，但必须配置路由权限

**来源**：session:6a2139882ec60aa1a7395093

### 商品 AI 文案生成 (Product AI Copywriting)

**功能概述**：Admin 端提供一键 AI 生成商品文案功能，商家输入商品基础信息后，后端调用 Doubao/Ark 大模型生成营销文案。

**技术架构**：
- **入口**：Admin 商品编辑页面 `frontend/admin/src/pages-new/GoodsEdit.tsx`
- **API**：`POST /api/v1/products/ai/copywriting`（经 Gateway 转发至 product-service）
- **后端实现**：`backend/product/handler/copywriting.go` + `backend/product/service/copywriting.go`
- **LLM 供应商**：`backend/shared/llm/` 抽象层，当前实现为 Doubao Provider

**关键约束**：
- **API 密钥管理**：生产环境 `ARK_API_KEY` 通过服务器环境文件（如 `/srv/auction/env/.env.demo`）配置，由 `product-service` 容器读取；Gateway 仅负责请求转发和鉴权，不直接调用 AI 接口
- **安全规范**：严禁将 API Key 提交至 Git 仓库或在对话中明文传输；更新密钥需修改服务器 `.env` 文件并重启对应服务
- **前端字段映射**：后端返回字段名为 `available_amount`（注意不是 `available` 或 `balance`），前端需正确读取避免余额显示为 0

**测试要点**：
- Admin 端单元测试覆盖 API 调用和错误处理 `frontend/admin/src/pages-new/__tests__/goodsEditAi.test.ts`
- 后端测试覆盖文案生成逻辑和 LLM 供应商封装 `backend/product/handler/copywriting_test.go`, `backend/shared/llm/doubao_test.go`

**模型选型经验**：
- 初始尝试 `Doubao-1.5-lite-32k` 为纯文本模型，不支持图片输入
- 最终选用 `doubao-seed-1-6-vision-250815` 支持多模态图片理解
- 排除 `doubao-seedance-2-0-fast-260128`（Seedance 视频生成模型，不适配 chat 协议）

**Admin 接入实现要点**：
- **UI 位置**：在 `GoodsEdit` 页面「基本信息」卡片内新增「AI 一键文案」按钮，位于描述输入框上方
- **状态管理**：使用本地 state 管理加载态（`isGenerating`）和错误提示，AI 生成失败不阻塞手动填写
- **字段预填**：AI 返回的 `title`、`description`、`suggested_start_price` 直接预填到现有表单字段，不新增商品表字段
- **API 封装**：在 `shared/api/product.ts` 新增 `generateCopywriting(params)` 方法，`shared/api/index.ts` 的内联 `productApi` 同步暴露同名方法

**开发流程**：
1. **brainstorming** → 确定方案 A（最小闭环接入）
2. **writing-plans** → 生成 TDD 实施计划
3. **sdd-run** → 隔离 worktree 执行，任务拆分：API 封装 → 组件集成 → 测试补全
4. **TRAE-code-review** → 重点审查请求封装裸 JSON 扩展和 UI 校验一致性
5. **finishing-a-development-branch** → 推送建 PR、CI 修复（`upload-artifact@v3` → `@v4`）、合并到 main

### 管理员 Dashboard 角色化展示 (Admin Dashboard Role-Based Display)

**问题背景**：商家和管理员共用同一 Dashboard 页面，但两者关注点完全不同。商家关注经营数据（待支付订单、今日收入等），管理员关注平台治理（全局统计、异常监控等）。

**角色化策略**：
1. **数据视角分离**：复用现有 `/statistics/overview` 接口，商家看自己的数据，管理员看平台全量数据
2. **UI 元素按需展示**：
   - 商家：显示「发布商品」「开启直播」等经营动作按钮
   - 管理员：隐藏经营动作按钮，替换为平台级指标卡片
3. **文案动态派生**：同一导航项根据 `user.role` 显示不同文案（如商家看到「我的直播间」，管理员看到「直播间列表」）

**实现要点**：
```tsx
// Layout.tsx 导航项按角色动态派生
const navItems = useMemo(() => {
  const baseItems = [...];
  if (user?.role === 1) { // 商家
    baseItems.push({ label: '我的直播间', path: '/live/list' });
  } else { // 管理员
    baseItems.push({ label: '直播间列表', path: '/live/list' });
  }
  return baseItems;
}, [user?.role]);
```

**来源**：session:6a22c06c2ec60aa1a73a1c37

### 直播间开播入口设计 (Live Stream Start Entry Design)

**设计背景**：PC 管理端需要控制直播间业务状态以跑通 H5 观看、竞拍、一口价交易链路，但真实推流能力计划二期在移动端实现。

**核心决策**：
1. **PC 定位**：PC 管理端是「直播经营控制台」而非「直播设备」，负责配置直播间、商品、竞拍、一口价、开播预告
2. **按钮文案**：保留 `开始直播`，但配合弱提示说明当前为演示开播状态
3. **入口位置**：从 Dashboard `window.prompt` 手输 ID 迁移到直播间详情页，商家在自己直播间详情页执行开播

**状态语义**：
```
未开播(0) → 开始直播 → 直播中(1) → 结束直播 → 已结束(2)
                 ↓
              封禁(3) ← 管理员治理操作
```

**角色权限边界**：
| 动作 | 商家 | 管理员 | 说明 |
|------|:---:|:---:|:---|
| 开始直播 | ✅ | ❌ | 商家只能开播自己拥有的直播间 |
| 结束直播 | ❌ | ✅ | 一期仅管理员可关闭，二期考虑商家自关闭 |
| 封禁直播间 | ❌ | ✅ | 管理员治理动作，封禁后直播间不可开播 |

**前端实现要点**：
1. **详情页 Scoped 接口**：使用 `/admin/live-streams/:id` 而非 `/live-streams/:id`，带 owner scope 校验，非 owner 在读取阶段 403
2. **封禁状态保护**：`status === 3` 时禁用开播按钮，不能只依赖后端拒绝
3. **角色展示控制**：商家展示开播说明 + 开始直播按钮，管理员展示封禁/关闭治理动作
4. **状态注释补全**：前端 `types.ts` 需补全 `3=已封禁` 注释，与后端枚举保持一致

**二期预留**：
- 移动端主播页推流凭证生成
- 推流成功回调触发开播状态
- 商家自关闭直播接口（如需）

**来源**：session:6a22c1da2ec60aa1a73a1cdd

### 平台管理员与商家直播间页面拆分 (Platform vs Merchant Live Room Pages)

**问题背景**：原直播间列表页面被商家和管理员复用，导致平台管理员看到「我的直播间」等商家专属文案，且无法区分「平台治理视角」与「商家经营视角」。

**拆分决策**：
1. **独立页面组件**：
   - `PlatformLiveList` — 平台管理员视角，标题为「平台直播间管理」，用于全站治理
   - `MerchantLiveList` — 商家视角，标题为「我的直播间」，用于经营自有直播间
2. **路由分离**：
   - 平台入口：`/live/list` → `PlatformLiveList`
   - 商家入口：`/live/my` → `MerchantLiveList`
   - 兼容处理：商家访问 `/live/list` 自动导向 `/live/my`
3. **功能边界**：
   - 平台管理员：看不见「创建直播间」按钮、「规则模板」入口、「创建竞拍」按钮
   - 商家：保留全部经营功能入口

**关键代码模式**：
```tsx
// 路由配置
<Route path="/live/list" element={
  <RequireRole allowedRoles={[2]}><PlatformLiveList /></RequireRole>
} />
<Route path="/live/my" element={
  <RequireRole allowedRoles={[1]}><MerchantLiveList /></RequireRole>
} />
```

**来源**：session:6a23e82e2ec60aa1a73a6073

### 前端禁用按钮与后端 API 可用性错位 (Frontend Disabled Button vs Backend API Availability)

**问题背景**：商家点击「创建直播间」按钮无响应，排查发现按钮被前端 `disabled` 且注释说明「后端无接口」，但实际上后端已提供 `POST /api/v1/admin/live-streams`。

**根因分析**：
- 前端遗留注释导致功能被误禁用
- 后端接口实际已存在且网关路由已配置
- 前后端信息同步不及时

**修复方案**：
1. 移除 `disabled` 属性，恢复按钮可点击状态
2. 验证后端接口可用性（`POST /api/v1/admin/live-streams` 仅允许商家调用）
3. 建立接口可用性检查清单，避免前端过早禁用功能

**来源**：session:6a23e82e2ec60aa1a73a6073

### Admin 统计页面身份维度实现 (Admin Statistics Role-Based Implementation)

**功能概述**：Admin 数据统计页面支持按登录身份自动切换数据范围，同一页面适配平台管理员和商家两种视角。

**实现状态**：

| 统计模块 | 平台管理员 | 商家 | 实现状态 |
|---------|:---------:|:---:|:--------:|
| 竞拍统计 | 全平台维度 | 我的竞拍维度 | 已实现 |
| 收入统计 | 全平台维度 | 我的收入维度 | 已实现 |
| 用户统计 | 平台用户洞察 | 不可见 | 已实现 |

**后端身份切换机制**：
- Gateway 允许 `merchant/admin` 都访问 `/statistics/auctions` 和 `/statistics/revenue`
- `admin` 角色：后端 `creatorID/sellerID = nil`，统计全平台数据
- `merchant` 角色：后端使用 `X-User-ID` 作为过滤条件，只统计当前商家数据

**前端角色视角说明**：
- 平台管理员显示：`全平台竞拍统计` + `平台维度` badge
- 商家显示：`我的竞拍统计` + `商家维度` badge
- 用户统计 Tab 仅对管理员可见（商家访问会被后端 403）

**关键代码位置**：
- Gateway 路由：`backend/gateway/router/router.go`
- 竞拍统计 Handler：`backend/auction/handler/statistics.go`
- 收入统计 Handler：`backend/product/handler/statistics.go`

**来源**：session:6a23e94b2ec60aa1a73a6199

### Admin 商品列表中文乱码修复 (Admin Product List Mojibake Fix)

**问题背景**：管理端商品列表界面出现中文乱码，后端返回的 UTF-8 文本被误解析为 Windows-1252。

**根因分析**：
- 后端数据库和接口实际使用 UTF-8 编码
- 某些环节（如数据库连接、中间件）可能导致编码被错误转换
- 前端接收到的文本是 Mojibake（如 `å•†å“` 应为 `商品`）

**修复方案**：
1. **后端编码修复**：在 `productApi` 返回边界对 `name`、`description` 等字段进行编码归一化
2. **前端解码工具**：使用 `decodePossibleMojibake` 函数修复后再渲染
3. **测试覆盖**：新增单元测试验证修复函数对各种 Mojibake 模式的处理能力

**关键代码位置**：
- `frontend/admin/src/shared/api/index.ts` - `productApi` 响应处理
- `frontend/admin/src/shared/api/encoding.ts` - 编码修复工具函数

**来源**：session:6a2134612ec60aa1a7394b27

### Admin 竞拍列表中文乱码修复 (Admin Auction List Mojibake Fix)

**问题背景**：竞拍列表界面展示 `auction.product?.name` 和 `auction.live_stream_name` 时出现中文乱码，而商品列表修复未覆盖嵌套字段。

**根因分析**：
- 上一轮商品列表修复只处理了 `productApi`，未处理 `auctionApi` 返回的嵌套 `product` 字段
- `AuctionList` 组件消费的是 `auction.product.name` 而非直接查询商品接口

**修复方案**：
1. **边界归一化**：在 `auctionApi` 返回边界统一修复 `product.name/description` 和 `live_stream_name`
2. **避免页面级补丁**：在 API 层统一处理，而非在每个渲染点单独修复
3. **失败用例先行**：先添加失败测试固定缺口，再实施修复

**关键代码位置**：
- `frontend/admin/src/shared/api/auctionEncoding.ts` - 竞拍数据编码修复
- `frontend/admin/src/shared/api/bidEncoding.ts` - 出价数据编码修复

**来源**：session:6a2134612ec60aa1a7394b27

### Admin 买家名归一化修复 (Admin Buyer Name Normalization Fix)

**问题背景**：管理端商家视角出现多处买家名显示异常：
1. 竞拍详情页出价记录 `td` 列用户名乱码（如 `æ¼”ç¤ºä¹°å®¶A` 应为 `演示买家A`）
2. 订单列表/详情买家名直接渲染 `order.user_name`，未走编码修复

**根因分析**：
- 之前仅修复了出价记录 API (`auctionApi.getBids`) 的买家名乱码
- 订单列表/详情 (`orderApi.list/get`) 仍在直接渲染原始 `user_name`
- 多个 API 边界各自处理编码，存在遗漏风险

**修复方案**：
1. **统一编码模块**：新增 `userEncoding.ts` 提供通用买家名归一化函数
2. **API 层归一化**：在数据进入 UI 前统一修复，而非页面局部处理
3. **双 API 覆盖**：同时修复 `orderApi` 和 `auctionApi` 的买家名字段
4. **测试先行**：先写失败测试固定缺口，再实施修复

**关键代码位置**：
- `frontend/admin/src/shared/api/userEncoding.ts` - 通用用户编码修复
- `frontend/admin/src/shared/api/index.ts` - orderApi/auctionApi 接入归一化

**测试覆盖**：
```bash
npm test -- --runInBand src/shared/api/__tests__/order.test.ts src/shared/api/__tests__/auction.test.ts
```

**关键提交**：
- `6a9f8a47 fix(admin): normalize buyer names and bid prices`

**来源**：session:6a25b6c30bfcee1b04fb138d

### 商家 AI 文案权限修复 (Merchant AI Copywriting Permission Fix)

**问题背景**：商家账号点击「一键 AI 文案」按钮时提示"当前账号没有使用 AI 文案的权限"，但管理员账号可正常使用。

**根因分析**：
- Gateway 的 `RequireMerchant()` 中间件已正确鉴权
- 但转发请求到 `product-service` 时未透传 `X-User-Role` Header
- `product-service` 的 `CopywritingHandler` 无法识别调用者角色，默认拒绝非管理员

**修复方案**：
1. **Gateway 透传角色**：在 `proxy.go` 中将 `X-User-Role` 透传给下游服务
2. **后端角色识别**：`product-service` 从 Header 读取角色，支持 `merchant`、`admin`、`streamer` 访问
3. **测试覆盖**：新增 `copywriting_route_test.go` 验证角色透传逻辑

**关键提交**：
- `f0d2b9af fix(gateway): forward merchant role to downstream`

**来源**：session:6a2134612ec60aa1a7394b27

### API 响应归一化修复 (API Response Normalization Fix)

**问题背景**：Admin 前端部分接口响应处理与后端实际返回结构不一致，导致数据无法正常展示。

**修复内容**：

1. **分类接口 `listCategories()`**
   - 后端返回：`{ list, total, page, page_size }`
   - 前端原假设：`Category[]`
   - 修复：解包 `list` 字段为数组

2. **收入统计接口 `getRevenueStats()`**
   - 后端返回：对象 `{ daily_revenue, monthly_revenue, category_distribution }`
   - 前端图表期望：数组格式
   - 修复：根据 `group_by` 参数解包为对应数组

**测试覆盖**：
```bash
npm test -- product.test.ts statisticsApi.test.ts --runInBand
# 2 suites passed, 9 tests passed
```

**来源**：session:6a23e94b2ec60aa1a73a6199

### Dashboard 统计概览接口契约对齐 (Dashboard Overview Statistics Contract Alignment)

**问题背景**：Admin Dashboard 的三个 KPI 卡片（总收入、今日成交、总订单数）中，「今日成交」和「总订单数」显示为 ¥0，因为后端 `/statistics/overview` 接口未返回前端期望的字段。

**根因分析**：
- 前端 `Dashboard.tsx` 读取 `statisticsApi.getOverview()`，期望字段：`total_revenue`、`today_revenue`、`total_orders`
- 后端 `OverviewStatistics` 只返回：`total_auctions`、`success_rate`、`total_revenue`、`total_users`、`active_users`
- 契约缺口：`today_revenue`（今日成交额）和 `total_orders`（订单总数）未实现

**修复方案**：
1. **后端补齐字段** (`backend/product/handler/statistics.go`)
   - `today_revenue`：统计今日已支付及后续状态订单的成交额
   - `total_orders`：统计订单总数（从 `orders` 表统计，不用 `total_auctions` 替代）

2. **测试先行** (`backend/product/handler/statistics_test.go`)
   - 新增测试断言响应必须包含 `total_orders` 和 `today_revenue`
   - 先跑失败测试确认契约缺口，再实现字段

**关键代码模式**：
```go
// DAO 层 SSOT 统计逻辑
type OverviewStatistics struct {
    TotalAuctions int64   `json:"total_auctions"`
    SuccessRate   float64 `json:"success_rate"`
    TotalRevenue  float64 `json:"total_revenue"`
    TodayRevenue  float64 `json:"today_revenue"`  // 新增
    TotalOrders   int64   `json:"total_orders"`   // 新增
    TotalUsers    int64   `json:"total_users"`
    ActiveUsers   int64   `json:"active_users"`
}
```

**验证命令**：
```bash
go test ./dao ./service ./handler -count=1
```

**来源**：session:6a25bce00bfcee1b04fb15bd

---

### 统计图表空态处理 (Statistics Chart Empty State Handling)

**问题背景**：竞拍热度分析图表在数据为空时显示空白容器，用户无法区分是加载中还是确实无数据。

**根因分析**：
- `Stats` 页面将竞拍、收入、用户三个接口放在同一个 `try/catch` 里顺序请求
- 任一接口失败会导致前面已获取的数据也被 catch 清空
- 空数组时 Recharts 仍渲染容器但不显示任何内容

**修复方案**：
1. **请求隔离**：三个统计请求独立执行，互不影响
2. **空态提示**：数据为空时显示明确的空态文案而非空白图表
3. **错误降级**：单个接口失败只影响对应模块，不阻断其他统计展示

**来源**：session:6a23e94b2ec60aa1a73a6199

### Admin 竞拍规则模板管理 (Admin Auction Rule Template Management)

**功能概述**：Admin 管理后台支持商家创建、编辑、删除竞拍规则模板，并在创建竞拍时选择模板自动应用规则参数。

**后端接口契约**：
- `GET /api/v1/admin/auction-rule-templates` — 模板列表
- `POST /api/v1/admin/auction-rule-templates` — 创建模板（HTTP 201 表示成功）
- `PUT /api/v1/admin/auction-rule-templates/:id` — 编辑模板
- `DELETE /api/v1/admin/auction-rule-templates/:id` — 删除模板
- `POST /api/v1/admin/products/:id/apply-rule-template` — 应用模板到商品

**前端响应码处理陷阱**：
- 后端创建成功返回 HTTP 201 + body `{ code: 201, data: {...} }`
- 前端 `request.ts` 若只把业务码 `0/200` 当成功，会导致 201 被误判为失败
- 修复：响应处理逻辑需兼容 `code: 201` 作为成功状态

**Upsert 语义修复**：
- 问题：旧规则有 `cap_price`，应用无封顶价模板后 `cap_price` 未被清空
- 根因：`Assign(rule).FirstOrCreate(rule)` 对 nil/zero 字段不够显式
- 修复：改为显式字段覆盖 — 按 `product_id` 存在则 `Updates` map，不存在则 `Create`

**模板应用链路**：
```
规则模板 CRUD → 选择模板 → apply-rule-template → 写入 auction_rules → 创建竞拍
```

**与测试平台的关系**：
- 独立测试平台不依赖 Admin 规则模板
- 测试平台直接调用 `POST /api/v1/products/:id/rules` 创建商品规则
- 模板功能仅影响 Admin 商家后台链路

**来源**：session:6a241e023eefb8c530aa78a6

---

### Admin 预约开拍功能 (Admin Scheduled Auction Start)

**功能概述**：Admin 管理后台支持商家在创建竞拍时设置预约开始时间，实现「立即开拍」与「预约开拍」两种模式。

**核心约束**：
1. **商家固定直播间约束**：Demo 环境复用固定商家直播间，不破坏现有直播间归属关系
2. **一商品一活跃竞拍约束**：同一商品在同一时间只能有一个活跃竞拍，预约时间到达前该商品不能被其他竞拍占用

**状态语义**：
```
待开始(upcoming) → 开始时间到达 → 进行中(ongoing) → 结束时间到达 → 已结束(ended)
```

**实现要点**：
1. **表单扩展**：创建竞拍表单新增 `start_time` 字段，支持日期时间选择器
2. **后端契约**：`POST /api/v1/admin/auctions` 请求体扩展 `start_time?: string`（ISO 8601 格式）
3. **状态判定**：列表查询从单纯 `status=1` 升级为 `status=1 AND start_time <= NOW()`
4. **时间源统一**：状态机时间由应用层统一传入，不依赖数据库 `NOW()`，避免时区问题

**关键代码模式**：
```go
// 列表查询：进行中竞拍需同时满足状态和时间条件
WHERE status = 1 AND start_time <= ? AND end_time > ?

// 创建竞拍：支持传入自定义开始时间
if req.StartTime != nil {
    auction.StartTime = *req.StartTime
} else {
    auction.StartTime = now // 立即开拍
}
```

**与 Demo Console 的关系**：
- Demo Console 的「正在竞拍」模式继续复用固定商家直播间
- Admin 预约开拍功能面向真实商家场景，支持灵活配置开拍时间
- 两者共享同一套后端状态机和时间判定逻辑

**来源**：session:6a25947b0bfcee1b04fb0946

### 商家订单管理后端化实现 (Merchant Order Management Backend Integration)

**功能概述**：将 Admin 商家订单管理界面从 mock/占位状态升级为完整后端对接，实现订单列表、搜索、状态统计和买家信息展示。

**实现范围**：
1. **后端接口扩展** (`backend/product/handler/admin_order.go`)
   - `GET /admin/orders` — 商家订单列表，支持 `search` 关键词和 `status` 筛选
   - 响应包含：订单基础信息 + 关联商品信息 + 买家信息（username/avatar）

2. **买家信息补齐** — 通过 `auction-service` 内部批量接口获取
   - 接口：`POST /internal/users/batch`
   - 入参：`user_ids` 数组
   - 响应：`map[user_id]{username, avatar}`

3. **状态统计** — 后端实时计算 4 个状态计数
   - `pending_payment` — 待支付
   - `pending_shipment` — 待发货
   - `shipped` — 已发货
   - `completed` — 已完成

**跨服务降级策略**：
```go
// enrichAdminOrderBuyers: 买家信息查询失败时静默跳过，不阻断订单列表
summaryMap, err := userClient.BatchGetUserSummaries(ctx, winnerIDs)
if err != nil {
    log.Printf("[WARN] batch get user summary failed: %v", err)
    // 降级：返回不含买家信息的订单列表
    return orders, nil
}
// 正常：回填买家信息
for _, order := range orders {
    if summary, ok := summaryMap[order.WinnerID]; ok {
        order.BuyerUsername = summary.Username
        order.BuyerAvatar = summary.Avatar
    }
}
```

**前端适配要点**：
- 订单列表 API 必须使用 `/admin/orders`，不能复用用户端 `/orders`
- 搜索框支持按商品名称/订单号模糊搜索
- 状态 Tab 与后端统计数字联动
- 买家信息字段：`buyer_username`（显示昵称）、`buyer_avatar`（显示头像）

**测试覆盖**：
- Handler 层：搜索、统计、商家隔离
- Service 层：enrichment 成功与降级路径
- Client 层：正常响应与错误处理

**来源**：session:6a2419153eefb8c530aa7658

### 商品 AI 文案生成 (Product AI Copywriting)
