# Live Room Layout Redesign - Design Spec

## 1. 概览 (Overview)

为了提供更加沉浸和符合用户心智的直播竞拍体验，我们计划对 H5 用户端直播间 (`frontend/h5/src/pages/Live/LiveRoomSlide.tsx`) 的 UI 布局进行重构。主要目标是释放屏幕中下部的视觉空间，优化辅助信息的展示位置，并确保与抖音等主流直播平台的心智模型保持一致。

## 2. 核心调整项 (Key Changes)

### 2.1 移除商品大卡片，提升出价排行
- **当前状态**：底部抽屉（`BidDock` 的 `.sheet`，仅 `sheet !== null` 时展开）内有一个 `<article className={styles.productCard}>` 占据了较大面积。注意它并非屏幕常驻元素，而是抽屉展开后才出现。
- **调整方案**：彻底移除该 `<article>` 标签及其包含的商品大卡片（商品图、名称、简介、抽屉内收藏行）。
- **关联影响**：在展开的抽屉内，商品卡片移除后下方的“出价排行” (`<section className={styles.rankingBlock}>`) 将自然上移，获得更多的展示空间，提升抽屉内布局适配性。

### 2.2 主播信息与收藏按钮融合 (左上角)
- **当前状态**：收藏按钮原本附着在商品卡片上。
- **调整方案**：
  - 增强 `styles.topBar` 的左侧部分 (`hostPill`)。
  - 将原本位于商品卡片内的“收藏直播间”按钮迁移至左上角，直接跟在主播信息（头像、名称、点赞/在线说明）的右侧。
  - 采用当前主题体系内的高亮色设计，提升辨识度；不得直接依赖未定义的 `var(--color-dy-pink)`。
  - 推荐新增局部 CSS 变量 `--live-follow-accent` / `--live-follow-accent-muted`，并分别在日/夜主题下给出可读、可点击的颜色值。

### 2.3 独立的在线人数与商品详情组件 (右上角)
- **当前状态**：在线人数附属于主播名称下方，商品详情无明确轻量化入口。
- **调整方案**：
  - **右上角第一行**：将“在线人数”提取为独立组件，置于右上角。左侧展示最多 3 个用户的重叠头像，右侧为半透明的数字胶囊，最右侧保留退出 (`X`) 按钮。
  - **右上角第二行 (方案 C 融合)**：在在线人数下方，新增一个主题适配的微胶囊样式“商品详情 >”按钮。
  - **头像数据规则**：从“主播/占位头像 MVP”升级为真实在线用户 presence。右上角头像区只展示服务端确认在线且已鉴权的真实用户头像；未登录、未鉴权或头像为空的连接只允许计入人数，不进入头像列表；不足 3 个时不使用本地伪造用户补位。
  - **在线人数规则**：H5 页面初始值可使用 `GET /api/v1/live-streams/:id` 的 `viewer_count` 兜底；WebSocket 建连后，以服务端下发的 `live_presence_update.viewer_count` 作为页面权威值。
  - **交互（已确认：小工作量方案）**：点击“商品详情”按钮直接跳转至现有商品详情页 `/detail?id=<auctionId>`（对应 `App.tsx` 中 `path="/detail"` 的 `ProductDetail` 页面，通过 `useSearchParams().get('id')` 接收 auctionId）。**不开抽屉、不新增侧边卡片、不新增 `sheet=info` 内容**。实现使用 react-router 的 `<Link to={`/detail?id=${auctionId}`}>`，离开直播间后用户可通过浏览器返回键回到直播间。

### 2.4 系统提示保留与强化
- **当前状态**：直播互动层已有 `ChatPanel`，视觉方案中出现“系统提示：欢迎来到直播间！”样式。
- **调整方案**：
  - 保留系统提示作为直播间氛围与引导信息的一部分。
  - 顶部栏、右上角观众组件、商品详情按钮不得遮挡系统提示区域。
  - 若实现中需要调整 `ChatPanel` 的提示样式，应保持轻量、半透明、可读，不引入新的业务接口。

## 3. 真实在线用户 Presence 设计

### 3.1 设计目标

右上角在线组件需要表达“当前直播间内真实在线观众”，不是静态直播间热度，也不是前端模拟数据。该能力应复用现有 `auction-service` WebSocket `LiveStreamRoom`，因为在线状态的产生和消失都来自 WebSocket 连接生命周期。

### 3.2 数据权威与边界

| 数据 | 权威来源 | 用途 | 备注 |
|---|---|---|---|
| `viewer_count` | `auction-service` WebSocket presence | H5 直播间右上角实时人数 | `product-service` 详情接口仅作为页面首屏兜底 |
| `viewers` | `auction-service` presence 聚合 + `users` 表头像信息 | H5 右上角最多 3 个头像 | 仅展示已鉴权真实用户 |
| `host_avatar` | `product-service.live_streams.streamer_avatar` | 主播头像兜底 | 详情接口不得继续返回空字符串占位 |

