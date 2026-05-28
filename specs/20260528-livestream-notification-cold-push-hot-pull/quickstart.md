# Quickstart: 直播间通知"冷推热拉"

---

## 前置条件

1. Redis服务正常运行
2. 数据库连接正常
3. 用户认证系统正常工作

---

## 快速测试流程

### 1. 创建测试直播间

```bash
# 创建冷门直播间（关注数 < 200）
curl -X POST http://localhost:8080/api/v1/live-streams \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "冷门直播间测试",
    "creator_id": 1,
    "scheduled_start_time": "2026-05-28T11:00:00Z"
  }'
```

### 2. 用户关注直播间

```bash
# 用户关注直播间
curl -X POST http://localhost:8080/api/v1/live-streams/1/follow \
  -H "Authorization: Bearer <user_token>"
```

验证 Redis:
```bash
redis-cli
> SMEMBERS user:1:followed_live_streams
> ZRANGE live_stream:cold:start_time 0 -1 WITHSCORES
> HGETALL live_stream:1:stats
```

### 3. 触发冷推任务

```bash
# 手动触发冷推（模拟定时任务）
curl -X POST http://localhost:8080/api/internal/cold-push/trigger \
  -H "Authorization: Bearer <internal_token>"
```

预期结果:
- 关注用户收到"直播即将开始"通知（蓝色）
- Redis ZSET中该直播间被移除

### 4. 测试热拉接口

```bash
# 用户触发热拉
curl -X POST http://localhost:8080/api/v1/notifications/hot-pull \
  -H "Authorization: Bearer <user_token>" \
  -H "Content-Type: application/json" \
  -d '{}'
```

预期结果:
- 返回用户关注的热门直播间通知
- `unread_count_delta` 正确更新

### 5. 商品提醒订阅

```bash
# 订阅商品提醒
curl -X POST http://localhost:8080/api/v1/products/1/remind \
  -H "Authorization: Bearer <user_token>" \
  -H "Content-Type: application/json" \
  -d '{"remind_type": "auction_start"}'
```

验证 Redis:
```bash
redis-cli
> ZRANGE user:1:product_reminders:start_time 0 -1 WITHSCORES
```

---

## 前端测试

### H5热拉触发测试

1. 登录H5应用
2. 打开浏览器开发者工具 → Console
3. 切换到其他标签页（模拟后台）
4. 切换回H5标签页
5. 观察Console日志：`hotPullNotifications` 调用

### 通知颜色验证

打开通知列表，验证不同类型通知显示正确颜色：

| 通知类型 | 预期颜色 |
|---------|---------|
| 正在直播 | 红色 |
| 竞拍即将开始 | 红色 |
| 直播即将开始 | 蓝色 |
| 竞拍成功 | 绿色 |
| 出价被超越 | 橙色 |
| 订单状态 | 棕色 |
| 竞拍未中标 | 灰色 |

---

## 热度变更测试

### 冷门→热门切换

```bash
# 增加关注人数到200+
for i in {1..200}; do
  curl -X POST http://localhost:8080/api/v1/live-streams/1/follow \
    -H "Authorization: Bearer <user_${i}_token>"
done
```

验证 Redis:
```bash
redis-cli
> ZRANGE live_stream:hot:start_time 0 -1 WITHSCORES
# 应看到直播间1已迁移到hot ZSET
> HGET live_stream:1:stats is_hot
# 应返回 "1"
```

---

## 监控验证

### Prometheus指标

```bash
# 查询冷推任务耗时
curl http://localhost:9090/api/v1/query?query=cold_push_latency_seconds

# 查询热拉接口耗时
curl http://localhost:9090/api/v1/query?query=hot_pull_latency_seconds

# 查询ZSET大小
curl http://localhost:9090/api/v1/query?query=zset_size
```

---

## 常见问题排查

### 问题：热拉返回空通知

**排查步骤**:
1. 检查用户是否关注了热门直播间
2. 检查 Redis `live_stream:hot:start_time` 是否有数据
3. 检查时间范围是否正确（now ~ now+1hour）

### 问题：冷推任务未执行

**排查步骤**:
1. 检查定时任务是否启动
2. 检查 Redis `live_stream:cold:start_time` 是否有数据
3. 检查时间范围是否正确（now ~ now+10min）

### 问题：热度状态不一致

**排查步骤**:
1. 比对 DB 关注数与 Redis follower_count
2. 检查 ZSET 中直播间所属集合
3. 手动触发状态同步