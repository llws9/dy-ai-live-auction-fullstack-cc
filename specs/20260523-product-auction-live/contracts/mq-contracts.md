# RabbitMQ Architecture: 消息队列架构设计

**Feature**: `20260523-product-auction-live`
**Date**: 2026-05-23

## 1. 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        Producer (API Layer)                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ 商品发布API  │  │ 竞拍开始API  │  │ 商品下架API  │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
                            ↓ 发送消息
┌─────────────────────────────────────────────────────────────────┐
│                        RabbitMQ Exchange                         │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  notification.delayed (x-delayed-message)                │   │
│  │    ├── new_product_delayed      (延迟 0s)                │   │
│  │    ├── auction_starting_delayed (延迟 30分钟)            │   │
│  │    └── auction_ended_delayed    (延迟 0s)                │   │
│  └─────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  notification.direct (direct)                            │   │
│  │    └── notification_immediate (立即发送)                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  notification.dlx (dead-letter-exchange)                 │   │
│  │    └── notification_failed (失败重试队列)                │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                            ↓ 消费消息
┌─────────────────────────────────────────────────────────────────┐
│                    Consumer (Notification Worker)                │
│  ┌──────────────────────────────────────────────────────┐      │
│  │  1. 解析消息                                          │      │
│  │  2. 查询关注用户列表（分批 1万/批）                   │      │
│  │  3. 批量推送通知                                      │      │
│  │  4. 更新推送状态                                      │      │
│  │  5. 发送 ACK 确认                                     │      │
│  └──────────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Exchange 和 Queue 设计

### 2.1 延迟交换机（核心）

**Exchange**: `notification.delayed`
**Type**: `x-delayed-message`
**Purpose**: 处理需要延迟发送的通知（如竞拍开始前30分钟）

```go
// 创建延迟交换机
err := ch.ExchangeDeclare(
    "notification.delayed",       // name
    "x-delayed-message",          // type (插件提供)
    true,                         // durable
    false,                        // auto-deleted
    false,                        // internal
    false,                        // no-wait
    amqp.Table{
        "x-delayed-type": "direct",  // 底层路由类型
    },
)
```

### 2.2 延迟队列

#### Queue 1: `new_product_delayed` - 新商品发布通知

**延迟时间**: 0s（立即发送）
**消费者**: Notification Worker
**消息体**:
```json
{
  "live_stream_id": 10,
  "type": "new_product",
  "product_id": 1,
  "product_name": "稀有珠宝",
  "creator_name": "张三",
  "created_at": "2026-05-23T10:00:00Z"
}
```

#### Queue 2: `auction_starting_delayed` - 竞拍即将开始通知

**延迟时间**: 30分钟（1800000ms）
**消费者**: Notification Worker
**消息体**:
```json
{
  "live_stream_id": 10,
  "type": "auction_starting",
  "product_id": 1,
  "auction_id": 100,
  "product_name": "稀有珠宝",
  "start_time": "2026-05-23T10:30:00Z",
  "created_at": "2026-05-23T10:00:00Z"
}
```

#### Queue 3: `auction_ended_delayed` - 竞拍结束通知

**延迟时间**: 0s（立即发送）
**消费者**: Notification Worker
**消息体**:
```json
{
  "live_stream_id": 10,
  "type": "auction_ended",
  "product_id": 1,
  "auction_id": 100,
  "winner_id": 5,
  "winner_name": "用户A",
  "final_price": 150.00,
  "created_at": "2026-05-23T10:35:00Z"
}
```

### 2.3 直接交换机（立即发送）

**Exchange**: `notification.direct`
**Type**: `direct`
**Purpose**: 处理需要立即发送的通知（如商品下架）

```go
// 创建直接交换机
err := ch.ExchangeDeclare(
    "notification.direct", // name
    "direct",              // type
    true,                  // durable
    false,                 // auto-deleted
    false,                 // internal
    false,                 // no-wait
    nil,                   // arguments
)
```

#### Queue: `notification_immediate`

**Routing Key**: `product_unpublished`
**消息体**:
```json
{
  "live_stream_id": 10,
  "type": "product_unpublished",
  "product_id": 1,
  "product_name": "稀有珠宝",
  "reason": "商品质量问题",
  "created_at": "2026-05-23T10:05:00Z"
}
```

