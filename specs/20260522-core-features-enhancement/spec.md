# Feature Specification: 直播竞拍系统核心功能完善

**Feature**: `20260522-core-features-enhancement`
**Created**: 2026-05-22
**Status**: Draft
**Input**: 本地技术提案文档: [core-features-enhancement_brainstorm.md](../brainstorm/core-features-enhancement_brainstorm.md)

## 需求背景

直播竞拍系统已完成基础功能实现，但在以下四个核心领域存在功能缺口：

1. **Redis状态同步未启用**：StateManager已创建但在WebSocket连接中未被调用，导致连接状态无法持久化，重连后无法恢复状态。
2. **用户历史记录返回模拟数据**：GetUserHistory方法返回硬编码数据，用户无法查看真实的竞拍历史。
3. **时间同步缺少周期性推送**：客户端倒计时依赖本地时间，存在时间偏差风险，影响竞拍公平性。
4. **权限验证不完整**：缺少RBAC权限控制，无法区分普通用户、主播和管理员的操作权限。

**设计原则**：采用方案B（完整重构方案），在现有架构基础上引入新的服务层组件，提升系统的可扩展性和可靠性，同时控制重构风险。

---

## User Scenarios & Testing

### User Story 1 - Redis状态同步与分布式锁 (Priority: P1)

**描述**：作为系统，需要在WebSocket连接时启用Redis状态同步，并在竞拍出价时使用分布式锁防止并发冲突，确保竞拍数据的一致性和可靠性。

**Why this priority**: 这是系统稳定性的基础，直接影响竞拍数据的正确性，是其他功能正常运行的前提。

**Technical Implementation**:

1. **新建文件**：
   - `auction/service/lock.go#DistributedLockService` - Redis分布式锁服务
   - `auction/websocket/manager.go#WebSocketManager` - 统一管理Hub和StateManager

2. **修改文件**：
   - `auction/websocket/hub.go#Hub` - 添加stateManager字段
   - `auction/websocket/client.go#Client` - 连接时保存状态到Redis
   - `auction/service/bid.go#PlaceBid` - 使用分布式锁保护出价操作
   - `auction/main.go#main` - 创建并注入新服务

3. **核心逻辑**：
```go
// DistributedLockService 分布式锁
type DistributedLockService struct {
    redis *redis.Client
}

func (s *DistributedLockService) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
    return s.redis.SetNX(ctx, key, "locked", ttl).Result()
}

func (s *DistributedLockService) ReleaseLock(ctx context.Context, key string) error {
    return s.redis.Del(ctx, key).Err()
}
```

4. **调用链路**：
```
出价请求 → BidHandler → DistributedLockService.AcquireLock → BidService.PlaceBid → BidDAO.Create → DB → ReleaseLock
WebSocket连接 → Client.Connect → WebSocketManager.Register → StateManager.SaveConnectionState → Redis
```

**Independent Test**: 可通过并发出价测试验证分布式锁的正确性，通过WebSocket重连测试验证状态恢复。

**Acceptance Scenarios**:

1. **Given** 用户发起竞拍出价，**When** 多个用户同时出价，**Then** 分布式锁确保出价操作串行执行，无数据竞争
2. **Given** 用户WebSocket连接断开，**When** 用户重新连接，**Then** 从Redis恢复连接状态，用户可继续参与竞拍
3. **Given** Redis不可用，**When** 系统尝试获取分布式锁，**Then** 降级为本地内存锁，记录错误日志，不影响主流程

---

### User Story 2 - 用户历史记录真实查询 (Priority: P2)

**描述**：作为用户，希望能查看真实的竞拍历史记录，包括参与过的竞拍、出价次数、是否中标等信息，以便了解自己的竞拍活动。

**Why this priority**: 提升用户体验，让用户能追溯历史活动，但不是核心竞拍流程的阻塞性问题。

**Technical Implementation**:

1. **新建文件**：
   - `product/dao/history.go#HistoryDAO` - 用户历史记录DAO
   - `product/service/history.go#HistoryService` - 历史记录服务

2. **修改文件**：
   - `product/service/order.go#GetUserHistory` - 调用HistoryService替代模拟数据

