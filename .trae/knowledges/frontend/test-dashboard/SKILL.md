---
name: knowledge-frontend-test-dashboard
description: >
  Covers Test Dashboard 的测试任务 API、WebSocket 进度流、Zustand 状态、A/B 对比、演示大屏和 Docker/Nginx 运行约束。
  Navigate when: modifying frontend/test-dashboard, debugging progress WebSocket, adding test scenarios, changing report polling, demo theater, Grafana-facing demos, or test-dashboard deployment.
  Excludes: Admin 管理后台; Admin context is in ../admin/SKILL.md.
  Keywords: frontend/test-dashboard, Dashboard, Screen, wsStore, testStore, discoverWS, VITE_WS_BASE, StepTimeline, AntiSnipeTimeline, demoTheater, Zustand, WebSocket
---

## Module Structure

Test Dashboard 是测试与演示控制台，负责任务启动、实时进度、报告查询、A/B 对比和演示大屏；它依赖 Gateway 暴露的 HTTP API 与 WebSocket discovery。

### Directory Layout
- `frontend/test-dashboard/src/api/test.ts` — 测试任务 API、报告查询、取消任务、WebSocket discovery。
- `frontend/test-dashboard/src/store/wsStore.ts` — WebSocket 连接状态、消息历史和清理逻辑。
- `frontend/test-dashboard/src/store/testStore.ts` — 当前测试任务状态。
- `frontend/test-dashboard/src/pages/Dashboard.tsx` — 主控制台页面。
- `frontend/test-dashboard/src/pages/Compare.tsx` — A/B 对比页面和轮询逻辑。
- `frontend/test-dashboard/src/pages/Screen.tsx` — 1920×1080 演示大屏模式。
- `frontend/test-dashboard/src/pages/demoTheater.ts` — 用户旅程事件到演示状态的映射模型。
- `frontend/test-dashboard/src/components/` — 进度、时间轴、状态机和演示组件。

### Key Entry Points
- `frontend/test-dashboard/src/App.tsx` — `/test` 与 `/test/screen` 路由入口。
- `frontend/test-dashboard/src/api/test.ts` — 所有测试 API 与 `discoverWS` 入口。
- `frontend/test-dashboard/src/store/wsStore.ts` — WebSocket 生命周期控制。
- `frontend/test-dashboard/vite.config.ts` — React 去重和开发代理配置。

## Gotchas
- 建立新 WebSocket 前必须先关闭旧连接，`connect()` 内部先调用 `disconnect()` 是防止连接泄漏和跨任务串消息的关键约束（`frontend/test-dashboard/src/store/wsStore.ts`）
- Dashboard 页面卸载时必须清理 WS 与全局 store，否则切换页面后会保留幻影进度和旧任务状态（`frontend/test-dashboard/src/pages/Dashboard.tsx`）
- `discoverWS` 使用独立 axios 实例，不走通用 `API_BASE`；修改 WebSocket discovery 时要同时考虑 `VITE_WS_BASE` 和 Nginx `/ws/` 反代（`frontend/test-dashboard/src/api/test.ts`, `deploy/demo/nginx-ip.conf`）
- `recharts` 等依赖可能引入第二份 React 导致 `Invalid hook call`，`vite.config.ts` 的 `resolve.dedupe: ['react', 'react-dom']` 不能随意删除（`frontend/test-dashboard/vite.config.ts`）
- WebSocket 消息历史有最大 200 条限制，新增高频消息类型时不能绕过 `wsStore` 直接无限追加到组件状态（`frontend/test-dashboard/src/store/wsStore.ts`）
- A/B 对比轮询用 ref 持有最新结果，不能把完整响应对象放入 effect 依赖导致 interval 反复重启（`frontend/test-dashboard/src/pages/Compare.tsx`）

## Architecture
- Test Dashboard 的数据流是启动测试拿 `test_id`、通过 `discoverWS(test_id)` 获取 WS URL、WebSocket 收 progress/step/metrics、最终轮询 `getReport(test_id)` 获取报告（`frontend/test-dashboard/src/api/test.ts`, `frontend/test-dashboard/src/store/wsStore.ts`）
- `/test` 是带侧栏的控制台路由，`/test/screen` 是无侧栏大屏模式；演示投屏相关修改应优先检查 Screen 路由而不是 Dashboard 主页面（`frontend/test-dashboard/src/App.tsx`, `frontend/test-dashboard/src/pages/Screen.tsx`）
- 测试类型覆盖压测、E2E、用户旅程、防狙击、回调投递、故障注入和 A/B 对比；新增测试场景应先落在 `src/api/test.ts` 的 API 层，再接入页面状态和可视化组件（`frontend/test-dashboard/src/api/test.ts`）

## Patterns
- `wsStore` 管连接和消息历史，`testStore` 管当前运行任务；跨组件状态不要在页面局部重复实现，否则清理和重连语义会分叉（`frontend/test-dashboard/src/store/wsStore.ts`, `frontend/test-dashboard/src/store/testStore.ts`）
- `StepTimeline` 对同名步骤做 `#N` 编号，后端新增重复 step 时前端不需要强制改名，应该保留编号展示语义（`frontend/test-dashboard/src/components/StepTimeline.tsx`）
- `demoTheater` 将 UserJourney 事件映射为演示状态，新增演示事件应在模型层映射，避免 Screen 组件直接理解后端原始事件细节（`frontend/test-dashboard/src/pages/demoTheater.ts`, `frontend/test-dashboard/src/pages/Screen.tsx`）

