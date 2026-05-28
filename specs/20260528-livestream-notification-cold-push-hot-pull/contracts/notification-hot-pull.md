# API Contracts: 直播间通知"冷推热拉"

---

## Hot Pull API

### POST /api/v1/notifications/hot-pull

用户主动拉取热门直播间通知。

**Request**:

```json
{
  "last_pull_time": "2026-05-28T09:00:00Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| last_pull_time | string | 否 | 上次拉取时间（ISO 8601） |

**Response** (200 OK):

```json
{
  "success": true,
  "notifications": [
    {
      "id": 123,
      "type": "live_starting",
      "title": "热门直播间即将开播",
      "content": "您关注的直播间「XXX」将在10分钟后开播",
      "color": "blue",
      "data": {
        "live_stream_id": 456,
        "product_ids": [1, 2, 3]
      },
      "created_at": "2026-05-28T10:00:00Z",
      "read_at": null
    },
    {
      "id": 124,
      "type": "live_now",
      "title": "直播已开始",
      "content": "您关注的直播间「XXX」正在直播",
      "color": "red",
      "data": {
        "live_stream_id": 456
      },
      "created_at": "2026-05-28T10:10:00Z",
      "read_at": null
    }
  ],
  "unread_count_delta": 3,
  "next_pull_after": "2026-05-28T10:30:00Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| notifications | array | 通知列表 |
| unread_count_delta | int | 未读数增量 |
| next_pull_after | string | 建议下次拉取时间 |

**Errors**:

| 状态码 | 说明 |
|--------|------|
| 401 | 未认证，需要 JWT |
| 429 | 调用过于频繁（限流） |

---

## Product Reminder API

### POST /api/v1/products/{product_id}/remind

订阅商品竞拍提醒。

**Path Parameters**:

| 参数 | 类型 | 说明 |
|------|------|------|
| product_id | int64 | 商品ID |

**Request**:

```json
{
  "remind_type": "auction_start"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| remind_type | string | 是 | 提醒类型：auction_start |

**Response** (200 OK):

```json
{
  "success": true,
  "reminder_id": 789,
  "message": "已设置提醒，竞拍开始前10分钟将收到通知"
}
```

**Errors**:

| 状态码 | 说明 |
|--------|------|
| 400 | 商品不存在或竞拍已开始 |
| 401 | 未认证 |
| 409 | 已订阅该商品 |

---

### DELETE /api/v1/products/{product_id}/remind

取消商品竞拍提醒。

**Path Parameters**:

| 参数 | 类型 | 说明 |
|------|------|------|
| product_id | int64 | 商品ID |

**Response** (200 OK):

```json
{
  "success": true,
  "message": "已取消提醒"
}
```

**Errors**:

| 状态码 | 说明 |
|--------|------|
| 401 | 未认证 |
| 404 | 未订阅该商品 |

---

## Live Stream Stats API (Internal)

### GET /api/internal/live-streams/{live_stream_id}/stats

获取直播间热度统计（内部接口）。

**Path Parameters**:

| 参数 | 类型 | 说明 |
|------|------|------|
| live_stream_id | int64 | 直播间ID |

**Response** (200 OK):

```json
{
  "follower_count": 350,
  "is_hot": true,
  "status": "offline",
  "scheduled_start_time": "2026-05-28T11:00:00Z"
}
```

---

## Notification Color Update

### PUT /api/v1/notifications/{id}

更新通知状态（现有接口，增加color返回）。

**Response** 增加字段:

```json
{
  "id": 123,
  "type": "live_starting",
  "color": "blue",
  ...
}
```

---

## Rate Limiting

### 热拉接口限流

| 参数 | 值 |
|------|------|
| 限流策略 | 每用户每分钟最多2次 |
| 实现方式 | Redis计数器 + 滑动窗口 |

```go
// 限流逻辑
key := fmt.Sprintf("hot_pull:ratelimit:%d", userID)
count := redis.Incr(key)
if count == 1 {
    redis.Expire(key, 60*time.Second)
}
if count > 2 {
    return errors.New("调用过于频繁")
}
```

---

## Frontend Integration

### 热拉触发时机

```typescript
// 1. 登录成功后
onLoginSuccess(() => {
  hotPullNotifications();
});

// 2. 页面可见性变化
document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'visible') {
    // 检查最小间隔30秒
    const lastPull = localStorage.getItem('last_hot_pull_time');
    if (lastPull && Date.now() - new Date(lastPull).getTime() < 30000) {
      return;
    }
    hotPullNotifications();
  }
});
```

### API调用示例

```typescript
async function hotPullNotifications() {
  const response = await fetch('/api/v1/notifications/hot-pull', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      last_pull_time: localStorage.getItem('last_hot_pull_time'),
    }),
  });

  const data = await response.json();
  
  // 更新本地状态
  setNotifications([...data.notifications, ...notifications]);
  setUnreadCount(unreadCount + data.unread_count_delta);
  
  // 记录拉取时间
  localStorage.setItem('last_hot_pull_time', new Date().toISOString());
}
```