3. **核心查询逻辑**：
```sql
-- 用户参与的竞拍历史
SELECT 
    a.id as auction_id,
    p.name as product_name,
    o.final_price,
    o.winner_id = ? as is_winner,
    COUNT(b.id) as bid_count,
    a.created_at
FROM auctions a
JOIN products p ON a.product_id = p.id
JOIN bids b ON a.id = b.auction_id AND b.user_id = ?
LEFT JOIN orders o ON a.id = o.auction_id
WHERE a.status = 3  -- 已结束
GROUP BY a.id
ORDER BY a.created_at DESC
LIMIT ? OFFSET ?
```

4. **调用链路**：
```
历史查询请求 → OrderHandler → HistoryService.GetUserHistory → HistoryDAO.QueryUserHistory → DB(auctions + bids + products)
```

**Independent Test**: 可通过API测试验证返回的真实数据，对比数据库记录确认查询正确性。

**Acceptance Scenarios**:

1. **Given** 用户已参与过竞拍，**When** 用户查询历史记录，**Then** 返回真实的竞拍历史，包含商品名、出价次数、是否中标
2. **Given** 用户未参与过任何竞拍，**When** 用户查询历史记录，**Then** 返回空列表
3. **Given** 用户查询历史记录，**When** 指定分页参数，**Then** 返回正确的分页数据

---

### User Story 3 - 时间同步周期性推送 (Priority: P2)

**描述**：作为用户，希望客户端倒计时与服务器时间保持同步，避免因本地时间偏差导致竞拍不公平。

**Why this priority**: 影响竞拍公平性，但可通过客户端校时暂时缓解，优先级与用户历史记录相当。

**Technical Implementation**:

1. **修改文件**：
   - `auction/service/scheduler.go#Scheduler` - 添加时间同步推送任务
   - `auction/websocket/time_sync.go#TimeSyncService` - 添加BroadcastTimeSync方法

2. **核心逻辑**：
```go
// Scheduler 中添加时间同步任务
func (s *Scheduler) startTimeSyncTask() {
    ticker := time.NewTicker(5 * time.Second)
    go func() {
        for {
            select {
            case <-ticker.C:
                s.broadcastTimeSync()
            case <-s.stopCh:
                ticker.Stop()
                return
            }
        }
    }()
}

func (s *Scheduler) broadcastTimeSync() {
    auctions, _ := s.auctionDAO.ListByStatus(context.Background(), model.AuctionStatusOngoing)
    for _, auction := range auctions {
        msg := s.timeSyncService.CreateTimeSyncMessage(auction.EndTime.UnixMilli())
        s.hub.BroadcastToRoom(auction.ID, msg)
    }
}
```

3. **调用链路**：
```
定时器(5s) → Scheduler.broadcastTimeSync → AuctionDAO.ListByStatus(ongoing) → TimeSyncService.CreateTimeSyncMessage → Hub.BroadcastToRoom → 所有客户端
```

**Independent Test**: 可通过客户端日志验证每5秒收到时间同步消息。

**Acceptance Scenarios**:

1. **Given** 有进行中的竞拍，**When** 每5秒触发时间同步，**Then** 向所有竞拍房间的客户端推送服务器时间
2. **Given** 客户端收到时间同步消息，**When** 计算倒计时，**Then** 与服务器时间保持一致（误差<500ms）
3. **Given** 没有进行中的竞拍，**When** 时间同步任务触发，**Then** 不推送任何消息

---

### User Story 4 - RBAC权限验证 (Priority: P3)

**描述**：作为系统管理员，希望能区分普通用户、主播和管理员的操作权限，确保敏感操作（创建/取消竞拍）只有授权用户可执行。

**Why this priority**: 安全性增强，但基础功能已可用，可在系统稳定后完善。

**Technical Implementation**:

1. **角色定义**：

| Role ID | 角色名称 | 说明 |
|---------|---------|------|
| 0 | 普通用户 | 参与竞拍 |
| 1 | 主播 | 管理自己的直播间和竞拍 |
| 2 | 平台管理员 | 管理所有资源 |

2. **权限矩阵**：

| 操作 | 普通用户 | 主播 | 平台管理员 |
|------|---------|------|-----------|
| 查看竞拍列表 | ✅ | ✅ | ✅ |
| 出价竞拍 | ✅ | ✅ | ✅ |
| 创建竞拍 | ❌ | ✅ (自己的) | ✅ |
| 取消竞拍 | ❌ | ✅ (自己的) | ✅ |

3. **新建文件**：
   - `gateway/middleware/rbac.go#RBACMiddleware` - 角色权限中间件
   - `auction/middleware/rbac.go#RBACMiddleware` - 角色权限中间件