## Conventions
- Test Dashboard 开发端口为 5174，用于避开 Admin/H5 常用端口；本地排障端口冲突时不要随意修改主干配置（`frontend/test-dashboard/SKILL.md`, `AGENTS.md`）
- `/api` 和 `/ws` 都应通过 Gateway/Nginx 入口代理，前端不应直连测试服务容器内部地址（`frontend/test-dashboard/vite.config.ts`, `deploy/demo/nginx-ip.conf`）
- **WebSocket 地址必须使用 Nginx 反向代理路径，严禁公网直连微服务私有端口**。服务发现返回的 `ws_url` 若包含公网不可达的直连 IP:Port（如 `:18092`），会导致浏览器 WS 连接超时挂起。应统一通过 Nginx 转发（如 `/test-ws`）并保留 `Upgrade` 头（`deploy/demo/nginx-ip.conf`, `docker-compose.demo.yml`）
- **部署脚本需校验 WS 地址格式**。`scripts/deploy-prod.sh` 和 `scripts/test-deploy-prod-scripts.sh` 应检查服务发现返回的 `ws_url` 是否为代理路径格式（以 `/` 开头），而非 `ws://ip:port` 直连格式，防止配置回退导致连接挂起
- 页面级组件放在 `src/pages/`，可复用可视化组件放在 `src/components/`，类型定义和 API 函数同置在 `src/api/test.ts`（`frontend/test-dashboard/src/pages/`, `frontend/test-dashboard/src/components/`, `frontend/test-dashboard/src/api/test.ts`）

## Testing Strategy
- 测试运行态是 HTTP 启动 + WebSocket 进度 + 报告轮询的组合，验证时不能只看启动接口 200，还要确认 WS discovery 返回 JSON 且包含 `ws_url`（`scripts/test-deploy-prod-scripts.sh`, `deploy/demo/MAIN_DEPLOY_QUICKSTART.md`）
- Demo Theater 依赖 UserJourney 的 prepare/enter_live/reminder/auction_bid/sky_lamp/fixed_price_purchase/verify/cleanup 等事件名，后端事件改名会直接影响大屏展示（`frontend/test-dashboard/src/pages/demoTheater.ts`）

## UX Enhancement Decisions

### 剧场模式 (Chaos Theater Mode)
- **设计方案**：采用「实况战情室 (Live War-Room)」风格，营造实时侦测分析的硬核终端感
- **决策过程**：使用 `ui-design-trio` Skill 进行三版方案推演——方案1（极简控制台/常规B端）、方案2（沉浸式放映/电影感）、方案3（实况战情室/仪表盘追踪），最终选定方案3
- **核心功能**：
  - **C1-a 一键剧本播放**：新增「开始演示」按钮，自动执行 baseline→inject→recover 完整流程，与现有手动模式并存
  - **C1-b 旁白字幕**：采用终端日志风格的打字机效果，浮于图表左上角，带 `> ` 提示符
  - **C1-c 曲线锚点 + 行内指标**：在 Recharts 曲线上标注注入时刻/SLA击穿/恢复拐点，相关指标（恢复耗时、峰值错误率、损失QPS）作为浮窗挂载在锚点旁
- **视觉约束**：
  - 使用等宽字体呈现旁白，锚点线使用红色虚线，行内指标卡跟随锚点出现
  - 锚点标签采用短文本 + 错层布局策略，避免多个标签在图表中重叠
- **主题说明**：Test Dashboard 为单套浅色主题，无需双主题适配
- **设计文档**：`docs/superpowers/specs/2026-06-09-chaos-theater-mode-design.md`
- **来源**：session:6a27ede70bfcee1b04fbc3b6

### UX 增强开发流程 (UX Enhancement Development Process)
对于复杂的 UX 增强任务，项目采用以下标准化流程：
1. **brainstorming** — 明确动机、边界和取舍，输出候选方案清单
2. **ui-design-trio** — 对视觉方案进行三版推演（如极简/赛博/仿生），浏览器预览后选定
3. **writing-plans** — 生成详细实施计划（含任务拆分、写集、测试哨兵）
4. **sdd-run** — 按 SDD 协议执行，使用独立 worktree 隔离开发
5. **verification-before-completion** — 本地验证后合并

该流程已在直播间战况热度条 (A1)、演示台剧场模式 (C1)、H5 个人中心重构、出价排行视觉优化、直播间倒计时与流拍动画、直播间互动 UI 升级中验证有效。

**来源**：session:6a27ede70bfcee1b04fbc3b6

### Recharts 锚点标签防重叠 (ReferenceLine Label Layout)
**问题背景**：在韧性曲线上使用 `ReferenceLine` 标注多个锚点（注入时刻、SLA击穿、恢复拐点）时，标签文字重叠在一起，影响可读性。

**根因分析**：Recharts 默认的 `ReferenceLine` label 不会自动避让，当多个 label 都画在图内同一高度附近且字符串过长时，必然重叠。

**解决方案**：
1. **短文本策略**：将长标签（如 `SLA: peak 68% error rate`）缩短为关键词（如 `Inject`、`SLA`、`Recover`）
2. **错层布局**：给不同锚点分配不同的 `position` 或 `dy` 偏移，使标签在垂直方向错开
3. **布局元信息**：在锚点数据结构中增加 `position: 'top' | 'bottom'` 或 `dy: number` 字段，控制每个标签的相对位置

**关键代码模式**：
```tsx
// 锚点数据结构增加布局元信息
interface Anchor {
  x: number;
  label: string;
  position: 'top' | 'bottom'; // 错层布局
  dy?: number; // 额外偏移
}

// ReferenceLine 使用自定义 label
<ReferenceLine
  x={anchor.x}
  label={{
    value: anchor.label,
    position: anchor.position,
    dy: anchor.dy,
    fill: '#dc2626',
    fontSize: 12,
  }}
/>
```

**教训**：使用 Recharts `ReferenceLine` 做多锚点标注时，必须通过短文本 + 错层布局（`position`/`dy`）主动控制标签位置，不能依赖默认布局。

**来源**：session:6a27ede70bfcee1b04fbc3b6

## Demo Console Features

### H5 Demo Console 架构设计 (H5 Demo Console Architecture)

**功能定位**：H5 端常驻演示控制面板，用于在移动端实时触发演示效果（出价、账号切换、充值等），解决独立测试平台无法展示 H5 端实时效果的问题。

