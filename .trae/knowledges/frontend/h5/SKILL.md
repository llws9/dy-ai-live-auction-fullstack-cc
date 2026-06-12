---
name: knowledge-frontend-h5
description: >
  Covers H5 用户端的页面结构、图片资源管理、兜底策略、足迹功能和移动端布局约束。
  Navigate when: modifying frontend/h5 pages, adding mobile features, changing image fallback logic, debugging mobile layout issues, or working with localStorage-based features.
  Excludes: Admin 管理后台和 Test Dashboard。
  Keywords: frontend/h5, H5, mobile, image fallback, default-auction-cover, localStorage, footprint, viewport, Home, LiveRoom, User
---

## Module Structure

H5 是面向用户的移动端前端，覆盖首页、直播间、竞拍、提醒、订单和个人中心等核心场景；核心风险集中在图片资源兜底、移动端布局适配和本地状态管理。

### Directory Layout
- `frontend/h5/src/pages/Home/` — 首页、商品列表和筛选器。
- `frontend/h5/src/pages/LiveRoom/` — 直播间、竞拍交互和实时消息。
- `frontend/h5/src/pages/User/` — 个人中心、订单列表和足迹。
- `frontend/h5/src/utils/` — 工具函数，包括图片兜底逻辑。
- `frontend/h5/public/assets/` — 静态资源，包括兜底图片。
- `frontend/h5/nginx/`、`frontend/h5/Dockerfile` — H5 容器和静态服务配置。

### Key Entry Points
- `frontend/h5/src/pages/Home/index.tsx` — 首页入口，包含商品卡片和图片兜底。
- `frontend/h5/src/utils/imageFallback.ts` — 图片加载失败兜底逻辑。
- `frontend/h5/src/pages/User/Index.tsx` — 个人中心入口。

## Gotchas
- **夜间模式 CSS 选择器适配**：项目实际暗色开关是 `<html data-theme="dark">`，不是 `.dark` class。使用 `:global(.dark)` 选择器在真实页面不会命中，导致夜间样式失效。应使用 `[data-theme="dark"]` 或 `:global([data-theme="dark"])` 选择器
- **公网环境图片兜底必须使用同源静态资源**。`copilot-cn.bytedance.net` 等内网域名在公网浏览器无法解析（ERR_NAME_NOT_RESOLVED），导致图片显示失败。兜底图应放在 `frontend/h5/public/assets/` 目录下（如 `default-auction-cover.svg`），使用相对路径引用（`frontend/h5/src/pages/Home/index.tsx`, `frontend/h5/src/utils/imageFallback.ts`）
- 直播间内动画及绝对定位组件必须使用 `position: absolute` 并基于容器尺寸计算相对坐标，禁止使用 `vw/vh`，否则在不同尺寸手机上会错位
- 依赖 WebSocket 的 Hook 必须随 auth token 变化执行重连，否则切换用户后仍会收到旧用户的消息
- 首页筛选器在「收藏」Tab 需隐藏，避免无意义的空筛选状态
- 个人中心足迹记录使用 `localStorage` 存储，结构为 `{id, name, cover, enteredAt}`，上限 10 条，新增时去重并置顶
- 个人中心服务区（钱包/地址/卖家申请/企业入驻）采用横向图标+文字排列，比竖向更紧凑且更像可点击菜单
- 个人中心数字角标（如竞拍记录提醒）应绝对定位到卡片右上角，避免挤偏文字导致不居中
- 个人中心"新"功能角标（如卖家申请/企业入驻）应外移到卡片右上角，不压标题文字
- 数字角标颜色必须与通用 Badge 组件保持一致，使用 `--touchpoint-badge-text: var(--bg-surface)` 变量，确保双主题下对比度正确
- **数字角标颜色覆盖**：当用户明确要求数字为白色时，需使用精确选择器 `.metricCard .metricBadge` 覆盖，避免被 `.metricCard span` 等更高优先级规则覆盖
- **日间模式主 CTA 悬浮样式陷阱**：主 CTA 按钮（如「中标待支付」）在日间模式下悬浮时，内部 `strong` 元素可能被全局 anchor hover 规则覆盖导致文字颜色变透明（与背景融合）。修复需显式锁定 `strong` 的 `color: var(--text-inverse)`，确保悬浮态文字可见
- **主 CTA 按钮日间悬浮修复**：日间模式下主 CTA 悬浮时文字不可见，根因是全局样式 `.primaryAuctionCta:hover strong` 被覆盖。需使用更精确选择器 `.primaryAuctionCta:hover strong { color: var(--text-inverse) !important; }` 强制锁定
- **直播间顶部栏布局对齐**：右侧「在线人数」胶囊应与左侧主播信息区 (`hostPill`) 保持同高（42px），而非固定宽度。删除右侧多余的 `×` 关闭链接（左侧已有返回按钮），保持顶部栏视觉平衡
- **直播间详情页字符编码修复**：`LiveRoomSlide.tsx` 中的 `host_name` 可能因后端编码问题出现乱码（mojibake），需使用 `repairUtf8Mojibake` 工具函数修复后再渲染（首页 `Home/index.tsx` 已实现此修复，直播间需保持一致）
- **直播间主播头像兜底**：`LiveRoomSlide.tsx` 中的 `host_avatar` 直接渲染后端返回的 URL，若该 URL 为内网域名（如 `copilot-cn.bytedance.net`）或已失效，会导致头像显示失败。应添加 `onError` 兜底逻辑，失败时切换到本地默认头像（`frontend/h5/public/assets/default-avatar.svg`）
- **足迹状态实时获取策略**：足迹仅存储进入时的快照（`id, name, cover, enteredAt`），不包含直播间当前状态。若需在个人中心显示直播间实时状态（直播中/即将开始/已结束），应在页面打开时基于 `footprints.map(live_stream_id)` 并发调用 `liveStreamApi.get(id)` 获取最新状态，而非修改 localStorage 契约。状态角标放在封面右上角，使用半透明深色胶囊，接口失败时显示「状态未知」或不显示，避免阻塞页面加载
- **底部导航栏媒体查询陷阱**：移动端浏览器在宽视口（如 CSS 视口宽度 >= 512px）下，`@media (min-width: 431px)` 可能将底部导航从 `position: fixed` 覆盖为 `position: absolute`，导致导航被推到页面内容底部，需滑到最底部才可见。修复时应使用更精确的媒体查询条件（如增加 `and (hover: hover) and (pointer: fine)` 区分桌面与移动端）
- **首页底部导航定位陷阱**：H5 首页在浏览器宽视口模式下，底部导航栏会跑到所有直播间卡片的最下方（而非固定在视口底部），但进入直播间后正常。根因是 `MobileShell.module.css` 中的媒体查询 `@media (min-width: 431px)` 将 `.bottomNav` 从 `position: fixed` 覆盖为 `position: absolute`，导致导航相对于 `.mobileShell` 定位而非视口。修复时应增加 `(hover: hover) and (pointer: fine)` 条件区分桌面与移动端，确保移动端始终使用 `position: fixed`
- **商品列表过滤原则**：H5 首页展示的商品列表依赖后端接口返回的数据，若后端未过滤 `status` 字段，可能导致管理端标记为"未发布"的商品对用户可见。此类问题需在后端修复（增加 `status=1` 过滤），H5 端无需额外处理，但需验证修复后列表数据正确性
- **首页分类 Tab 数据驱动原则**：H5 首页顶部 Tab 栏的"全部/收藏"是固定设计，动态分类来自 `GET /categories` 接口。若线上仅显示两个 Tab，需检查后端分类数据是否正确关联商品，以及 Admin 管理端是否正确维护 `category_id` 字段。分类数据问题需在后端/Admin 修复，H5 端仅负责按接口数据渲染
- **CSS 未闭合块导致 Vite 编译错误**：`Live.module.css` 末尾存在未闭合的 `@media (prefers-reduced-motion: reduce)` 块时，Vite dev server 会抛出 `vite-error-overlay` 错误。单测和 `tsc` 不会覆盖 CSS 解析，需通过 Vite 实际编译或生产构建 (`npm run build`) 才能暴露此类问题。修复时需补全完整的媒体查询块，而非仅添加闭合括号

## Architecture
- H5 使用 React + Vite 构建，独立部署在 `/` 路径下，与 Admin (`/admin/`) 和 Test Dashboard (`/test/`) 分离
- 图片兜底策略：加载失败时切换到本地 SVG 兜底图，而非外部 URL，确保公网环境可用性
- 移动端布局优先使用 Flexbox 和百分比，关键交互区域保证最小 44px 点击热区

## Patterns
- 图片组件统一使用 `onError` 处理加载失败，调用 `imageFallback.ts` 中的兜底逻辑
- 足迹功能采用「去重+置顶」策略，同一条记录再次访问时移到最前，不重复添加
- 足迹记录时机：进入直播间即触发，数据存储于 localStorage
- 个人中心信息架构采用「优先级瀑布流」：待付款 CTA → 交易统计 → 足迹 → 账户服务，按使用频率降序排列
- 服务区按钮采用图标左+文案右的横向布局，降低卡片高度，提高空间利用率
- **直播间空态布局**：避免使用 `align-items: center` 导致上半屏留白，改用 `flex-start` 贴近顶部
- 服务区功能入口应避免冗余：若上方已存在带数字的 Metric 统计（如收藏），下方二级菜单应替换为其他功能（如设置），避免重复入口
- 消息通知入口位置：从二级菜单移至「我的竞拍」三宫格中间位置，替换原「中标数量」卡片，右上角显示未读角标
- 二级菜单占位策略：当功能区菜单项较少时（如只有「设置」一项），可添加「帮助中心」「客服与反馈」「关于平台」等占位入口，保持视觉区域完整
- 筛选器状态按 Tab 隔离，切换 Tab 时重置筛选条件避免交叉污染

### H5 首页筛选器设计决策 (Homepage Filter Design Decision)

**决策背景**：用户需要在首页新增筛选维度（热度/价格区间）对用户进行分流。

**核心决策**：
- **热度定义**：采用出价次数聚合（而非在看人数）
  - 出价数据与 auctions 同库，单服务内 JOIN 即可
  - 所有场次状态（预告/直播中/已结束）都有出价数，排序都有意义
  - 在看人数只有"直播中"才有数据，预告/已结束恒为0，不适合做通用排序维度

- **价格过滤口径**：采用 `auctions.current_price`（而非起拍价）
  - 起拍价在跨服务的 product-service，需跨服务编排
  - current_price 同库，与热度排序在同一查询里实现，最简单

- **后端必做改动**：这不是前端独立改动，必须同步修改：
  1. `ListParams` 增加 `SortBy` 字段
  2. DAO 层支持 GROUP BY bids 计数排序
  3. 响应增加 `bid_count` 字段
  4. 前端选"最热"时跳过 `sortAuctionsForHome` 客户端重排

**来源**：session:6a26ce7a0bfcee1b04fb3ddc

- 足迹实时状态获取采用「快照存储+实时拉取」模式：localStorage 只存进入时的静态信息，打开个人中心时再并发拉取各直播间的实时状态，避免历史足迹数据迁移和状态过期问题

### H5 个人中心 UX 决策流程模式 (Profile Center UX Decision Process)

**问题背景**：个人中心页面信息层级混乱、空间利用率低，需要从动机出发重构。

**决策流程**：
1. **痛点定位**：通过 brainstorming 明确核心痛点（信息层级混乱 + 空间利用率低）
2. **核心动作识别**：确定用户最高频动作是「交易闭环」（看竞拍/中标 → 去付款）
3. **功能边界明确**：足迹功能采用纯前端实现（localStorage），避免把 UI 重构撑成大 feature
4. **布局方案对比**：在「优先级瀑布流」和「交易聚合+服务网格」间选择前者，更贴合现有结构
5. **细节迭代**：服务区横向图标+文字、角标外移到右上角、数字颜色统一等微观调优
6. **导航链路优化**：中标待支付 CTA 跳订单页、钱包入口跳独立钱包页、消息通知移到核心统计区

**关键产出**：
- 足迹数据契约：`{id, name, cover, enteredAt}`，上限 10 条，去重置顶
- 服务区布局：横向图标+文字，降低卡片高度
- 角标定位策略：数字角标和「新」角标都外移到卡片右上角，不压文字

**来源**：session:6a2854360bfcee1b04fbf604

### 直播间顶部栏布局调整模式 (Live Room Header Layout Adjustment)

**问题背景**：直播间顶部栏左右两侧元素（主播信息区 vs 在线人数胶囊）高度不一致，且右侧存在多余的关闭按钮，影响视觉平衡。

**调整原则**：
1. **高度对齐**：右侧「在线人数」胶囊的高度应与左侧主播信息区 (`hostPill`) 保持一致（42px = 34px 头像 + 4px + 4px padding）
2. **宽度自然**：在线人数胶囊不设置固定左右宽度，让内容自然撑开
3. **删除冗余**：移除右侧 `×` 关闭链接，因为左侧已有返回按钮，功能重复

**TDD 验证要点**：
- 组件测试：确认不再渲染 `aria-label="退出直播间"` 的链接
- CSS 测试：确认 `.viewersRow` 使用正确高度且 `.closeBtn` 样式被移除

**关键文件**：
- `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` — 顶部栏组件结构
- `frontend/h5/src/pages/Live/Live.module.css` — 顶部栏样式定义

**来源**：session:6a26d5860bfcee1b04fb3f61

## Conventions
- H5 开发端口为 5173，与 Admin (5175) 和 Test Dashboard (5174) 区分
- 静态资源放在 `public/assets/`，构建后会复制到 `dist/assets/`，保持同源访问
- 所有新 UI 组件必须适配双主题并使用项目指定设计 Token

## Testing Strategy
- H5 使用 Jest 进行单元测试，重点覆盖图片兜底逻辑和 localStorage 操作
- 移动端布局需在真机或 DevTools 设备模拟器中验证，不能仅依赖桌面浏览器

## Local Development Environment

### WebSocket 端口配置陷阱 (WebSocket Port Configuration)

**问题背景**：本地开发时 WebSocket 服务启动报错 `listen tcp: address 8083: missing port in address`。

**根因分析**：
- 代码中使用标准库 `http.Server{Addr: port}` 启动 WS 服务
- 配置中的 `WSPort` 默认值是 `"8083"`（没有冒号）
- Go 的 `net.Listen` 需要 `":8083"` 格式，而 Hertz 的 `WithHostPorts` 可以接受裸端口
- 这导致 HTTP 服务能启动，但 WS 服务启动失败

**修复方案**：
- 在 `startWebSocketServer` 函数中规范化端口格式
- 检查端口字符串是否以冒号开头，如果没有则添加

**关键代码模式**：
```go
func normalizePort(port string) string {
    if !strings.HasPrefix(port, ":") {
        return ":" + port
    }
    return port
}

// 使用
wsServer := &http.Server{
    Addr: normalizePort(cfg.Server.WSPort),
    Handler: mux,
}
```

**来源**：session:6a1c56f7959156a8dfc84fae

### Vite Proxy 与后端连接问题

**问题背景**：本地开发时前端登录失败，提示"查询用户失败"，但直接 curl 后端接口正常。

**根因分析**：
- Vite 配置中 `proxy.target` 使用 `http://localhost:8080`
- macOS 上 `localhost:8080` 可能命中另一个旧的本机 `gateway-service` 进程
- `127.0.0.1:8080` 才是当前 Docker 启动的 demo gateway
- 导致前端代理请求打到错误的后端实例，返回 401 或 500

**解决方案**：
- 根因是本机旧 `gateway-service` 进程残留，占用了 `localhost:8080`，不是主干配置错误
- 正解是清理旧进程后让 `localhost` 重新命中 Docker demo gateway：`lsof -ti:8080 | xargs kill -9`
- 禁止把主干 `proxy.target` 从 `localhost` 改为 `127.0.0.1` 来绕过本机环境问题（与 AGENTS.md 本地排障约束一致，亦与本文档「优先使用 localhost」保持一致）

**验证命令**：
```bash
# 验证后端实际监听地址
curl http://127.0.0.1:8080/api/v1/health
curl http://localhost:8080/api/v1/health  # 对比返回是否一致
```

**来源**：session:6a1fffdd867f95f321be0cfd

### 本地测试数据初始化流程

**场景**：本地启动后数据库为空，无法登录或看不到直播间数据。

