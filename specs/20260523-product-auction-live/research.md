# Research: 技术研究与决策

**Feature**: `20260523-product-auction-live`
**Date**: 2026-05-23
**Purpose**: 解决技术上下文中的未知项，确定最佳实践

## 研究任务列表

### 1. 消息队列技术选型

**问题**: 大V直播间百万用户通知推送，选择合适的消息队列技术

**决策**: RabbitMQ

**理由**:
- 功能丰富：支持延迟队列（竞拍开始前30分钟通知）、死信队列（失败重试）、优先级队列
- 可靠性高：消息持久化、确认机制、事务消息，保证消息不丢失
- 运维友好：提供Web管理界面，方便监控队列状态、消息堆积情况
- Go客户端成熟：`github.com/rabbitmq/amqp091-go` 稳定可靠
- 吞吐量适中：单机可达数万QPS，满足百万级用户通知场景

**备选方案**:
- Redis Stream: 轻量级但功能有限，缺乏延迟队列、死信队列等高级特性
- Kafka: 高吞吐但过于重量级，运维复杂度高，不适合当前规模
- RocketMQ: 阿里开源，适合电商场景，但社区相对较小

**实现方案**:
```go
// 1. 延迟队列配置（竞拍开始前30分钟通知）
args := amqp.Table{
    "x-delayed-type":     "direct",
    "x-dead-letter-exchange":    "notification.dlx",
    "x-dead-letter-routing-key": "failed",
}
ch.QueueDeclare(
    "auction_starting_delayed",  // 延迟队列
    true, false, false, false,
    args,
)

// 2. 消息结构
type NotificationMessage struct {
    LiveStreamID int64  `json:"live_stream_id"`
    Type         string `json:"type"`          // new_product, auction_starting, auction_ended
    ProductID    int64  `json:"product_id"`
    AuctionID    int64  `json:"auction_id"`
    BatchIndex   int    `json:"batch_index"`   // 批次索引
    TotalBatches int    `json:"total_batches"` // 总批次数
    CreatedAt    string `json:"created_at"`
}

// 3. 生产者发送消息
func (p *NotificationProducer) SendDelayed(msg *NotificationMessage, delayMs int64) error {
    body, _ := json.Marshal(msg)
    return p.channel.Publish(
        "notification.delayed",  // 延迟交换机
        "auction_starting",      // routing key
        false, false,
        amqp.Publishing{
            ContentType:  "application/json",
            Body:         body,
            DeliveryMode: amqp.Persistent,  // 持久化
            Headers: amqp.Table{
                "x-delay": delayMs,  // 延迟时间（毫秒）
            },
        },
    )
}

// 4. 消费者处理消息
func (c *NotificationConsumer) Start() {
    msgs, _ := c.channel.Consume(
        "notification_processing",  // 队列名
        "", false, false, false, false, nil,
    )
    
    for msg := range msgs {
        var notification NotificationMessage
        json.Unmarshal(msg.Body, &notification)
        
        // 处理通知推送
        if err := c.processNotification(&notification); err != nil {
            // 失败后进入死信队列
            msg.Nack(false, false)
        } else {
            // 成功确认
            msg.Ack(false)
        }
    }
}
```

**队列设计**:
```
Exchange: notification.delayed (x-delayed-message)
  ├── Queue: new_product_delayed      → 新商品发布通知
  ├── Queue: auction_starting_delayed → 竞拍开始通知（延迟30分钟）
  └── Queue: auction_ended_delayed    → 竞拍结束通知

Exchange: notification.dlx (死信交换机)
  └── Queue: notification_failed      → 失败消息重试队列

Exchange: notification.direct (直接交换机)
  └── Queue: notification_immediate   → 立即发送的通知（商品下架）
```

---

### 2. 批量通知推送策略

**问题**: 如何高效推送百万级用户通知，避免系统过载

**决策**: 分批次异步推送 + 限流策略

**理由**:
- 分批次处理避免数据库和推送服务过载
- 异步处理不阻塞主业务流程
- 限流保护系统稳定性

**实现细节**:
- 批次大小: 1万用户/批
- 批次间隔: 3-5秒
- 最大耗时: 10分钟（100万用户 = 10批）
- 优先级队列: 竞拍开始通知优先级最高