**核心架构决策**：
1. **AssistiveTouch 悬浮球模式**：借鉴 iOS 辅助触控的交互设计
   - 平时为小型悬浮球（毛玻璃质感、深色半透明）
   - 点击后扇形展开一级菜单（账号/演示/充值/商家）
   - 一级菜单点击后展开对应二级菜单，悬浮球颜色/图标变化作为反馈
   - 执行动作后自动收起，恢复到安静状态

2. **菜单结构边界划分**：
   - **账号（快速登录）**：纯粹改变当前 H5 端的身份
     - 二级菜单：买家A、商家、管理员、返回
   - **演示（外部动作）**：纯粹触发外部环境变化（不改变当前身份）
     - 二级菜单：他人跟价、并发压测、竞拍延时、防作弊演示、返回
   - **充值**：为当前登录账户充值固定金额
   - **商家**：商家专用演示动作
     - 二级菜单：即将开播、正在竞拍、一口价、返回

3. **环境隔离策略**：
   - 纯演示定位，代码直接写在项目中，不做 ENV 隔离
   - 后端演示接口（`/api/test/demo/*`）仅注册在 `test-service` 中
   - 演示动作使用固定演示账号（如买家B 13800000002）进行真实查库和出价

**交互设计要点**：
- 使用 SVG 线框图标替代 emoji，保持视觉一致性
- 全局变暗的毛玻璃背景遮罩，点击空白处自动收起
- 子菜单散开距离适当拉大，防止误触
- 非直播间页面点击需要直播间上下文的功能时，toast 提示而非禁用按钮

**技术实现边界**：
- 前端：`DemoConsole` 组件挂载在 `App.tsx` 全局
- 后端：`test-service` 新增 `/api/test/demo/*` 接口组
- 账号体系：统一使用 138 号段和固定密码 `Demo@123456`

**来源**：session:6a242c4900057ea64ca26316

### H5 Demo Console JWT 自愈机制 (Demo Console JWT Self-Healing)

**问题背景**：H5 Demo Console 点击按钮提示"JWT无效"，因为浏览器 localStorage 中存储的 token 已过期或失效。

**根因分析**：
1. 演示动作（如"正在竞拍"）调用 `/api/test/demo/*` 接口需要有效 JWT 鉴权
2. 用户浏览器中残留的旧 token 已过期
3. 仅重启 test-service 无法清理浏览器中的无效 token

**解决方案**：前端自愈机制
- Demo 动作遇到 401 时，自动使用演示买家 A 账号刷新 token
- 重试一次，避免用户看到错误后手动重新登录

**关键代码模式**：
```typescript
// DemoConsole.tsx - 演示动作错误处理
const handleDemoAction = async () => {
  try {
    await demoApi.triggerAction();
  } catch (err) {
    if (err.status === 401) {
      // 自动刷新 token 并重试
      await refreshDemoToken();
      await demoApi.triggerAction();
    }
  }
};
```

**关键文件**：
- `frontend/h5/src/components/DemoConsole/DemoConsole.tsx`
- `frontend/h5/src/api/demo.ts`

**来源**：session:6a2707bb0bfcee1b04fb6b6f

---

### H5 Demo Console 他人天灯演示 (H5 Demo Console Other Sky Lamp)

**功能目标**：在 H5 Demo Console 中提供「他人点天灯」演示功能，让当前用户（A视角）触发其他演示账号（B用户）对当前竞拍开启点天灯，用于演示多人同时点天灯的竞争场景。

**设计决策**：
1. **复用现有链路**：复用 `test-service` SDK 已有的 `SubscribeSkyLamp` 能力，不重复造轮子
2. **固定演示账号**：使用固定的演示买家 B（ID: 9102，手机号 13800138002）作为「他人」身份
3. **菜单结构**：作为「演示」一级菜单下的二级按钮，与「他人跟价」并列

**后端接口设计**：
- `POST /api/test/demo/sky-lamp` — 以演示买家 B 身份对当前竞拍开启点天灯
- 实现逻辑：使用固定买家 B 身份调用现有 `SubscribeSkyLamp` SDK 方法
- 权限：仅可在直播间页面触发，依赖当前 `auction_id` 上下文

**前端实现要点**：
- 在 `DemoConsole` 组件的「演示」菜单下新增「他人天灯」二级按钮
- 点击后调用 `/api/test/demo/sky-lamp` 接口
- 成功后在直播间显示天灯飘屏效果（由现有 WebSocket `sky_lamp` 事件驱动）

**来源**：session:6a25604000057ea64ca2d08d

---

### H5 Demo Console 商家菜单设计 (H5 Demo Console Merchant Menu)

**功能目标**：为商家角色提供快速创建竞拍和一口价商品的演示功能，支持在 H5 端一键触发商家动作。

**菜单结构**：
- **即将开播**：创建 1 分钟后开始的竞拍（pending 状态）
- **正在竞拍**：创建可立即进入/出价的竞拍（ongoing 状态）
- **一口价**：为当前直播间创建一口价商品
- **返回**：返回上级菜单

**关键设计决策**：

1. **重复点击处理策略**：
   - 每次点击都创建新的 demo 商品，避免"同一商品同一时间只能有一个活跃竞拍"的约束冲突
   - 商品命名使用 `DEMO_时间戳_随机后缀` 格式，天然不撞唯一性
   - 不需要取消旧竞拍，不会破坏演示现场
   - 每个竞拍 fixture 必须用独立商品（`uk_active_product` 唯一索引约束）

2. **直播间上下文处理**：
   - `即将开播` 和 `正在竞拍`：后端自动创建直播间 → 商品 → 规则 → 竞拍完整链路
   - `一口价`：依赖当前 H5 已进入的直播间，通过 `DemoContext` 传递 `currentLiveStreamId`
   - 不在直播间页面点击 `一口价` 时，前端 toast 提示 `请先进入直播间`，不禁用按钮

