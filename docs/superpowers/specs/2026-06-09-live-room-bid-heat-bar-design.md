# 直播间「战况热度条」设计 (A1)

- 日期: 2026-06-09
- 范围: 仅 H5 前端 `frontend/h5`，不改后端、不改数据契约
- 驱动: 演示表现力（营造竞拍激烈度的可感知氛围）

## 1. 目标

在直播间根据近期出价频率，实时表达「这场拍卖有多火」。把零散的出价飘屏升级为一条持续可感知的「战况热度条」，让用户（及评委）一眼看到竞拍激烈程度。

## 2. 非目标

- 不做边框呼吸光晕、不做背景粒子（成本高、易与已有飘屏视觉打架）。
- 不依赖后端新增字段或接口，纯前端基于现有实时事件与本地成功出价事件计算。
- 不改动出价提交逻辑（A2 已从本次范围移除）。

## 3. 激烈度算法

- 数据源（三处，缺一会漏算）:
  - `ws.on('bid_placed', ...)`（`LiveRoomSlide.tsx` L752）——主要是他人出价。
  - `handleBid` 成功分支（L888 附近）——本地用户走 REST 出价，不保证自身收到 `bid_placed` 回推，必须显式补记。
  - `ws.on('sky_lamp_auto_bid', ...)`（L779）——点天灯自动跟价。
  - 三处均调用 `markBid()` 记一次。不做去重：滑窗档位对偶尔重复计数不敏感，去重会增加复杂度（YAGNI）。
- 窗口: 10 秒滑动窗口，统计窗口内出价事件数 `count`。
- 分档（3 档）:
  - 冷静 (calm): `count <= 1`
  - 升温 (warming): `2 <= count <= 4`
  - 白热 (blazing): `count >= 5`
  - 阈值集中定义为常量，便于演示时微调。
- 回落: 窗口内无新出价时，随时间推移 `count` 自然衰减，热度平滑回落，避免长期停留在白热档失真。
- 实现方式: 维护一个出价时间戳数组，定时（约每 1s）裁剪掉 10s 前的时间戳并重算档位。组件卸载时清理定时器。

## 4. 组件设计

新增组件 `BidHeatBar`（`frontend/h5/src/components/LiveRoom/BidHeatBar.tsx` + 同名 `.module.css`）。

- 职责单一: 输入「当前档位 level」+「人数信息」，输出对应视觉的热度条。不感知 WS、不感知业务，纯展示。
- Props:
  - `level: 'calm' | 'warming' | 'blazing'`
  - `bidderCount: number`
  - `viewerCount: number`
- 视觉:
  - 复用现有 `TreasureProgressBar` 的样式风格与设计 Token，保持直播间视觉一致。
  - 档位越高，填充比例越高、配色越暖；白热档加流光动画（`transform`/`opacity` 实现，不用 `width`/`left` 动画以保性能）。
  - 文案由「档位文案 + 现有人数信息」组成，例如「战况白热 · 已有 N 人出价 · M 人围观」。
  - 适配双主题（light/dark）。
  - `prefers-reduced-motion` 下关闭流光动画，仅保留静态档位色。

## 5. 集成（升级现有 marquee，不新增布局）

- 现状: `LiveRoomSlide.tsx` L1181-L1185 的 `heatMarqueeContainer` 已是一条静态「🔥 已有 N 人出价 · M 人围观」热度文案，位于 BidDock 抽屉内。
- 方案: 用 `BidHeatBar` 替换 `heatMarqueeContainer`，复用其位置（抽屉内、价格块下方）。语义从静态文案升级为档位驱动的战况条，避免出现两个「热度」语义冲突，也不碰直播间主画面对齐约束。
- 计算逻辑放在自定义 Hook `useBidHeat`（`frontend/h5/src/hooks/useBidHeat.ts`），输出 `level` 与 `markBid()`，与渲染解耦。
- 在第 3 节所述三处出价来源调用 `markBid()`。

## 6. 数据流

```
WS 'bid_placed' / handleBid 成功 / 'sky_lamp_auto_bid' --> useBidHeat.markBid()
useBidHeat 内部 10s 滑窗 + 定时衰减 --> level
level + 人数 --> <BidHeatBar /> 渲染（替换原 heatMarquee 位置）
```

## 7. 错误与边界

- 进直播间初始无出价: 显示冷静档。
- 切换竞拍场次/重连: 重置时间戳窗口，避免跨场次累计。
- 高频出价: 时间戳数组仅保留 10s 内，内存可控。

## 8. 测试 (TDD)

- `useBidHeat`: 单元测试覆盖三档边界（1/2/4/5 次）、衰减回落、窗口裁剪、重置。用假定时器控制时间。
- `BidHeatBar`: 渲染测试覆盖三档 className/文案、`prefers-reduced-motion` 降级。
- 布局回归: 若放入直播间固定布局，补一条 CSS 布局断言，防止再次偏移（参考既有 `LiveLayoutCss.test.ts` 模式）。

## 9. 通用约束

- 双主题适配、使用设计 Token、`prefers-reduced-motion` 关闭动效。
- 先写失败测试，再最小实现，再验证（`npm test` + `npm run build`）。