前端仍必须经 `gateway-service` `/api/v1` 入口访问 HTTP 接口；WebSocket 沿用现有 `/api/v1/ws?auction_id=...&live_stream_id=...` 发现/连接路径。

### 3.3 WebSocket 消息契约

新增服务端到客户端消息类型：

```json
{
  "type": "live_presence_update",
  "timestamp": 1710000000000,
  "data": {
    "live_stream_id": 3,
    "viewer_count": 12,
    "viewers": [
      {
        "user_id": 1001,
        "name": "张三",
        "avatar_url": "https://cdn/u1001.png"
      }
    ]
  }
}
```

字段规则：
- `viewer_count` 按在线用户数去重，不按连接数计数。
- `viewers` 最多返回 3 个，用于右上角头像展示。
- `viewers[*].user_id` 仅用于前端 key，不展示完整用户标识。
- `avatar_url` 为空时前端使用 `name` 首字兜底；服务端不伪造头像 URL。
- `live_presence_update` 是瞬时状态消息，不进入 `LiveStreamRoom` 弹幕历史，不参与新进房历史回放。

### 3.4 服务端状态模型

在 `LiveStreamRoom` 内新增 presence 结构：

```go
type PresenceViewer struct {
    UserID    int64
    Name      string
    AvatarURL string
    Clients   map[string]struct{}
}
```

核心规则：
- `clients` 仍保存连接维度，用于广播和断线清理。
- presence 使用 `user_id -> PresenceViewer` 去重；同一用户多 Tab、多次重连只算 1 个在线用户。
- 注册连接时：
  - `LiveStreamID > 0` 才参与直播间 presence。
  - `Authenticated=true` 的连接进入实名 presence，可出现在 `viewers`。
  - 非鉴权连接不得出现在 `viewers`；如产品要求可单独计入匿名连接数，但默认不展示头像。
- 注销连接时：移除对应 `client_id`；当某用户最后一个连接断开，才从 presence 删除该用户。
- 每次注册/注销后广播一次 `live_presence_update`。
- 新连接注册成功后立即向该连接发送一次 snapshot，解决首屏等待下一次变更的问题。

### 3.5 鉴权与隐私

presence 必须只信任服务端验证过的 JWT。现有 WebSocket 兼容 `user_id` query 参数只能用于历史兼容，不得用于实名头像展示。原因：`user_id` query 可被客户端伪造，一旦用于头像列表，会暴露任意用户“在线”假象。

隐私最小化：
- 只下发 `user_id/name/avatar_url` 三个展示必需字段。
- 不下发手机号、邮箱、token、完整用户对象。
- 日志不得打印 presence payload 中的完整头像列表。

### 3.6 前端状态流

`LiveRoomSlide.tsx` 新增本地状态：

```ts
type PresenceViewer = {
  user_id: number;
  name: string;
  avatar_url?: string;
};

type LivePresence = {
  viewer_count: number;
  viewers: PresenceViewer[];
};
```

渲染规则：
- 首屏：使用 `liveStream.viewer_count` 和主播头像兜底，避免空白。
- WebSocket 收到 `live_presence_update` 且 `live_stream_id` 等于当前房间后，覆盖本地 presence。
- 右上角数字显示 `presence.viewer_count ?? liveStream.viewer_count ?? 0`。
- 头像优先展示 `presence.viewers`，最多 3 个；为空时可展示主播头像兜底，但不得构造虚假用户。
- 组件卸载或切换直播间时解绑 WS handler，避免旧房间 presence 覆盖新页面。

### 3.7 与现有 HTTP 字段的关系

`GET /api/v1/live-streams/:id` 仍返回 `viewer_count`，但定位调整为“初始值/降级值”。当前详情接口中的 `host_name` / `host_avatar` 不应继续写死为空，应与列表接口保持一致，分别返回 `StreamerName` / `StreamerAvatar`。

### 3.8 测试要求

后端：
- `LiveStreamRoom` 注册同一 `user_id` 的多个 client 时，`viewer_count` 只增加 1。
- 注销其中一个 client 不应移除用户；最后一个 client 注销后才移除。
- 新 client 注册后立即收到 snapshot。
- `live_presence_update` 不进入 `GetHistory()`。
- 非 `Authenticated` 连接不得出现在 `viewers`。

前端：
- 收到 `live_presence_update` 后，右上角人数覆盖 HTTP 初始值。
- 收到多个 `viewers` 时最多渲染 3 个头像。
- `live_stream_id` 不匹配的 presence 消息被丢弃。
- 组件卸载后取消订阅，避免后续消息触发 setState。