3. **后端接口设计**：
   - `POST /api/test/demo/merchant/auctions` - 创建竞拍
     - body: `{ "mode": "upcoming" | "ongoing" }`
     - 后端用演示商家账号（如 9103）创建完整链路
   - `POST /api/test/demo/merchant/fixed-price` - 创建一口价
     - body: `{ "live_stream_id": 123 }`
     - 不传或不在直播间时前端直接提示

4. **防狙击时间窗口配置**：
   - 普通规则默认 `trigger_delay_before=30` 秒
   - Demo Console 创建的商家竞拍配置为 `trigger_delay_before=10` 秒，方便演示"压到最后 10 秒再出价"的场景

**实现边界**：
- 复用现有 test-service SDK 的 `CreateProductAs`、`CreateAuctionRule`、`CreateAuctionAs` 等方法
- `即将开播` 需要扩展 auction create 接口支持 `start_time` 参数
- 创建链路使用真实业务路径，不绕过领域逻辑

**来源**：session:6a242c4900057ea64ca26316

### Demo Console "正在竞拍" 模式开播状态问题 (Demo Console Ongoing Mode Live Stream Start Issue)

**问题背景**：用户通过 Demo Console 的「正在竞拍」按钮创建竞拍后，重新登录没有看到开播提醒弹窗，且进入直播间时提示"资源不存在"。

**根因分析（双层问题）**：

**第一层：直播间未启动**
- `PostMerchantAuction`（`ongoing` 模式）只创建商品、直播间、规则和竞拍，然后等待竞拍状态变成 started
- **关键缺失**：没有调用 `/api/v1/live-streams/:id/start` 启动直播间
- 这导致 `live_stream:{id}:stats.started_at` 未写入，重新登录后查询不到待提醒的直播 session
- 直播间状态未正确设置为「直播中」，导致进入时资源校验失败

**第二层：直播间关注关系断裂（架构设计问题）**
- 原实现每次点击都**新建直播间**（`Demo 直播间 <productID>`），而非复用固定商家直播间
- 用户关注（Follow）是按 `live_stream_id` 维度，不是按 `merchant_id` 维度
- 即使用户之前关注了商家 9103 的旧直播间，新创建的直播间 `live_stream_id` 不同，关注关系不会自动继承
- 导致开播提醒查询"我关注的正在直播的直播间"时返回空

**修复方案**：
1. **数据修复**：补全缺失的直播间启动状态写入
2. **代码修复**：`PostMerchantAuction` 在创建 ongoing 竞拍后，显式调用 `StartLive` 启动直播间
3. **架构修正**：Demo Console 复用**固定商家直播间**，而非每次创建新直播间；新竞拍绑定到同一个固定的 `live_stream_id`
4. **测试覆盖**：增加测试断言确保"ongoing demo 必须启动直播间"

**固定商家直播间模型**：
```text
固定商家 9103
  -> 固定直播间 live_stream_id（复用，不新建）
  -> 每次 Demo 创建新商品（避免一商品一活跃竞拍冲突）
  -> 每次 Demo 创建新 auction
      - upcoming: start_time = now + 1min
      - ongoing: start_time = now（立即开始）
  -> auction.live_stream_id 都绑定同一个固定商家直播间
```

**关键洞察**：
- 关注关系绑定的是 `live_stream_id`，不是 `merchant_id`
- 开播提醒查询的是"我关注的直播间中哪些正在直播"，不是"我关注的商家是否有直播"
- Demo Console 必须完整模拟真实业务链路，包括直播间启动和固定直播间复用

**关键代码模式**：
```go
// PostMerchantAuction 修复后流程
func PostMerchantAuction(ctx context.Context, mode string) error {
    // 1. 创建商品
    product := CreateProductAs(...)
    // 2. 创建直播间
    liveStream := CreateLiveStream(...)
    // 3. 创建竞拍规则
    rule := CreateAuctionRule(...)
    // 4. 创建竞拍
    auction := CreateAuctionAs(...)
    
    // 5. 【修复】启动直播间（ongoing 模式）
    if mode == "ongoing" {
        if err := StartLive(ctx, liveStream.ID); err != nil {
            return err // 开播失败应报错，避免创建未开播的直播间
        }
    }
    return nil
}
```

**教训**：
- Demo Console 的「正在竞拍」模式必须完整模拟真实业务链路，包括直播间启动
- 竞拍在进行中但直播间未开播会导致状态不一致，影响开播提醒和直播间进入逻辑
- 测试 fixture 应验证直播间状态，而非仅验证竞拍记录创建

**来源**：session:6a257c8d0bfcee1b04fafe04, session:6a258fcd0bfcee1b04fb07fb

### H5 并发出价演示 (Concurrent Bids Demo)
**功能目标**：在 H5 Demo 控制台提供「一键抬价」按钮，由后端 test-service 发起快速连续出价，制造"价格被超越"场景，让用户体验竞价失败的业务反馈。

**核心设计决策**：
1. **串行非并发**：虽然按钮名为"并发压测"，但后端采用**串行快速递增**策略。原因：
   - 业务层有 `AuctionBidLock` + 乐观锁，真并发会导致大量"出价过于频繁"失败
   - 串行确保每笔出价基于上一笔成功后的最新价递增，抬价幅度可控
   - 演示效果更稳定，H5 飘屏顺序自然

2. **CapPrice 提前终止保护**：出价前检查，若下一笔金额 ≥ `cap_price` 则停止，避免触发 `handleCapPriceBid` 导致拍卖意外成交结束

3. **Increment 自动修正**：若调用方传入的 `increment` < 规则 increment，自动按规则 increment 处理，避免首笔即低于最低出价失败

4. **响应契约**：
   - 全失败 → HTTP 400 + `ok: false` + `last_error`
   - 有成功 → HTTP 200 + `ok: true` + `success_count/failure_count/highest_amount`

**技术边界**：
- 出价链路**不校验余额**，demo 用户无需预充值即可出价
- 同一用户连续递增出价合法，无禁止规则
- H5 端无需改动即可看到飘屏/排行/热度动画（由 `bid_placed` WS 事件驱动）

