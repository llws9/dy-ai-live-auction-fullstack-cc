# API Contracts: 接口契约

**Feature**: `20260523-product-auction-live`
**Date**: 2026-05-23
**Base URL**: `http://localhost:8080/api/v1`

---

## 1. 商品管理 API (Product Service)

### 1.1 发布商品

**Endpoint**: `POST /products/{id}/publish`

**权限**: 商家 (Role=1)

**请求参数**:
```json
{
  "start_time": "2026-05-23T10:00:00Z",  // 可选，竞拍开始时间，默认30分钟后
  "rule_id": 123                          // 可选，竞拍规则ID，不传则使用默认规则
}
```

**响应**:
```json
{
  "code": 200,
  "message": "发布成功",
  "data": {
    "product": {
      "id": 1,
      "name": "稀有珠宝",
      "status": 1,  // 已发布
      "created_at": "2026-05-23T09:00:00Z"
    },
    "auction": {
      "id": 100,
      "product_id": 1,
      "live_stream_id": 10,
      "status": 0,  // 待开始
      "start_time": "2026-05-23T10:00:00Z",
      "end_time": "2026-05-23T10:05:00Z"
    },
    "live_stream": {
      "id": 10,
      "name": "张三的直播间"
    }
  }
}
```

**错误响应**:
```json
{
  "code": 400,
  "message": "商品已发布，无法重复发布"
}
```

**业务规则**:
- 只有草稿状态的商品可以发布
- 自动创建竞拍记录（状态=待开始）
- 自动关联直播间
- 发送通知给关注用户

---

### 1.2 下架商品

**Endpoint**: `POST /products/{id}/unpublish`

**权限**: 商家 (Role=1)

**请求参数**:
```json
{
  "reason": "商品质量问题"  // 可选，下架原因
}
```

**响应**:
```json
{
  "code": 200,
  "message": "下架成功",
  "data": {
    "product": {
      "id": 1,
      "status": 2  // 已下架
    },
    "cancelled_auctions": [  // 取消的竞拍列表
      {
        "id": 100,
        "status": 4  // 已取消
      }
    ],
    "notified_users": 25  // 已通知的用户数
  }
}
```

**业务规则**:
- 只有已发布的商品可以下架
- 取消所有待开始和进行中的竞拍
- 通知已出价的用户
- 记录操作日志

---

## 2. 直播间管理 API (Auction Service)

### 2.1 关注直播间

**Endpoint**: `POST /live-streams/{id}/follow`

**权限**: 所有登录用户

**请求参数**: 无

**响应**:
```json
{
  "code": 200,
  "message": "关注成功",
  "data": {
    "live_stream": {
      "id": 10,
      "name": "张三的直播间",
      "followers_count": 1250
    },
    "follow": {
      "id": 1001,
      "user_id": 5,
      "live_stream_id": 10,
      "notification_enabled": true,
      "created_at": "2026-05-23T09:30:00Z"
    }
  }
}
```

**业务规则**:
- 用户可以关注多个直播间
- 默认开启通知
- 防止重复关注

---

### 2.2 取消关注直播间

**Endpoint**: `DELETE /live-streams/{id}/follow`

**权限**: 所有登录用户

**响应**:
```json
{
  "code": 200,
  "message": "取消关注成功"
}
```

---

### 2.3 获取直播间关注统计

**Endpoint**: `GET /live-streams/{id}/followers/stats`

**权限**: 商家（查看自己的直播间）、管理员

**响应**:
```json
{
  "code": 200,
  "data": {
    "total_count": 1250,
    "new_today": 15,
    "new_this_week": 120,
    "new_this_month": 450,
    "active_last_7_days": 800,
    "active_last_30_days": 1000,
    "participated_count": 650
  }
}
```

**业务规则**:
- 不返回具体用户信息，仅统计数据
- 商家只能查看自己直播间的统计

---

### 2.4 更新通知偏好

**Endpoint**: `PUT /live-streams/{id}/notification-preference`

**权限**: 所有登录用户

**请求参数**:
```json
{
  "notification_enabled": false
}
```

**响应**:
```json
{
  "code": 200,
  "message": "更新成功",
  "data": {
    "notification_enabled": false
  }
}
```

**业务规则**:
- 用户可以关闭特定直播间的通知，但保留关注关系
- 再次开启时恢复通知

---

### 2.5 获取用户关注的直播间列表

