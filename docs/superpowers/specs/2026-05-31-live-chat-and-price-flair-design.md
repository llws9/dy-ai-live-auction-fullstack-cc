# 直播间弹幕与高价飘屏设计 (B1)

- 版本: v1.0
- 日期: 2026-05-31
- 范围: C2C 直播拍卖平台的"直播间弹幕 + 高价飘屏"功能 MVP
- 关联代码:
  - [websocket/hub.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/websocket/hub.go)
  - [websocket/room.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/websocket/room.go)
  - [websocket/client.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/websocket/client.go)
  - [websocket/message.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/websocket/message.go)

---

## 1. 背景与目标

### 1.1 现状

当前直播竞拍平台已具备：
- WebSocket 单向广播能力（`AuctionRoom`，按 `auction_id` 隔离）
- 客户端仅能发送 `ping` / `sync_request`，**没有反向消息通道**
- 已有 JWT 鉴权、Redis、Hub 架构

### 1.2 目标

在 C2C 直播拍卖场景中，引入**直播间弹幕**与**高价成交飘屏**两个原子能力，使直播间从"单向看货"升级为"互动看货"，提升留存时长与情感卷入。

### 1.3 第一性原则约束

- **不引入持久化负担**：弹幕本质是即时消息，丢失成本远低于落库成本（MVP 不入 MySQL）
- **不破坏现有 Auction Room**：弹幕属于直播间生命周期，不应受拍品切换中断
- **WS 连接数最小化**：移动端对连接数敏感，单连接多订阅
- **能在不接入第三方服务的前提下交付 MVP**：内置黑词 + 频控

### 1.4 非目标 (Out of Scope)

| 项 | 原因 |
|---|---|
| 表情商城 / 自定义图片表情 | MVP 仅 unicode emoji，避免引入资源管理与审核成本 |
| 弹幕落库与历史回查 | 内存环形缓冲足以满足 MVP；落库需配套 Outbox + 审核留痕，体量不到位前是过度工程 |
| 礼物 / 打赏 / 连麦 | 属于 B2/B5，将复用本设计的 LiveStreamRoom 能力但不在本 spec |
| 主播运营控制台 | MVP 用配置文件管理黑词，控制台延后 |
| 跨直播间广场弹幕 | 飘屏只在所属直播间内可见 |
| 国际化与翻译 | 出海需求出现后再做 |

---

## 2. 关键决策摘要

| 决策点 | 选择 | 理由 |
|---|---|---|
| Room 抽象 | 新增 `LiveStreamRoom`（按 `live_stream_id`），与 `AuctionRoom` 平行 | 弹幕跨拍品连续，违背 auction 边界；正交可扩展 |
| WS 连接策略 | 单连接多订阅，握手 URL 携带 `live_stream_id` 与可选 `auction_id` | 移动端连接数最少 |
| 飘屏触发规则 | 起拍价×N **或** 单笔加价×M（任一命中） | 兼容 0 元起拍；同时鼓励豪爽出价 |
| 鉴权 | WS 升级时校验 JWT；游客可连接但**发送被拒** | 复用现有 JWT 中间件，发送门槛低但可控 |
| 黑词治理 | 内置词库（YAML 配置文件），冷启动 50–100 词 | 0 外部依赖；后续可热更新 |
| 频控 | Redis 计数器；单用户 1s ≤ 1 条；单房间 1s ≤ 20 条全局 | 已有 Redis；轻量足够 |
| 历史保留 | 内存环形缓冲，每 LiveStreamRoom 100 条；新进房回放 | 不入 DB |
| 单条消息长度 | ≤ 50 字 | Whatnot/抖音直播的工业经验值 |
| 表情 | 仅 unicode emoji，按字符长度计算 | MVP 无审核成本 |

---

## 3. 整体架构

### 3.1 模块划分

