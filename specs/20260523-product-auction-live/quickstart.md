# Quick Start: 开发环境搭建指南

**Feature**: `20260523-product-auction-live`
**Date**: 2026-05-23

## 前置条件

- Go 1.21+
- Node.js 18+
- MySQL 8.0+
- Redis 7+
- RabbitMQ 3.12+ (支持延迟队列插件)

## 1. 数据库准备

### 1.1 创建数据库

```sql
CREATE DATABASE IF NOT EXISTS live_auction DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 1.2 执行迁移脚本

迁移脚本位于 `scripts/migrations/` 目录：

```bash
# 执行基础迁移
mysql -u root -p live_auction < scripts/migrations/001_init_schema.sql
mysql -u root -p live_auction < scripts/migrations/002_add_auth_fields.sql

# 执行新增迁移（直播间功能）
mysql -u root -p live_auction < scripts/migrations/003_add_live_stream.sql
```

### 1.3 迁移脚本内容

`003_add_live_stream.sql`:

```sql
-- 创建直播间表
CREATE TABLE IF NOT EXISTS live_streams (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    creator_id BIGINT NOT NULL UNIQUE COMMENT '商家ID',
    name VARCHAR(128) NOT NULL COMMENT '直播间名称',
    description TEXT COMMENT '直播间描述',
    cover_image VARCHAR(256) COMMENT '封面图',
    status TINYINT DEFAULT 1 COMMENT '状态：0=禁用，1=正常',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='直播间表';

-- 创建用户关注直播间表
CREATE TABLE IF NOT EXISTS user_live_stream_follows (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    live_stream_id BIGINT NOT NULL COMMENT '直播间ID',
    notification_enabled TINYINT DEFAULT 1 COMMENT '是否接收通知',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '关注时间',
    UNIQUE KEY uk_user_live_stream (user_id, live_stream_id),
    INDEX idx_user_id (user_id),
    INDEX idx_live_stream_id (live_stream_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户关注直播间表';

-- 为auctions表新增live_stream_id字段
ALTER TABLE auctions
ADD COLUMN live_stream_id BIGINT NULL COMMENT '直播间ID' AFTER product_id,
ADD INDEX idx_live_stream_id (live_stream_id);

-- 为products表更新状态注释
ALTER TABLE products
MODIFY COLUMN status TINYINT DEFAULT 0 COMMENT '状态: 0=草稿, 1=已发布, 2=已下架';

-- 为现有商家创建直播间
INSERT INTO live_streams (creator_id, name, description, status, created_at)
SELECT
    id as creator_id,
    CONCAT(name, '的直播间') as name,
    CONCAT(name, '的个人直播间') as description,
    1 as status,
    created_at
FROM users
WHERE role = 1 -- 主播/商家
ON DUPLICATE KEY UPDATE
    name = VALUES(name);

-- 为现有竞拍记录设置live_stream_id
UPDATE auctions a
JOIN live_streams ls ON a.creator_id = ls.creator_id
SET a.live_stream_id = ls.id
WHERE a.live_stream_id IS NULL;
```

## 2. 后端服务启动

### 2.1 配置环境变量

在 `backend/` 目录下创建 `.env` 文件：

```env
# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=live_auction

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# RabbitMQ
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
RABBITMQ_VHOST=/

# JWT
JWT_SECRET=your_jwt_secret_key
JWT_EXPIRE=86400

# Service Ports
GATEWAY_PORT=8080
PRODUCT_SERVICE_PORT=8081
AUCTION_SERVICE_PORT=8082
```

### 2.2 启动 RabbitMQ

**安装延迟队列插件**（必需）：

```bash
# macOS (使用 Homebrew)
brew install rabbitmq
brew services start rabbitmq

# 启用延迟队列插件
rabbitmq-plugins enable rabbitmq_delayed_message_exchange

# 访问管理界面: http://localhost:15672
# 默认账号: guest / guest
```

**Docker 方式**（推荐）：

```bash
# 启动 RabbitMQ + 延迟队列插件
docker run -d --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  rabbitmq:3.12-management

# 进入容器启用插件
docker exec rabbitmq rabbitmq-plugins enable rabbitmq_delayed_message_exchange
```

**验证插件安装**：

```bash
# 检查插件列表
rabbitmq-plugins list | grep delayed

# 应该看到:
# [E*] rabbitmq_delayed_message_exchange
```

### 2.3 启动后端服务

```bash
# 启动 Product Service
cd backend/product
go run main.go

# 启动 Auction Service (新终端)
cd backend/auction
go run main.go

# 启动 Gateway (新终端)
cd backend/gateway
go run main.go
```

## 3. 前端启动

### 3.1 安装依赖

```bash
# 管理端
cd frontend/admin
npm install

# 用户端 (H5)
cd frontend/h5
npm install
```

### 3.2 启动开发服务器

```bash
# 管理端
cd frontend/admin
npm run dev

# 用户端 (新终端)
cd frontend/h5
npm run dev
```

## 4. API 测试示例

### 4.1 用户登录

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }'
```

响应：
```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 999,
      "name": "系统管理员",
      "email": "admin@example.com",
      "role": 2
    }
  }
}
```

### 4.2 发布商品

```bash
curl -X POST http://localhost:8080/api/v1/products/1/publish \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "start_time": "2026-05-23T15:00:00Z",
    "rule_id": 123
  }'
```

### 4.3 关注直播间

```bash
curl -X POST http://localhost:8080/api/v1/live-streams/10/follow \
  -H "Authorization: Bearer <token>"
```

### 4.4 获取直播间关注统计

```bash
curl -X GET http://localhost:8080/api/v1/live-streams/10/followers/stats \
  -H "Authorization: Bearer <token>"
```

## 5. 关键配置说明

### 5.1 角色定义

| 角色常量 | 值 | 说明 |
|---------|---|------|
| RoleUser | 0 | 普通用户 |
| RoleStreamer | 1 | 商家/主播 |
| RoleAdmin | 2 | 管理员 |

**代码位置**:
- `backend/product/model/user.go`
- `backend/auction/model/user.go`

### 5.2 商品状态

| 状态 | 值 | 说明 |
|------|---|------|
| Draft | 0 | 草稿 |
| Published | 1 | 已发布 |
| Unpublished | 2 | 已下架 |

### 5.3 竞拍状态

| 状态 | 值 | 说明 |
|------|---|------|
| Pending | 0 | 待开始 |
| Ongoing | 1 | 进行中 |
| Delayed | 2 | 延时中 |
| Ended | 3 | 已结束 |
| Cancelled | 4 | 已取消 |

### 5.4 通知配置

```go
// 通知类型
const (
    NotificationTypeNewProduct        = "new_product"         // 新商品发布
    NotificationTypeAuctionStarting   = "auction_starting"    // 竞拍即将开始
    NotificationTypeAuctionEnded      = "auction_ended"       // 竞拍结束
    NotificationTypeProductUnpublished = "product_unpublished" // 商品下架
)

// 批量推送配置
const (
    BatchSize         = 10000   // 每批1万用户
    BatchInterval     = 3       // 批次间隔3秒
    MaxPushDuration   = 600     // 最大推送时长10分钟
    AdvanceNotifyTime = 30      // 提前30分钟通知
)
```

## 6. 常见问题

### Q1: 管理员登录失败

**问题**: 提示"非管理员账号，无法登录"

**解决**: 检查数据库中用户的 `role` 字段是否为 `2`，密码哈希是否正确。

```sql
SELECT id, name, email, role FROM users WHERE email = 'admin@example.com';
```

### Q2: API 请求返回 401

**问题**: 请求返回"未授权"

**解决**: 检查 JWT token 是否正确传递，是否过期。

```bash
# 解码 JWT token
echo "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." | base64 -d
```

### Q3: 商品发布失败

**问题**: 发布商品时返回错误

**解决**:
1. 检查商品状态是否为"草稿"（status=0）
2. 检查商家是否有对应的直播间
3. 检查竞拍规则是否配置

```sql
-- 检查商品状态
SELECT id, name, status, creator_id FROM products WHERE id = 1;

-- 检查直播间
SELECT * FROM live_streams WHERE creator_id = <商家ID>;

-- 检查竞拍规则
SELECT * FROM auction_rules WHERE product_id = 1;
```

### Q4: 通知推送未触发

**问题**: 关注直播间后未收到通知

**解决**:
1. 检查 Redis 是否正常运行
2. 检查 `user_live_stream_follows` 表中的 `notification_enabled` 字段
3. 检查消息队列消费者是否启动

```bash
# 检查 Redis 连接
redis-cli ping

# 查看关注记录
SELECT * FROM user_live_stream_follows WHERE live_stream_id = 10;
```

## 7. 开发注意事项

### 7.1 权限验证

所有新增 API 端点必须添加权限中间件：

```go
// 商家权限验证
func RequireMerchant() app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        userRole := c.GetInt("user_role")
        if userRole != 1 && userRole != 2 {
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

// 管理员权限验证
func RequireAdmin() app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        userRole := c.GetInt("user_role")
        if userRole != 2 {
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

### 7.2 数据隔离

商家查询竞拍列表时，必须过滤数据：

```go
func (h *AuctionHandler) List(ctx context.Context, c *app.RequestContext) {
    userID := c.GetInt64("user_id")
    userRole := c.GetInt("user_role")

    if userRole == 1 { // 商家
        // 只查询该商家的直播间下的竞拍
        auctions = h.auctionDAO.GetByCreatorID(ctx, userID)
    } else if userRole == 2 { // 管理员
        // 查询所有竞拍
        auctions = h.auctionDAO.List(ctx, filters)
    }
}
```

### 7.3 批量推送性能

实现批量推送时，注意以下约束：

```go
func (s *NotificationService) BatchPush(liveStreamID int64, notification *Notification) error {
    // 1. 获取关注用户总数
    totalUsers := s.followDAO.CountByLiveStream(liveStreamID)

    // 2. 分批推送
    batchSize := 10000
    batches := (totalUsers + batchSize - 1) / batchSize

    for i := 0; i < batches; i++ {
        // 3. 获取当前批次用户
        offset := i * batchSize
        users := s.followDAO.GetFollowers(liveStreamID, offset, batchSize)

        // 4. 推送到消息队列
        for _, user := range users {
            s.pushToQueue(user.ID, notification)
        }

        // 5. 批次间隔（避免系统过载）
        if i < batches-1 {
            time.Sleep(3 * time.Second)
        }
    }

    return nil
}
```

## 8. 下一步

完成环境搭建后，请参考以下文档：

- **详细设计**: [data-model.md](./data-model.md)
- **API契约**: [contracts/api-contracts.md](./contracts/api-contracts.md)
- **技术决策**: [research.md](./research.md)
- **实施计划**: [plan.md](./plan.md)