### 3.9 项目亮点表达口径

本功能在答辩/项目亮点文档中不应表述为“把模拟头像换成真实头像”，而应表述为“直播间实时 Presence 能力”。核心技术难点是把 WebSocket 连接生命周期转换成可信的用户在线事实：

- **人数可信**：在线人数按 `user_id` 去重，不按连接数计数；同一用户多 Tab、多次重连只算 1 人，最后一个 client 断开才离线。
- **头像可信**：头像列表只展示服务端 JWT 鉴权过的真实用户；历史兼容的 query `user_id` 不得用于实名 presence，避免伪造任意用户在线。
- **消息边界清晰**：`live_presence_update` 是瞬时状态消息，只用于当前房间状态覆盖，不进入弹幕 history，不参与新进房历史回放。
- **前端状态可收敛**：H5 首屏使用 HTTP `viewer_count` 兜底，WebSocket 建连后以 `live_presence_update` 作为权威状态；切换直播间时按 `live_stream_id` 过滤旧消息。
- **隐私最小化**：下发字段限制为 `user_id/name/avatar_url`，且头像列表限制最多 3 个展示用户，不下发完整用户对象或敏感字段。

推荐在项目总览 HTML 中把它作为“第五大工程难点”展示，标题为：**真实在线 Presence：人数可信、头像可信、消息不污染历史**。

## 4. 组件结构更新示例 (Component Structure)

重构后的 `<header>` 区域大致结构如下：

```tsx
<header className={styles.topBar}>
  {/* 左侧：主播信息 + 收藏 */}
  <div className={styles.hostPill}>
    {/* 头像、名称、副标题 */}
    <div className={styles.hostInfo}>...</div>
    {/* 收藏按钮 */}
    <button className={styles.followBtn}>
      {following ? '已收藏' : '收藏'}
    </button>
  </div>

  {/* 右侧：观众区 + 详情入口 */}
  <div className={styles.rightActions}>
    {/* 观众区 */}
    <div className={styles.viewersRow}>
      <div className={styles.avatarsGroup}>...</div>
      <div className={styles.viewerCount}>{(liveStream?.viewer_count ?? 0).toLocaleString()}</div>
      <Link className={styles.closeBtn} to="/">X</Link>
    </div>

    {/* 商品详情入口 (方案C)，直接跳转现有详情页 */}
    <Link className={styles.productDetailBtn} to={`/detail?id=${auctionId}`}>
      商品详情 &gt;
    </Link>
  </div>
</header>
```

点击“商品详情”跳转至 `/detail?id=<auctionId>`，由现有 `ProductDetail` 页面渲染完整商品信息（商品图片、名称、简介、价格、加价信息等），无需在直播间内新增详情面板。底部抽屉（`BidDock` / `sheet`）经本次改造后仅承载出价相关内容（价格、倒计时、出价排行、出价框），不再包含商品大卡片。

## 5. 视觉与样式指南 (Visual & Styling Guidelines)

- **背景与层级**：顶部栏背景使用半透明底色或毛玻璃效果 (`backdrop-filter: blur(8px)`)，以保证在视频或封面图背景上文字的可读性。避免依赖复杂渐变作为唯一可读性来源。
- **色彩规范**：
  - 收藏按钮（未收藏状态）：推荐使用局部变量 `--live-follow-accent`，日间/夜间分别映射到当前主题中对比度足够的高亮色。
  - 已收藏/处理中状态：使用 `--live-follow-accent-muted` 或现有 disabled 语义样式，避免与未收藏主操作抢视觉焦点。
  - 商品详情胶囊：不得硬编码只适合单一主题的白底黑字。日间可使用浅色半透明底，夜间应切换为深色半透明底或主题 surface 色，并保证文字对比度。
- **头像层叠**：右侧观众头像使用负 `margin-left` 和递减的 `z-index` 实现层叠排列。
- **日/夜主题适配**：
  - 所有新增样式优先使用现有 CSS 变量（如 `--bg-*`、`--text-*`、`--border-*`、`--radius-*`、`--spacing-*`）。
  - 必须同时覆盖 `:global(:root[data-theme='dark'])` 和默认/日间主题下的显示效果。
  - 半透明底、主题适配胶囊、头像边框、关闭按钮、详情按钮均需在直播画面深色/浅色背景上保持可读。
  - 禁止为绕过主题问题直接写死只在夜间有效的颜色组合。

## 6. 后续步骤 (Next Steps)
1. **获取确认**：等待用户确认此 Design Spec。
2. **实施计划**：更新既有实施计划，纳入 `auction-service` presence、H5 WebSocket 消息接入、`product-service` 详情字段修正与测试。
3. **代码实施**：按 TDD 先补后端 presence 测试，再接 H5 UI 状态与真实浏览器验证。