**最短初始化路径**：
1. **创建测试用户**（若注册接口可用）：
   ```bash
   curl -X POST http://127.0.0.1:8080/api/v1/auth/register \
     -H "Content-Type: application/json" \
     -d '{"phone":"13900000001","password":"Demo@123456","username":"本地测试用户"}'
   ```

2. **验证登录链路**：
   ```bash
   # 登录获取 token
   curl -X POST http://127.0.0.1:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"phone":"13900000001","password":"Demo@123456"}'
   
   # 验证用户查询
   curl http://127.0.0.1:8080/api/v1/users/me \
     -H "Authorization: Bearer <token>"
   ```

3. **检查直播间数据**：
   ```bash
   # 确认 live_streams 和 auctions 表有数据
   curl http://127.0.0.1:8080/api/v1/live-streams
   curl http://127.0.0.1:8080/api/v1/auctions
   ```

**若缺少数据**：需运行后端 seed 脚本或手动插入演示数据（见后端文档）。

**来源**：session:6a1fffdd867f95f321be0cfd

## UX Enhancement Decisions

### 直播间排行榜夜间样式设计 (Live Room Ranking Dark Mode Design)

**决策背景**：夜间模式下排行榜区块原使用白色玻璃卡片压暗，红色光晕和黑色内卡叠在一起，层次显得脏乱，空榜时尤其明显。

**核心决策**：
- **配色方案**：采用「深色玻璃 + 暖金高光」统一配色体系，与拍卖/竞价主题契合
- **容器样式**：深色半透明背景 (`rgba(20,20,25,0.85)`) + 暖金色边框/光晕 (`#D4A853`)
- **空位占位态**：「虚位以待」使用暖金色文字，降低透明度 (0.6) 营造期待感
- **我的排位卡片**：避免纯黑压暗，使用深色玻璃 + 暖金边框突出显示

**技术实现要点**：
- CSS 选择器必须使用 `[data-theme="dark"]` 而非 `.dark`，确保与项目暗色开关一致
- 光晕效果使用 `box-shadow` 多层叠加实现暖金扩散感
- 保持与日间模式相同的 DOM 结构，仅通过 CSS 变量切换配色

**关键文件**：
- `frontend/h5/src/pages/Live/Live.module.css` — 排行榜容器样式
- `frontend/h5/src/pages/Live/__tests__/LiveLayoutCss.test.ts` — 夜间样式回归测试

**来源**：session:6a256c3a00057ea64ca2d96c

### H5 个人中心重构 (Profile Center Redesign)

**决策背景**：解决原个人中心信息层级混乱、空间利用率低的问题。

**核心改动**：
- **信息架构**：采用「优先级瀑布流」布局，按使用频率降序排列：待付款 CTA → 交易统计（竞拍/中标/收藏）→ 足迹 → 账户服务
- **Header 区域**：用户信息区采用无卡片设计，直接融入背景，减少视觉层级
- **交易区**：移除独立的「中标」按钮，统一收敛到「竞拍记录」，通过数字角标提示待支付数量
- **服务区（钱包/地址/卖家申请/企业入驻）**：
  - 采用横向图标+文字排列（图标左、文案右），比竖向更紧凑且更像可点击菜单
  - 数字角标（如竞拍记录提醒）应绝对定位到卡片右上角，避免挤偏文字导致不居中
  - "新"功能角标（如卖家申请/企业入驻）应外移到卡片右上角，不压标题文字
- **足迹功能**：基于 `localStorage` 实现，进入直播间即记录，数据结构 `{id, name, cover, enteredAt}`，上限 10 条，同房间再次访问时去重并置顶

**关键设计决策**：
- **中标待支付 CTA 跳转目标**：跳 `/orders`（我的订单页），而非竞拍记录页面 `/history`，因为它是订单支付动作而非浏览动作
- **消息通知入口位置**：从二级菜单移至「我的竞拍」三宫格中间位置，替换原「中标数量」卡片，右上角显示未读角标
- **二级菜单占位策略**：当功能区菜单项较少时（如只有「设置」一项），添加「帮助中心」「客服与反馈」「关于平台」等占位入口，保持视觉区域完整
- **收藏入口去重**：若上方交易统计区已有带数字的 Metric 统计（如收藏），下方二级菜单应替换为其他功能（如设置），避免重复入口

**来源**：session:6a2854360bfcee1b04fbf604

### H5 钱包页设计 (Wallet Ledger Page)

**决策背景**：个人中心点击钱包后需展示余额、冻结金额和收支流水。

**选定方案**：`B · 流水账本`
- 强调余额解释、冻结来源、收支流水追踪
- 顶部展示「可用余额」大数字，下方解释「冻结金额」原因
- 流水列表按时间倒序，区分收入/支出/冻结类型
- 个人中心钱包入口点击后跳转 `/wallet`
- 个人中心顶部「中标待支付」CTA 继续跳 `/orders`（订单支付动作，非钱包浏览）

**数据来源**：
- 余额使用 `userApi.getBalance()`，字段名为 `available_amount`（注意不是 `available` 或 `balance`）
- 流水数据先做前端派生演示数据，明确标注为演示用途

**关键修复**：
- 前端必须正确读取后端返回的 `available_amount` 字段，错误读取会导致余额显示为 ¥0
- 个人中心顶部「中标待支付」CTA 跳转目标应为 `/orders`（我的订单页），而非竞拍记录页面

**设计决策流程**：
- 使用 `ui-design-trio` Skill 进行三版方案推演：
  - A · 资产概览：强调「总资产 = 可用 + 冻结」的会计等式，适合金融类应用
  - **B · 流水账本（选中）**：强调余额解释、冻结来源、收支流水追踪
  - C · 支付行动：强调「充值/提现」快捷入口，适合高频支付场景

**来源**：session:6a2854360bfcee1b04fbf604

### 直播间战况热度条 (BidHeatBar)
- **设计方案**：采用「赛博流光 (Cyber Glow)」风格，通过动态渐变和发光效果表现竞拍激烈度
- **决策过程**：使用 `ui-design-trio` Skill 进行三版方案推演——方案1（极简几何/扁平化）、方案2（赛博流光/科技感）、方案3（仿生呼吸/情绪化），最终选定方案2
- **档位算法**：基于近10秒滑动窗口统计 bid 事件数，分为 calm（冷静）/ warming（升温）/ blazing（白热化）三档
- **数据源覆盖**：必须挂接三处出价来源——`bid_placed` WS回调、`handleBid` REST成功分支、`sky_lamp_auto_bid` 自动跟价
- **落点策略**：最终采用「半嵌入抽屉上边缘」设计——组件中线压在底部抽屉（BidDock）上边缘，上半部分露出在视频区，下半部分嵌入抽屉内；给抽屉顶部留出内容避让空间
- **视觉约束**：
  - 白热化档使用 `transform`/`opacity` 实现流光扫光动画，支持 `prefers-reduced-motion` 降级为静态配色
  - 外层进度条槽始终满宽显示，内部实时热度填充按档位比例计算（calm:24% / warming:62% / blazing:100%）
  - 移除外层灰色罩子，保持视觉通透
- **主题适配**：使用项目现有 CSS Token（`--color-primary-*`、`--color-accent-*` 等），通过 `data-theme` 属性实现双主题切换
- **设计文档**：`docs/superpowers/specs/2026-06-09-live-room-bid-heat-bar-design.md`
- **来源**：session:6a27ede70bfcee1b04fbc3b6

### 半嵌入抽屉组件动画同步 (BidDock Animation Sync)
**问题背景**：战况坞 (BidHeatBar) 作为 `topAddon` 半嵌入在底部抽屉 (BidDock) 上边缘，但抽屉收回时组件"卡"在屏幕中间突兀消失，没有跟随抽屉动画。

**根因分析**：抽屉关闭时 `.sheet` 从 `translateY(0)` 过渡到 `translateY(100%)`，但 `sheetDockAddon` 是兄弟节点，只有固定 `bottom: 50dvh` 和静态 `translateY(50%)`，没有对应的关闭态 transform，所以它停在屏幕中线直到 `renderedSheet` 定时卸载。

**解决方案**：
1. 让 `topAddon` 组件与抽屉共享 `isSheetOpen` 打开状态
2. 给 `sheetDockAddon` 配置与抽屉同步的 CSS transition 参数（`0.35s ease`）
3. 在关闭态时让 `topAddon` 跟随抽屉一起 `translateY` 动画

**关键代码模式**：
```tsx
// BidDock.tsx 中传递 sheet 打开状态给 topAddon
{topAddon && React.cloneElement(topAddon, { isSheetOpen })}

// CSS 中同步动画参数
.sheetDockAddon {
  transition: transform 0.35s ease; // 必须与 .sheet 的 transition 一致
}
.sheetDockAddon.sheetClosed {
  transform: translateY(100%); // 跟随抽屉一起下移
}
```

**教训**：兄弟节点组件若需跟随抽屉动画，必须共享打开状态并同步 CSS transition 参数（如 `0.35s ease`），防止关闭时产生"悬浮卡顿"感。

**来源**：session:6a27ede70bfcee1b04fbc3b6

### 底部抽屉打开动画缺失问题 (Bottom Sheet Open Animation Missing)

**问题背景**：点击打开底部抽屉时，抽屉直接"出现"在屏幕上，没有从底部滑入的动画效果。

**根因分析**：
- `BidDock` 组件在 `sheet !== null` 时才把抽屉 DOM 挂载上去
- 挂载时同时带有 `.sheetOpen` 类（设置 `transform: translateY(0)`）
- 浏览器第一次布局就看到最终状态，没有从 `translateY(100%)` 到 `translateY(0)` 的状态变化
- 关闭时也会立即卸载 DOM，导致关闭动画同样缺失

**解决方案**：
1. 让抽屉在 DOM 挂载后的下一帧（如使用 `requestAnimationFrame` 或 `setTimeout(..., 0)`）再添加 `.sheetOpen` 类
2. 关闭时先移除 `.sheetOpen` 类让抽屉执行关闭动画，等 CSS transition 结束后再卸载 DOM
3. 确保 `.sheet` 的 CSS transition 参数一致（如 `transition: transform 0.35s ease`）

**关键代码模式**：
```tsx
// 挂载后延迟一帧再打开，确保动画触发
useEffect(() => {
  if (sheet !== null) {
    requestAnimationFrame(() => {
      setIsOpen(true); // 添加 .sheetOpen 类
    });
  }
}, [sheet]);

// 关闭时先执行动画，再卸载
const closeSheet = () => {
  setIsOpen(false); // 移除 .sheetOpen 类
  setTimeout(() => {
    setSheet(null); // transition 结束后卸载 DOM
  }, 350); // 与 CSS transition 时长一致
};
```

**来源**：session:6a2560fc00057ea64ca2d1c0

---

### 直播间抽屉布局比例调整 (Live Room Sheet Layout Ratio)

**问题背景**：用户期望点击打开抽屉后，抽屉展开到屏幕的 80%，直播间视频区对应缩小到 20%；但实际效果是抽屉 70dvh、视频区 30%，与预期不符。

**解决方案**：
- 调整抽屉高度：从 `70dvh` 改为 `80dvh`
- 调整直播区压缩比例：从 `30%` 改为 `20%`
- 确保直播区和抽屉的 CSS transition 时长一致（如 `0.35s ease`），避免一个先动完一个后动完

**关键 CSS 调整**：
```css
/* 抽屉展开态 */
.sheetOpen {
  transform: translateY(0);
  height: 80dvh; /* 从 70dvh 调整 */
}

/* 直播区压缩态 */
.videoAreaCompact {
  height: 20%; /* 从 30% 调整 */
  transition: height 0.35s ease; /* 与抽屉动画同步 */
}
```

**来源**：session:6a2560fc00057ea64ca2d1c0

---

### 半嵌入抽屉组件动画同步 (BidDock Animation Sync)
**问题背景**：战况坞 (BidHeatBar) 作为 `topAddon` 半嵌入在底部抽屉 (BidDock) 上边缘，但抽屉收回时组件"卡"在屏幕中间突兀消失，没有跟随抽屉动画。

**根因分析**：抽屉关闭时 `.sheet` 从 `translateY(0)` 过渡到 `translateY(100%)`，但 `sheetDockAddon` 是兄弟节点，只有固定 `bottom: 50dvh` 和静态 `translateY(50%)`，没有对应的关闭态 transform，所以它停在屏幕中线直到 `renderedSheet` 定时卸载。

**解决方案**：
1. 让 `topAddon` 组件与抽屉共享 `isSheetOpen` 打开状态
2. 给 `sheetDockAddon` 配置与抽屉同步的 CSS transition 参数（`0.35s ease`）
3. 在关闭态时让 `topAddon` 跟随抽屉一起 `translateY` 动画

**关键代码模式**：
```tsx
// BidDock.tsx 中传递 sheet 打开状态给 topAddon
{topAddon && React.cloneElement(topAddon, { isSheetOpen })}

// CSS 中同步动画参数
.sheetDockAddon {
  transition: transform 0.35s ease; // 必须与 .sheet 的 transition 一致
}
.sheetDockAddon.sheetClosed {
  transform: translateY(100%); // 跟随抽屉一起下移
}
```

**教训**：兄弟节点组件若需跟随抽屉动画，必须共享打开状态并同步 CSS transition 参数（如 `0.35s ease`），防止关闭时产生"悬浮卡顿"感。

**来源**：session:6a27ede70bfcee1b04fbc3b6

### H5 底部导航栏选中态动效 (BottomNav Active Indicator)

**决策背景**：原底部导航选中态是静态胶囊高亮，用户希望增强「高端拍卖」气质，通过动效提升质感。

**核心决策**：
- 将 `div.tab.active` 外层胶囊和顶部金线抽象成 `nav` 内的**共享选中指示器**
- 点击不同 Tab 时，胶囊和金线沿着底部导航**横向移动**到对应位置
- 使用 `transform/opacity` 实现动画，便于适配 `prefers-reduced-motion`

**三版动效节奏**（通过 `ui-design-trio` 推演）：
- `A · 一体滑行`：胶囊和金线同步移动，最稳，推荐作为默认方案
- `B · 金线先导`：金线先到位，胶囊随后跟上，更精致但节奏需克制
- `C · 压感回弹`：移动后有轻微印章回弹，触感强但最容易过度

**技术要点**：
- 指示器使用绝对定位，`transform: translateX()` 实现移动
- 金线使用渐变 `linear-gradient(90deg, transparent, gold, transparent)`
- 胶囊使用 `backdrop-filter: blur()` 增强质感
- 必须同时适配 H5 双主题（dark/light），通过 `data-theme` 属性切换 Token

**位置计算审查要点**（关键技术约束）：
- **胶囊宽度策略**：推荐固定视觉宽度（如 `72px`），只 `translateX` 永不变宽；若采用可变宽度需同时过渡 `width` 与 `transform`，调参难度大
- **金线居中计算**：金线居中需 `x + width/2`，依赖胶囊宽度变量 `--nav-indicator-width`，spec 中需明确定义
- **初始测量时机**：必须在 `document.fonts.ready` 后重测，避免 web font 未加载导致 Tab 宽度变化后胶囊位置错位
- **React 实现方式**：需使用 Tab `ref` 数组 + `useLayoutEffect`（非 `useEffect`，避免首帧闪烁），路由变化需触发重测
- **定位基准约束**：`tabRect.left - navRect.left` 成立的前提是 `bottomNav` 左右无 border；若将来添加左右 border，absolute 定位会整体偏移

**定位公式定案**（固定宽度方案）：
```
x = (tabRect.left - navRect.left) + (tabRect.width - W) / 2
金线中心 = x + W/2
```
其中 `W` 为固定胶囊宽度（建议 `72px`），`tabRect` 和 `navRect` 通过 `getBoundingClientRect()` 获取。

**实现检查清单**：
- [x] 指示器 `z-index` 设置为 `0`（非 `-1`），Tab 设置为 `1`，避免被背景层遮挡
- [x] 首次定位禁用过渡动画，避免初始位置跳变
- [x] 监听 `resize` 和 `document.fonts.ready` 重新测量
- [x] 动效时长统一为 `260ms`，仅过渡 `transform` 属性
- [x] 支持 `prefers-reduced-motion` 媒体查询降级为无动画

**关键实现细节**：
- 胶囊固定宽度 `72px`，通过 DOM 测量计算每个 Tab 的中心位置
- 使用 `useLayoutEffect` 而非 `useEffect` 进行初始测量，避免首帧闪烁
- Tab 宽度通过 `getBoundingClientRect()` 获取，需考虑字体加载完成后的重测
- 路由变化时需重新触发测量，确保指示器位置正确
- 指示器和 Tab 的 `z-index` 层级：指示器 `z-index: 0`，Tab `z-index: 1`，确保 Tab 内容在指示器上方