```
backend/auction/websocket/
├── hub.go              （已有，扩展：增加 liveStreamRooms map）
├── room.go             （已有 AuctionRoom，不动）
├── livestream_room.go  （新增 LiveStreamRoom，含环形缓冲）
├── client.go           （已有，扩展：增加 LiveStreamID 字段、handleChat 分支）
├── message.go          （已有，扩展：新增 chat / price_flair 消息类型）
├── chat_filter.go      （新增：黑词过滤、长度校验、emoji 处理）
├── chat_throttle.go    （新增：基于 Redis 的频控）
└── chat_config.yaml    （新增：黑词与频控参数配置）

backend/auction/handler/
├── ws.go               （已有，扩展：升级时解析 live_stream_id）
└── chat.go             （新增：HTTP 兜底接口，例如管理端拉取最近黑词命中）
```

### 3.2 双 Room 模型

```
┌─ Hub ────────────────────────────────────────┐
│                                              │
│  auctionRooms      map[int64]*AuctionRoom    │  ← 已有（出价、延时、结束）
│  liveStreamRooms   map[int64]*LiveStreamRoom │  ← 新增（弹幕、飘屏、系统消息）
│  userRooms         map[int64]map[*Client]    │  ← 已有（个人通知）
│                                              │
└──────────────────────────────────────────────┘
        ▲                          ▲
        │                          │
   AuctionID                  LiveStreamID
        │                          │
        └──────── Client ──────────┘
                    │
              单 WS 连接
              同时订阅二者
```

### 3.3 数据流：弹幕

```
用户输入 → 前端长度/Emoji 预校验
       → WS Send(text=chat)
       → Client.handleMessage(MessageTypeChat)
       → ChatFilter.Validate(userID, text)
              │
              ├─ 长度超限 → Reply Error(40001)
              ├─ 黑词命中 → Reply Error(40002)（前端弹"消息含违规内容"）
              ├─ 频控命中 → Reply Error(40003)（前端 Toast"发送过快"）
              └─ 通过 → LiveStreamRoom.Broadcast(MessageTypeChatMessage)
                            ├─ 写入环形缓冲
                            └─ 广播给该 LiveStreamRoom 所有 Client
```

### 3.4 数据流：高价飘屏

```
出价成功（service/bid.go）→ 现有 BidPlaced 广播（不改）
                          │
                          └─ 新增：FlairChecker.CheckBid(auction, bid)
                                 │
                                 ├─ delta = bid.amount - prev_price
                                 ├─ 命中 起拍价×N 或 加价幅度×M
                                 └─ LiveStreamRoom.Broadcast(MessageTypePriceFlair)
                                        （直播间内全员收到）

竞拍成功结束（state machine → ended）→ 现有 AuctionEnded 广播
                                    │
                                    └─ FlairChecker.CheckEnded(auction, finalPrice)
                                           │
                                           └─ 同上规则命中 → 飘屏
```

> **关键**：飘屏消息走 **LiveStreamRoom**，不走 AuctionRoom。
> 理由：飘屏的观众范围 = 直播间观众 ⊇ 当前拍品观众。一个用户即便没专门盯当前拍品，也应看到"刚刚 9999 拍走了"。

---

## 4. 数据模型与协议

### 4.1 不新增数据库表

MVP 完全不入库。如未来需要审核留痕，再走 Outbox 模式补建表。

### 4.2 内存数据结构

```go
// LiveStreamRoom：直播间级 Room
type LiveStreamRoom struct {
    LiveStreamID int64
    clients      map[string]*Client
    clientsLock  sync.RWMutex

    // 环形缓冲：保存最近 N 条消息（用于新进房回放）
    history     [chatHistorySize]*Message
    historyHead int
    historyLen  int
    historyLock sync.RWMutex

    Register   chan *Client
    Unregister chan *Client
    Broadcast  chan *Message
    done       chan struct{}
}

const chatHistorySize = 100
```

### 4.3 WS 消息协议扩展

#### 4.3.1 新增客户端 → 服务端

