# 直播竞拍系统监控配置

## 概述

本监控系统基于 Prometheus + Grafana 架构,提供全方位的系统、应用、业务和数据库监控。

## 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        监控系统架构                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐  │
│  │  应用层   │    │  数据层   │    │  系统层   │    │  业务层   │  │
│  └─────┬────┘    └─────┬────┘    └─────┬────┘    └─────┬────┘  │
│        │               │               │               │         │
│  ┌─────┴────┐    ┌─────┴────┐    ┌─────┴────┐    ┌─────┴────┐  │
│  │ Backend  │    │PostgreSQL│    │   Node   │    │ Auction  │  │
│  │WebSocket │    │  Redis   │    │ Exporter │    │ Metrics  │  │
│  └─────┬────┘    └─────┬────┘    └─────┬────┘    └─────┬────┘  │
│        │               │               │               │         │
│        └───────────────┴───────────────┴───────────────┘         │
│                              │                                    │
│                       ┌──────┴──────┐                            │
│                       │ Prometheus  │                            │
│                       │   (TSDB)    │                            │
│                       └──────┬──────┘                            │
│                              │                                    │
│              ┌───────────────┼───────────────┐                    │
│              │               │               │                    │
│       ┌──────┴──────┐ ┌─────┴──────┐ ┌─────┴──────┐             │
│       │   Grafana   │ │Alertmanager│ │   Loki     │             │
│       │ (可视化)    │ │  (告警)    │ │ (日志)     │             │
│       └─────────────┘ └────────────┘ └────────────┘             │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## 组件说明

### 1. Prometheus
- **版本**: v2.45.0
- **端口**: 9090
- **功能**: 指标采集、存储、查询
- **配置**: `prometheus/prometheus.yml`

### 2. Grafana
- **版本**: v10.0.0
- **端口**: 3000
- **默认账号**: admin / admin123
- **功能**: 数据可视化、仪表板展示
- **仪表板**:
  - 系统总览 (`system-overview.json`)
  - 竞拍指标 (`auction-metrics.json`)
  - API性能 (`api-performance.json`)

### 3. Alertmanager
- **版本**: v0.25.0
- **端口**: 9093
- **功能**: 告警路由、去重、通知
- **配置**: `alertmanager/alertmanager.yml`

### 4. Exporters

#### Node Exporter
- **端口**: 9100
- **功能**: 系统指标(CPU、内存、磁盘、网络)

#### cAdvisor
- **端口**: 8080
- **功能**: 容器指标(CPU、内存、网络、文件系统)

#### PostgreSQL Exporter
- **端口**: 9187
- **功能**: PostgreSQL数据库指标

#### Redis Exporter
- **端口**: 9121
- **功能**: Redis缓存指标

#### Blackbox Exporter
- **端口**: 9115
- **功能**: HTTP/TCP探测、服务可用性

### 5. 日志系统

#### Loki
- **端口**: 3100
- **功能**: 日志聚合、查询

#### Promtail
- **功能**: 日志采集、发送到Loki

### 6. 分布式追踪

#### Jaeger
- **端口**: 16686 (UI), 14268 (HTTP)
- **功能**: 分布式追踪、调用链分析

## 快速开始

### 1. 启动监控系统

```bash
cd monitoring
docker-compose up -d
```

### 2. 访问服务

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin123)
- **Alertmanager**: http://localhost:9093
- **Jaeger**: http://localhost:16686

### 3. 查看仪表板

登录Grafana后,进入 `Live Auction` 文件夹,查看预置仪表板:
- 系统总览
- 竞拍业务指标
- API性能监控

## 监控指标

### 系统指标

| 指标名称 | 描述 | 告警阈值 |
|---------|------|---------|
| CPU使用率 | 实时CPU使用百分比 | >80% (warning), >95% (critical) |
| 内存使用率 | 实时内存使用百分比 | >85% (warning), >95% (critical) |
| 磁盘使用率 | 磁盘使用百分比 | >85% (warning), >95% (critical) |
| 网络流量 | 入站/出站网络流量 | - |
| 磁盘IO | 磁盘读写速度 | IO等待 >20% (warning) |

### 应用指标

| 指标名称 | 描述 | 告警阈值 |
|---------|------|---------|
| 请求QPS | 每秒请求数 | <1 req/s (warning), >1000 req/s (warning) |
| 响应时间P99 | 99%请求的响应时间 | >500ms (warning), >2s (critical) |
| 错误率 | 5xx错误比例 | >1% (critical) |
| 并发连接数 | 当前TCP连接数 | >1000 (warning) |