**测试要点**：
- 验证固定宽度胶囊在三个不同宽度 Tab 间的居中定位
- 验证 `document.fonts.ready` 后重测机制
- 验证 `prefers-reduced-motion` 降级
- 验证双主题下指示器颜色正确性

**来源**：session:6a28703b0bfcee1b04fc2ec6, session:6a203cf7867f95f321be373d

**部署与迁移关联**：
- 底部导航组件属于 H5 核心交互，其变更需通过 `scripts/deploy-prod.sh` 部署到线上
- 部署前需确保本地 `HEAD == origin/main` 且工作区干净，否则 plan 会被阻断
- 若线上正在进行 Compose Project 迁移（如 `app` → `auction-demo`），需等待迁移完成且 `.deploy-ref` 版本对齐后才能部署新变更

### H5 底部导航共享指示器实现细节 (BottomNav Shared Indicator Implementation)

**实现检查清单**（从设计到落地的关键验证点）：
- [x] 指示器 `z-index` 设置为 `0`（非 `-1`），Tab 设置为 `1`，避免被背景层遮挡
- [x] 首次定位禁用过渡动画，避免初始位置跳变
- [x] 监听 `resize` 和 `document.fonts.ready` 重新测量
- [x] 动效时长统一为 `260ms`，仅过渡 `transform` 属性
- [x] 支持 `prefers-reduced-motion` 媒体查询降级为无动画

**关键实现细节**：
- 胶囊固定宽度 `72px`，通过 DOM 测量计算每个 Tab 的中心位置
- 使用 `useLayoutEffect` 而非 `useEffect` 进行初始测量，避免首帧闪烁
- Tab 宽度通过 `getBoundingClientRect()` 获取，需考虑字体加载完成后的重测
- 路由变化时需重新触发测量，确保指示器位置正确
- 指示器和 Tab 的 `z-index` 层级：指示器 `z-index: 0`，Tab `z-index: 1`，确保 Tab 内容在指示器上方

**测试要点**：
- 验证固定宽度胶囊在三个不同宽度 Tab 间的居中定位
- 验证 `document.fonts.ready` 后重测机制
- 验证 `prefers-reduced-motion` 降级
- 验证双主题下指示器颜色正确性

**来源**：session:6a28703b0bfcee1b04fc2ec6

**迁移执行经验**（从本次会话沉淀）：
- 远端 Compose Project 迁移需先备份数据和静态资源，再停止旧 project 容器（不删卷），复制命名卷到新 project，最后启动新 project
- 卷复制时使用国内镜像源（如 `docker.m.daocloud.io/library/alpine:3.19`）避免拉取失败
- 迁移后需运行 `scripts/init-demo-users.sh` 初始化演示账号
- 必须保留旧 project 的命名卷作为回滚点，严禁执行 `down -v`
- 迁移完成后需验证 `.deploy-ref` 版本标识与本地 HEAD 对齐，否则后续 `scripts/deploy-prod.sh verify` 会报错阻断
- **迁移前置检查清单**：
  - [ ] 本地 `HEAD == origin/main` 且工作区干净（无未提交的非 ignored 改动）
  - [ ] 远端备份已完成（数据卷、静态资源、MySQL dump、`.deploy-ref`）
  - [ ] 确认目标 project 名称（`auction-demo`）与脚本期望一致
  - [ ] 确认命名卷映射关系（`app_*` → `auction-demo_*`），避免数据丢失
  - [ ] 迁移后验证 `.deploy-ref` 与本地 HEAD 一致，确保 verify 可通过

### H5 状态标识颜色分配策略 (Status Badge Color Strategy)

**决策背景**：「即将开始」状态卡片原使用灰色，与「已结束」状态视觉上难以区分，不利于用户快速辨识不同状态的竞拍。

**核心决策**：
- **直播中**：使用橙色/金色（暖色强调），代表活跃和紧迫感
- **即将开始**：使用蓝色（info 色系），代表预告和期待感，与灰色「已结束」形成明显区分
- **已结束**：保持灰色，代表非活跃状态

**实现要点**：
- 「即将开始」专属样式：圆点和边框改为 `info` 蓝色系
- 「直播中」的 `liveDot` 保持金色/橙色不变，占用暖色强调位
- 通过颜色建立状态心智：暖色=活跃/进行中，蓝色=预告/即将开始，灰色=结束/非活跃

**来源**：session:6a24479200057ea64ca27367

---

### H5 首页/详情页「即将开始」状态卡片按钮逻辑 (Upcoming Auction Card Buttons)

**决策背景**：「即将开始」状态的竞拍卡片不应显示「当前出价」「查看竞拍结果」「进入直播」等按钮，这些按钮在竞拍未开始时没有业务意义。

**核心决策**：
- **首页即将开始卡片**：按钮应为「详情」+「订阅」（而非「查看竞拍结果」+「进入直播」）
- **详情页即将开始状态**：
  - 不显示「当前出价」（起拍价为0时仍显示0不合理）
  - 不显示「截止XXX时间」，应显示「XXX时间开始竞拍」
  - 底部按钮应为「详情」+「订阅开拍提醒」，移除出价蒙层
  - 订阅按钮点击后调用 `POST /api/v1/products/{productId}/remind`

**状态判定增强**：
- 详情页状态判定需同时检查 `status` 和 `end_time`，避免「status=进行中但 end_time 已过期」仍显示进行中
- 列表查询的「进行中」判定应为 `status=1 AND end_time > NOW()`

**来源**：session:6a219f682ec60aa1a739b535

---

### H5 订阅开拍提醒功能 (Auction Reminder Subscription)

**功能概述**：用户可订阅即将开始的竞拍，在开拍时收到通知提醒。

**API 契约**：
- `POST /api/v1/products/{productId}/remind` — 订阅/取消订阅开拍提醒
- `GET /api/v1/users/me/reminders` — 获取当前用户的订阅列表（用于页面加载时回填按钮状态）

**前端实现要点**：
1. **状态回填**：页面加载时需调用 `GET /users/me/reminders` 获取已订阅列表，回填按钮「已订阅」状态
2. **幂等处理**：后端重复订阅应返回成功而非报错，前端收到「已订阅」响应后同步 UI 到「已订阅」态
3. **网关路由**：需确保 gateway-service 正确转发 `/api/v1/products/*/remind` 到 product-service

**关键修复**：
- 前端不能仅在点击成功后把 productId 放入内存 Set，刷新后状态会丢失
- 后端重复订阅不能返回 500，应返回幂等成功或明确的状态码

**来源**：session:6a219f682ec60aa1a739b535

---

### H5 已结束竞拍详情页按钮布局 (Ended Auction Detail Buttons)

**决策背景**：已结束竞拍详情页不应在底部蒙层显示「继续竞拍」按钮，这与竞拍已结束的业务状态冲突。

**核心决策**：
- 已结束竞拍详情页底部主按钮应为「查看竞拍结果」，放在内容区竞拍规则卡片下方
- 结果页底部 CTA 应为「返回首页」（而非「继续竞拍」）
- 移除 footer 蒙层出价栏，统一将行动按钮移到内容区

**来源**：session:6a219f682ec60aa1a739b535

---

### H5 首页直播间维度重构 (Homepage LiveRoom Dimension)

**决策背景**：原首页查询的是「竞拍维度」卡片，但业务设定是一个商家只能有一个直播间，一个直播间同一时间只能有一个正在竞拍和一个即将开始；这导致首页和直播间 feed 数据重叠但形态不同，需要明确心智区分。

**核心决策**：
- **首页 `/`** = 货架式概览（决策态）：直播间卡片 grid，展示「有哪些间、谁值得进、什么快开始」
- **直播间 feed `/live`** = 沉浸式播放器（消费态）：全屏单间直播流 + 抖音式上下滑切间
- 两者是「目录 → 内容流」的标准漏斗关系，不是重复

**技术方案**：
- 首页查询直播间维度数据，每个直播间显示「正在竞拍」或「即将开始」的代表性商品
- 已结束竞拍采用「最近 Y 分钟内结束」的滑动窗口策略，避免一个直播间历史已结束过多导致查询复杂
- Feed 几乎不用改动，复用现有 `/live?id=` 进入逻辑

**设计文档**：`docs/superpowers/specs/2026-06-08-h5-home-liveroom-dimension-design.md`

**来源**：session:6a25c5830bfcee1b04fb1c9e

### H5 首页乱码修复漏接问题 (Homepage Mojibake Fix Omission)

**问题背景**：直播间页和商品详情页已接入 `repairUtf8Mojibake` 修复乱码，但用户从首页 `/` 进入时仍看到乱码卡片标题。

**根因分析**：
- 乱码修复最初只覆盖了 `LiveRoomSlide.tsx` 和 `ProductDetail/index.tsx`
- 首页 `Home/index.tsx` 的直播间卡片列表未接入修复函数
- 导致从首页看到的商品名仍是接口原始 mojibake（如 `è€èœ...`），进入直播间后反而正常

**修复方案**：
- 在 `Home/index.tsx` 中对接口返回的 `product_name`、`live_stream_title` 等字段统一调用 `repairUtf8Mojibake`
- 保持与直播间页一致的修复策略

**教训**：多入口页面需检查所有数据渲染路径，避免修复遗漏导致用户体验不一致。

**来源**：session:6a203cf7867f95f321be373d

### H5 直播间状态不一致问题 (Live Stream Status Mismatch)

**问题背景**：首页显示「延时中」状态的竞拍卡片，用户点击进入直播间后显示为空状态或错误页面。

**根因分析**：
- 首页竞拍卡片基于 `auction.status=2`（延时中）判断可进入直播间
- 直播页 `LiveFeedPage` 或 `LiveRoomSlide` 只拉取 `status=1`（直播中）的直播流列表
- 两者状态判定标准不一致：首页按 `auction.status`，直播页按 `live_stream.status`
- 导致用户从「延时中」入口进入后，直播页认为该房间不在直播中，显示为空

**修复方案**：
- URL 携带 `id + auction_id` 时，即使直播列表没有返回该房间，也必须渲染目标直播间
- 或统一状态判定逻辑：直播页应支持按 `auction_id` 直接进入，不依赖直播流列表筛选

**关键代码模式**：
```typescript
// 直播间入口应优先使用 URL 传入的 auction_id 定位
const targetAuctionId = searchParams.get('auction_id');
if (targetAuctionId) {
  // 直接加载该直播间，不依赖列表筛选结果
  loadLiveRoomByAuctionId(targetAuctionId);
}
```

**教训**：列表页筛选条件与详情页准入逻辑必须保持一致，否则会出现「能点进去但看不到内容」的认知断裂。

**来源**：session:6a23de982ec60aa1a73a5af4

### H5 足迹状态角标设计 (Footprint Status Badge)

**决策背景**：用户希望在个人中心足迹卡片上显示直播间的当前状态（直播中/即将开始/已结束），但足迹数据存储在 localStorage 中，仅包含进入时的静态快照。

**核心决策**：
- **存储策略**：保持 localStorage 契约不变，仍只存 `{id, name, cover, enteredAt}`，避免历史数据迁移和脏数据问题
- **状态获取**：个人中心页面打开时，基于 `footprints.map(live_stream_id)` 并发调用 `liveStreamApi.get(id)` 获取最新直播间详情
- **状态映射**：使用接口返回的 `status` 字段映射为「直播中 / 预告中 / 已结束 / 暂不可用」，接口失败时显示「状态未知」或不显示
- **UI 位置**：状态角标放在 `.footprintCover` 内部右上角，使用半透明深色胶囊，既不遮主体图，也符合 H5 小卡片尺寸

**技术边界**：
- 状态获取是「尽力而为」，接口失败不阻塞足迹列表加载
- 足迹列表本身仍从 localStorage 读取，保持离线可用性
- 状态角标仅作为视觉提示，不强制要求实时准确

**来源**：session:6a289e7d0bfcee1b04fc5a92

### H5 首页收藏 Tab 接口对齐 (Home Favorites Tab API Alignment)

**问题背景**：首页顶部 Tab 栏点击「收藏」后显示"收藏接口待后端开放后接入"占位文案，但右上角「我的收藏」入口能正常展示收藏列表，两处体验不一致。

**根因分析**：
- `HomePage` 中 `activeTab === '收藏'` 时直接 `setAuctions([])` 并显示占位文案
- 右上角「我的收藏」入口调用 `followApi.getFollowedLiveStreams()` 获取真实数据
- 两处数据源/接口契约不一致，导致用户体验割裂

**修复方案**：
- 首页「收藏」Tab 点击时调用 `followApi.getFollowedLiveStreams()` 获取收藏直播间列表
- 未登录时正常返回 401，登录后展示已收藏直播间
- 无收藏时显示「暂无收藏直播间」空状态

**测试验证**：
- 新增回归测试：点击「收藏」tab 时必须调用 `followApi.getFollowedLiveStreams`
- 验证未登录态返回 401，登录后正确渲染收藏列表

**来源**：session:6a2145b92ec60aa1a7396725

### H5 我的收藏页面按钮文案复用陷阱 (Favorites Page Button Label Reuse)

**问题背景**：「我的收藏」页面中，「进入直播间」和「取消收藏」按钮后面都跟着「直播间」三个字，同时显示 `#undefined` 字样（如「进入直播间 直播间 #undefined」）。

**根因分析**：
- 页面将 `title` 字段同时用于内容标题和按钮补充说明
- 当后端返回的收藏项缺少 `title`/`name`/`id` 等字段时，前端生成兜底字符串「直播间 #undefined」
- 该兜底字符串被渲染到按钮文案中，导致重复和未定义值显示

**修复方案**：
- 字段解耦：区分「内容标题」和「按钮文案」两个独立用途，不再复用同一字段
- 按钮文案简化：移除对 `title` 的依赖，按钮本身不需要重复带直播间名称
- 兜底策略：字段缺失时显示简洁的默认文案，而非拼接未定义值

**关键文件**：
- `frontend/h5/src/pages/Follow/index.tsx` — 收藏列表页面，按钮文案渲染逻辑
- `frontend/h5/src/pages/Follow/__tests__/Following.test.tsx` — 回归测试

**教训**：
- 避免将同一字段用于多个语义不同的展示位置，尤其是涉及兜底/默认值时
- 按钮文案应保持简洁，不需要重复上下文已明确的信息（如「进入直播间」按钮无需再带直播间名称）
- 兜底字符串应经过 review，避免包含变量占位符（如 `#undefined`）直接暴露给用户

**来源**：session:6a214cd02ec60aa1a7396fd0

### H5 首页竞拍卡片图片字段兼容修复 (Home Auction Card Image Field Compatibility)

**问题背景**：管理端配置的竞拍品图片在管理端能正常显示，但在移动端首页卡片显示"暂无图片"。

**根因分析**：
- 管理端商品接口返回 `images[]` 数组字段
- 移动端首页竞拍列表接口返回的是 `product.image` 单字符串字段
- 移动端 `Home/index.tsx` 的 `getFirstImage` 函数只从 `product.images` 取图，没有兼容 `product.image`

**修复方案**：
1. **类型扩展**：`ProductSummary` 类型新增 `image?: string` 字段
2. **取图逻辑兼容**：`getFirstImage` 函数优先读取 `product.images[0]`，不存在时降级到 `product.image`

**关键代码模式**：
```typescript
// 类型定义
interface ProductSummary {
  // ... 其他字段
  images?: string[];  // 管理端使用的数组字段
  image?: string;     // 列表接口返回的单图字段（新增兼容）
}

// 取图函数兼容处理
function getFirstImage(product: ProductSummary): string | undefined {
  return product.images?.[0] ?? product.image;
}
```

**TDD 执行**：
1. **Red**：新增测试用例模拟后端只返回 `product.image`，断言渲染成 `<img>` 而非"暂无图片"
2. **Green**：修改类型定义和取图逻辑，让测试通过
3. **验证**：`npm run build` 确认 TypeScript 编译和打包正常

**来源**：session:6a22c45f2ec60aa1a73a1f2e

---

### H5 历史竞拍记录图片字段兼容修复 (History Auction Image Field Compatibility)

**问题背景**：移动端"我的竞拍记录"页面（`/history`）中，商品图片加载不出来，显示"暂无图片"。

