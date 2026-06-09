# 演示台「剧场模式」设计 (C1)

- 日期: 2026-06-09
- 范围: 仅 `frontend/test-dashboard`，不改后端、不改数据契约
- 驱动: 演示表现力（让评委零操作看完整韧性故事 + 量化结论）

## 1. 目标

把现有故障注入演示（`Chaos.tsx`）从「手动填参 + 启动」升级为可一键播放的「剧场模式」：自动按时间线跑 baseline → inject → recover，配阶段旁白字幕，并在曲线上标注关键事件锚点、给出量化结论指标卡。

## 2. 现状（已具备，不重做）

- 韧性曲线: 错误率 / 平均延迟 / 成功 QPS 三线 + baseline/inject/recover 三段背景色（`ResilienceCurve`）。
- 后端报告已返回: `baseline_error_rate` / `inject_error_rate` / `recover_error_rate` / `detection_latency_ms` / `recovery_latency_ms` / `all_ok`。
- 实时观测: WS `step` 阶段推进 + `buildLiveResilienceReport` 实时构建曲线。
- 各故障类型实现原理说明（`describeFaultImplementation`）。

## 3. 范围（本次新增）

### C1-a 一键剧本（与手动模式并存）

- 在 `Chaos.tsx` 新增「开始演示」按钮，与现有「启动 / 取消」手动模式并存（手动用于调试，一键用于演示）。
- 剧本类型: 单故障剧本，固定 `fault_type = 'error_rate'`，使用一组演示友好的预设参数（如 baseline 3s / inject 8s / recover 5s / error_rate 0.5）。
- 执行: 点击后用预设参数复用现有 `startChaos` 流程（复用 `start()` 中 WS 连接 + 轮询逻辑），无需手动填参。
- 前置: 依赖后端实验开始前已强制 `RecoverAll()` 的既有行为确保基线纯净（无需前端额外处理，但在 spec 中记录该依赖）。
- 运行互斥: `running` 只表示启动请求进行中，不能代表实验全生命周期。按钮禁用必须同时考虑 `testID`、`progress`、`step`，仅当无测试或测试已进入终态（`step === 'done'` / `step === 'failed'` / `progress >= 100`）才允许再次启动。

### C1-b 阶段旁白字幕

- 随 WS `step` 阶段切换，在曲线/进度区顶部显示一句旁白:
  - baseline: 「正在采集基线指标，建立健康水位…」
  - inject: 「正在注入约 50% 错误率，观察系统反应…」（文案按当前 error_rate 动态生成）
  - recover: 「故障已移除，正在观测系统自愈…」
  - 结束: 当 `report` 已返回，或 `step === 'done'` / `progress >= 100` 且已有可用报告时，用 `summarizeResilienceReport` 已有结论文案收尾。
- 仅手动模式与一键模式共用同一旁白逻辑（基于 `step` / `progress` / `report`，不区分入口）。

### C1-c 曲线锚点 + 量化指标卡

- 锚点（recharts `ReferenceLine`，叠加在现有 `ResilienceCurve` 上）:
  - 注入时刻: 第一个 `inject` bucket 的 index。
  - SLA 击穿: 错误率首次超过阈值（如 5%）的 index（若存在）。
  - 恢复拐点: 第一个 `recover` bucket 的 index。
- 量化指标卡（在现有 Metric 网格基础上新增，纯前端从 buckets 计算，不依赖新后端字段）:
  - 恢复耗时: 复用后端 `recovery_latency_ms`。
  - 峰值错误率: inject 阶段各 bucket 错误率最大值。
  - 损失 QPS: baseline 阶段平均成功 QPS − inject 阶段平均成功 QPS（小于 0 取 0）。

## 4. 组件与函数边界

- 旁白文案生成: 纯函数 `buildNarration({ step, progress, form, report })` → string，便于单测；结束态必须显式依赖 `report` 或终态进度，不能只看 `step`。
- 锚点计算: 纯函数 `buildCurveAnchors(buckets)` → `{ injectIndex?, slaBreachIndex?, recoverIndex? }`。
- 量化指标计算: 纯函数 `buildDemoMetrics(report)` → `{ peakErrorRatePct, lostQps, recoveryMs }`。
- 三个纯函数与渲染解耦，UI 仅消费其结果；`ResilienceCurve` 接收可选 anchors 渲染 `ReferenceLine`。

## 5. 数据流

```
点击「开始演示」 --> 预设参数 --> 复用 start()（startChaos + WS + poll）
WS step --> buildNarration --> 旁白字幕
buckets --> buildResilienceSeries + buildCurveAnchors --> 曲线 + ReferenceLine
report/buckets --> buildDemoMetrics --> 指标卡
```

## 6. 错误与边界

- 一键演示进行中按钮禁用，避免重复触发（需新增或复用类似 `isPressureStartDisabled` 的终态判断；不能只依赖当前 `running`）。
- 无 SLA 击穿（错误率从未超阈值）: 不画击穿锚点。
- 实时阶段 buckets 不全时: 锚点/指标卡按已有数据渐进显示，不报错。

## 7. 测试 (TDD)

- `buildNarration`: 覆盖 baseline/inject/recover/结束 四态及 error_rate 动态文案。
- `buildCurveAnchors`: 覆盖有/无击穿、阶段缺失场景。
- `buildDemoMetrics`: 覆盖峰值错误率、损失 QPS（含负值归零）、恢复耗时。
- 复用现有 `Chaos.test.ts` 模式补断言。

## 8. 通用约束

- 沿用 test-dashboard 现有内联样式风格与配色 Token。
- 一键模式与手动模式并存，互不干扰。
- 先写失败测试，再最小实现，再验证。