### 业务指标

| 指标名称 | 描述 | 告警阈值 |
|---------|------|---------|
| 在线用户数 | 当前在线用户总数 | <10 (info) |
| 活跃竞拍数 | 正在进行的竞拍数量 | =0 持续30分钟 (info) |
| 出价频率 | 每秒出价次数 | <0.1 bids/s (warning) |
| 订单转化率 | 出价转订单比例 | <5% (warning) |

### 数据库指标

| 指标名称 | 描述 | 告警阈值 |
|---------|------|---------|
| PostgreSQL连接数 | 数据库连接使用率 | >80% (warning) |
| 慢查询 | 平均查询时间 | >1s (warning) |
| 复制延迟 | 主从复制延迟 | >10s (critical) |
| Redis内存使用率 | Redis内存使用比例 | >85% (warning) |
| Redis命中率 | 缓存命中率 | <80% (warning) |

## 告警规则

### 告警级别

| 级别 | 说明 | 通知方式 |
|-----|------|---------|
| critical | 严重故障,需要立即处理 | 邮件 + Slack + PagerDuty |
| warning | 警告,需要关注 | 邮件 + Slack |
| info | 信息提示 | Slack |

### 告警分类

1. **系统告警** (`system_alerts`)
   - 高CPU/内存/磁盘使用率
   - 磁盘IO等待

2. **应用告警** (`application_alerts`)
   - 高错误率
   - 慢响应时间
   - 异常QPS

3. **业务告警** (`business_alerts`)
   - 低在线用户数
   - 无活跃竞拍
   - 低出价频率
   - 低转化率

4. **数据库告警** (`database_alerts`)
   - 高连接数
   - 慢查询
   - 复制延迟
   - Redis内存/命中率

5. **容器告警** (`container_alerts`)
   - 高容器资源使用
   - 容器重启
   - 异常退出

6. **可用性告警** (`service_availability_alerts`)
   - 服务不可用
   - 健康检查失败
   - SSL证书过期

## 仪表板使用

### 系统总览仪表板

展示系统核心资源使用情况:
- CPU/内存/磁盘使用率仪表盘
- 系统资源趋势图
- 网络流量和磁盘IO
- 服务健康状态

### 竞拍业务指标仪表板

展示竞拍业务关键指标:
- 在线用户数
- 活跃竞拍数
- 出价频率
- 订单转化率
- 用户行为分析
- 业务转化漏斗

### API性能监控仪表板

展示API性能指标:
- 请求QPS
- 响应时间分布(P50/P95/P99)
- 错误率趋势
- API端点性能排名
- 慢请求追踪
- 并发连接数

## 自定义指标

### 后端应用集成

```javascript
const metrics = require('./monitoring/exporters/backend-metrics');

// 1. 在Express应用中使用中间件
app.use(metrics.httpMetricsMiddleware);

// 2. 暴露指标端点
app.get('/metrics', async (req, res) => {
  res.set('Content-Type', metrics.getContentType());
  res.send(await metrics.getMetrics());
});

// 3. 记录业务指标
metrics.updateOnlineUsers(onlineUserCount);
metrics.recordBid(auctionId, userId);
metrics.recordOrder(auctionId, 'completed', amount);
```

### WebSocket服务集成

```javascript
const { WebSocketMonitor } = require('./monitoring/exporters/websocket-metrics');

const monitor = new WebSocketMonitor();

// 1. 记录连接
ws.on('connection', (socket, req) => {
  const connId = monitor.recordConnection(socket, userId);
});

// 2. 记录消息
monitor.recordMessage('bid', 'inbound', messageSize);

// 3. 记录房间加入
monitor.joinRoom(connectionId, auctionId);

// 4. 记录实时出价
monitor.recordLiveBid(auctionId, bidAmount, latency);
```

## 告警通知配置

### Email配置

编辑 `alertmanager/alertmanager.yml`:

```yaml
global:
  smtp_smarthost: 'smtp.example.com:587'
  smtp_from: 'alertmanager@example.com'
  smtp_auth_username: 'alertmanager@example.com'
  smtp_auth_password: 'your-password'
```

### Slack配置

