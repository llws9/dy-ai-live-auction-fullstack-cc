# Quickstart: 直播竞拍系统核心功能完善

**Feature**: `20260522-live-auction-core`
**Date**: 2026-05-22

## 概述

本指南帮助开发者快速开始实现核心功能完善模块，包括实时排名同步、断线重连、PC管理后台和体验优化。

---

## 前置条件

### 后端开发环境

```bash
# Go 版本
go version >= 1.21

# 依赖服务
MySQL >= 8.0
Redis >= 7.0

# 环境变量
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=
export DB_NAME=auction
export REDIS_ADDR=localhost:6379
export JWT_SECRET=your-secret-key
```

### 前端开发环境

```bash
# Node.js 版本
node version >= 18.0

# 包管理器
npm >= 9.0
# 或
pnpm >= 8.0
```

---

## 快速开始

### 1. 启动基础服务

```bash
# 启动 MySQL 和 Redis
docker-compose up -d mysql redis

# 等待服务就绪
docker-compose ps
```

### 2. 启动后端服务

```bash
# 启动 Product Service (端口 8081)
cd backend/product
go run main.go

# 启动 Auction Service (端口 8082 HTTP, 8083 WebSocket)
cd backend/auction
go run main.go

# 启动 Gateway Service (端口 8080)
cd backend/gateway
go run main.go
```

### 3. 启动前端服务

```bash
# 启动 H5 前端
cd frontend/h5
npm install
npm run dev

# 启动 Admin 前端（另一个终端）
cd frontend/admin
npm install
npm run dev
```

---

## 实施顺序

### 第一周：实时排名同步 + 断线重连

#### Day 1-2: 实时排名同步

**后端任务**：

1. **修改 `service/bid.go`** - 添加 `broadcastRanking` 方法

```go
// 在 PlaceBid 方法成功后调用
func (s *BidService) broadcastRanking(ctx context.Context, auctionID int64) error {
    rankings, err := s.bidDAO.GetRanking(ctx, auctionID, 10)
    if err != nil {
        return err
    }

    message := &websocket.Message{
        Type: "rank_update",
        Data: rankings,
    }

    s.hub.BroadcastToRoom(auctionID, message)
    return nil
}
```

2. **修改 `websocket/message.go`** - 添加消息类型

```go
const (
    MessageTypeBidPlaced    = "bid_placed"
    MessageTypeRankUpdate   = "rank_update"  // 新增
    MessageTypeOvertaken    = "overtaken"
    MessageTypeDelayTriggered = "delay_triggered"
    MessageTypeAuctionEnded = "auction_ended"
)
```

**前端任务**：

1. **修改 `services/websocket.ts`** - 处理排名消息

```typescript
case 'rank_update':
  this.onRankUpdate?.(data.rankings);
  break;
```

2. **修改 `pages/Auction/Ranking.tsx`** - 实时显示

```typescript
const [rankings, setRankings] = useState<RankingItem[]>([]);

// WebSocket 回调
websocket.onRankUpdate = (newRankings) => {
  setRankings(newRankings);
};
```

#### Day 3-4: 断线重连机制

**后端任务**：

1. **新增 `websocket/state_sync.go`** - 状态同步逻辑

```go
type SyncManager struct {
    redis  *redis.Client
    hub    *Hub
}

func (m *SyncManager) GetSyncState(auctionID int64) (*SyncState, error) {
    key := fmt.Sprintf("sync:state:%d", auctionID)
    // 从 Redis 获取最新状态
}

func (m *SyncManager) UpdateSyncState(auctionID int64, state *SyncState) error {
    key := fmt.Sprintf("sync:state:%d", auctionID)
    // 更新 Redis 状态
}
```

2. **修改 `websocket/client.go`** - 心跳优化

```go
const (
    pongWait       = 60 * time.Second
    pingPeriod     = 30 * time.Second
    maxMessageSize = 512
)

func (c *Client) readPump() {
    defer func() {
        c.hub.Unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    for {
        // 读取消息
    }
}
```

**前端任务**：

1. **新增 `hooks/useReconnect.ts`** - 重连逻辑

