# 用户触达 Metrics 打点设计文档

**日期**：2026-06-02
**适配仓库**：`dy-ai-live-auction-fullstack-cc`
**范围**：移动端 H5 触达事件、`gateway-service` 前端埋点入口、Prometheus/Grafana 指标链路

---

## 1. 背景与目标

用户触达一期已经具备红点、通知中心、热拉通知和开播提醒弹窗等能力，但当前实现只能回答“是否有待处理触达”或“是否已经提醒过”，不能回答“用户是否实际看到、点击、关闭、转化”。

本次目标是补齐有业务价值的触达打点，并复用项目已有 metric 链路：
- 前端统一通过 `trackEvent()` 上报触达事件。
- Gateway 复用已有 `POST /api/v1/track` 接口接收前端埋点。
- Gateway 将触达事件写入 Prometheus Counter。
- Grafana 通过 PromQL 查询触达曝光、点击和转化趋势。

不做事件明细落库，不引入新的第三方埋点 SDK，不把 `user_id`、`notification_id`、`live_stream_id` 等高基数字段作为 Prometheus label。

---

## 2. 当前仓库事实

现有链路已经包含基础 metrics 能力：
- `gateway-service` 在 `main.go` 初始化 `metrics.Init("gateway")`。
- `gateway-service` 已注册 `POST /api/v1/track`，处理函数为 `metrics.TrackEvent(m)`。
- `gateway-service` 已启动 Prometheus metrics server，监听 `:9090`。
- `gateway/pkg/metrics/handler.go` 已支持 `live_room_enter`、`bid_click`、`payment_start` 等老事件。
- `gateway/pkg/metrics/handler.go` 的 `default` 分支目前不记录未知事件，因此触达事件即使上报也不会进入 Prometheus。
- `frontend/h5/src` 下没有统一 tracking 模块，触达页面和 hook 也没有任何 `track/report/beacon` 调用。

关键约束：
- 前端业务 API 默认走 `/api/v1`，但现有埋点入口是 `/api/v1/track`，不能直接复用 `services/api.ts` 的普通业务请求封装。
- 触达打点不能阻塞 UI、不能影响通知展示、不能因为上报失败导致业务失败。
- Prometheus label 必须低基数，避免把用户、通知、直播间 ID 作为 label。

---

## 3. 推荐方案

采用“前端统一封装 + 复用 Gateway 埋点入口 + 新增触达 Prometheus Counter”的方案。

### 方案 A：只做前端 `trackEvent()` 并 console/debug

优点是改动最小，但无法进入现有 Grafana 面板，不能满足业务可视化分析诉求。

### 方案 B：前端统一 `trackEvent()`，Gateway 写 Prometheus Counter（推荐）

优点是复用现有 `/api/v1/track`、Prometheus 和 Grafana 链路，改动范围可控；指标足够支撑触达曝光、点击和转化漏斗分析。

### 方案 C：新增事件明细表或日志分析链路

优点是可追踪用户级和通知级明细，但范围更大，并且需要数据治理、隐私边界和查询入口设计。当前阶段先不做。

结论：采用方案 B。

---

## 4. 前端设计

新增统一封装，推荐路径为 `frontend/h5/src/utils/trackEvent.ts`。

职责：
- 统一生成事件 payload：`event_type`、`event_name`、`params`、`timestamp`。
- 统一上报到 `/api/v1/track`。
- 优先使用 `navigator.sendBeacon`，失败或不可用时 fallback 到 `fetch` + `keepalive`。
- 上报失败只在开发环境输出调试信息，不抛错、不影响 UI。

Payload 形态：

```json
{
  "event_type": "touchpoint_event",
  "event_name": "summary_exposed",
  "params": {
    "source": "bottom_nav",
    "entry": "profile_tab",
    "type": "all",
    "result": "success",
    "count_bucket": "1_5"
  },
  "timestamp": 1780300800000
}
```

前端只传低基数聚合字段：
- `source`：页面或组件来源，例如 `home`、`bottom_nav`、`profile`、`notification_center`、`mobile_shell`。
- `entry`：触达入口，例如 `notification_bell`、`profile_tab`、`auction_history`、`live_reminder_modal`。
- `type`：触达类型，例如 `all`、`pending_payment`、`outbid`、`ending_soon`、`live_start`。
- `result`：结果，例如 `success`、`failed`、`clicked`、`dismissed`、`debounced`。
- `count_bucket`：数量分桶，例如 `0`、`1`、`2_5`、`6_10`、`10_plus`。

---

## 5. 事件清单

首批接入这些触达事件：