**根因分析**：
- 历史记录前端组件只识别 `product_image`、`image`、`product.image`、`product.images[0]` 等特定字段名
- 后端历史 DAO 返回的商品图片字段可能是 `images` JSON 数组或其他命名，与前端期望不一致
- 字段命名不一致导致前端无法正确获取图片 URL

**修复方案**：
1. **后端接口扩展**：在历史记录接口响应中补充 `product_image` 字段，与前端期望对齐
2. **前端取图逻辑增强**：`getFirstImage` 兼容函数应覆盖更多字段命名变体（`images[]`、`image`、`product_image`、`cover_image` 等）

**关键代码模式**：
```typescript
// 历史记录响应类型
interface AuctionHistoryItem {
  // ... 其他字段
  product_image?: string;  // 明确提供单图字段供前端直接使用
  product?: {
    images?: string[];
    image?: string;
  };
}

// 增强版取图函数（兼容多种字段命名）
function getFirstImage(item: AuctionHistoryItem): string | undefined {
  return item.product_image 
    ?? item.product?.images?.[0] 
    ?? item.product?.image
    ?? item.image;
}
```

**来源**：session:6a242a7d00057ea64ca26118

---

### H5 竞拍记录未读标识逻辑 (Auction History Unread Indicator)

**问题背景**：「我的竞拍」页面显示未读数量，但进入「竞拍记录」页面后，卡片无法区分已读和未读状态。

**核心设计决策**：
- **未读定义**：`待处理中标订单` = 未读（`status=0`，待支付状态）
- **已读定义**：`已处理中标订单` = 已读（`status=1`，已支付/已取消等完结状态）
- **非中标记录**：不参与未读/已读区分，只展示「未中标」状态

**数据契约**：
- 后端 `/orders/history` 接口必须返回 `status` 字段（订单状态）
- 前端根据 `isWon(record) && isPendingOrderStatus(record.status)` 判定是否为待处理中标

**卡片状态映射**：
| 条件 | 卡片样式 | 文案 |
|------|----------|------|
| `isWon && isPending` | 高亮边框 + 待处理标签 | 「待处理 / 竞拍成功 / 去支付」 |
| `isWon && !isPending` | 普通样式 | 「已处理 / 竞拍成功 / 查看结果」 |
| `!isWon` | 普通样式 | 「未中标」 |

**关键实现点**：
```typescript
// 待处理状态判定
const isPendingOrderStatus = (status: number) => status === 0; // 0=待支付

// 卡片渲染分支
const isPendingWonRecord = isWon(record) && isPendingOrderStatus(record.status);
```

**后端契约修复**：
- `backend/product/dao/history.go` 的 `UserHistoryItem` 结构体需添加 `Status` 字段
- SQL 查询需从 `orders` 表获取 `status` 字段
- 修复 SQLite 兼容性：`DATE_FORMAT` 改为 `created_at as created_at`

**来源**：session:6a2464ce00057ea64ca286e5

---

### H5 竞拍结果页交互优化 (Auction Result Page UX Enhancement)

**问题背景**：竞拍结果页存在以下体验问题：
1. 本人中标时显示两个 disabled 按钮（「订单待生成」「订单生成中」），无明确行动指引
2. 中标人名称显示乱码（mojibake）
3. 出价时间标签与值的位置不符合「key 在下，value 在上」的预期

**优化方案**：

**1. 按钮逻辑重构**
- **本人中标**：只保留一个「去支付」按钮，点击弹出「支付链路待完善」提示弹窗
- **非本人中标**：只保留「返回首页」链接按钮
- **移除 disabled 按钮**：不再显示「订单待生成」「订单生成中」等无意义状态

**2. 支付弹窗设计**
- 背景蒙层：`backdrop-filter: blur(12px)` 毛玻璃效果
- 弹窗主体：质感渐变背景 + 双层光影 + 顶部金色渐变光束
- 文本可读性：强制使用 `var(--text-primary)`，字体放大到 15px
- 按钮样式：幽灵按钮（Ghost Button）质感，微金色背景 + 金色细边框

**3. 乱码修复**
- 中标人名称统一使用 `repairUtf8Mojibake` 修复后再渲染
- 头像首字也使用修复后的名称首字

**4. 信息层级调整**
- 出价时间区域：值（时间）在上，标签（"出价时间"）在下
- DOM 顺序：`strong`（值）在前，`span`（标签）在后

**关键文件**：
- `frontend/h5/src/pages/Result/index.tsx` — 结果页主组件
- `frontend/h5/src/pages/Result/Result.module.css` — 弹窗样式

**来源**：session:6a2464ce00057ea64ca286e5

---

### H5 首页竞拍卡片观看人数显示 (Home Auction Viewer Count)

**决策背景**：用户希望在首页普通竞拍卡片上显示真实的直播间观看人数，而非静态占位。

**核心决策**：
- **采用「真实快照人数」方案**：后端通过扩展 batch 接口批量回填 `viewer_count`，前端仅做展示，不引入 WebSocket 实时订阅
  - 避开 N+1 请求风险和多连接 WebSocket 的复杂度
  - 保证数据真实性，同时控制工程复杂度
- **判定主语明确化**：显示与否只看 `auction.status`（前端 `statusInfo.live`），与 `live_stream.status` 无关
- **后端不做 status 过滤**：batch 接口对**所有** status 的直播间都返回真实 viewer_count，过滤逻辑完全交前端处理，保持接口职责单一

**技术边界与约束**：
- **直播间维度语义**：`viewer_count` 是直播间维度而非竞拍维度，同一直播间挂多个竞拍时多张卡片显示相同人数是**预期行为**
- **Batch 上限无需分批**：`internalLiveStreamBatchMaxIDs=200`，而首页 `page_size=20` 去重后 stream id ≤20，永不触发上限，**不实现分批逻辑**
- **降级策略**：product 服务不可用时，后端聚合层应降级为 `viewer_count=0` 并记录 Warn 级日志（每请求最多 1 条），严禁导致整个列表接口 5xx
- **软依赖原则**：列表项非核心元数据（如观看人数）应设为软依赖，后端聚合时若该字段查询失败应降级并记录日志，严禁导致整个列表接口报错

**设计审核关键发现**（方案定稿前必须澄清的边界）：
1. **「进行中」的判定主语**：存在两个 status（`auction.status` 和 `live_stream.status`），必须明确显示条件只看 `auction.status`（即前端 `statusInfo.live`），与 `live_stream.status` 无关
2. **同直播间多竞拍卡片的 viewer_count 重复**：`viewerByStream` 是直播间维度，同 `live_stream_id` 的多个 auction 卡片会显示同一数值，这是预期行为而非 bug
3. **Batch 接口职责范围**：batch 接口对所有 status 的直播间都返回真实 viewer_count，不过滤、不聚合，保持职责单一可复用

**实现要点**：
- `auction-service` 列表组装阶段调用 `product-service` 内部 batch 接口批量获取直播间摘要
- 使用 Redis 优先的批量回填逻辑（Redis 命中则跳过 DB 查询）
- 前端展示逻辑：`statusInfo.live && viewer_count > 0` 时才显示观看人数角标

**设计文档**：`docs/superpowers/specs/2026-06-10-h5-home-auction-viewer-count-design.md`

**来源**：session:6a28a0a30bfcee1b04fc5ce6

### 观看人数零值显示策略 (Viewer Count Zero Display)

**决策背景**：用户要求直播间进行中时，即使观看人数为 0 也要显示 "0 观看"，而非隐藏角标。

**核心决策**：
- **展示条件变更**：从 `statusInfo.live && auction.viewerCount > 0` 改为仅判断 `statusInfo.live`
- **零值展示**：`viewerCount === 0` 时显示 `"0 观看"`，非零时仍使用 `toLocaleString()` 格式化（如 `"1,314 观看"`）
- **后端语义不变**：`viewer_count` 字段含义和降级策略保持不变

**TDD 执行模式**：
1. **Red**：先修改 Jest 测试断言，期望显示 `"0 观看"`，验证当前实现会失败
2. **Green**：修改生产代码展示条件，让测试通过
3. **Refactor**：保持改动最小化，仅调整展示层逻辑

**来源**：session:6a28a0a30bfcee1b04fc5ce6

### 一口价上架入场动画设计决策 (Fixed Price Listing Animation Design)

**决策背景**：商家一口价商品上架时需要添加入场动画，动画结束后过渡到右下角固定位置，需适配日/夜双主题。

**三版方案对比**：

| 方案 | 风格 | 动画路径 | 特点 |
|------|------|----------|------|
| A | 强吸引力 | 中心放大 → 抛物线飞入右下角 | 视觉冲击强，适合强调新品上架 |
| B（最终选定）| 平滑滑入 | 顶部滑入停留 → 直线匀速缩小至右下角 | 体验温和，过渡自然，与直播节奏协调 |
| C | 礼物掉落感 | 侧边弹跳入场 → 旋转吸入右下角 | 互动感好，游戏化风格 |

**技术实现要点**：
- 使用 CSS transform + opacity 实现动画，确保 60fps
- 支持 `prefers-reduced-motion` 媒体查询降级
- 使用 CSS Variables 实现日/夜双主题切换

**触发源正确性**（代码审查发现）：
- 必须使用 WebSocket `fixed_price_listed` 实时上架事件触发
- 不能使用"列表从空变非空"推断，避免 REST 加载的已有商品误判为新上架
- **Hook 层信号收敛**：将触发信号收敛到 `useFixedPriceItems` 的 `fixed_price_listed` 事件输出，而非从 REST 后的列表差异推断

**代码审查关键发现**：
1. **触发源错误**：原逻辑"列表从空变非空就动画"会把初始 REST 加载出的已有一口价商品误判为"新上架"
2. **动画位移性能**：应使用 `transform/opacity` 实现位移，并补充 `prefers-reduced-motion` 降级
3. **测试覆盖**：需补"REST 初始商品不播放动画 / WS 上架才播放"的回归测试

**修复方案**：
- 将触发信号收敛到 `useFixedPriceItems` 的 `fixed_price_listed` 事件输出
- 动画使用 CSS transform 实现，添加无障碍降级支持
- 补充回归测试锁定动画触发时机

**关键文件**：
- `frontend/h5/src/components/FixedPriceCard/` — 一口价卡片组件
- `frontend/h5/src/pages/Live/Live.module.css` — 动画样式
- `frontend/h5/src/hooks/useFixedPriceItems.ts` — 事件处理 Hook

**来源**：session:6a2707bb0bfcee1b04fb6b6f

---

### 一口价动画时序 Bug (Fixed Price Animation Timing Bug)

**问题背景**：一口价商品出现的时机不对，动画还没播放完，商品卡片就提前出现在右下角列表。

**根因分析**：
- `useFixedPriceItems` 在收到 `fixed_price_listed` WebSocket 事件时，立即将商品写入 `items` 状态
- 页面直接渲染 `fixedPriceItems`，导致动画层和右下角卡片会并存
- 正确边界应在 `LiveRoomSlide` 层做「展示门控」：listed 事件先进入动画队列，动画完成后才放行到列表

**修复方案**：
1. **状态分离**：动画状态和列表状态分离管理
2. **门控逻辑**：`fixed_price_listed` 事件触发时，先启动入场动画，动画完成后再将商品加入列表状态
3. **时序保证**：确保动画播放期间，商品卡片不会提前出现在右下角固定位置

**关键代码模式**：
```typescript
// LiveRoomSlide.tsx - 动画完成回调
const handleAnimationComplete = (item: FixedPriceItem) => {
  // 动画完成后才将商品加入列表
  dispatchFixedPriceItems({ type: 'ADD_ITEM', payload: item });
};

// 事件监听 - 只触发动画，不直接更新列表
useEffect(() => {
  const handler = (msg: WSMessage) => {
    if (msg.type === 'fixed_price_listed') {
      // 先播放动画，动画完成后再更新列表
      playEntryAnimation(msg.payload, handleAnimationComplete);
    }
  };
  ws.onMessage(handler);
}, []);
```

**与「卡片未实时显示」问题的区别**：
- 「动画时序 Bug」：卡片出现太早（动画未结束就显示）
- 「卡片未实时显示」：卡片出现太晚（需要刷新才显示）
- 两者修复方向相反，但都需要正确处理动画与列表状态的时序关系

**来源**：session:6a274d560bfcee1b04fba6a8

---

### H5 直播间宝箱进度条 UI 设计决策 (Live Room Treasure Box Progress Bar)

**决策背景**：直播间「看直播领宝箱」功能需要设计一个悬浮进度条组件，展示用户观看时长进度和可领取状态，需适配日/夜双主题。

**设计探索流程**：
使用 `ui-design-trio` Skill 进行三版方案推演，通过浏览器可视化预览日/夜主题效果后由用户选定：

| 方案 | 风格 | 设计逻辑 | 适用场景 |
|------|------|----------|----------|
| **A · 现代玻璃态 (Glassmorphism)** | 半透明毛玻璃 | 深色半透明底板+流光渐变进度条，悬浮不遮挡直播画面 | 常规直播间，追求精致感 |
| B · 游戏化弹动 (Gamified) | 高对比立体 | 去除底板束缚，加粗进度条+放大宝箱元素，可领状态时弹动幅度大 | 大促活动，最大化刺激点击 |
| C · 极简质感 (Minimalist) | 抽象线条 | 摒弃具象宝箱图形，采用极细进度轴+发光节点 | 高端品牌直播间，清爽克制 |

**最终选定**：**方案 A（现代玻璃态）**
- 采用深色半透明毛玻璃底板（`backdrop-filter: blur`）
- 细长进度条带有流光渐变色
- 宝箱节点在可领状态下有呼吸动画
- 整体质感轻量、现代，不喧宾夺主

**技术实现要点**：
- 组件名：`TreasureProgressBar`
- 位置：直播间底部悬浮，与底部导航/出价区不重叠
- 主题适配：使用 CSS Variables 支持 `data-theme="dark/light"` 切换
- 动画：进度填充使用 CSS transition，宝箱呼吸使用 `@keyframes breathe`

**交互体验设计**：
- 宝箱节点处于可领状态时展示呼吸动画吸引点击
- 点击领取后触发 `+ 300` 金色上浮动画
- 总金币余额实时滚动更新
- 支持 `prefers-reduced-motion` 媒体查询降级

**来源**：session:6a26f8b10bfcee1b04fb5768

---

### H5 直播间聊天发送失败问题 (Live Chat Send Failure)

**问题背景**：直播间快捷聊天气泡和输入框点击后无法发送消息。

**根因分析**：
- 主直播间 WebSocket 在 `active/auctionId` 变化时创建，但未把 `token` 纳入依赖
- 用户先未登录进入直播间，再通过 DemoConsole 自动登录后，聊天 WS 仍是旧连接
- `sendChat` 因缺少有效连接或 `liveStreamId` 返回 false，导致消息发送失败

**修复方案**：
- WebSocket Hook 依赖数组必须包含 `token`，确保登录态变化后触发重连
- `sendChat` 调用前校验连接状态和 `liveStreamId` 有效性

**关键代码模式**：
```typescript
// LiveRoomSlide.tsx - WS 依赖
useEffect(() => {
  if (!token || !liveStreamId) return;
  const ws = createWebSocketConnection({ token, liveStreamId });
  // ...
}, [token, liveStreamId]);  // token 变化时重建连接
```

**来源**：session:6a26ce7a0bfcee1b04fb3ddc

---

### H5 直播间弹幕功能 (Live Chat)

**功能概述**：直播间支持实时弹幕聊天，包括消息发送、接收、历史回放和频控。

**技术架构**：
- **双 Room 模型**：物理直播间 (`live_stream_id`) 与逻辑房间 (`room_id`) 分离，支持一个直播间内多场次竞拍隔离
- **WebSocket 消息类型**：
  - `chat_send` — 客户端发送弹幕
  - `chat_message` — 服务端广播弹幕
  - `chat_error` — 发送失败错误码 (40001/40002/40003)
- **频控策略**：Redis 实现，单用户 1 秒内限 1 条，单房间 1 秒内限 10 条
- **内容校验**：长度 1-50 字符，黑词过滤（如"微信""QQ"等）

**飘屏规则 (Price Flair)**：
- **R1 高价飘屏**：出价金额 ≥ 当前价 150% 且 ≥ 1000 元时触发
- **R2 成交飘屏**：竞拍结束时最高出价者触发
- **R3 一口价飘屏**：一口价购买成功时触发（`fixed_price_flair`）