```go
const MessageTypeChatSend MessageType = "chat_send"

// ChatSendData
type ChatSendData struct {
    LiveStreamID int64  `json:"live_stream_id"` // 必填
    Text         string `json:"text"`           // ≤ 50 字
    ClientMsgID  string `json:"client_msg_id"`  // 客户端生成，幂等用
}
```

#### 4.3.2 新增服务端 → 客户端

```go
const MessageTypeChatMessage MessageType = "chat_message"

// ChatMessageData：广播给所有直播间观众
type ChatMessageData struct {
    LiveStreamID int64  `json:"live_stream_id"`
    UserID       int64  `json:"user_id"`
    UserName     string `json:"user_name"`
    AvatarURL    string `json:"avatar_url,omitempty"`
    Text         string `json:"text"`
    SentAt       int64  `json:"sent_at"`        // ms
    ClientMsgID  string `json:"client_msg_id"`  // 回显，便于发送方去重
}

const MessageTypePriceFlair MessageType = "price_flair"

// PriceFlairData：高价飘屏
type PriceFlairData struct {
    LiveStreamID int64   `json:"live_stream_id"`
    AuctionID    int64   `json:"auction_id"`
    UserID       int64   `json:"user_id"`
    UserName     string  `json:"user_name"`
    Amount       float64 `json:"amount"`              // 触发飘屏的金额
    Reason       string  `json:"reason"`              // "high_bid"（R1）| "auction_won"（R2）
    StartPrice   float64 `json:"start_price,omitempty"`
    BidDelta     float64 `json:"bid_delta,omitempty"` // 本次出价超出当前价的幅度
}
```

#### 4.3.3 错误码扩展

| Code | 含义 | 客户端处理 |
|------|------|------------|
| 40001 | 弹幕长度超限 | 输入框红框提示 |
| 40002 | 命中违禁词 | 通用 Toast "消息含违规内容" |
| 40003 | 发送过快（频控） | 显示冷却时间倒计时（同一秒内的重复点击灰化） |
| 40101 | 未登录 | 跳转登录 |
| 40301 | 用户被禁言 | 灰化输入框（MVP 不实现，预留） |

### 4.4 WS 握手 URL 扩展

```
现有: ws://gateway/api/v1/ws?auction_id=123
新增: ws://gateway/api/v1/ws?auction_id=123&live_stream_id=456
      （live_stream_id 可选；不带则不订阅弹幕流）
```

兼容性：旧客户端不带 `live_stream_id` 时，行为完全保持现状。

### 4.5 黑词配置

```yaml
# backend/auction/websocket/chat_config.yaml
chat:
  max_text_length: 50
  user_rate_limit:
    interval_ms: 1000     # 每 1 秒
    max_messages: 1       # 1 条
  room_rate_limit:
    interval_ms: 1000     # 每 1 秒
    max_messages: 20      # 全房间 20 条
  blocked_words:
    # 引流类
    - "微信"
    - "weixin"
    - "vx"
    - "qq"
    - "电话"
    # 涉政、辱骂等敏感词条由运营补齐
  flair_rules:
    high_bid_multiplier: 5      # 单笔出价 ≥ 加价幅度 × 5 触发
    auction_won_multiplier: 5   # 成交价 ≥ 起拍价 × 5 触发
    min_absolute_amount: 100    # 兜底：金额过低不飘（避免 0 元起拍场景刷屏）
```

> 配置文件通过 Nacos 加载，热更新由现有 `pkg/nacos` 负责。

---

## 5. 详细行为定义

### 5.1 进房流程（含历史回放）

1. 客户端打开直播间页面，构造 WS URL（带 `live_stream_id` + 当前 `auction_id`）
2. 服务端 WS 升级中间件：
   - 校验 JWT（**可选**：游客也允许通过，仅 `UserID=0`）
   - 解析 `live_stream_id`，校验直播间存在且非"已下播"