**设计演变记录**：
- **初始方案**：使用 goroutine 真并发，但发现会导致大量"出价过于频繁"失败，演示效果不稳定
- **方案 A（最终采用）**：串行快速递增，每笔基于上一笔成功后的最新价递增，`failure_count` 接近 0，`highest_amount` 确定
- **核心权衡**：放弃"并发压测"语义，换取"让用户用旧价稳定失败"的演示目标

**实现检查清单**：
- [x] test-service 读取 `rules.cap_price` 并实现 clamp 逻辑
- [x] 出价金额公式：`baseline + increment*(i+1)`，每笔成功后更新 baseline
- [x] 检测到下一笔 ≥ `cap_price` 时提前停止，返回已成功的出价统计
- [x] 响应体统一包含 `ok` 字段，与 HTTP status code 语义一致
- [x] H5 端通过 `demo:concurrent-bids-completed` 事件同步价格到直播间

**测试要点**：
- `DemoConsole` 测试：验证并发出价按钮触发正确 API 调用，成功后广播事件
- `LiveRoomSlide` 测试：验证监听 `demo:concurrent-bids-completed` 事件并正确更新价格和出价输入
- 后端测试：验证串行递增逻辑、CapPrice 提前终止、increment 自动修正

**相关文档**：
- 设计文档：`docs/superpowers/specs/2026-06-10-h5-demo-concurrent-bids-design.md`
- 实现计划：`docs/superpowers/plans/2026-06-10-h5-demo-concurrent-bids-plan.md`

### H5 出价抽屉价格同步问题 (Bid Drawer Price Sync)

**问题背景**：用户点击「并发出价」后，H5 出价抽屉显示的 `bidAmount/minBid` 与实际 `current_price` 不一致。例如：抽屉显示最低出价 440，但实际当前价已被抬到 460，导致用户出价 450 一直失败。

**根因分析**：
1. 出价抽屉的 `minBid` 基于 H5 本地状态 `current_price` 计算
2. 并发出价成功后，`DemoConsole` 只弹 toast，没有主动同步最新价格给直播间
3. 直播间价格状态主要依赖 `bid_placed` WebSocket 事件更新
4. 在本地演示/高频点击场景下，WS 可能延迟或丢失最后几笔状态，导致 H5 状态滞后

**解决方案**：
并发出价成功后，应将后端返回的 `highest_amount` 作为权威价格同步给 LiveRoom，而不是只等 WS。

**具体实现**：
1. 新增 `demo:concurrent-bids-completed` 自定义事件，由 `DemoConsole` 在并发出价成功后广播
2. 事件 payload 携带 `highest_amount`（后端返回的权威最高价）
3. `LiveRoomSlide` 监听该事件，收到后向上修正 `current_price`，确保出价抽屉的 `minBid` 同步刷新
4. 同时更新 `bidAmount` 输入框为新的最低出价，避免用户基于旧价格出价失败

**关键代码模式**：
```tsx
// DemoConsole.tsx - 并发出价成功后广播事件
window.dispatchEvent(new CustomEvent('demo:concurrent-bids-completed', {
  detail: { highest_amount: response.highest_amount }
}));

// LiveRoomSlide.tsx - 监听事件并同步价格
useEffect(() => {
  const handler = (e: CustomEvent) => {
    const newPrice = e.detail.highest_amount;
    if (newPrice > currentPrice) {
      setCurrentPrice(newPrice);
      setBidAmount(newPrice + minIncrement);
    }
  };
  window.addEventListener('demo:concurrent-bids-completed', handler);
  return () => window.removeEventListener('demo:concurrent-bids-completed', handler);
}, [currentPrice, minIncrement]);
```

**教训**：
- 高频出价场景下不能仅依赖 WebSocket 做状态同步，需要主动拉取或接口返回权威状态
- Demo 控制台与直播间之间的状态同步需要考虑异步延迟
- 出价抽屉的 `minBid` 计算应基于最新权威价格，而非本地可能滞后的状态
- 跨组件状态同步优先考虑自定义事件（CustomEvent），避免引入复杂的状态管理库

**来源**：session:6a2879d10bfcee1b04fc3745

### 故障注入判定逻辑与 FAIL 根因分析 (Chaos Report AllOK Logic)

**问题背景**：故障注入实验结束后显示 `结论 = FAIL`，但用户不确定是系统真实故障还是判定逻辑问题。

**判定规则源码**（`backend/test/scenario/chaos/chaos.go`）：
```go
rep.AllOK = rep.InjectErrorRate > rep.BaselineErrorRate &&
    rep.RecoverErrorRate <= rep.InjectErrorRate
```

**判定标准**：
- 注入阶段错误率必须**高于**基线阶段
- 恢复阶段错误率必须**低于或等于**注入阶段

**常见 FAIL 场景**：

| 场景 | 原因 | 解决方案 |
|------|------|----------|
| **延迟注入被判 FAIL** | 延迟注入只抬升 `avg_latency_ms`，不产生错误，`inject_error_rate` 仍为 0 | 按故障类型分判定：`latency` 看延迟抬升/恢复，而非错误率 |
| **基线已污染** | 上一次 chaos profile 未完全恢复，或探测目标本身不稳定，`baseline_error_rate` 已很高 | 每次 chaos 开始前执行 `RecoverAll()`，保证 baseline 从干净状态开始 |

**修复方案**：
1. **按故障类型分判定**：
   - `error_rate` / `disconnect`：看错误率变化
   - `latency`：看延迟抬升和恢复
2. **实验前清理**：`Run` 启动前清空 broker，确保 baseline 干净

**来源**：session:6a27c83e0bfcee1b04fbb4a9

---

### 韧性曲线实时渲染模式 (Resilience Curve Live Rendering)

**问题背景**：故障注入图表在实验结束后才渲染，无法实时观察故障注入过程。

**解决方案**：图表数据优先来自 WebSocket `history` 实时消息，结束后再用报告兜底。