**代码示例**:
```go
func (s *NotificationService) BatchPush(liveStreamID int64, notification *Notification) error {
    // 获取关注用户总数
    totalUsers := s.followDAO.CountByLiveStream(liveStreamID)

    // 计算批次数
    batchSize := 10000
    batches := (totalUsers + batchSize - 1) / batchSize

    // 分批推送
    for i := 0; i < batches; i++ {
        offset := i * batchSize
        users := s.followDAO.GetFollowers(liveStreamID, offset, batchSize)

        // 推送到消息队列
        for _, user := range users {
            s.pushToQueue(user.ID, notification)
        }

        // 批次间隔
        if i < batches - 1 {
            time.Sleep(3 * time.Second)
        }
    }

    return nil
}
```

---

### 3. 直播间统计数据实时性方案

**问题**: 直播间关注数、竞拍数等统计数据如何保证实时性和性能

**决策**: Redis缓存 + 定时更新

**理由**:
- 避免实时COUNT查询影响性能
- Redis提供高性能缓存
- 定时更新保证数据最终一致性

**实现方案**:
```go
// Redis Key设计
live_stream:{id}:followers_count  // 关注数
live_stream:{id}:active_auctions  // 活跃竞拍数
live_stream:{id}:total_revenue    // 总成交额

// 更新策略
1. 关注/取消关注: INCR/DECR操作
2. 竞拍状态变化: 实时更新活跃竞拍数
3. 竞拍结束: 实时累加成交额
4. 定时任务: 每小时从数据库重新计算，更新缓存（兜底）
```

**一致性保证**:
- 写操作优先更新缓存
- 定时任务从DB同步，修复不一致
- 读取时优先缓存，缓存不存在时查DB并回写

---

### 4. 用户隐私数据保护策略

**问题**: 商家查看关注统计数据时，如何保护用户隐私

**决策**: 仅统计聚合数据，不存储和展示用户明细

**理由**:
- 符合用户隐私保护要求
- 避免数据滥用风险
- 满足商家运营需求

**实现方案**:
```go
type FollowStats struct {
    TotalCount         int64 // 关注总数
    NewToday           int64 // 今日新增
    NewThisWeek        int64 // 本周新增
    NewThisMonth       int64 // 本月新增
    ActiveLast7Days    int64 // 近7天活跃用户数
    ActiveLast30Days   int64 // 近30天活跃用户数
    ParticipatedCount  int64 // 参与过竞拍的关注用户数
}

// API: GET /api/v1/live-streams/{id}/followers/stats
// 权限: 仅商家和管理员可访问
// 返回: FollowStats结构，不包含任何用户明细
```

**数据库查询**:
- 关注总数: COUNT(*) FROM user_live_stream_follows WHERE live_stream_id=?
- 活跃度统计: COUNT(DISTINCT user_id) FROM bids WHERE user_id IN (关注用户) AND created_at > ?
- 无需JOIN用户表，不暴露用户信息

---

### 5. 前端直播间入口设计

**问题**: MVP阶段如何设计用户端直播间入口

**决策**: 独立列表页 + 首页推荐

**理由**:
- 独立列表页方便用户主动浏览
- 首页推荐提高直播间曝光率
- 实现简单，满足MVP需求

**MVP实现**:
1. **直播间列表页** (`/live-streams`):
   - 展示所有直播间（分页）
   - 显示关注状态、关注数量、当前竞拍数
   - 支持搜索（按名称）
   - 点击进入直播间详情

2. **首页推荐**:
   - 显示热门直播间（按关注数排序）
   - 显示最近活跃直播间（按竞拍活动排序）
   - 快速关注按钮

**二期优化**:
- 个性化推荐（根据用户历史行为）
- 竞拍商品页关联直播间信息
- 分类标签（珠宝、鞋服、艺术品等）

---

### 6. 通知去重和优先级策略

**问题**: 用户短时间内收到多条通知如何处理

**决策**: MVP阶段全量推送，二期支持用户配置

**MVP实现**:
- 所有通知都发送，不做去重
- 通知列表按时间倒序排列
- 实现简单，用户可自行筛选

**二期实现**:
```typescript
// 用户通知偏好设置
interface NotificationPreference {
  mode: 'all' | 'aggregated' | 'important_only';
  // all: 全量接收
  // aggregated: 智能聚合（10分钟内同一直播间多条通知聚合为一条）
  // important_only: 仅重要通知（竞拍开始、竞拍结束、出价被超）
}

// 聚合逻辑
function aggregateNotifications(notifications: Notification[]): Notification[] {
  // 按直播间分组
  // 时间窗口10分钟内的聚合
  // 保留最新通知，标记为"有N条新动态"
}
```

