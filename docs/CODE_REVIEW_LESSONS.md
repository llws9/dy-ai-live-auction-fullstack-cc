# 代码审查问题记录

> 本文档记录代码审查中发现的问题，供团队参考，避免重复犯同样错误。

---

## 消息通知系统 (Cold Push Hot Pull)

### Critical 严重问题

#### 1. WebSocket推送方法错误

**问题描述**:
在 `service/notification.go:139` 中，使用 `BroadcastToRoom` 推送用户通知，但这会将通知广播到竞拍房间（所有观众），而非用户的专属房间。

**错误代码**:
```go
s.hub.BroadcastToRoom(notification.UserID, msg)  // 错误
```

**正确代码**:
```go
s.hub.BroadcastToUserRoom(notification.UserID, msg)  // 正确
```

**根因分析**:
- `BroadcastToRoom(roomID)` 将消息广播到指定房间，roomID 是竞拍ID
- `BroadcastToUserRoom(userID)` 将消息推送到用户专属房间
- 混淆了两种方法的用途

**预防措施**:
- 编写WebSocket推送逻辑时，先确认目标房间类型
- 用户级通知必须使用 `BroadcastToUserRoom`
- 竞拍级广播（如排名更新）使用 `BroadcastToRoom`

---

#### 2. 关注服务Redis同步缺失

**问题描述**:
`service/follow.go` 中 Follow/Unfollow 方法未同步到 Redis，导致热拉通知过滤无法正确获取用户关注的直播间列表。

**根因分析**:
- 热拉通知功能依赖 Redis 存储 `user:{uid}:followed_live_streams` 集合
- 关注/取消关注只更新了数据库，未更新Redis缓存
- 数据不一致导致热拉通知过滤失效

**修复代码**:
```go
// Follow 方法中添加
if err := dao.AddUserFollowedLiveStream(ctx, userID, liveStreamID); err != nil {
    fmt.Printf("Warning: failed to sync follow to Redis: %v\n", err)
}

// Unfollow 方法中添加
if err := dao.RemoveUserFollowedLiveStream(ctx, userID, liveStreamID); err != nil {
    fmt.Printf("Warning: failed to sync unfollow to Redis: %v\n", err)
}
```

**预防措施**:
- 任何涉及缓存的功能，需确保数据库和缓存一致性
- 设计阶段明确数据流向：写数据库 → 写缓存
- 添加单元测试验证缓存同步逻辑

---

### Important 重要问题

#### 3. 热拉通知缺少数据库持久化

**问题描述**:
`HotPullNotifications` 生成的通知直接推送，未保存到数据库。

**影响**:
- 用户查看历史通知时无法看到热拉通知
- 无法标记已读/未读
- 服务重启后通知丢失

**建议修复**:
在推送前调用 `notificationDAO.Create` 保存通知实体。

---

#### 4. 缺少错误处理和幂等性保护

**问题描述**:
- 关注服务缺少幂等性检查（重复关注）
- 热拉通知 Redis 操作失败时未正确处理

**预防措施**:
- 写操作前检查状态（如关注前检查是否已关注）
- Redis 操作失败应有降级策略或重试机制

---

## 点天灯功能 (SkyLamp)

### Important 重要问题

#### 1. 缺少服务层和Handler实现

**问题描述**:
只有 Model 和 DAO 层，缺少 Service 和 Handler 层，无法通过API调用。

**修复**:
- 创建 `service/sky_lamp.go` 实现业务逻辑
- 创建 `handler/sky_lamp.go` 实现API接口
- 在 `main.go` 中注册服务并添加路由

---

#### 2. 未集成到BidService出价流程

**问题描述**:
点天灯触发逻辑未集成到 `BidService.PlaceBid`，无法在被超越时自动跟价。

**修复**:
- 在 `BidService` 添加 `SkyLampTrigger` 接口
- 在出价成功后调用 `TriggerAutoBid`
- 使用异步触发避免阻塞主流程

---

#### 3. 缺少竞拍结束时的订阅清理

**问题描述**:
竞拍结束时，活跃的点天灯订阅未自动停止。

**建议修复**:
在 `AuctionService.EndAuction` 中调用 `skyLampDAO.StopByAuction`。

---

### Minor 小问题

#### 4. 缺少并发保护

**问题描述**:
`TriggerAutoBid` 获取所有订阅后逐个出价，无并发限制。

**建议**:
- 使用 goroutine pool 或限流器控制并发
- 防止短时间内大量出价请求

---

## 种子数据生成器

### Important 重要问题

#### 1. 二进制文件被提交到git

**问题描述**:
`backend/auction/auction-service` 和 `backend/seed/seed` 等二进制文件（11MB）被提交到git仓库。

**修复**:
```bash
git rm backend/auction/auction-service
git rm backend/seed/seed
```

**预防措施**:
- 在 `.gitignore` 中添加二进制文件排除规则
- 提交前检查 `git status`，排除编译产物

---

#### 2. 通知类型不完整

**问题描述**:
种子数据只生成3种通知类型，缺少 `auction_starting`、`live_stream_starting_soon`、`live_stream_now_live`。

**修复**:
扩展通知类型列表，覆盖所有业务场景。

---

## 通用建议

### 代码质量预防措施

1. **接口设计一致性**
   - 同类功能使用相同的接口签名
   - 参考 `NotificationSender` 接口设计模式

2. **数据一致性**
   - 涉及缓存的操作必须同步更新
   - 设计阶段明确数据流向和存储策略

3. **错误处理**
   - 所有外部依赖调用（DB/Redis/API）需有错误处理
   - 关键操作需有降级策略

4. **并发安全**
   - 分布式锁保护竞拍出价等关键操作
   - 异步任务使用限流器防止过载

5. **测试覆盖**
   - 核心业务逻辑必须有单元测试
   - 涉及外部依赖的代码需有集成测试

---

## 更新日志

| 日期 | 版本 | 更新内容 |
|------|------|----------|
| 2026-05-28 | 1.0 | 初版，记录消息通知和点天灯代码审查问题 |

---

> **注意**: 发现新问题时应及时更新本文档，确保团队持续学习改进。