4. **修改文件**：
   - `gateway/router/router.go` - 对敏感路由添加RBAC中间件
   - `auction/model/user.go` - 添加Role常量和方法

5. **DB变更**：
   - `users.role` - 新增字段，用户角色，默认0（普通用户）
   - `auctions.creator_id` - 新增字段，竞拍创建者ID（主播），用于归属检查

6. **核心逻辑**：
```go
// RBACMiddleware 权限中间件
func RBACMiddleware(requiredRole int) app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        userRole := c.GetInt("user_role") // 从 JWT 解析

        if userRole < requiredRole {
            c.JSON(403, map[string]interface{}{
                "code":    403,
                "message": "权限不足",
            })
            c.Abort()
            return
        }
        c.Next(ctx)
    }
}
```

7. **调用链路**：
```
请求 → JWTMiddleware(解析token) → RBACMiddleware(检查角色) → Handler → Service
```

**Independent Test**: 可通过不同角色的API调用测试验证权限控制。

**Acceptance Scenarios**:

1. **Given** 普通用户尝试创建竞拍，**When** 调用创建竞拍API，**Then** 返回403权限不足
2. **Given** 主播创建竞拍，**When** 竞拍创建成功，**Then** auctions.creator_id设置为当前用户ID
3. **Given** 主播尝试取消他人竞拍，**When** 检查归属，**Then** 返回403无权操作他人资源
4. **Given** 平台管理员取消竞拍，**When** 权限检查通过，**Then** 竞拍成功取消

---

### Edge Cases

1. **Redis不可用**：分布式锁应降级为本地内存锁，WebSocket状态同步应记录错误但不影响连接
2. **数据库查询超时**：用户历史记录查询应设置超时，返回空列表并记录日志
3. **时间同步消息丢失**：客户端应实现本地倒计时，服务器推送作为校准依据
4. **权限配置错误**：用户角色默认为普通用户，手动提权

---

## Requirements

### Functional Requirements

- **FR-001**: System MUST 在WebSocket连接时保存连接状态到Redis
- **FR-002**: System MUST 在竞拍出价时使用分布式锁防止并发冲突
- **FR-003**: System MUST 返回真实的用户竞拍历史记录
- **FR-004**: System MUST 每5秒向进行中的竞拍推送服务器时间
- **FR-005**: System MUST 根据用户角色控制操作权限
- **FR-006**: System MUST 记录竞拍创建者ID用于归属检查
- **FR-007**: System MUST 在Redis不可用时降级为本地内存锁

### Key Entities

- **DistributedLock**: 分布式锁实体，包含key、value、ttl属性
- **ConnectionState**: WebSocket连接状态，包含clientID、auctionID、userID、connectedAt
- **UserHistory**: 用户历史记录，包含auctionID、productName、finalPrice、isWinner、bidCount
- **Role**: 用户角色，包含ID和权限列表

---

## Success Criteria

### Measurable Outcomes

- **SC-001**: 并发出价测试通过，无数据竞争（100次并发测试全部成功）
- **SC-002**: WebSocket重连后状态恢复成功率100%
- **SC-003**: 用户历史记录查询响应时间<500ms
- **SC-004**: 时间同步推送间隔准确（误差<100ms）
- **SC-005**: 权限验证准确率100%，无越权操作
- **SC-006**: 所有现有测试保持通过，无回归问题

---

## Involved Projects

| Service (PSM) | Project Path | Change Type |
|---------------|--------------|-------------|
| auction-service | backend/auction | Modified |
| product-service | backend/product | Modified |
| gateway-service | backend/gateway | Modified |

---

## Risk Points

1. **破坏现有功能**: 重构可能影响现有测试。缓解措施：保持所有现有测试通过，增量添加测试。
2. **Redis不可用**: 分布式锁依赖Redis。缓解措施：降级为本地内存锁，记录错误日志，不影响主流程。
3. **权限配置错误**: 用户角色未正确设置。缓解措施：默认为普通用户，手动提权。
4. **并发竞拍冲突**: 分布式锁实现不当可能导致数据竞争。缓解措施：使用PEXPIRE自动续期，锁超时后自动释放。
5. **主播归属检查遗漏**: 主播可能操作他人资源。缓解措施：在敏感操作（创建/取消竞拍）中强制检查creator_id。