### 2.4 死信交换机（失败重试）

**Exchange**: `notification.dlx`
**Type**: `direct`
**Purpose**: 处理推送失败的消息

```go
// 创建死信交换机
err := ch.ExchangeDeclare(
    "notification.dlx", // name
    "direct",           // type
    true,               // durable
    false,              // auto-deleted
    false,              // internal
    false,              // no-wait
    nil,                // arguments
)
```

#### Queue: `notification_failed`

**消息体**: 与原始消息相同，新增错误信息字段
```json
{
  "live_stream_id": 10,
  "type": "new_product",
  "product_id": 1,
  "error": "database connection timeout",
  "retry_count": 3,
  "created_at": "2026-05-23T10:00:00Z"
}
```

---

## 3. 生产者实现

### 3.1 初始化连接

```go
package mq

import (
    amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationProducer struct {
    conn    *amqp.Connection
    channel *amqp.Channel
}

func NewNotificationProducer(url string) (*NotificationProducer, error) {
    conn, err := amqp.Dial(url)
    if err != nil {
        return nil, err
    }

    ch, err := conn.Channel()
    if err != nil {
        return nil, err
    }

    // 创建延迟交换机
    ch.ExchangeDeclare(
        "notification.delayed",
        "x-delayed-message",
        true, false, false, false,
        amqp.Table{"x-delayed-type": "direct"},
    )

    // 创建直接交换机
    ch.ExchangeDeclare(
        "notification.direct",
        "direct",
        true, false, false, false, nil,
    )

    // 创建死信交换机
    ch.ExchangeDeclare(
        "notification.dlx",
        "direct",
        true, false, false, false, nil,
    )

    return &NotificationProducer{conn: conn, channel: ch}, nil
}
```

### 3.2 发送延迟消息

```go
// 发送延迟消息（竞拍开始前30分钟通知）
func (p *NotificationProducer) SendAuctionStartingNotification(
    liveStreamID int64,
    auctionID int64,
    productID int64,
    productName string,
    startTime time.Time,
) error {
    // 计算延迟时间 = 开始时间 - 30分钟 - 当前时间
    delayMs := startTime.Add(-30 * time.Minute).Sub(time.Now()).Milliseconds()
    if delayMs < 0 {
        delayMs = 0 // 如果已经过了30分钟前，立即发送
    }

    msg := NotificationMessage{
        LiveStreamID: liveStreamID,
        Type:         "auction_starting",
        ProductID:    productID,
        AuctionID:    auctionID,
        ProductName:  productName,
        StartTime:    startTime,
        CreatedAt:    time.Now(),
    }

    body, _ := json.Marshal(msg)

    return p.channel.Publish(
        "notification.delayed",   // exchange
        "auction_starting",       // routing key
        false, false,
        amqp.Publishing{
            ContentType:  "application/json",
            Body:         body,
            DeliveryMode: amqp.Persistent,
            Headers: amqp.Table{
                "x-delay": delayMs,  // 延迟时间（毫秒）
            },
        },
    )
}
```

### 3.3 发送立即消息

```go
// 发送立即消息（商品下架通知）
func (p *NotificationProducer) SendProductUnpublishedNotification(
    liveStreamID int64,
    productID int64,
    productName string,
    reason string,
) error {
    msg := NotificationMessage{
        LiveStreamID: liveStreamID,
        Type:         "product_unpublished",
        ProductID:    productID,
        ProductName:  productName,
        Reason:       reason,
        CreatedAt:    time.Now(),
    }

    body, _ := json.Marshal(msg)

    return p.channel.Publish(
        "notification.direct",      // exchange
        "product_unpublished",      // routing key
        false, false,
        amqp.Publishing{
            ContentType:  "application/json",
            Body:         body,
            DeliveryMode: amqp.Persistent,
        },
    )
}
```

---

## 4. 消费者实现

### 4.1 初始化消费者