3. `Hub.RegisterClient` 同时把客户端注册到对应的 AuctionRoom 与 LiveStreamRoom
4. LiveStreamRoom 注册成功后：
   - 立即把环形缓冲中的最近 100 条 `chat_message` 一次性回放给该客户端（避免空荡荡的进房体验）
5. 客户端进入正常接收状态

### 5.2 发送弹幕流程

1. 用户在前端输入 → 前端预校验（长度、emoji 数量）
2. 前端通过现有 WS 连接发送 `chat_send`
3. 服务端 `Client.handleMessage` 增加分支：
   ```go
   case MessageTypeChatSend:
       c.handleChatSend(msg)
   ```
4. `handleChatSend` 流程：
   - 若 `UserID == 0` → Reply Error(40101) 立即 return
   - 长度 / Unicode 安全校验 → 失败 Reply Error(40001)
   - 黑词命中 → Reply Error(40002)
   - 频控（用户 + 房间双层）→ 失败 Reply Error(40003)
   - 全部通过 → 构造 `ChatMessageData` 投递到 `LiveStreamRoom.Broadcast`
5. LiveStreamRoom 接到广播：
   - 先 `pushHistory(msg)` 写入环形缓冲
   - 再向所有客户端 `Send <- msg`

### 5.3 高价飘屏判定规则

设：
- `startPrice` = 拍品起拍价
- `bidIncrement` = 拍品当前加价幅度
- `bidAmount` = 本次出价金额
- `prevPrice` = 上一次最高价

**规则 R1（豪爽出价）**：
```
delta = bidAmount - prevPrice
触发: delta >= bidIncrement × high_bid_multiplier  AND  bidAmount >= min_absolute_amount
理由: 鼓励远超最小加价幅度的"豪气出价"
```

**规则 R2（高价成交）**：
```
触发: finalPrice >= startPrice × auction_won_multiplier  AND  finalPrice >= min_absolute_amount
理由: 起拍价被翻倍 5x 以上的成交本身就是"故事"
```

**规则 R3（0 元起拍兜底）**：
```
当 startPrice == 0 时，R2 永远命中（任意正成交都满足 0 × 5 = 0）。
为避免刷屏，加 min_absolute_amount 兜底：finalPrice 必须 ≥ 100 才飘。
```

任一规则命中即广播 1 条 `price_flair` 到 LiveStreamRoom。**同一拍品同一种 reason 在 30 秒内仅飘一次**（去重），避免连续超出阈值出价导致刷屏。

### 5.4 频控具体实现

使用 Redis 原子计数：

```
Key:  chat:rate:user:{user_id}
TTL:  1s
Op:   INCR；若 > max_messages → 拒绝；否则放行

Key:  chat:rate:room:{live_stream_id}
TTL:  1s
Op:   INCR；若 > max_messages → 拒绝；否则放行
```

两道防线先用户后房间，任何一个被卡都拒绝。失败原因不区分（统一 40003），避免对手摸出限流策略。

### 5.5 直播间下播 / Room 销毁

- 当 LiveStreamRoom 内最后一个 Client 离开 → 触发计时器（30s）
- 30s 内有新 Client 加入 → 取消销毁
- 30s 后仍空 → 销毁 Room（释放环形缓冲）
- 如果直播间显式标记下播（管理操作）→ 立即销毁，发送 `auction_ended` 风格的房间关闭事件

---

## 6. 鉴权与安全

| 关注点 | 处理 |
|---|---|
| 谁能看 | 任何人（包括游客）：握手时不强制 JWT |
| 谁能发 | 必须登录用户：`UserID == 0` 时直接拒绝 |
| 防恶意刷屏 | 用户级 + 房间级双层频控 |
| 防注水广告 | 黑词过滤（联系方式、引流词） |
| 防 XSS | 服务端不存储不解析 HTML；前端用 `textContent` 渲染，禁止 `innerHTML` |
| 防大消息攻击 | 长度硬上限 50 字 + WS 单帧 ReadLimit 已是 512 字节 |
| 防伪造 user_name | 服务端从 JWT/DB 取，不信任客户端字段 |

