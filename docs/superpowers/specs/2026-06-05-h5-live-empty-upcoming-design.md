# H5 直播间空态优化设计

## 背景

H5 底部导航进入 `/live` 时，`LiveFeedPage` 当前只展示正在竞拍的直播间。若没有符合条件的直播间，页面只返回纯文本空态，例如“暂无正在竞拍的直播间”。这个状态没有解释原因，也没有给用户下一步动作，导致直播间页在低峰时段显得像异常页面。

本次优化只处理“没有正在竞拍直播间”的展示与转化路径，不改变直播间滑动 feed、出价、点天灯、一口价、WebSocket 等直播内核心链路。

## 目标

1. 当没有正在竞拍的直播间时，展示可解释、可行动的空态。
2. 若存在即将开始的竞拍，最多展示最近 2 条预告。
3. 用户可以在空态内直接订阅开拍提醒。
4. 日间/夜间模式均保持可读性和按钮层级。
5. 无预告数据或接口失败时降级到“去首页看拍品”，不展示假预告。

## 非目标

1. 不新增“全部预告”入口。
2. 不新增直播间订阅模型。
3. 不做推荐算法或个性化排序。
4. 不改变管理员/商家端开播权限与直播间管理逻辑。
5. 不让 H5 前端直连后端子服务，所有流量仍经 `gateway-service` 的 `/api/v1` 入口。

## 用户体验

### 有即将开播预告

页面文案：

- 主标题：`下一场竞拍正在准备`
- 说明：`当前没有正在竞拍的直播间。先订阅感兴趣的预告场次，开拍前会提醒你回来。`
- 分区标题：`即将开播`

预告列表规则：

- 最多展示 2 条。
- 展示最近开始的 2 条待开始竞拍。
- 每条展示开始时间、拍品/专场名称、起拍价或简短说明、订阅按钮。
- 条目非按钮区域点击跳转商品详情页：`/detail?id={auctionId}`。
- 订阅按钮点击只执行订阅动作，并阻止冒泡，避免同时跳转。

按钮状态：

- 未订阅：`订阅`
- 请求中：`订阅中...`
- 已订阅：`已订阅`，禁用或弱化展示

### 无预告或接口失败

显示轻行动空态：

- 主标题：`当前没有竞拍直播`
- 说明：`可以先看看正在预热的拍品，开拍提醒会第一时间通知你。`
- 主按钮：`去首页看拍品`，跳转 `/`

失败时不弹错误 Toast，避免把正常低峰状态包装成故障。

## 数据设计

### 预告数据

前端需要一个“即将开播竞拍”数据源。查询语义：

- `auction.status = 0`
- `auction.start_time > now`
- 按 `auction.start_time ASC`
- `limit = 2`
- 需要返回 `auction_id`、`product_id`、`product.name`、`start_time`、`start_price/current_price`、`live_stream_id`

优先复用现有竞拍列表接口能力。如果现有 `GET /api/v1/auctions` 已能按 `status=0&page=1&page_size=2` 返回待开始竞拍并按开始时间排序，则前端直接复用 `auctionApi.list({ status: '0', page: 1, page_size: 2 })`。如果现有排序或字段不满足，需要在后端补齐该查询语义，但仍通过 Gateway 暴露在 `/api/v1` 下。

### 订阅数据

订阅开拍提醒复用现有商品提醒能力：

- 订阅：`POST /api/v1/products/:productId/remind`
- 取消订阅：本次页面不提供
- 状态回填：`GET /api/v1/users/me/reminders`

这避免新建直播间级订阅模型，也与首页、商品详情页已有“订阅开拍提醒”行为保持一致。

## 前端设计

### 组件边界

在 `frontend/h5/src/pages/Live/LiveFeedPage.tsx` 中引入空态分支，但建议将展示拆成局部组件，避免 `LiveFeedPage` 继续膨胀：

- `LiveEmptyState`
  - 输入：`upcomingAuctions`、`subscribedProductIds`、`pendingProductId`
  - 输出：点击订阅、点击预告条目、点击去首页
- `UpcomingAuctionCard`
  - 输入：单个预告竞拍、订阅状态
  - 负责阻止订阅按钮冒泡

样式放在 `frontend/h5/src/pages/Live/Live.module.css`，复用现有主题 token，保证日间/夜间模式一致。