```go
package mq

type NotificationConsumer struct {
    conn    *amqp.Connection
    channel *amqp.Channel
    svc     *NotificationService
}

func NewNotificationConsumer(url string, svc *NotificationService) (*NotificationConsumer, error) {
    conn, err := amqp.Dial(url)
    if err != nil {
        return nil, err
    }

    ch, err := conn.Channel()
    if err != nil {
        return nil, err
    }

    // 设置 QoS（每个消费者最多处理10条未确认消息）
    ch.Qos(10, 0, false)

    // 创建队列
    queues := []string{
        "new_product_delayed",
        "auction_starting_delayed",
        "auction_ended_delayed",
        "notification_immediate",
    }

    for _, q := range queues {
        _, err := ch.QueueDeclare(
            q,
            true,  // durable
            false, // auto-delete
            false, // exclusive
            false, // no-wait
            amqp.Table{
                "x-dead-letter-exchange":    "notification.dlx",
                "x-dead-letter-routing-key": "failed",
            },
        )
        if err != nil {
            return nil, err
        }
    }

    // 绑定队列到交换机
    ch.QueueBind("new_product_delayed", "new_product", "notification.delayed", false, nil)
    ch.QueueBind("auction_starting_delayed", "auction_starting", "notification.delayed", false, nil)
    ch.QueueBind("auction_ended_delayed", "auction_ended", "notification.delayed", false, nil)
    ch.QueueBind("notification_immediate", "product_unpublished", "notification.direct", false, nil)

    return &NotificationConsumer{conn: conn, channel: ch, svc: svc}, nil
}
```

### 4.2 消费消息

```go
// 启动消费者
func (c *NotificationConsumer) Start() error {
    queues := []string{
        "new_product_delayed",
        "auction_starting_delayed",
        "auction_ended_delayed",
        "notification_immediate",
    }

    for _, queue := range queues {
        go c.consume(queue)
    }

    return nil
}

func (c *NotificationConsumer) consume(queueName string) {
    msgs, err := c.channel.Consume(
        queueName,
        "",     // consumer tag
        false,  // auto-ack (false = 手动确认)
        false,  // exclusive
        false,  // no-local
        false,  // no-wait
        nil,    // args
    )
    if err != nil {
        log.Printf("Failed to register consumer for %s: %v", queueName, err)
        return
    }

    for msg := range msgs {
        var notification NotificationMessage
        if err := json.Unmarshal(msg.Body, &notification); err != nil {
            log.Printf("Failed to unmarshal message: %v", err)
            msg.Ack(false) // 格式错误的消息直接确认，不重试
            continue
        }

        log.Printf("Processing notification: %+v", notification)

        // 处理通知推送
        if err := c.svc.ProcessNotification(&notification); err != nil {
            log.Printf("Failed to process notification: %v", err)
            // 消费失败，进入死信队列
            msg.Nack(false, false)
        } else {
            // 成功确认
            msg.Ack(false)
        }
    }
}
```

---

## 5. 通知服务实现

### 5.1 批量推送逻辑