---

## 7. 前端集成（H5）

### 7.1 新增组件

```
frontend/h5/src/
├── components/
│   ├── LiveChat/
│   │   ├── ChatPanel.tsx       （滚动列表 + 输入框，固定底部）
│   │   ├── ChatBubble.tsx      （单条气泡）
│   │   └── PriceFlairLayer.tsx （全屏覆盖层，CSS 动画从右滑入）
│   └── LiveRoom/
│       └── index.tsx           （挂载 ChatPanel + PriceFlairLayer）
├── store/
│   └── liveChatStore.ts        （Zustand：history、connected、send 方法）
└── api/
    └── liveChatWs.ts           （WS 连接管理，扩展现有 ws 客户端）
```

### 7.2 飘屏 UX 规范

- 触发后从屏幕右侧滑入，停留 4s，淡出
- 同时最多 1 条飘屏；后到的入队，逐条播放
- 队列上限 5 条，超出丢弃最旧
- 主色：金黄渐变背景 + 用户头像 + "@用户名 以 ¥9,999 拍下！"
- 移动端 44px 触摸高度规范

### 7.3 弹幕 UX 规范

- 默认半透明黑底气泡，从底向上滚动
- 距离底部 80px（避开输入框）
- 每条停留 5s，超过列表上限（屏幕约 6 条）顶部最旧消失
- 自己发的弹幕用蓝色边框区分
- 输入框 emoji 按钮使用系统 keyboard（不内置 picker）

### 7.4 错误处理

- 40001/40002 → 输入框抖动 + Toast 提示
- 40003 → 发送按钮灰化 1 秒，倒计时显示
- WS 断开 → 复用现有重连逻辑，重连成功后**自动重新订阅 LiveStreamRoom**（带 live_stream_id 重新握手即可）

---

## 8. 测试策略（TDD 大纲）

### 8.1 后端单元测试

| 文件 | 用例 |
|---|---|
| `chat_filter_test.go` | 长度边界（49/50/51）、unicode emoji 计数、黑词命中、组合命中 |
| `chat_throttle_test.go` | 用户级单条通过、第二条被拒、TTL 过期重置；房间级聚合限流 |
| `livestream_room_test.go` | 注册/注销、广播分发、环形缓冲覆盖（>100 条）、回放正确性、Room GC |
| `flair_checker_test.go` | R1/R2/R3 各自触发条件、min_absolute_amount 兜底、30s 去重 |
| `client_chat_send_test.go` | 游客拒绝、登录用户通过、错误码正确性、ClientMsgID 回显 |

### 8.2 集成测试

| 场景 | 验证 |
|---|---|
| 进房历史回放 | 房间已有 50 条，新连接进入应在握手后立即收到 50 条 |
| 双 Room 隔离 | LiveStreamA 弹幕不应进入 LiveStreamB |
| 拍品切换 | 同一直播间切换 auction_id 时弹幕流不中断 |
| 飘屏与出价时序 | bid_placed → price_flair 在同一直播间，时序正确，且 auction_room 内的 bid_placed 不受影响 |

### 8.3 前端测试

| 用例 | 工具 |
|---|---|
| ChatBubble 渲染（含 emoji、含特殊字符不破布局） | Jest + React Testing Library |
| 频控错误后 1 秒灰化按钮 | Jest 模拟 timers |
| 飘屏队列：连续 5 条只播 5 条，第 6 条丢弃最旧 | Jest |

### 8.4 压测预期

- 单 LiveStreamRoom 1000 人在线，房间级 20 条/秒上限 → 单连接每秒接收 20 条 = 20 × 200B ≈ 4KB/s，完全在 WS 容量内
- 全平台 100 个直播间同时高峰 → 100 × 20 = 2000 条/秒，单 Hub goroutine + Redis INCR 足以承担