**关键实现细节**：
- 历史回放：新用户进入直播间时推送最近 ≤100 条消息
- 房间隔离：不同 `live_stream_id` 的消息互不可见
- 用户身份：从 JWT 派生 `user_id` 和 `user_name`，拒绝 guest (UserID==0)

**本地开发 WS 代理配置**：
```ts
// vite.config.ts
proxy: {
  '/api/v1/ws': { target: 'ws://localhost:8083', ws: true, changeOrigin: true },
  '/api': { target: 'http://localhost:8080', changeOrigin: true },
}
```

**来源**：session:6a1c56f7959156a8dfc84fae

### H5 直播间聊天面板布局定位 (LiveChat Panel Positioning)

**问题背景**：直播间底部出价区域出现一条"黑线"横贯输入框，影响视觉体验。

**根因分析**：
- 选中的元素是 `LiveChat` 的输入栏 `.inputBar`，位于 `ChatPanel.module.css`
- `.inputBar` 设置了 `background: linear-gradient(to top, rgba(0, 0, 0, 0.6), transparent)`，底部 60% 透明度黑色向上渐变到透明，在深色背景上形成黑色横带
- 更深层原因：`.inputBar` 的父级 `.panel` 使用了 `position: absolute`，而 `ChatPanel` 被放在竞拍抽屉 `.sheet` 内部
- `.sheet` 也是 `position: absolute`，导致 `ChatPanel` 相对整个抽屉绝对定位，而非正常文档流
- 结果是聊天输入栏的选中边界/输入栏位置错乱，横穿出价区域

**解决方案**：
- 将 `ChatPanel` 从"悬浮绝对定位"改为抽屉内的普通布局
- `.panel` 改为 `position: relative`
- `.inputBar` 改为 `position: static`
- 移除黑色渐变背景，避免视觉干扰

**关键文件**：
- `frontend/h5/src/components/LiveChat/ChatPanel.module.css` — 样式定义
- `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` — 组件使用位置

**教训**：组件复用时需检查定位上下文，`position: absolute` 在嵌套绝对定位容器中会产生意外的视觉叠加问题。

**来源**：session:6a204726867f95f321be3c6f, session:6a1fffc7867f95f321be0ce6

### H5 直播间聊天区域布局重构 (LiveChat Overlay Refactor)

**问题背景**：原聊天面板位于底部抽屉内，只有打开抽屉才能看到聊天记录，不符合主流直播平台的交互习惯。

**核心决策**：
- 将聊天从底部抽屉内移到直播画面层的常驻 overlay
- 位置占据商品 Dock 左上方区域，右侧留给一口价商品浮层
- 聊天记录做成无外框的半透明气泡，输入框做成短圆角毛玻璃条

**布局要点**：
- 聊天区域作为直播画面的常驻 overlay，不依赖抽屉打开状态
- 输入框固定在商品 Dock 上方，与一口价卡片左右分区
- 聊天记录列表采用半透明样式，不遮挡直播画面

**关键文件**：
- `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` — 聊天面板挂载位置
- `frontend/h5/src/components/LiveChat/ChatPanel.tsx` — 聊天面板组件
- `frontend/h5/src/components/LiveChat/ChatPanel.module.css` — 样式定义

**来源**：session:6a2165022ec60aa1a73987df

### H5 直播间聊天用户名乱码修复 (Chat Username Mojibake Fix)

**问题背景**：聊天列表中的用户名显示乱码（如 `è€èœ...`），但商品名/主播名显示正常。

**根因分析**：
- `ChatBubble` 组件直接渲染 `msg.user_name`，没有经过 `repairUtf8Mojibake` 修复
- 商品名/主播名在渲染前已统一调用修复函数，但聊天消息渲染边界遗漏

**修复方案**：
- 在 `ChatBubble` 组件中对 `user_name` 调用 `repairUtf8Mojibake` 后再渲染
- 保持与商品名/主播名一致的修复策略

**关键文件**：
- `frontend/h5/src/components/LiveChat/ChatBubble.tsx` — 聊天消息渲染组件

**教训**：多入口数据渲染需检查所有字段的编码修复，避免遗漏导致用户体验不一致。

**来源**：session:6a2165022ec60aa1a73987df

---

### H5 直播间飘窗用户名乱码修复 (Live Flair Username Mojibake Fix)

**问题背景**：点天灯之后的飘窗提醒中，用户名显示乱码（如 `è€èœ...`）。

**根因分析**：
- 前端已有 `repairUtf8Mojibake` 工具函数用于修复直播弹幕用户名乱码
- 但点天灯提醒走的是另一条 WebSocket/飘窗链路，没有复用这个归一化函数
- 导致飘窗直接渲染后端返回的原始字符串，出现编码问题

**修复方案**：
- 在飘窗组件中对 `user_name` 字段统一调用 `repairUtf8Mojibake` 修复后再渲染
- 检查其他飘窗/飘屏组件（如 `FixedPriceFlair`、直播间通知 toast 的 `title/content`）是否存在同类问题
- 确保所有「飘窗/飘屏/Toast/Notice/Flair」类组件渲染外部文本时都经过编码修复

**关键文件**：
- `frontend/h5/src/components/LiveFlair/` — 点天灯飘窗组件
- `frontend/h5/src/components/FixedPriceFlair/` — 一口价购买飘屏组件
- `frontend/h5/src/utils/repairUtf8Mojibake.ts` — 编码修复工具函数

**同类问题审查清单**：
- [ ] 点天灯飘窗 (`LiveFlair`) — 用户名修复
- [ ] 一口价购买飘屏 (`FixedPriceFlair`) — 用户名/商品名修复
- [ ] 直播间通知 Toast — `title`/`content` 字段修复
- [ ] 其他飘屏/飘窗组件

**教训**：
- 编码修复必须在所有渲染外部文本的边界统一接入，不能假设某条链路数据是"干净的"
- 新增飘窗/飘屏类组件时，默认接入 `repairUtf8Mojibake` 应作为 checklist 项

**来源**：session:6a24568600057ea64ca279d0

### H5 直播间一口价卡片布局优化 (Fixed Price Card Layout)

**问题背景**：一口价商品卡片位置不佳（压在左侧大卡片上）、标题样式不明显、库存标签溢出。

**布局演进**：
1. **初始位置**：铺满左侧并压在聊天区域
2. **调整方案**：移到右下角悬浮位（输入框左侧、商品简介 Dock 上方）
3. **最终布局**：右侧 156px 宽度，与左侧聊天框（210px）保留 6-10px 间隙

**样式优化要点**：
- 卡片内部采用紧凑电商布局：小图 + 强标题 + 大价格 + 全宽按钮
- 「限时一口价」badge 横跨图片和标题两列，位于卡片第一行
- 库存标签单独占一行并截断，避免和价格挤在一行溢出
- 禁止横向滚动条，避免白条出现

**关键文件**：
- `frontend/h5/src/components/FixedPriceCard/index.tsx` — 一口价卡片组件
- `frontend/h5/src/components/FixedPriceCard/index.module.css` — 样式定义
- `frontend/h5/src/pages/Live/Live.module.css` — 直播间布局样式

**来源**：session:6a2165022ec60aa1a73987df

### H5 一口价上架后卡片未实时显示问题 (Fixed Price Listed Card Missing)

**问题背景**：一口价动画出现并收到 WebSocket 事件后，右下角一口价卡片没有出现，需要刷新页面才显示。

**根因分析**：
- 本地 `fixed_price_listed` 事件只触发 `LiveRoomSlide` 播放入场动画
- 动画播放后没有将新商品写入 `useFixedPriceItems` 的 `items` 列表状态
- 刷新后 REST 请求拉取新数据，卡片才出现

**修复方案**：
- 将本地上架事件接入 `useFixedPriceItems` 的同一个 reducer
- 让「动画播放」和「卡片列表更新」共用同一个状态源
- 确保动画结束后卡片立即出现在列表中

**关键代码模式**：
```typescript
// useFixedPriceItems.ts - reducer 处理本地事件
case 'LOCAL_ITEM_LISTED':
  return {
    ...state,
    items: [action.payload, ...state.items],  // 新商品置顶
  };

// LiveRoomSlide.tsx - 事件监听
useEffect(() => {
  const handler = (msg: WSMessage) => {
    if (msg.type === 'fixed_price_listed') {
      // 同时触发动画和更新列表
      playEntryAnimation(msg.payload);
      dispatchFixedPriceItems({ type: 'LOCAL_ITEM_LISTED', payload: msg.payload });
    }
  };
  ws.onMessage(handler);
}, []);
```

**来源**：session:6a26ce7a0bfcee1b04fb3ddc

---

### H5 一口价确认弹窗优化 (Fixed Price Purchase Modal)

**问题背景**：一口价确认弹窗存在以下体验问题：
1. 文案泄露直播间号（`直播间 {liveStreamId} 限时一口价`）
2. 标题颜色不明显，看不清
3. 缺少商品图片展示

**优化方案**：
- **文案脱敏**：移除直播间号，改为「限时一口价」
- **视觉增强**：标题、商品名、价格标签做高对比，确认按钮采用强购买确认样式
- **商品图片**：弹窗左侧展示商品封面图，优先使用 `product_brief.cover_image`，无图时显示兜底占位

**关键文件**：
- `frontend/h5/src/components/FixedPricePurchaseModal/index.tsx` — 确认购买弹窗

**来源**：session:6a2165022ec60aa1a73987df

### H5 余额不足弹窗与全局 Toast 冲突 (Insufficient Balance Dialog Toast Conflict)

**问题背景**：点击抢购后，余额不足弹窗正常弹出，但顶部同时出现「发生未知错误」的全局 Toast。

**根因分析**：
- 余额不足返回 HTTP 402，弹窗组件能识别并打开「余额不足」二级弹窗
- 但底层 `post()` 默认 `showError=true`，在错误抛给组件前已触发全局错误 toast
- 通用错误映射没有 402，所以显示成「发生未知错误」

**修复方案**：
- 在一口价购买 API 层关闭请求层通用 toast（`showError: false`）
- 让业务弹窗独占处理 402，避免双层提示

**关键代码模式**：
```typescript
// 关闭全局错误提示，由业务层处理
await fixedPriceApi.purchase(itemId, { showError: false });
```

**关键文件**：
- `frontend/h5/src/api/fixedPrice.ts` — API 封装
- `frontend/h5/src/components/FixedPricePurchaseModal/index.tsx` — 弹窗错误处理

**来源**：session:6a2165022ec60aa1a73987df

### 观看人数数据源差异 (Viewer Count Data Source Discrepancy)

**问题背景**：用户反馈直播间显示"有人观看"，但首页卡片显示"0 观看"，存在数据源不一致。

**根因分析**：
- **直播间页**：使用 `live_presence_update` WebSocket 消息中的 `viewer_count`，来自 Presence 系统的实时在线人数
- **首页卡片**：使用 product-service 批量接口返回的 `live:viewer:{id}` Redis 快照或 DB 数据

**关键发现**：
- Presence 系统可能没有回写 `live:viewer:{id}` Redis key，导致首页读取的仍是旧快照
- 两个数据源独立维护，存在同步延迟或写入缺失的可能

**排查路径**：
1. 检查 Presence 系统是否写入 `live:viewer:{id}`
2. 检查 product-service 批量接口是否正确读取 Redis 优先、DB 兜底
3. 确认两个数据源的更新频率和一致性策略

**教训**：涉及跨系统数据一致性时，需明确写入路径和回写机制，避免"有数据但不同源"的隐性 bug。

**来源**：session:6a28a0a30bfcee1b04fc5ce6

### 直播间 Presence 实时在线用户接入 (Live Presence Integration)

**功能概述**：H5 直播间页通过 WebSocket 接入实时在线用户列表（头像、用户名、观看人数），替换前端模拟数据。

**数据来源**：
- **HTTP 初始加载**：`GET /api/v1/live-streams/:id` 返回 `viewer_count` 和 `host_avatar` 作为初始值
- **WebSocket 实时更新**：`live_presence_update` 消息推送最新在线状态

**消息处理要点**：
```typescript
// WebSocket 消息监听
ws.onMessage((msg) => {
  if (msg.type === 'live_presence_update') {
    // 更新观看人数
    setViewerCount(msg.payload.viewer_count);
    // 更新在线用户头像列表
    setViewers(msg.payload.viewers || []);
  }
});
```

**隐私安全边界**：
- 未鉴权用户（游客模式）不会收到包含实名信息的 `viewers` 列表
- 前端需处理 `viewers` 字段缺失或为空数组的情况

**测试覆盖要点**：
- 正常鉴权用户能收到完整 presence 更新
- 用户头像列表按 `user_id` 去重显示
- 连接断开后重新建立时能正确恢复 presence 状态

**来源**：session:6a26bb690bfcee1b04fb3791

### 列表接口非核心元数据软依赖原则 (List Interface Soft Dependency Principle

**决策背景**：首页竞拍列表需要聚合直播间观看人数等非核心元数据，这些数据的查询失败不应导致整个列表接口 5xx。

**核心原则**：
1. **软依赖定义**：非核心元数据（如 `viewer_count`）是「锦上添花」而非「必不可少」，查询失败时应降级处理而非报错
2. **降级策略**：依赖服务不可用时，字段降级为默认值（如 `viewer_count=0`），并记录 Warn 级日志（每请求最多 1 条）
3. **接口稳定性**：列表接口的核心职责是返回主数据（竞拍列表），非核心元数据查询失败不应破坏主流程

**实现模式**（以观看人数批量回填为例）：
```go
// 批量获取直播间摘要（Redis 优先/DB 兜底）
summaryMap, err := liveStreamClient.BatchGetSummary(ctx, streamIDs)
if err != nil {
    // 降级：记录 WARN 日志，返回空 map，让 viewer_count 默认为 0
    log.Printf("[WARN] batch get live stream summary failed: %v", err)
    summaryMap = make(map[int64]*LiveStreamSummary)
}
// 组装响应时，summaryMap 中不存在的 ID 自动使用零值
```

**前端配合**：
- 展示逻辑应处理零值/空值情况（如 `viewer_count > 0` 才显示角标）
- 不过度依赖非核心字段的存在性

**来源**：session:6a28a0a30bfcee1b04fc5ce6

### H5 直播间底部导航安全区域适配 (Live Room Bottom Nav Safe Area)

**决策背景**：直播间画面被底部导航栏遮挡，需要调整直播视口高度避免遮挡。

**核心决策**：
- **方案 B（推荐）**：直播视频容器的 `bottom` 停在底部导航栏上边框，而非继续铺到屏幕底部
- 不隐藏底部导航，保持 `/live` 作为 Tab 页的结构完整性
- 商品卡片仍按直播页底部定位，但直播页底部本身已经停在导航上边框

**技术实现**：
- 将「底部导航高度」抽成统一语义变量
- 直播页内容高度 = 视口高度 - TabBar 高度
- 通过 CSS 调整 `.liveContainer` 或视频容器的 `height`/`bottom` 属性

**来源**：session:6a1fd603867f95f321bde97f

### H5 直播间空状态设计 (Live Room Empty State Design)

**决策背景**：当用户进入直播间但没有正在进行的竞拍时，需要优雅的空状态引导，而非空白页面或错误提示。

**核心决策**：
- **采用「预告时间线」方案（方案 B）**：展示即将开播的直播间列表，将用户留在直播转化链路
- **数据查询**：`auction.status = 0` 且 `start_time > now`，按 `start_time ASC` 取最近 2 条
- **条目交互**：整行点击跳转商品详情页 `/detail?id={auctionId}`，订阅按钮独立处理并阻止冒泡
- **降级策略**：无预告或接口失败时，显示主按钮「去首页看拍品」

**设计要点**：
- 最多展示 2 条即将开播记录，避免信息过载
- 移除「全部预告」入口，保持界面简洁
- 订阅按钮调用商品开拍提醒接口 `productReminderApi.subscribe/list`
- 复用首页和商品详情已有的提醒状态逻辑

**UX 设计流程经验**：
- 使用 `brainstorming` Skill 进行需求澄清和三版方案推演（A-轻行动空态/B-预告时间线/C-推荐拍品卡）
- 通过浏览器视觉辅助预览日/夜双主题效果，降低决策成本
- 最终选择 B 方案的关键理由：「即将开播/预告场次」数据可从后端获取，空态不应只做逃离按钮，而应把用户留在直播转化链路