**Endpoint**: `GET /user/followed-live-streams`

**权限**: 所有登录用户

**查询参数**:
- `page`: 页码，默认1
- `page_size`: 每页数量，默认20

**响应**:
```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": 10,
        "name": "张三的直播间",
        "cover_image": "https://example.com/cover.jpg",
        "followers_count": 1250,
        "active_auctions": 3,
        "notification_enabled": true,
        "followed_at": "2026-05-23T09:30:00Z"
      }
    ],
    "total": 5,
    "page": 1,
    "page_size": 20
  }
}
```

---

## 3. 竞拍管理 API (Auction Service)

### 3.1 获取竞拍列表（增强）

**Endpoint**: `GET /auctions`

**权限**: 商家、管理员

**新增查询参数**:
- `status`: 竞拍状态筛选（0=待开始, 1=进行中, 2=延时中, 3=已结束）
- `live_stream_id`: 按直播间ID筛选
- `live_stream_name`: 按直播间名称模糊搜索
- `search`: 关键词搜索（直播间名称）

**响应（管理员视角）**:
```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": 100,
        "product_id": 1,
        "product_name": "稀有珠宝",
        "live_stream_id": 10,           // 新增字段
        "live_stream_name": "张三的直播间",  // 新增字段
        "creator_id": 5,
        "creator_name": "张三",
        "status": 1,
        "current_price": 150.00,
        "bid_count": 12,
        "start_time": "2026-05-23T10:00:00Z",
        "end_time": "2026-05-23T10:05:00Z",
        "remaining_time": "3分25秒",
        "winner_name": "用户A"
      }
    ],
    "total": 50,
    "page": 1,
    "page_size": 20
  }
}
```

**响应（商家视角）**:
```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": 100,
        "product_id": 1,
        "product_name": "稀有珠宝",
        // 不包含 live_stream_id 和 live_stream_name 字段
        "status": 1,
        "current_price": 150.00,
        "bid_count": 12,
        "start_time": "2026-05-23T10:00:00Z",
        "end_time": "2026-05-23T10:05:00Z"
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20
  }
}
```

**业务规则**:
- 商家只能看到自己直播间的竞拍
- 管理员可以看到所有竞拍
- 管理员视角包含直播间信息

---

## 4. 通知 API (Auction Service)

### 4.1 批量推送通知（内部API）

**Endpoint**: `POST /internal/notifications/batch-push`

**权限**: 内部服务调用

**请求参数**:
```json
{
  "live_stream_id": 10,
  "type": "new_product",
  "title": "新商品上架",
  "content": "直播间【张三的直播间】发布了新商品【稀有珠宝】，快来参与竞拍吧！",
  "data": {
    "product_id": 1,
    "auction_id": 100
  }
}
```

**响应**:
```json
{
  "code": 200,
  "message": "推送任务已创建",
  "data": {
    "task_id": "task_20260523_123456",
    "target_count": 1250,
    "estimated_duration": "5分钟"
  }
}
```

**业务规则**:
- 使用消息队列异步推送
- 分批次处理（每批1万用户）
- 返回任务ID用于查询推送状态

---

## 5. 错误码定义

| 错误码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 409 | 资源冲突（如重复关注） |
| 500 | 服务器内部错误 |

---

## 6. 权限矩阵

| API端点 | 普通用户 | 商家 (Role=1) | 管理员 (Role=2) |
|---------|---------|--------------|----------------|
| POST /products/{id}/publish | ❌ | ✅ (自己的商品) | ✅ |
| POST /products/{id}/unpublish | ❌ | ✅ (自己的商品) | ✅ |
| POST /live-streams/{id}/follow | ✅ | ✅ | ✅ |
| DELETE /live-streams/{id}/follow | ✅ | ✅ | ✅ |
| GET /live-streams/{id}/followers/stats | ❌ | ✅ (自己的直播间) | ✅ |
| GET /auctions | ❌ | ✅ (自己的直播间) | ✅ |

---

## 7. 性能要求

| API端点 | 响应时间要求 | 备注 |
|---------|-------------|------|
| POST /products/{id}/publish | < 500ms | 不包含推送时间 |
| POST /products/{id}/unpublish | < 1s | 包含取消竞拍和发送通知 |
| GET /auctions | < 1s | 包含搜索和筛选 |
| POST /internal/notifications/batch-push | < 100ms | 仅创建推送任务 |