---

## 9. 监控与可观测性

新增 Prometheus 指标（复用 `pkg/metrics`）：

| Metric | 类型 | 标签 | 说明 |
|---|---|---|---|
| `chat_messages_total` | Counter | `live_stream_id`, `result` | result = sent/blocked_word/throttled/length |
| `chat_rooms_active` | Gauge | — | 活跃 LiveStreamRoom 数量 |
| `chat_room_clients` | Gauge | `live_stream_id` | 单房间在线人数 |
| `price_flairs_total` | Counter | `reason` | reason = high_bid/auction_won |

日志：所有黑词命中与频控触发记录到结构化日志（用户 ID、文本前缀），便于事后审计。

---

## 10. 渐进交付里程碑

| 里程碑 | 内容 | 价值 |
|---|---|---|
| **M1：基础双 Room** | LiveStreamRoom 抽象 + WS 握手扩展 + 单元测试 | 不可见但解锁后续所有能力 |
| **M2：弹幕 MVP** | chat_send / chat_message + 黑词 + 频控 + 历史回放 + H5 输入框与气泡 | 直播间互动从 0 到 1 |
| **M3：高价飘屏** | FlairChecker + 飘屏消息 + H5 全屏动画 | 戏剧性瞬间，留存 + 攀比心理 |
| **M4：监控与配置中心化** | Prom 指标 + Nacos 配置热更新 | 可运营、可观测 |

每个里程碑独立可上线、可灰度。建议按 M1→M2→M3→M4 顺序，M2 完成时即可对内部用户开放体验。

---

## 11. 风险与应对

| 风险 | 概率 | 影响 | 应对 |
|---|---|---|---|
| 黑词词库初版漏过广告 | 高 | 中 | 上线后 1 周内人工审日志补齐；后续接入外部内容安全 API |
| 单连接订阅两个 Room 导致心跳/超时逻辑复杂 | 中 | 中 | 复用现有 ping/pong 机制；房间订阅信息存于 Client 结构体 |
| 飘屏频繁触发刷屏 | 中 | 低 | 30s 去重 + 队列上限 + 客户端动画排队 |
| 直播间下播时 Room 内存泄漏 | 低 | 中 | 30s 空闲销毁 + 显式下播立即销毁，单元测试覆盖 |
| WS 反向消息引入新 attack surface | 中 | 中 | ReadLimit + 长度校验 + 频控三层保护；JWT 强制校验发送者 |
| 现有 AuctionRoom 与新 LiveStreamRoom 在 Hub 中混用导致死锁 | 低 | 高 | 两个 map 各自独立的 Lock；不持有交叉锁 |

---

## 12. 后续可扩展方向（不在本 spec）

- B2 礼物/打赏：复用 LiveStreamRoom，新增 `gift_sent` 消息
- B5 主播置顶问题：复用 LiveStreamRoom，新增 `pinned_question` 消息
- B6 土豪榜：基于 `chat_messages_total` + `bids` 离线聚合
- 主播控制台：禁言、删除单条、运营公告飘屏
- 弹幕落库：仅"飘屏触发"事件落库，普通弹幕不落
- 外部内容安全 API（如违规率超过阈值再切换）

---

## 13. 决策日志（供未来追溯）

- **2026-05-31**：选定 LiveStreamRoom 与 AuctionRoom 平行，而非合并：原因是用户选择"按直播间连续不断"，弹幕生命周期 ⊋ 拍品生命周期。
- **2026-05-31**：飘屏走 LiveStreamRoom 而非 AuctionRoom：飘屏的合理观众是直播间观众，不是单场拍品观众。
- **2026-05-31**：MVP 不入库：弹幕实时性 >> 留痕需求；落库需配套 Outbox 与审核能力，体量未到不做。
- **2026-05-31**：游客可看不可发：降低连接门槛拉留存，发送门槛维持身份可追溯。