| 事件名 | 触发时机 | 必要参数 |
| --- | --- | --- |
| `summary_exposed` | 红点/触达摘要被渲染且有有效响应 | `source`, `entry`, `type`, `result`, `count_bucket` |
| `entry_clicked` | 用户点击触达入口 | `source`, `entry`, `type`, `result` |
| `notification_list_exposed` | 通知中心列表加载完成 | `source`, `entry`, `type`, `result`, `count_bucket` |
| `notification_item_clicked` | 用户点击通知卡片 | `source`, `entry`, `type`, `result` |
| `mark_read` | 单条、全部或分类标记已读成功/失败 | `source`, `entry`, `type`, `result` |
| `hot_pull_triggered` | 登录成功或切回前台触发热拉 | `source`, `entry`, `type`, `result`, `count_bucket` |
| `live_reminder_exposed` | 开播提醒弹窗实际打开 | `source`, `entry`, `type`, `result` |
| `live_reminder_clicked` | 点击“立即前往” | `source`, `entry`, `type`, `result` |
| `live_reminder_dismissed` | 点击“稍后再看”或遮罩关闭 | `source`, `entry`, `type`, `result` |

不在首批接入：
- 每个通知卡片的逐条曝光明细。
- 用户级、通知级、直播间级明细分析。
- 跨端统一埋点协议。

---

## 6. Gateway Metrics 设计

在 `gateway/pkg/metrics.Metrics` 中新增一个低基数 Counter：

```text
touchpoint_event_total{event, source, entry, type, result}
```

字段含义：
- `event`：事件名，例如 `summary_exposed`。
- `source`：页面或组件来源。
- `entry`：触达入口。
- `type`：触达类型。
- `result`：事件结果。

处理规则：
- `TrackEvent` 收到 `event_type=touchpoint_event` 时读取参数并记录 Counter。
- 未知事件名允许记录，但 label 值必须经过白名单或归一化，非法值落到 `unknown`。
- 缺失参数使用默认值：`source=unknown`、`entry=unknown`、`type=unknown`、`result=unknown`。
- 不把 `user_id` 写入 label；如 request 里携带 `user_id`，后端忽略或仅用于日志调试，不进入 Prometheus。

---

## 7. Grafana 查询示例

总曝光量：

```promql
sum(rate(touchpoint_event_total{event=~".*_exposed"}[5m]))
```

触达入口点击量：

```promql
sum by (entry) (rate(touchpoint_event_total{event="entry_clicked"}[5m]))
```

开播提醒点击率：

```promql
sum(rate(touchpoint_event_total{event="live_reminder_clicked"}[5m]))
/
sum(rate(touchpoint_event_total{event="live_reminder_exposed"}[5m]))
```

通知中心点击率：

```promql
sum(rate(touchpoint_event_total{event="notification_item_clicked"}[5m]))
/
sum(rate(touchpoint_event_total{event="notification_list_exposed"}[5m]))
```

---

## 8. 错误处理

前端：
- `trackEvent()` 不抛业务可见错误。
- `sendBeacon` 返回 `false` 时 fallback 到 `fetch`。
- `fetch` 失败只在开发环境输出调试日志。
- 页面卸载、跳转和关闭弹窗时仍尽量上报，但不阻塞交互。

后端：
- 请求体非法返回 `400`。
- 合法但未知的触达参数归一化为 `unknown`，避免 panic。
- metrics 记录失败不影响业务 API，因为 `/api/v1/track` 是独立入口。

---

## 9. 测试策略

前端测试：
- `trackEvent()` 能按协议发送 `/api/v1/track`。
- `sendBeacon` 可用时优先使用 beacon。
- `sendBeacon` 不可用或返回 `false` 时 fallback 到 `fetch`。
- 触达关键组件在对应时机调用 `trackEvent()`。

后端测试：
- `TrackEvent` 收到 `touchpoint_event` 后递增 `touchpoint_event_total`。
- 缺失参数不会 panic，并落到 `unknown`。
- 高基数字段不会进入 Prometheus label。

手动验证：
- 启动 gateway 后触发 H5 触达行为。
- 访问 Prometheus metrics，确认存在 `touchpoint_event_total`。
- 在 Grafana Explore 中用 PromQL 查询事件趋势。

---

## 10. 风险与边界

主要风险：
- 事件过多导致前端重复上报。通过 React effect 依赖、去重 ref 和 count bucket 控制。
- Prometheus label 高基数。通过白名单和字段归一化控制。
- `/api/v1/track` 不在 `/api/v1` 下，前端封装必须显式走该路径。

边界：
- 本设计只做聚合指标，不提供用户级行为明细。
- `live_stream_reminder_receipts` 继续作为业务去重 receipt，不替代分析埋点。
- 后续若需要精细化归因，再设计日志或事件明细表，不在本次范围内。