**数据流设计**：
```
Chaos WS (每秒推送)
  ↓
history: {ok_count, fail_count, avg_latency_ms, phase}
  ↓
前端转成 Bucket 并追加到曲线数据
  ↓
Recharts 实时渲染韧性曲线
  ↓
最终报告回来 → 切换为最终汇总数据
```

**关键实现**：
- `Chaos` 组件订阅 WS `history` 消息
- 每秒消息到达即转成 `Bucket` 并渲染
- 最终报告回来后切换为汇总数据（更精确）

**故障类型说明展示**：
- 随当前选择的故障类型动态展示文字说明
- 说明故障的实现方式（如延迟注入通过 RoundTripper 延迟响应实现）

**来源**：session:6a27c83e0bfcee1b04fbb4a9

---

### 并发出价设计决策演变 (Concurrent Bids Design Evolution)

**初始方案（已废弃）**：
使用 goroutine 真并发发起出价，期望模拟高并发竞争场景。

**问题发现**：
- 业务层有 `AuctionBidLock` + 乐观锁，真并发会导致大量"出价过于频繁"失败
- 并发下"第 i 笔金额"与"当时的 current_price"关系不确定，小金额的会撞"已被超越"失败
- `failure_count` 不可控，`highest_amount` 不确定，演示效果不稳定

**最终方案（方案 A - 串行快速递增）**：
- 后端改为**串行循环出价**，每笔等上一笔成功再下一笔
- 每笔基于上一笔成功后的最新价递增
- `failure_count` 接近 0，`highest_amount` 确定，H5 飘屏顺序自然

**核心权衡**：
放弃"并发压测"的语义准确性，换取"让用户用旧价稳定失败"的演示目标。按钮文案可保留「并发压测」，但实质是"一键快速抬价"。

**来源**：session:6a2879d10bfcee1b04fc3745

### 并发出价价格同步的 Postmortem 复盘

**问题现象**：高频并发出价后，H5 出价抽屉显示的最低起拍价与实际当前价不一致，导致用户以"旧价"出价时频繁收到"已被超越"失败提示。

**根因层级**：
1. **直接原因**：`DemoConsole` 并发出价成功后仅弹 toast，未主动同步权威价格给直播间
2. **系统原因**：直播间价格状态主要依赖 `bid_placed` WebSocket 事件更新，本地演示/高频场景下 WS 可能延迟或丢失最后几笔状态
3. **设计原因**：出价抽屉的 `minBid` 计算基于本地可能滞后的 `current_price`，而非权威后端状态

**修复方案**：
1. 新增 `demo:concurrent-bids-completed` 自定义事件，由 `DemoConsole` 在并发出价成功后广播
2. 事件 payload 携带后端返回的 `highest_amount` 作为权威最高价
3. `LiveRoomSlide` 监听事件并向上修正 `current_price`，同步刷新出价抽屉的 `minBid` 和输入框值

**经验沉淀**：
- 高频出价场景下不能仅依赖 WebSocket 做状态同步，需要主动通过接口返回或事件机制同步权威状态
- Demo 控制台与直播间之间的状态同步需考虑异步延迟，跨组件状态同步优先考虑自定义事件（CustomEvent）
- 出价相关 UI 应基于最新权威价格计算，而非本地可能滞后的缓存状态

**来源**：session:6a2879d10bfcee1b04fc3745

---

### 线上并发问题记录案例 (Online Concurrency Issue Case)

**问题背景**：线上 Demo 环境出现的真实并发一致性问题，用于在亮点 HTML 和提交文档中展示工程实践中的真实踩坑记录。

**现象描述**：
用户点击「并发出价」后，H5 出价抽屉显示旧价 440，实际最低已到 460，导致用户出价 450 失败。

**根因分析**：
1. H5 出价抽屉的 `minBid` 基于本地状态 `current_price` 计算
2. 并发出价成功后，`DemoConsole` 仅弹 toast，未主动同步最新价格给直播间
3. 直播间价格状态主要依赖 `bid_placed` WebSocket 事件更新
4. 在高频点击/本地演示场景下，WS 可能延迟或丢失最后几笔状态，导致 H5 状态滞后于后端真实最高价

**修复方案**：
1. 新增 `demo:concurrent-bids-completed` 自定义事件
2. 由 `DemoConsole` 在并发出价成功后广播，携带后端返回的 `highest_amount` 作为权威最高价
3. `LiveRoomSlide` 监听该事件，收到后向上修正 `current_price`，确保出价抽屉的 `minBid` 同步刷新

**展示价值**：
- 体现项目在高频出价场景下的真实问题发现与修复能力
- 展示 WebSocket 状态同步的边界情况和兜底方案设计
- 可作为亮点 HTML `#websocket` 章节的实战案例卡片

**来源**：session:6a2875380bfcee1b04fc33e8（引用 session:6a2879d10bfcee1b04fc3745 的问题记录）

### E2E 演示剧场设计决策 (E2E Demo Theater Design Decision)

**问题背景**：独立测试平台的 E2E 测试直观性不强，演示后看不出业务价值，评委无法建立因果感。

**核心洞察**：
- 当前 E2E 页更像"CI 结果面板"：进度、步骤、最终 ID 有了，但缺少"业务状态如何被推动"的可视化证据
- 问题不只是缺场景，而是缺**演示叙事层**

**设计决策**：
1. **入口边界**：复用 `/test/screen` 作为演示剧场主入口，而非新增第四个相似页面
2. **叙事结构**：角色与初始条件 → 实时业务动画 → 最终证明
3. **核心目标**：让非技术观众在 30 秒内理解系统自动创建直播竞拍并验证并发、交易、订单、库存一致性
4. **一键启动**：打开大屏 → 点开始 → 自动造数 → 实时播放 → 最后给结论，评委无需理解 seller_id/bidder_id/duration 等测试参数

**技术实现方向（混合方案）**：
- 不在 test-dashboard 里完全重写 H5 页面（重复成本高）
- 不直接 iframe 普通 H5 页面（容易被登录态、路由、数据加载卡住）
- 最佳方案：给 H5 增加专用 `demo/live-room` 展示路由，使用 gateway `/api/v1` 数据和测试事件驱动
- `/test/screen` 左侧嵌入 H5 demo route，右侧消费 test-service 的 progress WS 和 report