**来源**：session:6a22ba2e2ec60aa1a73a185a

---

### H5 直播间空态布局修复 (Live Room Empty State Layout Fix)

**问题背景**：直播间空态页在屏幕上半部分留下大块空白，推荐内容被压到下半屏，用户需要滑动才能看到。

**根因分析**：
- `.liveEmptyPage` 使用了 `align-items: center`，在手机容器高度较大时会把卡片垂直居中
- 导致推荐内容（即将开播列表）被推到屏幕下半部分，上半屏留白

**修复方案**：
- 将空态卡片从「垂直居中」改为「贴近顶部但保留安全区和呼吸感」
- 调整 CSS 使推荐内容进入首屏，消除上半屏空白

**关键代码模式**：
```css
/* 修复前：垂直居中导致上半屏空白 */
.liveEmptyPage {
  align-items: center; /* 问题根源 */
}

/* 修复后：贴近顶部 */
.liveEmptyPage {
  align-items: flex-start;
  padding-top: var(--safe-area-top, 44px);
}
```

**来源**：session:6a242bf200057ea64ca26264

### H5 直播间 Tab 入口逻辑 (Live Room Tab Entry Logic)

**决策背景**：用户从底部导航栏点击「直播间」Tab 进入 `/live` 时，原逻辑提示"请从首页或详情页进入"，体验不佳。应直接展示推荐的正在竞拍的直播间。

**核心决策**：
- **数据来源**：调用 `GET /api/v1/live-streams?page=1&page_size=20&status=1` 获取「直播中」的直播间列表
- **筛选逻辑**：前端筛选 `current_auction_id > 0` 的直播间，展示第一个可竞拍直播间
- **非推荐逻辑**：当前不是调用首页推荐接口，而是直播中列表 + 前端筛选，排序由后端控制

**技术边界**：
- 若当前没有正在竞拍的直播间（所有直播间 `current_auction_id <= 0`），需优雅处理空状态
- 该逻辑与从首页/详情页进入直播间不同，后者携带明确的 `auction_id` 参数

**未来优化方向**：
若产品需要「直播间 Tab」成为"推荐的正在竞拍直播间"，应后端提供明确语义接口：
```
GET /api/v1/live-streams/recommended?auction_status=active
```
或让现有接口支持：
```
GET /api/v1/live-streams?status=1&has_current_auction=true
```

**来源**：session:6a1fd687867f95f321bde9cd

### BadgeDot 与 Toast 组件 (User Touchpoint UI Components)

**组件概述**：用户触达系统的核心 UI 组件，支持红点提醒和 Toast 通知，采用 theme-ready 设计支持双主题切换。

**BadgeDot 组件**：
- **位置**：`frontend/h5/src/components/BadgeDot/`
- **Props 接口**：
  ```ts
  interface BadgeDotProps {
    count?: number;      // 数字，0 时不展示
    max?: number;        // 最大显示值，默认 99，超出显示 "99+"
    dot?: boolean;       // 纯红点模式（无数字）
    ariaLabel?: string;  // 无障碍标签
    className?: string;  // 自定义类名
  }
  ```
- **四种状态**：
  1. 纯红点：`dot={true}`
  2. 数字：`count={5}`
  3. 99+：`count={120}` 超出 max
  4. 不展示：`count={0}` 或 `count={undefined}`

**Toast 组件**：
- **位置**：`frontend/h5/src/components/Toast/`
- **调用方式**：
  ```ts
  // 兼容旧签名
  showToast(message: string, type?: ToastType, duration?: number)
  
  // 对象签名（推荐）
  showToast({
    type: 'success' | 'warning' | 'danger' | 'error' | 'info' | 'loading',
    title?: string,
    message: string,
    duration?: number,
    actionText?: string,
    onAction?: () => void
  })
  ```
- **特性**：
  - 最多 3 条堆叠显示
  - 支持标题、描述、操作按钮、关闭按钮
  - 触控按钮最小 44px 点击热区
  - 滑入/滑出动画

**主题适配**：
- 使用 CSS 变量：`--touchpoint-badge-bg`、`--touchpoint-badge-text`
- 通过 `data-theme` 属性切换，无需 JavaScript 介入
- 支持 `prefers-reduced-motion` 媒体查询降级

**来源**：session:6a1a57f7959156a8dfc8139e

### H5 底部导航红点与消息页状态同步修复 (BottomNav Badge Sync)

**问题背景**：底部导航栏的「我的」Tab 右上角显示红点（未读通知数），但用户点击进入个人中心/消息通知页并标记已读后，底部导航栏的红点没有同步消失。

**根因分析**：
- 底部导航 `BottomNav` 使用 `useTouchpointNotifications` Hook 获取未读汇总（红点数）
- 消息通知页 `NotificationsPage` 标记已读时只更新自身局部的 `unreadCount` 状态
- 两个组件各自缓存同一份未读汇总数据，没有建立状态同步机制
- 底部导航在 `/notifications` 页面隐藏，但状态仍保留旧值，返回后显示过期红点

**修复方案**：
建立前端内部失效事件机制：
1. 消息页标记已读成功后，广播 `touchpoint:summary:invalidate` 自定义事件
2. `useTouchpointNotifications` Hook 监听该事件，收到后重新拉取未读汇总
3. 所有消费该 Hook 的组件（包括 BottomNav）自动同步最新状态

**关键代码模式**：
```typescript
// NotificationsPage.tsx - 标记已读后广播失效事件
const markAllAsRead = async () => {
  await notificationApi.markAllAsRead();
  setUnreadCount(0);
  // 广播事件，通知所有监听者重新拉取汇总
  window.dispatchEvent(new CustomEvent('touchpoint:summary:invalidate'));
};

// useTouchpointNotifications.ts - 监听失效事件
useEffect(() => {
  const handleInvalidate = () => {
    refetchSummary(); // 重新拉取未读汇总
  };
  window.addEventListener('touchpoint:summary:invalidate', handleInvalidate);
  return () => {
    window.removeEventListener('touchpoint:summary:invalidate', handleInvalidate);
  };
}, []);
```

**测试验证**：
- Red 测试：验证 BottomNav 收到失效事件后重新拉取并清除红点
- Red 测试：验证消息页「全部已读」会发出失效事件
- Green 测试：修复后两个测试通过，红点在标记已读后自动消失

**来源**：session:6a217e2c2ec60aa1a739910a, session:6a23dfd42ec60aa1a73a5be7

### H5 首页与底部导航通知数不一致修复 (Home vs BottomNav Badge Count Mismatch)

**问题背景**：首页右上角通知按钮显示 84 个未读，但底部导航栏「我的」Tab 只显示 83 个，两处通知数不一致。

**根因分析**：
- 首页右上角通知按钮独立调用 `hotPull()` 后再读取 `/notifications/unread-count` 获取未读数
- 底部「我的」Tab 读取共享的 `/notifications/summary` 获取未读汇总
- 如果 `hotPull` 新拉到 1 条通知，首页会立即显示新数量，但底部导航仍停留在 hotPull 前的旧值
- 两个入口使用不同接口且没有建立同步机制，导致数据不一致

**修复方案**：
让首页红点也读取同一个 touchpoint summary，并在 `hotPull` 后触发共享汇总刷新：
1. 首页通知红点组件改用 `useTouchpointNotifications` Hook 获取未读数（与底部导航共享同一数据源）
2. `hotPull` 完成后广播 `touchpoint:summary:invalidate` 事件
3. 所有消费该 Hook 的组件自动重新拉取最新汇总，确保首页和底部导航数字一致

**关键原则**：
- 同一业务语义（未读通知数）应使用统一的数据源和 Hook
- 避免不同组件各自调用独立接口获取同一语义的数据
- 热拉（hotPull）等增量更新操作后必须触发共享数据刷新

**测试验证**：
- Red 测试：`hotPull` 之后首页红点必须来自共享 summary 的最新值，而非自己单独读取 `unread-count`
- Green 测试：修复后首页和底部导航通知数保持一致

**来源**：session:6a23dfd42ec60aa1a73a5be7

---

### H5 底部导航与页面内未读数口径对齐 (BottomNav vs Page Unread Count Alignment)

**问题背景**：底部导航栏「我的」Tab 显示 83 个未读，但进入「我的」页面后，「我的竞拍」显示 4 条未读，「消息通知」显示 83 条未读。两处统计口径不一致导致用户困惑。

**根因分析**：
- 底部导航「我的」Tab badge 直接使用 `useTouchpointNotifications().unreadTotal`，即消息通知总未读数（83 条）
- 「我的竞拍」卡片显示的未读数是竞拍成功（中标待支付）的数量（4 条）
- 两者统计维度不同：一个是「消息通知未读」，一个是「中标待支付订单」

**修复方案**：
1. **明确统计口径**：
   - 底部导航「我的」Tab badge 应显示「我的竞拍」未读数（中标待支付），而非消息通知未读数
   - 「我的竞拍」未读只统计竞拍成功（中标待支付）的维度，而非全部竞拍记录
2. **数据源切换**：
   - 底部导航改用 `/notifications/summary` 中的 `pending_payment_count` 或类似字段
   - 与「我的竞拍」卡片使用同一数据源，确保数字一致

**关键原则**：
- 同一入口的不同展示位置（底部导航 badge vs 页面内统计）必须使用同一统计口径
- 「我的竞拍」未读应聚焦「待支付」这一高优先级动作，而非全部竞拍记录
- 消息通知未读应独立展示，不与竞拍未读混用

**代码审查要点**：
- 检查 `useTouchpointNotifications()` 返回的 `unreadTotal` 是否被正确用于「我的」Tab
- 确认「我的竞拍」卡片的数字来源是中标待支付订单数，而非消息通知数
- 验证两处使用同一接口字段（如 `summary.pending_payment_count`）

**来源**：session:6a242ae400057ea64ca2617e

### H5 订单列表 UI 设计决策流程 (Order List UI Design Process)

**决策背景**：H5 用户端订单列表需要支持日/夜双主题，同时展示商品图片、名称、商家名称和订单状态，信息层级需要清晰区分。

**设计迭代过程**：
1. **初始静态页**：先实现静态视觉页，数据入口按后端真实 `/orders` 预留，日/夜主题用现有 token/class 体系
2. **浏览器预览选型**：通过独立 HTML 预览页展示多版本设计，用户在浏览器中直接比较后选定
3. **价格标签优化**：
   - 初始：纵向堆叠「成交价」标签和金额，占用过多垂直空间
   - 优化：改为横向紧凑排列「成交价 ¥110」在同一行，减少卡片底部高度
4. **卡片结构定型**：最终采用「票据凭证」风格（C 方案）
   - 顶部：「AUCTION ORDER」票据标识 + 日期
   - 主体：商品图片 + 商品名称 + 商家名称
   - 底部：成交价（竖线强调样式）、状态标签、查看订单按钮

**关键设计决策**：
- **价格标签样式**：采用「竖线强调」方案（B 方案），左侧竖线 + 「成交价」标签 + 金额横向排列
- **卡片风格**：票据凭证风格，顶部有票据标识，整体呈卡片式票据形态
- **信息层级**：商品图和名称作为主体，价格和操作作为底部行动区

**来源**：session:6a2416b73eefb8c530aa74a2

---

### H5 订单列表数据展示优化 (Order List Data Display Enhancement)

**问题背景**：订单列表最初只显示「商品 #id」和「竞拍场次 #id」的 fallback 文案，没有展示真实商品图片、名称和商家名称。

**根因分析**：
- 后端 `/orders` 接口最初只返回订单模型（`id/auction_id/product_id/final_price/status/created_at`）
- 用户侧接口没有 join 商品表，前端只能显示技术 ID
- Admin 订单列表已有 `orders LEFT JOIN products` 视图，但 H5 用户列表走纯 `OrderDAO.List`

**解决方案**：
1. **后端扩展**：用户订单列表接口改为「订单 + 本服务商品展示信息」的视图
   - 新增返回字段：`product_name`、`product_image`（首图）、`seller_name`
   - 通过 `orders -> products -> users` 链路获取商家名称
   - 不跨服务查 auction 表，保持服务边界清晰
2. **前端适配**：
   - 有商品图时渲染真实图片，无图时保留 LOT 占位
   - 商品名称替代「商品 #id」fallback
   - 商家名称替代「竞拍场次 #id」fallback

**数据归属原则**：
- `product_name`、`product_image`、`seller_name` 都属于 product-service 自己的数据
- 不需要跨服务查 auction 表或 live_streams 表
- 直播间名称不应在订单列表展示（数据所有权不属于 product-service）

**来源**：session:6a2416b73eefb8c530aa74a2

---

### H5 直播间出价排行视觉设计 (Bid Ranking Visual Design)

**决策背景**：原出价排行视觉冲击不足，需要增强紧迫感与社会认同感，同时突出用户自身出价状态。

**核心决策**：
- **采用「琉璃微光」风格（Glassmorphism）**：带光晕的毛玻璃效果 (`backdrop-filter: blur`)，保证文字对比度同时不遮挡直播画面
- **前三名荣誉展示**：使用金、银、铜徽章与文字渐变强化荣誉感；第一名数字采用「呼吸闪烁」动画
- **固定三席位**：始终展示前三名位置，空缺时显示「虚位以待」占位态（视觉变灰、透明度降低）
- **底部「我的出价」**：采用悬浮轻量卡片设计，半透明毛玻璃背景，左侧带微渐变圆形包裹排位数字

**呼吸动画规格**：
- 动画周期：2.5s
- 效果组合：透明度变化 + 轻微缩放（Scale 1 → 1.15）+ 光晕扩散（Box Shadow）
- 关键帧：`@keyframes breathe` 自定义动画，比 Tailwind `animate-pulse` 更丰富的视觉效果

**上榜用户亲切化**：
- 当前用户上榜（第一/二/三名）时，名称显示为「我自己 (当前领先)」而非真实用户名
- 上榜行添加微弱高亮边框，帮助用户一眼在榜单中找到自己

**占位态样式**：
- 名称显示「虚位以待」
- 价格显示 `-`
- 视觉变灰、透明度降低（约 0.4）

**来源**：session:6a24637400057ea64ca28666

---

### H5 直播间出价成功飘窗动画 (Bid Success Flair)

**功能概述**：用户出价成功后，在底部抽屉收起的同时触发全屏飘窗动画，提供即时视觉反馈，增强竞拍临场感。

**设计决策**：
- **与点天灯分离**：普通出价飘窗与点天灯飘屏是独立链路，互不干扰
- **动画时机**：抽屉收起后延迟 300ms 触发飘窗，避免与抽屉动画冲突
- **展示时长**：飘窗停留 2.8s 后自动上浮淡出消失
- **视觉风格**：深色磨砂玻璃 + 金色描边，金额使用流光渐变文本

**触发链路**：
```
用户点击出价 → bidApi.placeBid 成功 → closeSheet 收起抽屉 → 延迟 300ms → 触发 bidSuccessFlair → 展示 2.8s → 自动清理
```

**关键实现点**：
- 飘窗状态管理：`showBidSuccessFlair` + `lastBidAmount` 控制显示内容和动画
- 抽屉状态同步：通过 `isSheetOpen` 状态确保飘窗与抽屉动画节奏一致
- 点天灯隔离：`handleStartSkyLamp` 链路不触发普通出价飘窗，避免重复动画

**被超价通知链路（Bid Outbid Notification）**：
- **运行态触发位置**：`LiveRoomSlide` 的 WebSocket 通知回调里
- **链路**：`LiveRoomSlide` 建立房间级 WS 连接 → 注册 `ws.onNotification(...)` → 当通知 `type === 'bid_outbid'` 时映射成 toast 配置 → 调用 `showGlobalToast(...)` 弹出全局提示
- **关键代码位置**：
  - WebSocket 订阅：`frontend/h5/src/pages/Live/LiveRoomSlide.tsx` 的 `ws.onNotification` 注册点
  - 通知处理：同文件中将 `bid_outbid` 映射为 toast 配置的逻辑
  - 服务分发：`frontend/h5/src/services/websocket.ts` 收到 `message.type === 'notification'` 时转给 `onNotification` 订阅者
- **注意**：热拉 `hotPullNotifications()` 只更新通知中心/未读数，**不会触发被超价 toast**

**来源**：session:6a228ebf2ec60aa1a739fdcb

---

### H5 竞拍成功成交动画设计 (Auction Success Animation Design)

**功能概述**：竞拍成功后展示全屏成交动画，提供强烈的情绪价值和结果确认，增强用户成就感。

