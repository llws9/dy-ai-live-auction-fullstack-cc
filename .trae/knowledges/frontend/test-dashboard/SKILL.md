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

## Project Highlight Integration

### 故障注入的架构表达
在「5端 + 可观测」架构图中，故障注入应体现为**独立测试平台发起的横切控制面**，而非侵入业务服务主路径：

- **位置**：位于 `test-dashboard` 背后的 `test-service / chaos scenario`
- **注入链路**：Test Dashboard 发起 → test-service 执行 chaos scenario → 进程内 RoundTripper / probe client 注入 latency / error_rate / disconnect → 探测 gateway / health / API → 结果回流到测试大屏和可观测栈
- **架构价值**：突出测试平台不是展示页，而是可发起压测、混沌、回调、反狙击等测试的**控制面**
- **边界说明**：业务流量仍走 `gateway /api/v1`，混沌测试是旁路探测与注入，不污染业务服务

来源：session:6a25c5830bfcee1b04fb1c9e