```go
package service

type NotificationService struct {
    followDAO   *dao.UserLiveStreamFollowDAO
    notifyDAO   *dao.NotificationDAO
    batchSize   int
    batchDelay  time.Duration
}

func NewNotificationService(followDAO *dao.UserLiveStreamFollowDAO, notifyDAO *dao.NotificationDAO) *NotificationService {
    return &NotificationService{
        followDAO:  followDAO,
        notifyDAO:  notifyDAO,
        batchSize:  10000,                // 1万用户/批
        batchDelay: 3 * time.Second,      // 批次间隔3秒
    }
}

func (s *NotificationService) ProcessNotification(msg *NotificationMessage) error {
    // 1. 获取关注用户总数
    totalUsers, err := s.followDAO.CountByLiveStream(msg.LiveStreamID)
    if err != nil {
        return err
    }

    // 2. 计算批次数
    batches := (totalUsers + int64(s.batchSize) - 1) / int64(s.batchSize)

    log.Printf("Total users: %d, batches: %d", totalUsers, batches)

    // 3. 分批推送
    for i := 0; i < int(batches); i++ {
        offset := i * s.batchSize
        users, err := s.followDAO.GetFollowers(msg.LiveStreamID, offset, s.batchSize)
        if err != nil {
            log.Printf("Failed to get followers (batch %d): %v", i, err)
            continue
        }

        // 批量创建通知记录
        notifications := make([]*model.Notification, 0, len(users))
        for _, user := range users {
            notifications = append(notifications, &model.Notification{
                UserID:  user.UserID,
                Type:    model.NotificationType(msg.Type),
                Title:   s.generateTitle(msg),
                Content: s.generateContent(msg),
                Data:    s.generateData(msg),
            })
        }

        // 批量插入数据库
        if err := s.notifyDAO.BatchCreate(notifications); err != nil {
            log.Printf("Failed to create notifications (batch %d): %v", i, err)
            continue
        }

        log.Printf("Batch %d/%d completed: %d users notified", i+1, batches, len(users))

        // 批次间隔
        if i < int(batches)-1 {
            time.Sleep(s.batchDelay)
        }
    }

    return nil
}

func (s *NotificationService) generateTitle(msg *NotificationMessage) string {
    switch msg.Type {
    case "new_product":
        return "新商品上架"
    case "auction_starting":
        return "竞拍即将开始"
    case "auction_ended":
        return "竞拍已结束"
    case "product_unpublished":
        return "商品已下架"
    default:
        return "通知"
    }
}

func (s *NotificationService) generateContent(msg *NotificationMessage) string {
    switch msg.Type {
    case "new_product":
        return fmt.Sprintf("直播间发布了新商品【%s】，快来参与竞拍吧！", msg.ProductName)
    case "auction_starting":
        return fmt.Sprintf("商品【%s】的竞拍即将在30分钟后开始，起拍价%.2f元！", msg.ProductName, msg.StartPrice)
    case "auction_ended":
        return fmt.Sprintf("商品【%s】的竞拍已结束，中标者：%s，成交价：%.2f元", msg.ProductName, msg.WinnerName, msg.FinalPrice)
    case "product_unpublished":
        return fmt.Sprintf("商品【%s】已被商家下架，原因：%s", msg.ProductName, msg.Reason)
    default:
        return "您有一条新通知"
    }
}

func (s *NotificationService) generateData(msg *NotificationMessage) map[string]interface{} {
    data := map[string]interface{}{
        "live_stream_id": msg.LiveStreamID,
        "product_id":     msg.ProductID,
    }

    if msg.AuctionID > 0 {
        data["auction_id"] = msg.AuctionID
    }

    return data
}
```

---

## 6. 集成到业务逻辑

### 6.1 商品发布时发送通知

```go
// Product Service
func (h *ProductHandler) Publish(c *app.RequestContext) {
    // ... 创建竞拍记录 ...

    // 发送新商品通知（立即发送）
    h.producer.SendNewProductNotification(
        liveStream.ID,
        product.ID,
        product.Name,
        creator.Name,
    )

    // 发送竞拍开始通知（延迟30分钟-竞拍时长，确保在开始前30分钟送达）
    h.producer.SendAuctionStartingNotification(
        liveStream.ID,
        auction.ID,
        product.ID,
        product.Name,
        auction.StartTime,
    )

    c.JSON(200, map[string]interface{}{
        "code":    200,
        "message": "发布成功",
        "data": map[string]interface{}{
            "product":  product,
            "auction":  auction,
            "live_stream": liveStream,
        },
    })
}
```

### 6.2 商品下架时发送通知

```go
// Product Service
func (h *ProductHandler) Unpublish(c *app.RequestContext) {
    // ... 取消竞拍记录 ...

    // 发送商品下架通知（立即发送）
    h.producer.SendProductUnpublishedNotification(
        liveStream.ID,
        product.ID,
        product.Name,
        reason,
    )

    c.JSON(200, map[string]interface{}{
        "code":    200,
        "message": "下架成功",
    })
}
```

---

## 7. 监控与运维

### 7.1 RabbitMQ 管理界面

访问 `http://localhost:15672`（默认账号：guest/guest）

**监控指标**:
- Queue 消息堆积情况
- Consumer 处理速率
- Dead Letter Queue 消息数量
- Network throughput

### 7.2 日志记录

```go
// 消费者日志
log.Printf("[%s] Received message: %s", queueName, msg.MessageId)
log.Printf("[%s] Processing batch %d/%d, users: %d", msg.Type, i+1, batches, len(users))
log.Printf("[%s] Batch completed in %v", msg.Type, duration)

// 错误日志
log.Printf("[ERROR] Failed to process notification: %v", err)
log.Printf("[ERROR] Message sent to DLX: %s", msg.MessageId)
```

### 7.3 告警规则