**关键转变**：
从"测试看板"变成"自动播放的直播竞拍验收短片"，右边挂技术证据。

**来源**：session:6a22eaa12ec60aa1a73a3e14

### 压测场景设计决策 (Pressure Test Scenario Design)

**两种核心场景**：

| 场景 | 目标 | 负载模型 | 预期失败类型 |
|------|------|----------|--------------|
| **单拍卖热点冲突** | 演示业务冲突率 | 多 worker 盲压同一拍卖 | 大量 400（出价被超越/金额不足） |
| **吞吐压测** | 测系统吞吐能力 | 多拍卖分片分散压力 | 接近 0 失败 |

**关键设计决策**：
1. **单拍卖热点场景保留**："100 人同时盲压一个拍卖"是真实业务场景，用于演示竞价激烈程度
2. **吞吐场景新增**：自动创建多个 auction fixture 分片，worker 固定压自己的拍卖，避免业务冲突混入吞吐评估
3. **fixture_count=0 语义**：按并发用户数创建分片（100 并发 → 100 个拍卖），最大程度减少业务冲突

**并发用户数与 QPS 关系**：
```
QPS ≈ 并发数 / 单次请求平均耗时

示例：
- 500 并发 × 500ms 平均耗时 = 1000 QPS
- 500 并发 × 250ms 平均耗时 = 2000 QPS
```

**来源**：session:6a22eaa12ec60aa1a73a3e14

### 压测错误码可解释性设计 (Pressure Test Error Code Interpretability)

**错误码中文解释映射**：

| Code | 中文含义 | 常见原因 | 是否系统异常 |
|------|----------|----------|--------------|
| `400` | 业务拒绝 | 出价被超越、金额不足、出价过于频繁、请求参数错误 | 否（业务预期） |
| `429` | 网关限流 | 触发 IP 限流阈值（默认 1000 req/s/IP） | 否（保护机制） |
| `500` | 服务端内部错误 | DB/Redis/事务/锁异常 | 是（需排查） |
| `502` | 网关上游错误 | Gateway 连接 upstream 失败、upstream 未返回完整响应 | 是（需排查） |
| `0` | 客户端未收到 HTTP 响应 | 请求超时、连接中断、context cancel、端口耗尽 | 视情况 |

**Code 0 细分（优化后）**：
- `context_canceled`：压测正常结束时的尾部请求，应从失败统计剔除
- `timeout`：客户端超时（如 5s）但服务端仍在处理，需关注
- `connection_reset`：连接被重置，可能服务端过载
- `cant_assign_address`：本机临时端口耗尽（本地特有）

**来源**：session:6a22eaa12ec60aa1a73a3e14

### Gateway 连接池治理 (Gateway Connection Pool Governance)

**问题现象**：
高 QPS 压测下出现大量 `Code 0` 和 `502`，日志显示 `can't assign requested address`。

**根因分析**：
1. 没有显式连接池时，高并发请求频繁新建 TCP 连接
2. 本机到 `auction-service`、`Redis` 都是 `127.0.0.1`，短时间大量建连消耗本机临时端口
3. 端口耗尽后报错 `can't assign requested address`

**解决方案**：
```go
// Gateway proxy 连接池配置
transport := &http.Transport{
    MaxConnsPerHost:     512,  // 到同一 upstream 的最大连接数
    MaxIdleConnsPerHost: 512,  // 最大空闲连接数
    IdleConnTimeout:     90 * time.Second,
}
```

**机制解释**：
1. **连接复用**：大量请求复用已有长连接，不再每次都新建连接，避免临时端口打爆
2. **有界并发 + 排队背压**：当到同一 upstream 的并发连接达到上限后，新请求在 `net/http.Transport` 内等待可用连接，而不是无限建连

**代价**：
- 连接池保护系统，但可能降低瞬时 QPS 或增加排队延迟
- 如果排队时间超过客户端 timeout，仍然可能看到 `code=0`

**云上 vs 本地差异**：
- **本地特有**：`can't assign requested address`（单机所有服务共享端口池）
- **云上通用**：连接池无界/太小导致的排队、超时、`502/504`

**来源**：session:6a22eaa12ec60aa1a73a3e14

### 压测客户端超时与排队延迟 (Client Timeout vs Server Queuing)

**问题现象**：
高 QPS 下 `Code 0` 数量大，P99 延迟达到 10000ms（客户端 timeout）。

**根因分析**：
- 压测客户端 HTTP timeout：5s
- Gateway 日志显示请求排队：8~17s
- 客户端先放弃（Code 0），但服务端后面还在继续处理

**关键公式**：
```
客户端感知失败 = 服务端排队延迟 > 客户端超时
```

**优化方向**：
1. **调整客户端 timeout**：与预期服务端处理时间匹配
2. **有界等待/快速失败**：热点场景下服务端排队过长时主动拒绝，而非无限等待
3. **排队延迟指标**：监控 `p95/p99` 排队时间，作为容量规划依据

**来源**：session:6a22eaa12ec60aa1a73a3e14

### 本地 vs 云上环境问题分析 (Local vs Cloud Environment Issues)

**本地特有、云上大概率缓解的问题**：
- `can't assign requested address`：本机压测端、gateway、auction、Redis 都跑在同一台机器，所有 `127.0.0.1` 短连接消耗同一台机器的临时端口
- macOS 本机端口/FD/网络栈参数偏保守，高 QPS 下更容易先打到本机资源限制
- Vite dev server、`go run`、本地 Docker/MySQL/Redis 混跑在一台机器上，放大抖动