**设计方案选定**：「一锤定音」前置动画 + V1 经典欢庆卡片

**动画序列**：
1. **一锤定音 (The Gavel Smash)**：
   - 纯 SVG 绘制高质感拍卖锤（木质纹理 + 纯金镶边）
   - 夸张张力曲线（Cubic-bezier）：向后蓄力 → 极速猛砸
   - 落锤瞬间触发全屏镜头震动（Camera Shake）
   - 金色能量冲击波（Shockwave）从落锤点向外扩散

2. **彩带绽放 (Confetti Burst)**：
   - 80 个多形态彩带（长条/圆点/方块）漫天飞舞
   - 模拟真实物理重力：抛物线向上炸开 → 自然散落
   - 与卡片浮现时机精准衔接（0.8s 处卡片顺势浮现）

3. **V1 经典欢庆卡片 (Stamp & Pop)**：
   - 重力「印章」盖下效果
   - 彩屑喷射庆祝
   - 「一锤定音」的直白成就感反馈

**技术实现**：
- 纯 CSS 动画（`@keyframes` + `cubic-bezier`），零重型外部依赖
- 60fps 流畅度保证，跨端（移动端 Webview）兼容
- CSS Variables 支持日/夜双主题无缝切换

**UX 设计流程**：
1. 使用 `ui-design-trio` Skill 推演三版方案（V1 经典欢庆/V2 尊享高奢/V3 游戏动感）
2. 独立 HTML 原型验证（`bid_success_animations.html`），浏览器预览日/夜双主题
3. 用户确认后合入真实业务代码

**来源**：session:6a2464ce00057ea64ca286e5

---

### H5 竞拍结束 Section 设计决策 (Auction End Section Design)

**决策背景**：原竞拍结束悬浮卡片视觉冲击力不足，需要更强的视觉刺激来传达「竞拍结束」的仪式感和结果确认。

**设计探索流程**：
1. **首轮三版方案**（视觉刺激不足被否决）：
   - V1 极简杂志风：无边框文本流，依赖自然对比
   - V2 沉浸式毛玻璃通栏：底部 Drawer 态，占满屏幕宽度
   - V3 不对称光晕排版：左侧信息 + 右侧极限放大成交价

2. **次轮三版高刺激方案**（浏览器预览后选定）：
   - V4 呼吸极光风：持续呼吸脉冲的径向渐变光晕，56px 极限成交价
   - **V5 黑金典藏风（最终选定）**：金属卡片扫光动效 + 斜体 SOLD 水印 + 衬线排版
   - V6 冲击波视效：CSS skewX 斜切变形 + 弹性弹出动画

**V5 黑金典藏风核心特征**：
- **扫光动效**：持续循环的金属反光扫过卡片表面，使用 `var(--focus-ring)` 自适应日/夜主题
  - 夜间：金色光泽奢华质感
  - 日间：香槟色反光，明显但不喧宾夺主
- **SOLD 水印**：巨大斜体倾斜文字作为背景层，营造典藏证书感
- **衬线排版**：类似报纸/典藏证书的字体风格，传递贵气感
- **扫光材质自适应**：从固定白色半透明改为品牌辅助色高亮变量，解决 Light Mode 下白色扫光隐形问题

**UX 设计流程**：
1. 使用 `ui-ux-pro-max` 生成独立 HTML 原型（`preview.html`）
2. 浏览器预览日/夜双主题效果
3. 用户选定 V5 后细化扫光在 Light Mode 下的可见性
4. 确认后合入真实业务代码（`LiveRoomSlide.tsx` + `Live.module.css`）

**关键实现文件**：
- `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` — 竞拍结束渲染层 DOM 结构
- `frontend/h5/src/pages/Live/Live.module.css` — 黑金扫光动效和排版样式

**来源**：session:6a25658100057ea64ca2d626

### UX 原型验证模式 (UX Prototype Validation)

**问题背景**：在合入真实业务代码前，需要快速验证动画效果、交互节奏和视觉方案，避免在复杂业务逻辑中反复调试 UI 细节。

**核心流程**：
1. **独立原型**：在项目根目录创建独立 HTML 文件（如 `bid-flair-prototype.html`），脱离复杂业务逻辑
2. **视觉验证**：使用 `web-design-engineer` Skill 快速构建可交互原型，在浏览器中验证动画曲线、时机和视觉效果
3. **确认后合入**：用户确认效果后，再将核心动画代码抽离到真实 React 组件和 CSS Module 中

**关键原则**：
- 原型文件仅用于预览，不修改真实业务代码（`LiveRoomSlide.tsx`、`BidDock.tsx` 等）
- 真实合入阶段只做最小改动：抽离动画组件、在成功回调中触发、保持原有抽屉逻辑不变
- 原型与生产代码分离，避免污染主干

**来源**：session:6a228ebf2ec60aa1a739fdcb

---

### H5 直播间竞拍结束后 UI 元素隐藏控制 (Auction End UI Element Hiding)

**问题背景**：竞拍结束后，直播间仍显示聊天输入框和一口价商品列表，与竞拍已结束的业务状态不符。

**根因分析**：
- `BidDock` 组件已根据 `hasEnded` 条件正确隐藏
- 但聊天面板 (`ChatPanel`) 和一口价商品列表 (`fixedPriceList`) 的渲染逻辑仅检查 `fixedPriceItems.length`，未考虑竞拍结束状态
- 导致结束摘要出现后，聊天框和一口价卡片仍浮在页面上

**修复方案**：
在 `LiveRoomSlide` 组件中统一使用 `hasEnded` 条件控制：
1. 竞拍结束后不渲染 `fixedPriceList`（一口价商品列表）
2. 竞拍结束后不渲染 `liveChatOverlay`（聊天输入框区域）

**关键代码模式**：
```tsx
// LiveRoomSlide.tsx 中统一控制
const hasEnded = displayStatus === 'ended' || displayStatus === 'settled';

// 一口价列表渲染条件
{!hasEnded && fixedPriceItems.length > 0 && (
  <FixedPriceList items={fixedPriceItems} />
)}

// 聊天面板渲染条件
{!hasEnded && (
  <ChatPanel />
)}
```

**测试覆盖**：
- 回归测试验证：过期结束和收到 `auction_end` 两种场景下，都不展示聊天输入框和一口价商品列表
- 断言 DOM 中不存在 `说点什么...` 输入框和 `一口价商品列表` 元素

**来源**：session:6a258c390bfcee1b04fb0581

---

### H5 Demo Console 缩短倒计时与流拍动画问题 (Demo Console Shorten Countdown)

**问题背景**：通过 Demo Console 缩短倒计时后，用户没有看到流拍动画。

**根因分析**：
- Demo Console 的「缩短倒计时」操作传入的是旧 `auction_id`
- 如果直播间已切换到新 `current_auction_id`，页面上下文里的旧 `auction_id` 与当前直播间不匹配
- 用户看到的当前竞拍不会收到结束事件，自然没有流拍/成交动画

**修复方案**：
- Demo Console 缩短倒计时前应确认当前直播间的 `current_auction_id`
- 确保缩短的是用户当前正在观看的竞拍，而非历史 auction

**关键代码模式**：
```typescript
// DemoConsole.tsx - 缩短倒计时前校验
const shortenCountdown = async () => {
  const currentAuctionId = liveStream.current_auction_id;
  if (!currentAuctionId) {
    showToast('当前直播间无进行中的竞拍');
    return;
  }
  await demoApi.shortenAuctionEndTime(currentAuctionId);
};
```

**来源**：session:6a26ce7a0bfcee1b04fb3ddc

---

### H5 直播间倒计时与流拍动画设计 (Live Room Countdown & Unsold Animation Design)

**功能概述**：竞拍结束前 5 秒触发全屏沉浸倒计时动画，流拍时触发碎裂消散动画，增强竞拍临场感和仪式感。

**流拍动画触发机制（本地归零立即触发）**：
- **问题背景**：流拍动画仅在收到 `auction_end/auction_ended` WebSocket 事件时触发，但本地倒计时归零后可能延迟等待后端事件，导致动画出现滞后
- **解决方案**：前端本地倒计时归零且当前无 `winner_id` 时，应立即触发流拍动画，不依赖 WebSocket 事件到达
- **关键代码模式**：
```typescript
// 本地倒计时归零即触发，不等后端事件
useEffect(() => {
  if (countdown <= 0 && !auction.winner_id && auction.status === 'active') {
    setShowUnsoldAnimation(true);
  }
}, [countdown, auction.winner_id, auction.status]);
```
- **来源**：session:6a274a280bfcee1b04fba36e

**设计决策流程**：
使用 `ui-design-trio` Skill 进行三版方案推演，浏览器可视化预览后由用户选定：

**倒计时动画（最后 5 秒触发）**：
| 方案 | 风格 | 视觉效果 | 选择 |
|------|------|----------|------|
| V1 心跳脉冲 | 极简 | 数字放大 + 心跳脉冲光晕 | - |
| V2 冲击波扩散 | 动感 | 数字放大 + 环形冲击波扩散 | - |
| **V3 故障艺术 (Glitch)** | **赛博** | **文字 glitch 抖动 + RGB 分离 + 扫描线** | **✓ 选中** |

**流拍动画（Unsold）**：
| 方案 | 风格 | 视觉效果 | 选择 |
|------|------|----------|------|
| V1 印章盖下 | 传统 | 灰色「流拍」印章盖下 | - |
| **V2 碎裂消散 (Shatter)** | **艺术** | **卡片碎裂成碎片后消散** | **✓ 选中** |
| V3 淡入淡出 | 极简 | 整体透明度渐变消失 | - |

**触发条件**：
- **倒计时动画**：仅在底部抽屉未展开时触发，数字从底部浮现至中心、变红放大、驻留后消散
- **流拍动画**：竞拍结束且无 winner 时触发，取代成交动画

**技术实现要点**：
- 纯 CSS 动画（`@keyframes` + `cubic-bezier`），零外部依赖
- 使用 CSS Variables 支持日/夜双主题切换
- 支持 `prefers-reduced-motion` 媒体查询降级
- 动画组件使用 `position: absolute` 基于容器定位，禁止使用 `vw/vh`

**关键文件**：
- `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` — 动画触发逻辑
- `frontend/h5/src/pages/Live/Live.module.css` — 动画关键帧定义
- `frontend/h5/src/components/UnsoldAnimation/` — 流拍动画组件

**来源**：session:6a26f0ae0bfcee1b04fb4e40

---

### H5 开播提醒弹窗误触发问题 (Live Reminder Popup Misfire)

**问题背景**：创建一口价商品后，错误地触发了直播开播提醒弹窗。

**根因分析**：
- 一口价是「商家动作」，但 Demo Console 的 401 重试逻辑统一登录买家 A（`role=0`）
- 登录态切换到买家 A 后，`MobileShell` 按正常用户链路拉取 `pending_live_reminder`
- 于是出现直播开播弹窗，与一口价实时消息无关

**修复方案**：
- 商家菜单动作（如创建一口价）的 401 重试应登录「商家」账号（`role=1`），而非买家 A
- 区分「商家动作」和「买家动作」的重试账号策略

**来源**：session:6a26ce7a0bfcee1b04fb3ddc

---

### H5 Demo Console 401 重试与 API 错误类型 (Demo Console Auth Retry)

**问题背景**：Demo Console 点击按钮后返回 401，但前端没有自动登录 demo 账号重试。

**根因分析**：
- `demoApi` 将 HTTP 错误包装成普通 `Error`，而非 `ApiError`
- `DemoConsole` 的 `runWithDemoAuthRetry` 只识别 `ApiError.status === 401`
- 错误类型不匹配导致重试逻辑未触发

**修复方案**：
- `demoApi` 层将 HTTP 错误转换为携带 `status` 的 `ApiError`
- 保留原有错误文案，确保组件层能正确识别 401 并触发重试

**关键代码模式**：
```typescript
// demoApi.ts - 错误包装
if (!resp.ok) {
  const err = new ApiError(`Demo API error: ${resp.status}`);
  err.status = resp.status;  // 关键：携带 status 供重试逻辑识别
  throw err;
}
```

**来源**：session:6a26ce7a0bfcee1b04fb3ddc

---

### H5 开播提醒弹窗链路修复 (Live Reminder Popup Fix)

**问题背景**：用户通过 Demo Console 创建「正在竞拍」后，买家登录后看不到开播提醒弹窗，但数据库中已生成提醒回执记录。

**根因分析**：
1. **API 响应解析问题（主因）**：前端 API 层只按 `application/json` 解析响应，但 `/live/pending-reminder` 返回的是 `text/plain; charset=utf-8` 包裹 JSON，导致组件拿到的 `hasReminder` 是 `undefined`
2. **并发请求竞态**：React StrictMode 或路由重渲染触发重复请求，第一次请求 claim 了提醒，第二次请求拿到空结果
3. **Demo 链路不完整**：`PostMerchantAuction` 创建 ongoing 竞拍后未调用 `StartLive`，导致直播间状态不完整

**修复方案**：
1. **API 层解析修复**：`frontend/h5/src/services/api.ts` 增加对 `text/plain` 包裹 JSON 的解析支持
2. **前端并发去重**：模块级维护 `pendingLiveReminderRequest`，同一 token 的请求复用同一个 Promise
3. **token 级 stale 校验**：请求前记录 token 快照，返回后比较，若用户已切号则丢弃旧请求结果
4. **后端 Demo 链路补全**：创建 ongoing 竞拍后显式调用 `StartLive` 启动直播间

**关键代码模式**：
```typescript
// 并发去重 + token 级 stale 校验
let pendingLiveReminderRequest: { token: string; promise: Promise<ReminderResponse> } | null = null;

const fetchPendingReminder = async (token: string) => {
  // 去重：同一 token 请求在飞行中则复用
  if (pendingLiveReminderRequest?.token === token) {
    return pendingLiveReminderRequest.promise;
  }
  
  const promise = api.get('/live/pending-reminder').then(res => {
    // stale 校验：返回后检查 token 是否仍有效
    if (currentToken !== token) return null;
    return res.data;
  });
  
  pendingLiveReminderRequest = { token, promise };
  return promise;
};
```

**GrowthBook 相关说明**：
- GrowthBook 本地请求失败（`localhost:3200` 拒绝连接）时走默认值
- Feature `live-start-popup-visibility` 只有明确返回 `false` 才会隐藏弹窗；失败/默认值都允许展示
- 不是本次问题的根因

**来源**：session:6a25985a0bfcee1b04fb0bd2

---

### H5 成交动画定位修复 (Bid Success Animation Positioning)

**问题背景**：竞拍成功「一锤定音」动画不在手机屏幕中间，偏移到左侧或受浏览器视口影响。

**根因分析**：
1. **定位上下文错误**：动画外层 `.shake-trigger` 使用 `position: fixed; inset: 0`，中心点取的是浏览器窗口而非手机预览容器
2. **transform 锚点问题**：锤子 SVG 以 `bottom right` 为旋转轴旋转，视觉重心被甩到左边
3. **容器宽度计算问题**：卡片容器使用 `auto` 宽度参与百分比计算，导致居中偏移

**修复方案**：
1. **收敛定位上下文**：将动画从浏览器视口 (`fixed`) 改为手机容器内 (`absolute`) 定位
2. **修正视觉锚点**：锤子动画使用明确的「以手机容器中心为基准点」计算 transform
3. **统一容器宽度**：卡片容器改为 100% 宽度内居中，避免 `auto` 宽度参与百分比计算

**关键 CSS 模式**：
```css
/* 修复前：取浏览器视口中心 */
.shake-trigger {
  position: fixed;
  inset: 0;
}

/* 修复后：取手机容器中心 */
.shake-trigger {
  position: absolute;
  inset: 0;
}

/* 确保父级有定位上下文 */
.page {
  position: relative;
  width: 100%;
}
```

**教训**：
- 动画组件必须使用 `position: absolute` 并基于容器尺寸计算相对坐标，禁止使用 `vw/vh`
- `rotate + scale` 后的 transform 轨迹需要把视觉中心校回中线
- 父级容器必须建立定位上下文 (`position: relative`)，否则 `absolute` 会向上查找直到视口

**来源**：session:6a25985a0bfcee1b04fb0bd2

---

### H5 商品详情页返回导航修复 (Product Detail Back Navigation Fix)

**问题背景**：从直播间点击商品详情后，点击返回按钮直接回到首页而非上一页直播间。

