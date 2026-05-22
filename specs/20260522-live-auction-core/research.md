# Research: 直播竞拍系统核心功能完善

**Feature**: `20260522-live-auction-core`
**Date**: 2026-05-22

## Research Tasks

基于技术上下文分析，识别出以下需要研究的技术决策点：

### 1. WebSocket 断线重连策略

**研究问题**：如何设计高可靠的 WebSocket 重连机制？

**决策**：采用指数退避 + 心跳保活策略

**理由**：
- 指数退避避免网络恢复瞬间的大量重连请求
- 心跳保活及时检测连接状态
- 最大重试次数限制防止无限重连

**备选方案**：
1. **固定间隔重连**：简单但可能造成重连风暴
2. **立即重连**：过于激进，可能导致服务器压力

**实现细节**：
```go
// 后端心跳检测
type Client struct {
    send            chan []byte
    pingTicker      *time.Ticker
    pongWait        time.Duration // 60秒
    pingPeriod      time.Duration // 30秒
}

// 前端重连逻辑
const reconnectDelays = [1, 2, 4, 8, 16, 30, 30, 30, 30, 30] // 秒
```

**最佳实践参考**：
- Socket.IO reconnect algorithm
- WebSocket RFC 6455 ping/pong mechanism

---

### 2. 实时排名广播性能优化

**研究问题**：高并发下如何保证排名广播的性能和准确性？

**决策**：采用消息节流 + 批量广播策略

**理由**：
- 避免短时间内大量排名更新消息
- 批量处理提高吞吐量
- 最终一致性保证数据准确

**备选方案**：
1. **每次出价立即广播**：实时性最好，但并发高时性能差
2. **定时批量广播**：性能好，但延迟可能过高
3. **节流 + 批量**：平衡实时性和性能 ✅

**实现细节**：
```go
// 消息节流器
type Throttler struct {
    interval    time.Duration // 200ms
    pendingMsgs map[int64]*Message
    mu          sync.Mutex
}

// 批量广播
func (t *Throttler) Flush() {
    // 每200ms推送一次最新排名
}
```

**性能指标**：
- 目标延迟：< 200ms
- 目标吞吐：1000 msg/s
- 目标并发：100+ concurrent bids

---

### 3. 服务端时间同步机制

**研究问题**：如何保证多客户端倒计时显示的一致性？

**决策**：采用服务端时间下发 + 客户端校准策略

**理由**：
- 服务端时间是唯一可信源
- 客户端定期校准消除时钟漂移
- 网络延迟补偿提升精度

**备选方案**：
1. **客户端本地倒计时**：简单但各客户端不同步
2. **服务端定时推送**：网络延迟导致不同步
3. **服务端时间 + 客户端校准**：精度最高 ✅

**实现细节**：
```go
// 服务端时间同步消息
type TimeSyncMessage struct {
    Type        string `json:"type"`
    ServerTime  int64  `json:"server_time"`  // Unix 毫秒
    EndTime     int64  `json:"end_time"`     // 竞拍结束时间
}
```

```typescript
// 客户端时间校准
const useServerTime = (serverEndTime: number) => {
  const [countdown, setCountdown] = useState(0);
  const serverTimeOffset = useRef(0);

  // 定期校准
  useEffect(() => {
    const syncInterval = setInterval(() => {
      // WebSocket 接收服务端时间
      // 计算时钟偏差
    }, 10000); // 每10秒校准一次
  }, []);
};
```

**精度目标**：
- 时钟同步误差：< 100ms
- 倒计时显示精度：毫秒级

---

### 4. 前端动画性能优化

**研究问题**：如何在低端设备上保证动画流畅性？

**决策**：采用 CSS 硬件加速 + 自动降级策略

**理由**：
- GPU 加速提升动画性能
- 自动降级保证低端设备可用性
- 可配置关闭动画

**备选方案**：
1. **JavaScript 动画**：灵活但性能差
2. **纯 CSS 动画**：性能好，缺乏控制
3. **CSS + React Transition Group**：平衡性能和控制 ✅

**实现细节**：
```typescript
// 动画降级策略
const useAnimation = () => {
  const [fps, setFps] = useState(60);
  const shouldAnimate = fps >= 30;

  useEffect(() => {
    // FPS 监控
    const monitor = new PerformanceMonitor((fps) => {
      setFps(fps);
    });
  }, []);
};
```

```css
/* CSS 硬件加速 */
.animate-price-change {
  transform: translateZ(0);
  will-change: transform, opacity;
  transition: transform 0.3s ease-out, opacity 0.3s ease-out;
}
```

**性能指标**：
- 目标帧率：> 60fps
- 降级阈值：< 30fps
- 降级策略：禁用动画或简化动画

---

### 5. 管理后台权限控制

**研究问题**：如何实现安全的权限验证机制？

**决策**：采用 JWT + RBAC 权限模型

**理由**：
- JWT 无状态，适合微服务架构
- RBAC 灵活，支持细粒度权限控制
- 已有 JWT 中间件，复用现有实现

**备选方案**：
1. **Session + Cookie**：需要状态存储，不适合微服务
2. **API Key**：简单但权限控制粗糙
3. **JWT + RBAC**：平衡安全性和灵活性 ✅

**实现细节**：
```go
// JWT 中间件
func AuthMiddleware() app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        token := string(c.GetHeader("Authorization"))
        claims, err := jwt.ParseToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }
        c.Set("user_id", claims.UserID)
        c.Set("role", claims.Role)
        c.Next(ctx)
    }
}

// RBAC 权限检查
func RequirePermission(permission string) app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        role := c.GetString("role")
        if !rbac.HasPermission(role, permission) {
            c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
            return
        }
        c.Next(ctx)
    }
}
```

**权限模型**：
- `admin`：完全权限
- `operator`：查看 + 编辑权限
- `viewer`：仅查看权限

---

## Research Summary

所有技术决策点已研究完成，无需 CLARIFICATION 标记。核心决策如下：

| 技术点 | 决策方案 | 主要理由 |
|--------|----------|----------|
| 断线重连 | 指数退避 + 心跳保活 | 避免重连风暴，及时检测断线 |
| 排名广播 | 消息节流 + 批量广播 | 平衡实时性和性能 |
| 时间同步 | 服务端时间 + 客户端校准 | 保证多客户端一致性 |
| 动画性能 | CSS 硬件加速 + 自动降级 | 兼顾性能和兼容性 |
| 权限控制 | JWT + RBAC | 安全灵活，适合微服务 |

**Next Step**: Proceed to Phase 1 - Data Model & Contracts Design
