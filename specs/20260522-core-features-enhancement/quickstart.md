# Quickstart: 直播竞拍系统核心功能完善

**Feature**: `20260522-core-features-enhancement`
**Date**: 2026-05-22

## 快速启动指南

### 1. 环境准备

#### 1.1 启动 Redis

```bash
# 方式1: 使用 Docker Compose (推荐)
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc
docker-compose up -d redis

# 验证 Redis 连接
docker-compose exec redis redis-cli ping
# 输出: PONG
```

#### 1.2 数据库迁移

```bash
# 连接 MySQL 执行迁移
mysql -h localhost -u root -p auction << 'EOF'
ALTER TABLE users ADD COLUMN role INT DEFAULT 0 COMMENT '用户角色: 0=普通用户, 1=主播, 2=平台管理员';
ALTER TABLE auctions ADD COLUMN creator_id BIGINT DEFAULT NULL COMMENT '竞拍创建者ID';
CREATE INDEX idx_auctions_creator_id ON auctions(creator_id);
EOF
```

---

### 2. 启动服务

#### 2.1 后端服务

```bash
# 设置环境变量
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=root
export DB_NAME=auction
export REDIS_ADDR=localhost:6379
export JWT_SECRET=your-secret-key

# 启动 auction-service
cd backend/auction
go run main.go

# 启动 product-service (新终端)
cd backend/product
go run main.go

# 启动 gateway-service (新终端)
cd backend/gateway
go run main.go
```

#### 2.2 使用 Docker Compose 启动全部

```bash
docker-compose up -d
```

---

### 3. 测试场景

#### 3.1 测试分布式锁

```bash
# 并发出价测试
for i in {1..10}; do
  curl -X POST http://localhost:8080/api/v1/auctions/1/bids \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer <token>" \
    -d '{"amount": 100}' &
done
wait

# 检查日志确认无数据竞争
docker-compose logs auction | grep "lock acquired"
```

#### 3.2 测试 WebSocket 状态同步

```javascript
// 使用 wscat 测试
wscat -c "ws://localhost:8083/ws?auction_id=1&token=<jwt_token>"

// 发送消息测试
> {"type":"sync_request"}
// 预期收到同步响应
```

#### 3.3 测试用户历史记录

```bash
# 获取用户历史记录
curl http://localhost:8080/api/v1/orders/history \
  -H "Authorization: Bearer <token>"

# 预期返回真实数据，非模拟数据
```

#### 3.4 测试时间同步推送

```javascript
// WebSocket 客户端应每5秒收到时间同步消息
// 消息格式:
{
  "type": "time_sync",
  "timestamp": 1716370800000,
  "data": {
    "server_time": 1716370800000,
    "end_time": 1716374400000
  }
}
```

#### 3.5 测试 RBAC 权限

```bash
# 普通用户尝试创建竞拍 (应返回403)
curl -X POST http://localhost:8080/api/v1/auctions \
  -H "Authorization: Bearer <user_token>" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1, "start_time": "2026-05-22T12:00:00Z", "end_time": "2026-05-22T13:00:00Z"}'

# 主播创建竞拍 (应成功)
curl -X POST http://localhost:8080/api/v1/auctions \
  -H "Authorization: Bearer <streamer_token>" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1, "start_time": "2026-05-22T12:00:00Z", "end_time": "2026-05-22T13:00:00Z"}'
```

---

### 4. 验证清单

| 功能 | 验证方法 | 预期结果 |
|------|---------|---------|
| Redis连接 | `docker-compose exec redis redis-cli ping` | PONG |
| 分布式锁 | 并发出价测试 | 无数据竞争 |
| 状态同步 | WebSocket重连 | 状态恢复 |
| 历史记录 | API查询 | 返回真实数据 |
| 时间同步 | WebSocket监听 | 每5秒收到消息 |
| RBAC权限 | 不同角色API调用 | 权限正确控制 |

---

### 5. 常见问题

#### Q: Redis 连接失败怎么办？

```bash
# 检查 Redis 容器状态
docker-compose ps redis

# 重启 Redis
docker-compose restart redis

# 如果是本地开发，可以降级为本地内存锁
# 代码会自动降级，无需手动配置
```

#### Q: 如何设置用户角色？

```sql
-- 设置用户为主播
UPDATE users SET role = 1 WHERE id = <user_id>;

-- 设置用户为平台管理员
UPDATE users SET role = 2 WHERE id = <user_id>;
```

#### Q: 如何查看时间同步消息？

```javascript
// 前端 WebSocket 监听
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === 'time_sync') {
    console.log('Server time:', msg.data.server_time);
    console.log('Remaining:', msg.data.end_time - msg.data.server_time);
  }
};
```