**云上仍然可能出现的问题**：
- 压测工具和 gateway 同机，或 gateway 到 auction/Redis 仍然大量短连接，临时端口耗尽依然会出现
- 非 2xx response body 不 drain，HTTP 连接复用失效，云上也会继续大量新建连接
- 连接池无界或配置太小，仍会表现为排队、超时、`502/504`
- 限流 key TTL 丢失，`429` 永久化误伤在云上也会发生
- 单拍卖热点冲突下的 `400` 是业务逻辑，不管本地还是云上都会存在，只是比例会随延迟和并发调度变化

**修复的通用性**：
这次修复不是"只为本地绕过"，修的是通用工程问题：
- 客户端 drain body，保证连接复用
- gateway/auction Redis 使用有界连接池，避免连接风暴
- 压测统计剔除结束窗口外的 in-flight 噪音
- 限流 key 补 TTL，避免永久限流
- 页面把 `400/429/500/0` 做可解释展示

**云上部署后建议基线**：
- `hot_auction`：验证业务冲突率、429 是否符合预期
- `throughput`：验证系统吞吐、P95/P99、0/500/502 是否为 0 或极低

**判断标准**：云上可以让性能更好，但不能依赖云环境掩盖连接复用、连接池、限流 TTL 这类代码问题。

**来源**：session:6a22eaa12ec60aa1a73a3e14

### 防狙击测试场景修复 (Anti-Snipe Test Scenario Fix)

**问题现象**：
防狙击测试场景失败，`prepare: create_auction: HTTP 403`。

**根因分析**：
防狙击场景的 `factory.go` 仍然直接 `CreateAuction(ctx, sellerID, ...)`：
1. 按默认普通用户角色发请求（非商家）
2. 没有先创建 `auction_rule`
3. 被当前后端契约 403 拒绝

**修复方案**：
按 E2E/压测已修过的链路统一修复：
1. 先创建商家账号（`role=1`）
2. 创建商品
3. 创建竞拍规则（`auction_rule`）
4. 再创建拍卖

**来源**：session:6a22eaa12ec60aa1a73a3e14

### 测试平台 API 契约审查要点 (Test Platform API Contract Review)

**问题背景**：
代码审查发现测试平台与后端服务间的 API 契约存在两个潜在问题：
1. `CreateForCreator` 返回码语义不明确（201 vs 200）
2. `GetAuctionResult` 对业务信封 `code` 字段缺乏失败关闭校验

**审查发现**：
1. **创建返回码问题**：`CreateForCreator` 无论新建还是幂等命中都返回 201，调用方无法区分"首次创建"和"重复调用"
2. **信封校验问题**：测试 SDK 对 `{code,data}` 响应信封没有失败关闭，遇到 HTTP 200 但 `code!=0/200` 时会被零值掩盖

**修复方案**：
1. `CreateForCreator` 返回 `created bool` 给 handler 决定 201/200
2. `GetAuctionResult` 对业务信封 `code` 做 fail-closed 校验，避免零值结果被当作成功

**教训**：
- 幂等创建接口应区分"首次创建"和"重复调用"的语义
- 业务信封响应必须做 fail-closed 校验，不能仅依赖 HTTP status code
- 测试 SDK 作为调用方也需要防御性编程，不能假设服务端总是返回符合契约的数据

**来源**：session:6a22ceee2ec60aa1a73a25ef

### 测试平台部署启动顺序 (Test Platform Deployment Startup Order)

**启动依赖链**：
测试平台需要按以下顺序启动：
1. 基础设施（MySQL/Redis/RabbitMQ）
2. 业务后端（gateway/product/auction）
3. test-service（独立测试服务）
4. test-dashboard 前端（Vite dev server）

**常见问题**：
1. **404 错误**：test-service 未启动或 Gateway 未配置 `/api/test` 代理路由
2. **500 错误**：`user_journey` 准备阶段调用下游服务失败，如 `prepare create_live_stream` 返回 400/500

**排查路径**：
1. 确认 `backend/test` 服务已启动并监听 `18090`
2. 确认 Gateway 已配置 `/api/test` 路由转发到 test-service
3. 查看 test-service 日志定位具体失败的下游调用
4. 检查下游服务（product-service/auction-service）是否正常运行

**验证命令**：
```bash
# 检查 test-service 是否监听
curl -s http://127.0.0.1:18090/health

# 检查 Gateway 代理是否通
curl -s http://127.0.0.1:8080/api/test/scenarios

# 触发最小 user-journey 验证全链路
curl -X POST http://127.0.0.1:8080/api/test/user-journey -d '{"duration_ms":1000}'
```

**来源**：session:6a22ceee2ec60aa1a73a25ef

### WebSocket 链接中断显示优化 (WebSocket Connection Status Display)

**问题现象**：
单拍卖热点冲突压测结束时，页面显示"链接中断"错误。

**根因分析**：
- WebSocket 的 `client closed` 是前端切换测试或页面关闭后的**正常断开**，不是后端异常
- 前端把测试完成后的 WS close 误显示为错误状态

**修复方案**：
把正常完成后的 close 当成**完成态**，而不是错误态：
1. 测试正常完成后，WS 关闭应显示"测试完成"而非"链接中断"
2. 区分"异常断开"（网络错误、服务端错误）和"正常关闭"（测试完成、页面切换）

**来源**：session:6a22eaa12ec60aa1a73a3e14

## Project Highlight Integration

### 故障注入的架构表达
在「5端 + 可观测」架构图中，故障注入应体现为**独立测试平台发起的横切控制面**，而非侵入业务服务主路径：

- **位置**：位于 `test-dashboard` 背后的 `test-service / chaos scenario`
- **注入链路**：Test Dashboard 发起 → test-service 执行 chaos scenario → 进程内 RoundTripper / probe client 注入 latency / error_rate / disconnect → 探测 gateway / health / API → 结果回流到测试大屏和可观测栈
- **架构价值**：突出测试平台不是展示页，而是可发起压测、混沌、回调、反狙击等测试的**控制面**
- **边界说明**：业务流量仍走 `gateway /api/v1`，混沌测试是旁路探测与注入，不污染业务服务

来源：session:6a25c5830bfcee1b04fb1c9e