1. **消息堆积告警**: `notification_delayed` 队列消息数 > 1000
2. **死信队列告警**: `notification_failed` 队列消息数 > 100
3. **消费者掉线告警**: Consumer 数量 < 预期数量

---

## 8. 性能优化

### 8.1 批量插入优化

```go
// 使用批量插入提高数据库写入性能
func (d *NotificationDAO) BatchCreate(notifications []*model.Notification) error {
    if len(notifications) == 0 {
        return nil
    }

    return d.db.CreateInBatches(notifications, 1000).Error
}
```

### 8.2 并发消费者

```go
// 启动多个消费者实例
for i := 0; i < 5; i++ {
    go c.consume("new_product_delayed")
}
```

### 8.3 连接池

```go
// RabbitMQ 连接池
type ConnectionPool struct {
    connections []*amqp.Connection
    channels    []*amqp.Channel
    mu          sync.Mutex
    index       int
}

func (p *ConnectionPool) GetChannel() *amqp.Channel {
    p.mu.Lock()
    defer p.mu.Unlock()

    p.index = (p.index + 1) % len(p.channels)
    return p.channels[p.index]
}
```

---

## 9. 故障处理

### 9.1 生产者故障

**场景**: RabbitMQ 连接断开

**处理**:
1. 自动重连机制
2. 消息持久化到本地文件
3. 重连后重新发送

```go
func (p *NotificationProducer) reconnect() {
    for {
        time.Sleep(5 * time.Second)

        conn, err := amqp.Dial(p.url)
        if err != nil {
            log.Printf("Reconnect failed: %v", err)
            continue
        }

        p.conn = conn
        p.channel, _ = conn.Channel()
        log.Println("Reconnected to RabbitMQ")
        break
    }
}
```

### 9.2 消费者故障

**场景**: Worker 崩溃，消息未确认

**处理**:
1. RabbitMQ 自动将未确认消息重新入队
2. 其他 Worker 接管处理
3. 监控告警，快速恢复

### 9.3 数据库故障

**场景**: 批量插入失败

**处理**:
1. 消息 Nack，进入死信队列
2. 死信队列消费者重试（最多3次）
3. 最终失败记录到错误日志表

```go
// 死信队列消费者
func (c *NotificationConsumer) consumeDLQ() {
    msgs, _ := c.channel.Consume("notification_failed", "", false, false, false, false, nil)

    for msg := range msgs {
        var notification NotificationMessage
        json.Unmarshal(msg.Body, &notification)

        retryCount := msg.Headers["retry_count"].(int32)
        if retryCount >= 3 {
            // 记录到错误日志表
            c.svc.LogError(notification, "max retries exceeded")
            msg.Ack(false)
            continue
        }

        // 重试
        notification.RetryCount = retryCount + 1
        body, _ := json.Marshal(notification)

        c.channel.Publish(
            "notification.delayed",
            notification.Type,
            false, false,
            amqp.Publishing{
                ContentType:  "application/json",
                Body:         body,
                DeliveryMode: amqp.Persistent,
                Headers: amqp.Table{
                    "retry_count": retryCount + 1,
                },
            },
        )

        msg.Ack(false)
    }
}
```

---

## 10. 扩展性

### 10.1 多机房部署

```
Master (北京) ──────┐
                    ├── Federation ──> Slave (上海)
Slave (广州)  ──────┘
```

### 10.2 消息分片

按直播间ID分片，提高并发处理能力：

```go
// 按直播间ID分片到不同队列
shardIndex := liveStreamID % 10
queueName := fmt.Sprintf("notification_shard_%d", shardIndex)
```

---

## 11. 对比 Redis Stream

| 特性 | RabbitMQ | Redis Stream |
|------|----------|--------------|
| 延迟队列 | ✅ 原生支持（插件） | ❌ 需要手动实现 |
| 死信队列 | ✅ 原生支持 | ❌ 需要手动实现 |
| 管理界面 | ✅ Web UI | ❌ 命令行 |
| 消息确认 | ✅ ACK机制 | ✅ XACK机制 |
| 吞吐量 | 数万QPS | 数十万QPS |
| 运维复杂度 | 中等 | 低（复用Redis） |
| 适用场景 | 企业级业务 | 轻量级场景 |

**结论**: RabbitMQ 功能更完善，适合当前百万级用户通知场景，特别是延迟队列和死信队列是刚需。
