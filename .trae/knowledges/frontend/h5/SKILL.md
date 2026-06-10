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
- **日间模式主 CTA 悬浮样式**：主 CTA 按钮（如「中标待支付」）需确保悬浮时文字颜色不与背景融合导致不可见，需显式锁定内部 `strong` 元素的 `color: var(--text-inverse)`，防止全局 anchor hover 规则覆盖导致文字变透明
- **直播间详情页字符编码修复**：`LiveRoomSlide.tsx` 中的 `host_name` 可能因后端编码问题出现乱码（mojibake），需使用 `repairUtf8Mojibake` 工具函数修复后再渲染（首页 `Home/index.tsx` 已实现此修复，直播间需保持一致）
- **直播间主播头像兜底**：`LiveRoomSlide.tsx` 中的 `host_avatar` 直接渲染后端返回的 URL，若该 URL 为内网域名（如 `copilot-cn.bytedance.net`）或已失效，会导致头像显示失败。应添加 `onError` 兜底逻辑，失败时切换到本地默认头像（`frontend/h5/public/assets/default-avatar.svg`）
- **足迹状态实时获取策略**：足迹仅存储进入时的快照（`id, name, cover, enteredAt`），不包含直播间当前状态。若需在个人中心显示直播间实时状态（直播中/即将开始/已结束），应在页面打开时基于 `footprints.map(live_stream_id)` 并发调用 `liveStreamApi.get(id)` 获取最新状态，而非修改 localStorage 契约。状态角标放在封面右上角，使用半透明深色胶囊，接口失败时显示「状态未知」或不显示，避免阻塞页面加载

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
- 服务区功能入口应避免冗余：若上方已存在带数字的 Metric 统计（如收藏），下方二级菜单应替换为其他功能（如设置），避免重复入口
- 消息通知入口位置：从二级菜单移至「我的竞拍」三宫格中间位置，替换原「中标数量」卡片，右上角显示未读角标
- 二级菜单占位策略：当功能区菜单项较少时（如只有「设置」一项），可添加「帮助中心」「客服与反馈」「关于平台」等占位入口，保持视觉区域完整
- 筛选器状态按 Tab 隔离，切换 Tab 时重置筛选条件避免交叉污染
- 足迹实时状态获取采用「快照存储+实时拉取」模式：localStorage 只存进入时的静态信息，打开个人中心时再并发拉取各直播间的实时状态，避免历史足迹数据迁移和状态过期问题

## Conventions
- H5 开发端口为 5173，与 Admin (5175) 和 Test Dashboard (5174) 区分
- 静态资源放在 `public/assets/`，构建后会复制到 `dist/assets/`，保持同源访问
- 所有新 UI 组件必须适配双主题并使用项目指定设计 Token

## Testing Strategy
- H5 使用 Jest 进行单元测试，重点覆盖图片兜底逻辑和 localStorage 操作
- 移动端布局需在真机或 DevTools 设备模拟器中验证，不能仅依赖桌面浏览器

## UX Enhancement Decisions

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

**来源**：session:6a28703b0bfcee1b04fc2ec6

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