### 状态流

1. `LiveFeedPage` 首屏仍先拉取直播间列表。
2. 若 `auctionRooms.length > 0`，保持现有直播 feed。
3. 若 `rooms.length === 0` 或 `auctionRooms.length === 0`，进入空态数据加载。
4. 拉取最近 2 条待开始竞拍。
5. 若用户已登录，同时拉取 `productReminderApi.list()` 回填订阅状态。
6. 有预告则展示预告空态；无预告或请求失败则展示“去首页看拍品”降级空态。

订阅行为：

1. 未登录用户点击订阅，跳转登录页，`redirect` 回 `/live`。
2. 已登录用户点击订阅，按钮变为 `订阅中...`。
3. 成功后将 `product_id` 加入本地 `subscribedProductIds`。
4. 若后端返回“已经订阅”，前端视为成功并回填 `已订阅`。
5. 其他错误保持按钮可重试，可用 Toast 提示“订阅失败，请稍后重试”。

## 后端设计

若现有 `GET /api/v1/auctions` 无法满足“待开始、未来时间、开始时间升序、limit 2”的语义，需要补齐查询能力：

- 在 `auction-service` 的列表查询中支持 `status=0`。
- 对待开始场次默认过滤 `start_time > now`，或新增明确参数如 `upcoming=true`。
- 返回字段必须包含 `product_id` 与商品摘要，供 H5 订阅和跳转使用。
- Gateway 只转发 `/api/v1/auctions`，不允许 H5 绕过 Gateway 直连服务。

排序要求必须由后端保证，前端只做防御性截断到 2 条，避免分页或排序语义漂移。

## 埋点

建议复用现有业务事件入口 `POST /api/v1/events`：

- `live_empty_upcoming_exposed`
  - 触发：展示预告空态
  - metadata：`auction_ids`、`count`
- `reminder_subscribe`
  - 触发：点击订阅按钮
  - source：`live_empty_upcoming`
  - metadata：`auction_id`、`product_id`
- `product_detail_click`
  - 触发：点击预告条目非按钮区域
  - source：`live_empty_upcoming`

如果当前后端事件白名单不包含新事件名，需要同步扩展 Gateway 事件校验。若为了降低实现成本，也可以首版只复用已有 `reminder_subscribe`，不新增曝光与详情点击事件。

## 测试策略

### 前端单元测试

覆盖 `LiveFeedPage`：

1. 无正在竞拍直播间但有 2 条待开始竞拍时，展示 `即将开播` 与两条预告。
2. 返回超过 2 条预告时，只展示最近 2 条。
3. 点击预告条目非按钮区域跳转 `/detail?id={auctionId}`。
4. 点击订阅按钮调用 `productReminderApi.subscribe(productId)`，不触发行点击跳转。
5. 订阅成功后按钮变为 `已订阅`。
6. 获取预告失败时展示 `去首页看拍品`。
7. 日/夜模式样式使用主题 token，不写死破坏夜间可读性的颜色。

### 后端测试

若修改竞拍列表查询：

1. `status=0` 只返回待开始竞拍。
2. `upcoming` 查询只返回 `start_time > now`。
3. 返回结果按 `start_time ASC`。
4. `page_size=2` 只返回 2 条。
5. 返回项包含 `product_id` 与商品摘要。

## 验收标准

1. `/live` 在没有正在竞拍直播间时不再只显示纯文本。
2. 有预告时最多展示最近 2 条，每条有独立订阅按钮。
3. 点击预告卡片非按钮区域进入商品详情页。
4. 点击订阅按钮不会跳转详情页。
5. 无预告或接口失败时展示 `去首页看拍品`。
6. 日间/夜间模式下文字、按钮、卡片边界均清晰可读。
7. 前端测试覆盖核心分支；若后端改查询，后端测试覆盖排序与过滤语义。

## 风险与约束

1. 如果现有竞拍列表接口不保证排序，必须后端修正，不能让前端猜测最近两条。
2. 如果未登录用户订阅跳登录，登录后需要回到 `/live`，否则转化路径会断。
3. 订阅基于 `product_id`，预告数据缺失 `product_id` 时该条不能展示订阅按钮。
4. 空态不应展示过多运营内容；超过 2 条会重新制造选择负担。
