# 日志输出示例

## 1. 竞拍创建日志

```json
{
  "timestamp": "2026-05-23T02:50:00.123456789Z",
  "service_name": "auction-service",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "operation_type": "create",
  "object_type": "auction",
  "object_id": "123",
  "success": true,
  "duration": "25ms",
  "request_data": {
    "product_id": 789,
    "start_time": "2026-05-23T10:00:00Z",
    "end_time": "2026-05-23T12:00:00Z"
  },
  "response_data": {
    "id": 123,
    "product_id": 789,
    "status": "pending",
    "current_price": 0,
    "start_time": "2026-05-23T10:00:00Z",
    "end_time": "2026-05-23T12:00:00Z"
  }
}
```

## 2. 出价操作日志

```json
{
  "timestamp": "2026-05-23T11:30:15.987654321Z",
  "service_name": "auction-service",
  "request_id": "660e8400-e29b-41d4-a716-446655440001",
  "operation_type": "bid",
  "object_type": "bid",
  "object_id": "456",
  "user_id": "789",
  "user_name": "张三",
  "success": true,
  "duration": "150ms",
  "request_data": {
    "auction_id": 123,
    "user_id": 789,
    "amount": 150.50,
    "rank": 1
  },
  "response_data": {
    "success": true,
    "message": "出价成功",
    "current_price": 150.50,
    "rank": 1,
    "winner_id": 789
  }
}
```

## 3. 商品创建日志

```json
{
  "timestamp": "2026-05-23T09:00:00.123456789Z",
  "service_name": "product-service",
  "request_id": "770e8400-e29b-41d4-a716-446655440002",
  "operation_type": "create",
  "object_type": "product",
  "object_id": "999",
  "success": true,
  "duration": "18ms",
  "request_data": {
    "name": "iPhone 15 Pro",
    "description": "最新款苹果手机",
    "images": [
      "https://example.com/image1.jpg",
      "https://example.com/image2.jpg"
    ]
  },
  "response_data": {
    "id": 999,
    "name": "iPhone 15 Pro",
    "description": "最新款苹果手机",
    "status": "draft",
    "created_at": "2026-05-23T09:00:00Z"
  }
}
```

## 4. 订单支付日志

```json
{
  "timestamp": "2026-05-23T14:30:45.123456789Z",
  "service_name": "product-service",
  "request_id": "880e8400-e29b-41d4-a716-446655440003",
  "operation_type": "pay",
  "object_type": "order",
  "object_id": "567",
  "success": true,
  "duration": "320ms",
  "request_data": {
    "order_id": 567,
    "auction_id": 123,
    "product_id": 999,
    "winner_id": 789,
    "final_price": 150.50
  }
}
```

## 5. Gateway请求日志

```json
{
  "timestamp": "2026-05-23T11:30:15.123456789Z",
  "service_name": "gateway-service",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "operation_type": "create",
  "method": "POST",
  "path": "/api/v1/auctions",
  "status": 201,
  "latency": "28ms",
  "client_ip": "192.168.1.100",
  "success": true,
  "user_id": "789",
  "user_name": "张三"
}
```

## 6. 错误日志示例

### 出价失败日志
```json
{
  "timestamp": "2026-05-23T11:35:20.123456789Z",
  "service_name": "auction-service",
  "request_id": "990e8400-e29b-41d4-a716-446655440004",
  "operation_type": "bid",
  "object_type": "auction",
  "object_id": "123",
  "user_id": "888",
  "success": false,
  "error_msg": "出价金额不足，最低出价为 160.00 元",
  "request_data": {
    "auction_id": 123,
    "user_id": 888,
    "attempted_amount": 155.00,
    "min_required": 160.00
  }
}
```

### 订单状态错误日志
```json
{
  "timestamp": "2026-05-23T14:35:50.123456789Z",
  "service_name": "product-service",
  "request_id": "aa0e8400-e29b-41d4-a716-446655440005",
  "operation_type": "pay",
  "object_type": "order",
  "object_id": "567",
  "success": false,
  "error_msg": "订单状态不允许支付",
  "request_data": {
    "order_id": 567,
    "current_status": "paid"
  }
}
```

## 7. 敏感信息脱敏示例

### 用户注册日志（密码已脱敏）
```json
{
  "timestamp": "2026-05-23T08:45:30.123456789Z",
  "service_name": "auction-service",
  "request_id": "bb0e8400-e29b-41d4-a716-446655440006",
  "operation_type": "create",
  "object_type": "user",
  "success": true,
  "duration": "45ms",
  "request_data": {
    "username": "zhangsan",
    "email": "zhangsan@example.com",
    "password": "***"
  }
}
```

## 日志查询示例

### 查询特定用户的所有操作
```bash
# 使用jq查询用户ID为789的所有操作
cat app.log | jq 'select(.user_id == "789")'
```

### 查询所有失败的出价操作
```bash
# 查询所有失败的出价操作
cat app.log | jq 'select(.operation_type == "bid" and .success == false)'
```

### 查询响应时间超过100ms的操作
```bash
# 查询慢操作
cat app.log | jq 'select(.duration_ms > 100)'
```

### 查询特定竞拍的所有操作
```bash
# 查询竞拍ID为123的所有操作
cat app.log | jq 'select(.object_id == "123" and .object_type == "auction")'
```

### 统计各操作类型的数量
```bash
# 统计操作类型分布
cat app.log | jq -r '.operation_type' | sort | uniq -c
```

## 监控和告警建议

### 1. 错误率监控
- 监控 `success: false` 的日志比例
- 设置错误率阈值告警（如 > 5%）

### 2. 性能监控
- 监控 `duration_ms` 字段
- 设置响应时间阈值告警（如 > 1000ms）

### 3. 操作频率监控
- 监控特定操作的频率
- 检测异常操作模式

### 4. 业务指标监控
- 出价成功率
- 订单完成率
- 商品创建数量
- 竞拍活跃度
