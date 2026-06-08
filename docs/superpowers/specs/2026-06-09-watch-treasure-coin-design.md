# 观看时长宝箱 + 金币资产 设计

**日期**：2026-06-09

**前端入口**：[LiveRoomSlide.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L967-L1015)（`topBar` 内 `hostPill` 下方）

---

## 1. 本质与边界

- **金币 = 虚拟娱乐积分**，与现金余额 `user_balances`（decimal/CNY，只读）完全隔离，新建独立模型。本期金币仅展示/累积，**不可消费、不抵扣竞拍或一口价**。
- **发币与时长判定是后端职责**：前端计时与动画只负责吸引点击，门槛达标判定、金额发放、幂等防重全部后端兜底。改前端无法刷币。
- **不做**：金币消费/兑换链路；金币与资金/订单联动；多档位可配置面板（金额以后端常量落地）。

## 2. 需求定档（已与用户确认）

| 维度 | 取值 |
|---|---|
| 金币用途 | 纯娱乐积分，仅展示/累积 |
| 宝箱重置周期 | 每日 0 点重置（按 `stat_date` 分桶） |
| 开箱金额 | 固定：3min→**100**，10min→**300**，30min→**800** |
| 时长口径 | **今日跨直播间累计**，每日重置；首页/其他页面停留不计（心跳只在直播间页面发） |
| 防刷 | 后端心跳累加 + 门槛校验 + 唯一键幂等 |

## 3. 数据模型（归属 auction-service）

```sql
-- 金币资产：1 用户 1 行，永久累积（整数，无小数）
CREATE TABLE user_coins (
  user_id     BIGINT       NOT NULL PRIMARY KEY,
  balance     BIGINT       NOT NULL DEFAULT 0,
  updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 今日观看时长：按天分桶，每日 0 点天然失效
CREATE TABLE user_watch_duration (
  user_id        BIGINT   NOT NULL,
  stat_date      DATE     NOT NULL,
  total_seconds  INT      NOT NULL DEFAULT 0,
  updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, stat_date)
);

-- 宝箱领取记录：唯一键 (user_id, stat_date, tier) 即幂等保证
CREATE TABLE treasure_claims (
  user_id    BIGINT  NOT NULL,
  stat_date  DATE    NOT NULL,
  tier       TINYINT NOT NULL,   -- 0=3min, 1=10min, 2=30min
  coins      BIGINT  NOT NULL,   -- 当次发放额，留存审计
  claimed_at DATETIME NOT NULL,
  PRIMARY KEY (user_id, stat_date, tier)
);
```

## 4. 接口契约（前端经 gateway `/api/v1`）

| Method/Path | 鉴权 | 用途 | 防刷点 |
|---|---|---|---|
| `POST /api/v1/watch/heartbeat` | JWT | 前端每 30s 上报一次，累加今日时长 | 服务端按「上次心跳时间差」累加，单次封顶 30s，丢弃异常大跳变 |
| `GET /api/v1/treasure/status` | JWT | 返回今日时长 + 3 宝箱状态 + 金币余额 | 状态完全由后端时长算出，前端不可篡改 |
| `POST /api/v1/treasure/claim` `{tier}` | JWT | 领取某宝箱并发币 | 校验 `total_seconds≥门槛` + 唯一键幂等；不达标/重复领取返回明确错误（失败关闭） |

**`GET /treasure/status` 响应**：

```json
{
  "code": 200,
  "data": {
    "stat_date": "2026-06-09",
    "watched_seconds": 640,
    "coin_balance": 400,
    "tiers": [
      { "tier": 0, "threshold_seconds": 180,  "coins": 100, "state": "claimed" },
      { "tier": 1, "threshold_seconds": 600,  "coins": 300, "state": "unlockable" },
      { "tier": 2, "threshold_seconds": 1800, "coins": 800, "state": "locked" }
    ]
  }
}
```

- `state` 枚举：`locked`（未达时长）/ `unlockable`（达标未领）/ `claimed`（已领）。
- `claim` 成功返回 `{ "code":200, "data": { "coins":300, "coin_balance":700 } }`。

## 5. 时长与计时口径

- 累计粒度：今日跨房间累计，0 点重置（`stat_date`）。
- 心跳节流 30s；仅在直播间页面（`LiveRoomSlide`）发送，离开/首页不发。
- 页面 `visibilitychange` 隐藏时停发心跳（不计后台时长），符合真实观看。
- 进入/重进直播间先 `GET /treasure/status` 拉权威态，避免本地计时与后端漂移导致「显示能领却领取失败」。

## 6. 前端组件设计

挂载于 `hostPill` 下方的浮条 `TreasureProgressBar`：

- 一条横向进度条，按 `watched_seconds / 1800` 百分比填充；3 个宝箱节点固定落在 3/10/30 分钟对应轴位（10%、33.3%、100%）。
- 单宝箱状态机：
  - `locked`：灰暗、半透明。
  - `unlockable`：进度条填充头到达后，宝箱**跳动 + 盖子微开 + 高光脉冲**吸引点击。
  - `claimed`：置灰 + 勾选，不可再点。
- 点击 `unlockable` → 调 `claim` → 开箱动画（盖弹开 + 金币迸射 + `+N` 飘字）→ 顶部金币数字滚动增长。
- 未登录：仅展示进度条，点击引导登录（金币需绑定用户落库）。
- 必须适配日/夜双主题（`:root[data-theme='dark']`）；不得遮挡下方 `fixedPriceList` 与 `liveChatOverlay`。

## 7. 资产可见性

用户中心 [User/Index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/User/Index.tsx) 增加金币入口/展示，复用 `GET /treasure/status` 的 `coin_balance`，让资产「可见」。

## 8. 测试要点

- 后端：心跳累加封顶、门槛未达拒发、重复领取幂等、跨日重置。
- 前端：`LiveLayoutCss.test.ts` 验证浮条不破坏既有布局；双主题渲染；状态机三态切换。