```typescript
export const useReconnect = (ws: WebSocket) => {
  const [reconnectCount, setReconnectCount] = useState(0);
  const delays = [1, 2, 4, 8, 16, 30, 30, 30, 30, 30];

  const reconnect = useCallback(() => {
    if (reconnectCount >= 10) {
      console.error('Max reconnect attempts reached');
      return;
    }

    const delay = delays[reconnectCount] * 1000;
    setTimeout(() => {
      ws.reconnect();
      setReconnectCount(prev => prev + 1);
    }, delay);
  }, [reconnectCount, ws]);

  return { reconnect, reconnectCount };
};
```

---

### 第二周：PC管理后台

#### Day 1-2: 商品管理

**后端任务**：

1. **新增 `product/handler/product.go`**

```go
type ProductHandler struct {
    productService *service.ProductService
}

func (h *ProductHandler) List(c context.Context, ctx *app.RequestContext) {
    // 商品列表
}

func (h *ProductHandler) Create(c context.Context, ctx *app.RequestContext) {
    // 创建商品
}

func (h *ProductHandler) Update(c context.Context, ctx *app.RequestContext) {
    // 更新商品
}

func (h *ProductHandler) Delete(c context.Context, ctx *app.RequestContext) {
    // 删除商品
}
```

**前端任务**：

1. **新增 `pages/Product/List.tsx`**
2. **新增 `pages/Product/Create.tsx`**
3. **新增 `pages/Product/Edit.tsx`**

#### Day 3-4: 竞拍管理 & 订单管理

类似结构，参考 API 合约文档

---

### 第三周：体验优化

#### Day 1-2: 动画效果

**新增 `utils/animations.ts`**：

```typescript
export const animations = {
  bidSuccess: {
    keyframes: [
      { transform: 'scale(1)' },
      { transform: 'scale(1.2)' },
      { transform: 'scale(1)' }
    ],
    options: {
      duration: 300,
      easing: 'ease-out'
    }
  },
  priceChange: {
    keyframes: [
      { opacity: 1, transform: 'translateY(0)' },
      { opacity: 0.8, transform: 'translateY(-10px)' },
      { opacity: 1, transform: 'translateY(0)' }
    ],
    options: {
      duration: 200
    }
  }
};
```

#### Day 3: 倒计时精度

**新增 `hooks/useServerTime.ts`**：

```typescript
export const useServerTime = (serverEndTime: number) => {
  const [countdown, setCountdown] = useState(0);
  const offset = useRef(0);

  useEffect(() => {
    let frameId: number;

    const update = () => {
      const now = Date.now() + offset.current;
      const remaining = Math.max(0, serverEndTime - now);
      setCountdown(remaining);

      if (remaining > 0) {
        frameId = requestAnimationFrame(update);
      }
    };

    frameId = requestAnimationFrame(update);
    return () => cancelAnimationFrame(frameId);
  }, [serverEndTime]);

  return countdown;
};
```

#### Day 4: 历史记录

**后端 API**：`GET /api/v1/users/me/history`

**前端页面**：`pages/History/index.tsx`

---

## 测试验证

### 单元测试

```bash
# 后端测试
cd backend/auction
go test ./service -v
go test ./websocket -v

# 前端测试
cd frontend/h5
npm test
```

### 集成测试

```bash
# WebSocket 连接测试
wscat -c ws://localhost:8083/ws?auction_id=1&user_id=10001

# API 测试
curl http://localhost:8080/api/v1/products
curl http://localhost:8080/api/v1/auctions
```

### 性能测试

```bash
# 并发出价测试
cd scripts
go run concurrent_bids.go -n 100 -auction-id 1
```

---

## 常见问题

### 1. WebSocket 连接失败

**检查项**：
- Auction Service 是否启动 (端口 8083)
- auction_id 是否存在
- user_id 或 token 是否正确

### 2. 排名不同步

**检查项**：
- Redis 连接是否正常
- WebSocket Hub 是否运行
- `broadcastRanking` 是否被调用

### 3. 重连失败

**检查项**：
- 最大重试次数是否超过 10 次
- 网络是否恢复
- 服务端状态是否正常

---

## 参考文档

- [spec.md](./spec.md) - 功能规格说明
- [data-model.md](./data-model.md) - 数据模型设计
- [contracts/](./contracts/) - API 合约定义
- [CONSTITUTION.md](../../docs/CONSTITUTION.md) - 项目宪法

---

## 下一步

完成本指南后，执行 `/adk:sdd:tasks` 生成详细的任务分解和实施计划。