```yaml
global:
  slack_api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'

receivers:
  - name: 'team-critical'
    slack_configs:
      - channel: '#critical-alerts'
        send_resolved: true
```

### Webhook配置

```yaml
receivers:
  - name: 'webhook'
    webhook_configs:
      - url: 'http://backend:3000/api/webhooks/alerts'
        send_resolved: true
```

## 日志查看

### Grafana中查看日志

1. 进入Grafana
2. 选择 `Explore` -> `Loki`
3. 输入查询语句:

```logql
# 查看应用日志
{job="application"}

# 按级别过滤
{job="application"} |= "error"

# 按服务过滤
{service="backend"} |= "auction"

# 提取字段
{job="application"} | json | level="error"
```

## 分布式追踪

### 在应用中集成

```javascript
const { trace } = require('@opentelemetry/api');
const { NodeTracerProvider } = require('@opentelemetry/sdk-trace-node');
const { JaegerExporter } = require('@opentelemetry/exporter-jaeger');

// 配置Jaeger导出器
const exporter = new JaegerExporter({
  endpoint: 'http://localhost:14268/api/traces',
});

// 创建追踪器
const provider = new NodeTracerProvider({
  exporters: [exporter],
});
provider.register();

// 创建span
const tracer = trace.getTracer('live-auction');
const span = tracer.startSpan('process-bid');
// ... 业务逻辑
span.end();
```

### 在Jaeger中查看

1. 访问 http://localhost:16686
2. 选择服务: `live-auction-backend`
3. 查看调用链和延迟分析

## 性能优化

### Prometheus优化

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

# 存储优化
command:
  - '--storage.tsdb.retention.time=30d'
  - '--storage.tsdb.retention.size=10GB'
```

### Grafana优化

```yaml
environment:
  - GF_RENDERING_MODE=clustered
  - GF_RENDERING_CLUSTERING_MAX_CONCURRENCY=5
```

### 查询优化

使用记录规则预计算常用指标:

```yaml
# recording_rules.yml
groups:
  - name: recording_rules
    rules:
      - record: instance:cpu_usage:rate5m
        expr: 100 - (avg by(instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)
```

## 故障排查

### Prometheus无法启动

```bash
# 检查配置
docker exec prometheus promtool check config /etc/prometheus/prometheus.yml

# 检查日志
docker logs prometheus
```

### Grafana数据源连接失败

```bash
# 检查网络
docker exec grafana ping prometheus

# 检查数据源配置
curl http://localhost:3000/api/datasources
```

### 告警未触发

```bash
# 检查规则
curl http://localhost:9090/api/v1/rules

# 检查告警状态
curl http://localhost:9093/api/v2/alerts
```

## 备份与恢复

### 备份Prometheus数据

```bash
# 创建快照
curl -X POST http://localhost:9090/api/v1/admin/tsdb/snapshot

# 复制数据
docker cp prometheus:/prometheus/snapshots ./prometheus-backup/
```

### 备份Grafana仪表板

```bash
# 导出仪表板
curl -H "Authorization: Bearer <API_KEY>" \
  http://localhost:3000/api/dashboards/db/<dashboard-uid> \
  > dashboard.json
```

## 安全建议

1. **修改默认密码**: 修改Grafana、Prometheus等默认密码
2. **启用TLS**: 为所有服务配置HTTPS
3. **网络隔离**: 使用Docker网络隔离监控组件
4. **访问控制**: 配置Grafana的RBAC权限
5. **日志脱敏**: 避免在日志中记录敏感信息

## 扩展功能

### 自定义Dashboard

1. 在Grafana中创建仪表板
2. 导出JSON
3. 保存到 `grafana/dashboards/` 目录

### 自定义告警规则

1. 编辑 `prometheus/alerts.yml`
2. 重载配置: `curl -X POST http://localhost:9090/-/reload`

### 集成更多Exporter

1. 添加Exporter到 `docker-compose.yml`
2. 在 `prometheus.yml` 中添加抓取配置
3. 创建对应的Dashboard

## 参考资源

- [Prometheus文档](https://prometheus.io/docs/)
- [Grafana文档](https://grafana.com/docs/)
- [Alertmanager配置](https://prometheus.io/docs/alerting/latest/configuration/)
- [PromQL查询](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Loki日志查询](https://grafana.com/docs/loki/latest/logql/)
- [Jaeger分布式追踪](https://www.jaegertracing.io/docs/)
