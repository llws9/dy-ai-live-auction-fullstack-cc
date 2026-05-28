# Observability 平台

基于 Grafana 生态的轻量级可观测性平台，包含日志收集、指标监控和可视化功能。

## 🏗️ 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        可观测性平台                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Gateway    │  │   Product    │  │   Auction    │          │
│  │   Service    │  │   Service    │  │   Service    │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                 │                 │                  │
│         │ 日志 (stdout)   │ 指标 (/metrics) │ 埋点 API        │
│         └─────────────────┼─────────────────┘                  │
│                           │                                    │
│         ┌─────────────────┼─────────────────┐                  │
│         ▼                 ▼                 ▼                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │  Promtail    │  │  Prometheus  │  │  Track API   │          │
│  │  日志采集    │  │  指标收集    │  │  埋点接收    │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                 │                 │                  │
│         ▼                 ▼                 │                  │
│  ┌──────────────┐  ┌──────────────┐         │                  │
│  │    Loki      │  │ Prometheus   │◄────────┘                  │
│  │  日志存储    │  │  指标存储    │                            │
│  └──────┬───────┘  └──────┬───────┘                            │
│         │                 │                                    │
│         └─────────┬───────┘                                    │
│                   ▼                                            │
│           ┌──────────────┐                                     │
│           │   Grafana    │                                     │
│           │   可视化     │                                     │
│           └──────────────┘                                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 📦 组件

| 组件 | 版本 | 端口 | 说明 |
|------|------|------|------|
| Grafana | 10.4.2 | 3002 | 统一可视化界面 |
| Prometheus | 2.52.0 | 9090 | 指标收集与存储 |
| Loki | 2.9.4 | 3100 | 日志聚合存储 |
| Promtail | 2.9.4 | - | 日志采集代理 |

## 🚀 快速开始

### 启动所有服务

```bash
# 方式一：使用启动脚本
./start.sh start

# 方式二：使用 docker-compose
docker compose up -d
```

### 访问界面

| 服务 | URL | 账号 |
|------|-----|------|
| Grafana | http://localhost:3002 | admin/admin |
| Prometheus | http://localhost:9090 | - |
| Loki | http://localhost:3100 | - |

## 📊 指标监控

### 已定义的业务指标

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `live_room_enter_total` | Counter | 直播间进入次数 |
| `live_room_current_viewers` | Gauge | 当前观看人数 |
| `auction_bid_total` | Counter | 出价次数 |
| `auction_completed_total` | Counter | 竞拍完成次数 |
| `order_completed_total` | Counter | 订单成交次数 |
| `order_amount` | Histogram | 订单金额分布 |
| `payment_completed_total` | Counter | 支付完成次数 |
| `user_register_total` | Counter | 用户注册数 |
| `websocket_connections` | Gauge | WebSocket连接数 |

### PromQL 查询示例

```promql
# 1小时内的直播间进入次数
sum(increase(live_room_enter_total[1h]))

# 当前 WebSocket 连接数
websocket_connections

# 请求成功率
sum(rate(http_requests_total{status=~"2.."}[5m]))
  / sum(rate(http_requests_total[5m]))

# P95 延迟
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# 成交金额（1小时）
sum(increase(order_amount_sum[1h]))
```

## 🎯 前端埋点

### 初始化

```typescript
import { initTracking } from '@/shared/tracking';

// 在应用入口初始化
initTracking({
  endpoint: '/api/track',  // 埋点 API 地址
  debug: true,             // 开发环境开启调试
  batchSize: 10,           // 批量发送大小
  flushInterval: 5000,     // 发送间隔（毫秒）
});
```

### 使用示例

```typescript
import { getTracker } from '@/shared/tracking';

// 直播间进入
getTracker().trackLiveRoomEnter('room-123', 'vip');

// 出价点击
getTracker().trackBidClick('auction-456', 999);

// 支付发起
getTracker().trackPaymentStart('order-789', 999, 'alipay');

// 自定义事件
getTracker().trackCustom('button_click', {
  button_name: 'submit_bid',
  page: 'auction_detail',
});
```

### 支持的事件类型

| 事件类型 | 方法 | 参数 |
|----------|------|------|
| 直播间进入 | `trackLiveRoomEnter(roomId, userType)` | room_id, user_type |
| 直播间离开 | `trackLiveRoomLeave(roomId, duration)` | room_id, duration_seconds |
| 竞拍浏览 | `trackAuctionView(auctionId, productId)` | auction_id, product_id |
| 出价点击 | `trackBidClick(auctionId, currentPrice)` | auction_id, current_price |
| 支付发起 | `trackPaymentStart(orderId, amount, method)` | order_id, amount, method |
| 用户注册 | `trackUserRegister(source)` | source |
| 用户登录 | `trackUserLogin(method)` | method |
| 页面浏览 | `trackPageView(pageName)` | page, url, referrer |
| 自定义 | `trackCustom(eventName, params)` | 自定义参数 |

## 📝 日志查询

### LogQL 示例

```logql
# 按 request_id 查询全链路日志
{service_name=~".+"} |= "your-request-id"

# 查看错误日志
{service_name=~".+", success="false"}

# 统计请求量
sum by (service_name) (count_over_time({service_name=~".+"}[1h]))
```

## 🔧 配置文件

```
observability/
├── docker-compose.yaml       # Docker Compose 配置
├── start.sh                  # 启动脚本
├── prometheus/
│   └── prometheus.yml        # Prometheus 配置
├── loki/
│   └── loki-config.yaml      # Loki 配置
├── promtail/
│   └── promtail-config.yaml  # Promtail 配置
└── grafana/
    └── provisioning/
        ├── datasources/      # 数据源配置
        └── dashboards/       # 仪表板配置
            ├── microservices-logs.json   # 日志仪表板
            └── business-metrics.json     # 业务监控仪表板
```

## 🛠️ 管理命令

```bash
./start.sh start     # 启动服务
./start.sh stop      # 停止服务
./start.sh restart   # 重启服务
./start.sh logs      # 查看日志
./start.sh status    # 查看状态
./start.sh clean     # 清理所有数据
```

## 📈 预置仪表板

### 业务监控仪表板

包含以下面板：
- 直播间进入次数、当前观看人数
- 成交次数、成交金额
- 请求速率 (QPS)
- 请求延迟 P95
- 出价趋势、成交趋势
- 支付统计、用户统计

### 微服务日志仪表板

包含以下面板：
- 实时日志流
- 请求统计
- 请求趋势图
- 操作类型分布
- 错误日志

## 🐛 故障排查

### 指标未显示

```bash
# 检查 Prometheus 是否正常
curl http://localhost:9090/-/healthy

# 检查服务 metrics 端点
curl http://localhost:8080/metrics
```

### 日志未采集

```bash
# 检查 Promtail 日志
docker logs promtail

# 检查 Loki 状态
curl http://localhost:3100/ready
```

## 📚 相关文档

- [日志实现文档](../backend/LOGGING_IMPLEMENTATION.md)
- [启动指南](../STARTUP_GUIDE.md)