**根因分析**：
- 商品详情页 header 的返回按钮硬编码为 `href="/"`
- 直播间进入详情时未携带来源状态
- 无论从哪里进入详情页，返回都跳 `/`

**修复方案**：
1. **直播间入口携带来源**：`Link to="/detail?id=..." state={{ from: 'live' }}`
2. **详情页检测来源**：读取 `location.state?.from`，为 `'live'` 时执行 `navigate(-1)`
3. **直接打开详情页**：仍返回首页，保持原有行为

**关键代码模式**：
```tsx
// LiveRoomSlide.tsx - 进入详情时携带来源
<Link to={`/detail?id=${productId}`} state={{ from: 'live' }}>
  商品详情
</Link>

// ProductDetail/index.tsx - 返回按钮处理
const handleBack = () => {
  if (location.state?.from === 'live') {
    navigate(-1);  // 返回上一页（直播间）
  } else {
    navigate('/'); // 返回首页
  }
};
```

**测试覆盖**：
- 从直播间带 state 进入商品详情，点击返回回到直播间
- 直接打开商品详情，点击返回回到首页

**来源**：session:6a25c4110bfcee1b04fb1b82

---

### H5 商品详情页价格显示为0问题 (Product Detail Price Zero Display Issue)

**问题背景**：从直播间跳转到商品详情页时，价格显示为 ¥0，但实际竞拍有出价记录或已成交。

**根因分析**：
- `ProductDetail` 组件仅从 `auction.current_price` 或 `auction.start_price` 获取价格
- 已结束竞拍的权威价格来源是 `/auctions/:id/result` 接口的 `final_price` / `won_bid.amount`
- 详情页未读取 `auction.rules` 字段，导致起拍价显示为0
- 流拍场景（无 winner/无出价）被误标为"成交价 ¥0"

**修复方案**：
1. **已结束竞拍优先使用 result 接口**：
   - 调用 `auctionApi.getResult(auctionId)` 获取成交价
   - 优先使用 `result.final_price`，其次 `result.won_bid.amount`
2. **起拍价从 rules 读取**：`auction.rules.start_price` 而非 `auction.start_price`
3. **流拍状态识别**：`winner_id === null && final_price === 0` 时显示"未成交/流拍"

**关键代码模式**：
```typescript
// 价格计算逻辑
let displayPrice: number;
let priceLabel: string;

if (auction.status === 'ended') {
  const result = await auctionApi.getResult(auction.id);
  if (result.winner_id) {
    displayPrice = result.final_price || result.won_bid?.amount || 0;
    priceLabel = '成交价';
  } else {
    displayPrice = 0;
    priceLabel = '未成交'; // 流拍
  }
} else {
  displayPrice = auction.current_price || auction.rules?.start_price || 0;
  priceLabel = '当前价';
}
```

**测试覆盖要点**：
- 已结束有成交价的竞拍显示正确成交价
- 已结束无成交价的竞拍显示"未成交"而非"成交价 ¥0"
- 进行中的竞拍从 `current_price` 或 `rules.start_price` 显示

**来源**：session:6a26d54c0bfcee1b04fb3f38

---

### H5 竞拍结束摘要显示修复 (Auction End Summary Display Fix)

**问题背景**：竞拍流拍后，结束摘要卡片显示"已成交"，语义错误。

**根因分析**：
- 结束事件只把 `status/current_price` 写入前端状态，没有同步 `winner_id`
- 结束态摘要只能固定显示"成交价"，无法区分成交与流拍

**修复方案**：
- 结束事件把 `winner_id` 一起写入 auction 状态
- 按是否有 winner 决定显示"成交价"还是"流拍"
- 避免在"本地倒计时刚到 0、服务端还没确认结算"时提前判定流拍

**关键代码模式**：
```typescript
// 结束事件处理
const handleAuctionEnd = (msg: WSMessage) => {
  setAuction(prev => ({
    ...prev,
    status: 'ended',
    winner_id: msg.payload.winner_id,  // 关键：同步 winner_id
    final_price: msg.payload.final_price,
  }));
};

// 摘要渲染
const isUnsold = auction.status === 'ended' && !auction.winner_id;
return (
  <div className="endSummary">
    <span>{isUnsold ? '流拍' : '成交价'}</span>
    {!isUnsold && <strong>¥{auction.final_price}</strong>}
  </div>
);
```

**来源**：session:6a26ce7a0bfcee1b04fb3ddc

---

### H5 流拍状态展示修复 (Unsold Auction State Display Fix)

**问题背景**：竞拍流拍后（无 winner/无出价），首页卡片显示"成交"，结果页显示"已成交/最终成交价 ¥0"，语义错误。

**根因分析**：
- `Home/index.tsx` 对所有 `status === 'ended'` 的竞拍都显示"成交"
- `Result/index.tsx` 对所有结果都显示"已成交/最终成交价"
- 未区分"有成交价的结束"和"无成交价的流拍"

**修复方案**：
1. **首页卡片**：`ended && winner_id` 显示"成交时间"，否则显示"已结束/流拍"
2. **结果页**：`winner_id === null || final_price === 0` 时显示"未成交/流拍"
3. **详情页**：同上逻辑，主价格区显示"未成交"而非"成交价 ¥0"

**关键判定逻辑**：
```typescript
// 流拍判定
const isUnsold = auction.status === 'ended' && 
                 (!auction.winner_id || auction.final_price === 0);

// 首页标签
const statusTag = isUnsold ? '流拍' : 
                  auction.status === 'ended' ? '成交' : 
                  auction.status === 'live' ? '直播中' : '即将开始';
```

**来源**：session:6a26d54c0bfcee1b04fb3f38

### H5 直播间点天灯功能 (Sky Lamp / Automatic Bid Guard)

**功能概述**：直播间内用户可开启「点天灯」自动跟价守护，系统会在有人出价时自动以最小加价幅度跟进，确保用户保持领先位置。

**UI 设计决策**：
- **方案选择**：通过 `ui-design-trio` Skill 进行三版方案推演（A-极简/ C-确认层/ D-悬浮），最终采用 **A+C 组合方案**
- **按钮布局**：底部抽屉内横向双按钮，「点天灯」占 30%，「立即出价」占 70%
- **Icon 设计**：天灯 icon 悬浮在「点天灯」按钮左上角，位置偏外 (`left: -10px; top: -8px`)
- **成功态视觉**：
  - 按钮变为不可点击，显示「守护中」文案
  - 天灯 icon 做上下浮动动画 (`@keyframes float`)
  - 抽屉自动收起
  - 底部 Dock 外层加金色光圈 (`box-shadow` 脉冲动画)
  - 商品图片左上角出现天灯角标
  - 直播间上方飘过「XXX 开启点天灯，自动守住领先」飘窗

**状态管理关键决策**：
1. **幂等处理**：后端返回「已有活跃的点天灯订阅」时，前端应视为幂等成功，同步 UI 到「守护中」态，而非报错
2. **错误处理**：`skyLampApi.startSubscription` 必须设置 `{ showError: false }`，让组件层统一处理成功/失败提示，避免请求层提前弹出全局错误 toast
3. **状态持久化**：页面加载时需调用 `GET /api/v1/sky-lamp/subscriptions?status=1` 查询活跃订阅，回填 UI 状态，防止刷新后状态丢失

**API 契约**：
- `POST /api/v1/sky-lamp/subscriptions` — 开启点天灯（需 `showError: false`）
- `GET /api/v1/sky-lamp/subscriptions?status=1` — 查询活跃订阅（页面加载时回填状态）

**关键实现文件**：
- `frontend/h5/src/components/BidDock/BidDock.tsx` — 点天灯按钮与出价按钮横排布局
- `frontend/h5/src/components/BidDock/SkyLampButton.tsx` — 点天灯按钮组件（含 icon 悬浮位）
- `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` — 状态查询与成功态动画触发
- `frontend/h5/src/api/skyLamp.ts` — API 封装（`showError: false` 配置）

**开发模式**：
- 使用 `feat/h5-sky-lamp-entry` 隔离 worktree 开发
- TDD 模式：先写失败测试（重复订阅应进入守护态），再最小实现

**来源**：session:6a2161e62ec60aa1a7398232

---

### H5 开发端口冲突排查 (H5 Dev Server Port Conflict)

**问题背景**：Vite dev server 显示启动成功但浏览器看到旧代码，或页面显示与设计不符。

**根因分析**：
- 同一端口被多个 Node 进程占用
- macOS 上 `localhost` 可能解析到 IPv6 `[::1]`，而旧进程监听 IPv4 `127.0.0.1`
- 导致同一端口实际有两个服务，浏览器可能命中旧进程

**排查命令**：
```bash
lsof -i :<port> | grep LISTEN
```

**解决方案**：
1. 清理旧进程：`kill -9 <PID>`
2. 优先使用 `localhost` 而非 `127.0.0.1` 访问
3. 强刷浏览器：`Cmd + Shift + R`

**来源**：session:6a2161e62ec60aa1a7398232

---

### H5 直播间点天灯飘窗用户名乱码修复 (Sky Lamp Flair Username Mojibake)

**问题背景**：点天灯成功后的飘窗提醒中，用户名显示乱码（如 `è€èœ...`）。

**根因分析**：
- 飘窗组件直接渲染后端返回的 `user_name`，未经过 `repairUtf8Mojibake` 修复
- 其他组件（如商品名/主播名）已统一调用修复函数，但飘窗渲染边界遗漏

**修复方案**：
- 在飘窗组件中对 `user_name` 字段统一调用 `repairUtf8Mojibake` 修复后再渲染
- 检查其他飘窗/飘屏组件是否存在同类问题

**关键文件**：
- `frontend/h5/src/components/LiveFlair/` — 点天灯飘窗组件
- `frontend/h5/src/utils/repairUtf8Mojibake.ts` — 编码修复工具函数

**教训**：编码修复必须在所有渲染外部文本的边界统一接入，新增飘窗/飘屏类组件时，默认接入 `repairUtf8Mojibake` 应作为 checklist 项

**来源**：session:6a2161e62ec60aa1a7398232, session:6a24568600057ea64ca279d0

### H5 直播间上下滑 Feed 改造 (Live Room Vertical Swipe Feed)

**决策背景**：原 `/live` 是单直播间详情页，向下滑动会露出底部抽屉的半截价格，不符合用户「向下滑进入下一个直播间」的心智预期。

**核心决策**：
- **Feed 模式**：`/live` 改造为抖音式全屏纵向 Feed，每个直播间占满一屏
- **手势优先级**：上下滑负责切直播间（手指上滑=下一个，下滑=上一个），抽屉不再响应滑动打开
- **抽屉交互变更**：从「滑动露出」改为「点击打开」，通过 `出价` 按钮或商品卡片显式触发

**技术架构**：
- **LiveFeedPage**：管理直播间列表、当前索引、URL 同步、分页加载
- **LiveRoomSlide**：单个直播间渲染，包含视频区、BidDock、出价抽屉
- **数据流**：`GET /api/v1/live-streams` 获取直播间列表 → 每个 slide 独立获取详情和 WS 连接

**关键边界决策**：
1. **URL 同步策略**：滑动切房使用 `replace`（避免浏览器返回在直播间间跳转），从首页/详情进入使用 `push`
2. **返回键策略**：抽屉打开时 `push` 一个 `?sheet=bid/info` 状态，返回键先消费该状态收起抽屉，而非离开页面
3. **出价中锁房**：出价请求 pending 时锁定当前房间，防止结果回写到错误 slide
4. **WS 归属校验**：切房时断开旧连接，WS 消息按 `auctionId/liveStreamId` 校验后再更新，避免旧房间延迟消息覆盖新房间
5. **抽屉收起方式**：点击遮罩、下拉抽屉 handle 区域（`deltaY >= 56px`）、出价成功后自动收起

**数据契约**：
- 直播间列表接口需返回 `current_auction_id`（当前可竞拍 ID），否则 feed 只能看直播不能出价
- `auction_id` 归属校验：URL 传入的 `auction_id` 必须校验 `auction.live_stream_id === 当前 liveStreamId`，不匹配则回退使用 `current_auction_id`

**手势判定参数**：
- 纵向滑动阈值：`>= 72px`
- 切房判定：纵向位移 > 横向位移，且超过阈值
- 抽屉下拉收起阈值：`>= 56px`，且仅在 `scrollTop <= 0` 时触发

**设计文档**：`docs/superpowers/specs/2026-06-02-h5-live-feed-swipe-design.md`

**来源**：session:6a1ea710959156a8dfc8a36e

### H5 直播间 Feed 分页加载竞态修复 (Live Feed Prefetch Race Condition)

**问题背景**：`LiveFeedPage` 的 prefetch effect 中，`loadingMoreRef` 用于防止并发加载，但在 effect cleanup 时未重置，导致快速切房后 ref 被卡死。

**根因分析**：
- 用户在 prefetch API 返回前滑动切房，React 执行旧 effect 的 cleanup（`cancelled = true`）
- 新 effect 执行时看到 `loadingMoreRef.current === true` 直接 return
- API 响应到达时 `.finally()` 因 `cancelled === true` 跳过重置，ref 永远为 true

**修复方案**：
```typescript
useEffect(() => {
  let cancelled = false;
  // ... prefetch logic
  return () => {
    cancelled = true;
    loadingMoreRef.current = false;  // 关键修复：cleanup 中重置 ref
  };
}, [currentIndex]);
```

**教训**：ref 的清理逻辑不应依赖异步回调的 `.finally()`，必须在 effect cleanup 中同步重置。

**来源**：session:6a1ea710959156a8dfc8a36e

### 一口价秒杀功能（Fixed Price Sale）前端集成

**功能概述**：直播间支持一口价秒杀商品，用户点击立即购买后扣减余额直接下单，库存实时同步通过 WebSocket 广播。

**组件结构**：
- `frontend/h5/src/components/FixedPriceCard/` — 一口价商品卡片，展示商品信息、价格、剩余库存
- `frontend/h5/src/components/FixedPricePurchaseModal/` — 确认购买弹窗，处理购买流程和余额不足提示
- `frontend/h5/src/components/FixedPriceFlair/` — 购买成功全屏飘屏，复用 B1 弹幕飘屏组件
- `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` — 直播间入口，挂载一口价卡片列表

**核心交互流程**：
1. 直播间加载时调用 `GET /api/v1/live-streams/{id}/fixed-price/items` 获取商品列表
2. 用户点击商品卡片 → 弹出确认购买弹窗 → 显示商品信息、价格、余额
3. 用户确认购买 → 前端生成 `X-Idempotency-Key` (uuid v4) → 调用 `POST /api/v1/fixed-price/items/{id}/purchase`
4. 后端返回结果：
   - 200 成功 → Toast「购买成功」+ 卡片置灰 + 触发飘屏 `fixed_price_flair`
   - 402 余额不足 → 弹出独立弹窗显示「当前余额 ¥XX，差 ¥YY」+ 跳转充值按钮
   - 409 错误（售罄/已购买/已下架）→ Toast 显示原因 + 按钮置灰
5. WebSocket 实时推送库存变化 `fixed_price_stock` → 前端更新剩余库存显示

**WebSocket 消息类型（复用 LiveStreamRoom）**：
- `fixed_price_listed` — 主播上架新商品
- `fixed_price_stock` — 库存变化（节流 1 条/秒/item）
- `fixed_price_sold_out` — 商品售罄
- `fixed_price_offline` — 商品下架
- `fixed_price_flair` — 购买成功飘屏（复用弹幕飘屏渠道）

**关键设计决策**：
- **余额不足交互区分**：402 余额不足使用独立弹窗（引导充值），其他 409 错误使用 Toast（不可恢复，无需弹窗）
- **幂等 key 生成**：前端点击时生成 uuid v4，重试复用同一个 key，避免网络抖动重复扣款
- **飘屏复用**：一口价购买成功飘屏复用 B1 弹幕飘屏组件，不新建独立实现
- **列表聚合**：`i_bought` 字段由后端在 auction-service 内查询 `fixed_price_purchases` 得出，无跨服务调用

**测试要点**：
- 抢购按钮点击必须生成 idempotency key 并复用
- 402 响应必须触发 InsufficientBalanceDialog，含跳转充值按钮
- 409 sold_out 响应必须按钮置灰
- WS `fixed_price_stock` 消息必须实时刷新剩余库存
- WS `fixed_price_flair` 消息必须触发飘屏

**来源**：session:6a1c5b0b959156a8dfc850b7