---

### 7. 商品发布时的竞拍记录创建策略

**问题**: 商品发布时如何创建竞拍记录，竞拍规则如何设置

**决策**: 配置规则后自动创建，使用默认规则

**理由**:
- 符合业务流程：配置规则 → 发布 → 创建竞拍
- 避免重复操作
- 提供默认值降低使用门槛

**实现流程**:
```go
func (s *ProductService) Publish(productID int64, creatorID int64) error {
    // 1. 检查商品状态
    product := s.productDAO.GetByID(productID)
    if product.Status != ProductStatusDraft {
        return errors.New("商品状态不正确")
    }

    // 2. 获取或创建直播间
    liveStream := s.getOrCreateLiveStream(creatorID)

    // 3. 获取竞拍规则
    rule := s.ruleDAO.GetByProductID(productID)
    if rule == nil {
        // 使用默认规则
        rule = s.createDefaultRule(productID)
    }

    // 4. 创建竞拍记录
    auction := &model.Auction{
        ProductID:     productID,
        LiveStreamID:  liveStream.ID,
        CreatorID:     creatorID,
        Status:        model.AuctionStatusPending,
        CurrentPrice:  rule.StartPrice,
        StartTime:     time.Now().Add(30 * time.Minute), // 30分钟后开始
        EndTime:       time.Now().Add(time.Duration(rule.Duration) * time.Second),
    }
    s.auctionDAO.Create(auction)

    // 5. 更新商品状态
    product.Status = ProductStatusPublished
    s.productDAO.Update(product)

    // 6. 发送通知
    s.notificationService.NotifyFollowers(liveStream.ID, "new_product", product)

    return nil
}
```

**默认规则**:
- 起拍价: 0元
- 加价幅度: 10元
- 竞拍时长: 300秒（5分钟）
- 延时时长: 30秒
- 封顶价: 无

---

## 未解决问题

无。所有技术上下文中的未知项已解决。

## 技术债务记录

1. **二期优化 - 通知聚合**:
   - 当前: 全量推送
   - 优化: 智能聚合和优先级过滤
   - 影响: 用户体验提升，减少打扰

2. **二期优化 - 个性化推荐**:
   - 当前: 按热度排序
   - 优化: 根据用户行为推荐
   - 影响: 提高关注转化率

3. **二期优化 - 关注用户数量限制**:
   - 当前: 无限制
   - 优化: 单个用户最多关注100个直播间
   - 影响: 防止恶意刷关注，提高数据质量

4. **二期优化 - 反垃圾机制**:
   - 当前: 无
   - 优化: 异常关注行为检测（短时间大量关注/取消）
   - 影响: 提高系统安全性

---

## 用户端开发技术决策（2026-05-23更新）

### 8. 用户认证方案

**问题**: 用户端如何实现登录认证，确保出价和关注功能的用户身份验证

**决策**: JWT token存储在localStorage，Axios拦截器自动添加认证头

**理由**:
- 后端已实现JWT认证中间件
- JWT token包含user_id和user_role，满足业务需求
- localStorage持久化存储，用户刷新页面无需重新登录
- Axios拦截器统一处理，避免重复代码

**实现方案**:
```typescript
// API拦截器配置
axios.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});
```

---

### 9. WebSocket实时推送方案

**决策**: 原生WebSocket API + 自动重连机制

**理由**: 无需额外依赖，延迟 < 100ms，后端已支持

---

### 10. 出价输入验证方案

**决策**: 前端实时验证 + 后端二次验证

**前端验证规则**: 出价金额 > 当前最高价 + 最小加价幅度

---

### 11. 关注功能UI交互方案

**决策**: 乐观更新 + 异步同步

**流程**: 点击 → 立即更新UI → 发送请求 → 成功保持/失败回滚

---

### 12. 移动端适配方案

**决策**: 响应式设计 + 触摸优化

**关键点**: 触摸目标44px，字体16px，平滑滚动

---

### 13. 状态管理方案

**决策**: React Context + useReducer（全局）+ useState（组件）

**理由**: 项目规模中等，不需要Redux等复杂方案

---

## 决策总结

所有技术决策已完成：

1. ✅ JWT认证方案确定
2. ✅ WebSocket实时推送方案确定
3. ✅ 状态管理方案确定
4. ✅ UI交互模式确定
5. ✅ 移动端适配方案确定
6. ✅ 出价验证方案确定

**Status**: Phase 0研究完成，可以执行Phase 1设计。
