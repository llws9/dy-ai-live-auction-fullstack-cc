# 运维手册

## 目录

1. [日常运维](#日常运维)
2. [监控告警](#监控告警)
3. [性能优化](#性能优化)
4. [故障处理](#故障处理)
5. [备份恢复](#备份恢复)
6. [安全加固](#安全加固)
7. [容量规划](#容量规划)

---

## 日常运维

### 日常检查清单

#### 每日检查（自动化）

- [ ] 服务健康状态
- [ ] 数据库连接状态
- [ ] Redis连接状态
- [ ] 磁盘使用率
- [ ] 错误日志数量
- [ ] API响应时间
- [ ] Prometheus指标端点可用性（`curl http://localhost:9090/metrics`）
- [ ] Gateway QPS正常范围（无异常峰值）

#### 每周检查（手动）

- [ ] 系统资源使用情况
- [ ] 慢查询日志分析
- [ ] 数据库表空间使用
- [ ] WebSocket连接数趋势
- [ ] 出价成功率统计

#### 每月检查

- [ ] 系统性能报告
- [ ] 容量评估
- [ ] 安全漏洞扫描
- [ ] 依赖包更新
- [ ] 备份有效性验证

### 服务巡检脚本

```bash
#!/bin/bash
# scripts/health_check.sh

echo "====================================="
echo "  系统健康检查 - $(date)"
echo "====================================="
echo ""

# 检查服务状态
echo "【1】服务状态"
services=("product:8081" "auction:8082" "websocket:8083" "gateway:8080")
for service in "${services[@]}"; do
    name=$(echo $service | cut -d: -f1)
    port=$(echo $service | cut -d: -f2)
    
    if curl -s http://localhost:$port/health > /dev/null 2>&1; then
        echo "✅ $name (:$port)"
    else
        echo "❌ $name (:$port) - 服务异常"
    fi
done
echo ""

# 检查数据库
echo "【2】数据库状态"
if mysql -u root -proot -e "SELECT 1" > /dev/null 2>&1; then
    echo "✅ MySQL"
    
    # 查询连接数
    connections=$(mysql -u root -proot -e "SHOW STATUS LIKE 'Threads_connected'" | tail -1 | awk '{print $2}')
    echo "   活跃连接数: $connections"
else
    echo "❌ MySQL - 连接失败"
fi
echo ""

# 检查Redis
echo "【3】Redis状态"
if redis-cli ping > /dev/null 2>&1; then
    echo "✅ Redis"
    
    # 查询内存使用
    memory=$(redis-cli info memory | grep used_memory_human | cut -d: -f2 | tr -d '\r')
    echo "   内存使用: $memory"
else
    echo "❌ Redis - 连接失败"
fi
echo ""

# 检查磁盘
echo "【4】磁盘使用"
df -h | grep -E "(Filesystem|/dev/)" | awk '{print $1": "$3" / "$2" ("$5")"}'
echo ""

# 检查网络连接
echo "【5】网络连接"
echo "ESTABLISHED连接数: $(netstat -an | grep ESTABLISHED | wc -l)"
echo "TIME_WAIT连接数: $(netstat -an | grep TIME_WAIT | wc -l)"
echo ""

echo "====================================="
echo "  检查完成"
echo "====================================="
```

### 日志管理

#### 日志轮转配置

创建 `/etc/logrotate.d/auction`：

```
/var/log/auction/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 auction auction
    sharedscripts
    postrotate
        docker-compose restart > /dev/null 2>&1 || true
    endscript
}
```

#### 日志分析脚本

```bash
#!/bin/bash
# scripts/log_analysis.sh

LOG_DIR="/var/log/auction"
TODAY=$(date +%Y-%m-%d)

echo "日志分析报告 - $TODAY"
echo "====================================="
echo ""

# 错误统计
echo "【错误统计】"
grep -r "ERROR" $LOG_DIR/*.log 2>/dev/null | \
    awk '{print $1}' | \
    sort | uniq -c | sort -rn | head -10
echo ""

# 慢请求统计
echo "【慢请求统计 (>1s)】"
grep -r "took.*[0-9]\{4,\}ms" $LOG_DIR/*.log 2>/dev/null | \
    wc -l
echo ""

# API调用统计
echo "【API调用TOP 10】"
grep -r "POST\|GET\|PUT\|DELETE" $LOG_DIR/*.log 2>/dev/null | \
    awk '{print $7}' | \
    sort | uniq -c | sort -rn | head -10
echo ""
```

---

## 监控告警

### Prometheus配置

创建 `prometheus.yml`：

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'auction-gateway'
    static_configs:
      - targets: ['localhost:9090']  # Gateway Prometheus 指标端口
      
  - job_name: 'auction-product'
    static_configs:
      - targets: ['localhost:8081']
    
  - job_name: 'auction-service'
    static_configs:
      - targets: ['localhost:8082']
    
  - job_name: 'mysql'
    static_configs:
      - targets: ['localhost:9104']
    
  - job_name: 'redis'
    static_configs:
      - targets: ['localhost:9121']
```

### Gateway 指标端点

Gateway 服务在独立端口 **9090** 提供 Prometheus 指标数据：

```bash
# 访问指标端点
curl http://localhost:9090/metrics

# 或在浏览器打开
open http://localhost:9090/metrics
```

#### 可用的 HTTP 指标

| 指标名称 | 类型 | 说明 | 标签 |
|----------|------|------|------|
| `http_requests_total` | Counter | HTTP请求总数 | `service`, `method`, `path`, `status` |
| `http_request_duration_seconds` | Histogram | HTTP请求耗时分布 | `service`, `method`, `path` |

#### 可用的业务指标

| 指标名称 | 类型 | 说明 | 标签 |
|----------|------|------|------|
| `live_room_enter_total` | Counter | 直播间进入次数 | `room_id`, `user_type` |
| `live_room_current_viewers` | Gauge | 直播间当前观看人数 | - |
| `live_room_peak_viewers` | Gauge | 直播间峰值观看人数 | `room_id` |
| `auction_created_total` | Counter | 竞拍创建总数 | `product_id`, `status` |
| `auction_bid_total` | Counter | 出价次数统计 | `auction_id`, `status` |
| `auction_bid_amount` | Histogram | 出价金额分布 | `auction_id` |
| `auction_completed_total` | Counter | 竞拍完成总数 | `auction_id`, `has_winner` |
| `order_created_total` | Counter | 订单创建总数 | `auction_id`, `product_id` |
| `order_completed_total` | Counter | 订单完成总数 | `auction_id`, `product_id` |
| `order_amount` | Histogram | 订单金额分布 | `status` |
| `user_register_total` | Counter | 用户注册总数 | `source` |
| `user_login_total` | Counter | 用户登录总数 | `method` |
| `websocket_connections` | Gauge | WebSocket当前连接数 | - |
| `websocket_messages_total` | Counter | WebSocket消息总数 | `type`, `direction` |
| `payment_initiated_total` | Counter | 发起支付次数 | `method` |
| `payment_completed_total` | Counter | 支付完成次数 | `method` |
| `payment_failed_total` | Counter | 支付失败次数 | `method`, `error_code` |

#### PromQL 查询示例

**QPS 计算（每秒请求数）**：
```promql
# 全局 QPS
rate(http_requests_total[1m])

# 按路径分组的 QPS
sum by (path) (rate(http_requests_total[1m]))

# 按服务分组的 QPS
sum by (service) (rate(http_requests_total[1m]))

# 按状态码分组的 QPS（只看成功请求）
sum by (status) (rate(http_requests_total{status="200"}[1m]))

# 错误请求 QPS（5xx）
sum(rate(http_requests_total{status=~"5.."}[1m]))
```

**响应时间分析**：
```promql
# P50 响应时间
histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# P99 响应时间
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# 平均响应时间
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])
```

**业务指标查询**：
```promql
# 每秒竞拍创建数
rate(auction_created_total[1m])

# 每秒出价数（成功）
rate(auction_bid_total{status="success"}[1m])

# 出价金额分布（平均）
rate(auction_bid_amount_sum[5m]) / rate(auction_bid_amount_count[5m])

# WebSocket 连接数
websocket_connections

# 支付成功率
rate(payment_completed_total[5m]) / rate(payment_initiated_total[5m])
```

#### 指标中间件实现

指标通过 Gateway 中间件自动采集：

```go
// backend/gateway/middleware/metrics.go
func MetricsMiddleware(serviceName string, m *metrics.Metrics) app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        start := time.Now()
        c.Next(ctx)
        
        duration := time.Since(start).Seconds()
        method := string(c.Method())
        path := string(c.URI().Path())
        status := c.Response.StatusCode()
        
        // 记录请求计数和耗时
        m.RequestsTotal.WithLabelValues(serviceName, method, path, strconv.Itoa(status)).Inc()
        m.RequestDuration.WithLabelValues(serviceName, method, path).Observe(duration)
    }
}
```

指标初始化位置：

```go
// backend/gateway/main.go
func main() {
    // 初始化 Prometheus 指标
    m := metrics.Init("gateway")
    
    // 应用指标中间件
    h.Use(middleware.MetricsMiddleware("gateway", m))
    
    // 启动独立 Prometheus 服务（端口 9090）
    go func() {
        promServer := &http.Server{
            Addr:    ":9090",
            Handler: metrics.Handler(),
        }
        log.Printf("Prometheus metrics server starting on :9090")
        promServer.ListenAndServe()
    }()
}
```

### 告警规则

创建 `alert_rules.yml`：

```yaml
groups:
  - name: auction_alerts
    rules:
      # 服务宕机告警
      - alert: ServiceDown
        expr: up == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "服务 {{ $labels.job }} 宕机"
          description: "服务 {{ $labels.instance }} 已宕机超过1分钟"
      
      # 高错误率告警
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "高错误率"
          description: "5xx错误率超过5%"
      
      # 响应时间告警
      - alert: HighResponseTime
        expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 0.2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "P99响应时间过长"
          description: "P99响应时间超过200ms"
      
      # 数据库连接数告警
      - alert: HighDBConnections
        expr: mysql_global_status_threads_connected > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "数据库连接数过高"
          description: "当前连接数: {{ $value }}"
      
      # Redis内存告警
      - alert: HighRedisMemory
        expr: redis_memory_used_bytes / redis_memory_max_bytes > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Redis内存使用过高"
          description: "Redis内存使用率超过80%"
```

### Grafana Dashboard

导入以下Dashboard：

#### 1. Gateway QPS Dashboard（核心）

| 面板名称 | PromQL 查询 | 说明 |
|----------|-------------|------|
| **总 QPS** | `sum(rate(http_requests_total[1m]))` | 全局每秒请求数 |
| **按路径 QPS** | `sum by (path) (rate(http_requests_total[1m]))` | 各路径请求量 |
| **成功率** | `sum(rate(http_requests_total{status="200"}[5m])) / sum(rate(http_requests_total[5m])) * 100` | HTTP 成功请求占比 |
| **P50 响应时间** | `histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))` | 50% 请求响应时间 |
| **P99 响应时间** | `histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))` | 99% 请求响应时间 |
| **错误 QPS** | `sum(rate(http_requests_total{status=~"5.."}[1m]))` | 5xx 错误每秒数 |

#### 2. 服务概览 Dashboard

- 请求数/秒（各服务）
- 响应时间分布
- 错误率
- 活跃连接数

#### 3. 业务指标 Dashboard

| 面板名称 | PromQL 查询 | 说明 |
|----------|-------------|------|
| **竞拍创建速率** | `rate(auction_created_total[5m])` | 每分钟竞拍创建数 |
| **出价成功率** | `rate(auction_bid_total{status="success"}[5m]) / rate(auction_bid_total[5m])` | 出价成功比例 |
| **WebSocket 连接数** | `websocket_connections` | 当前 WS 连接数 |
| **支付成功率** | `rate(payment_completed_total[5m]) / rate(payment_initiated_total[5m])` | 支付成功比例 |

#### 4. SQL 查询监控（新增）

| 面板名称 | PromQL 查询 | 说明 |
|----------|-------------|------|
| **SQL 耗时 P50** | `histogram_quantile(0.50, sum by (le, service) (rate(sql_query_duration_seconds_bucket[5m])))` | 50% SQL 查询耗时 |
| **SQL 耗时 P95** | `histogram_quantile(0.95, sum by (le, service) (rate(sql_query_duration_seconds_bucket[5m])))` | 95% SQL 查询耗时 |
| **SQL QPS** | `sum by (service, operation) (rate(sql_query_total[5m]))` | 每秒 SQL 查询数 |
| **SQL 错误率** | `sum(rate(sql_query_errors_total[5m]))` | SQL 查询错误数 |
| **操作分布** | `sum by (operation) (increase(sql_query_total[1h]))` | SELECT/INSERT/UPDATE/DELETE 分布 |
| **表热点** | `sum by (table) (increase(sql_query_total[1h]))` | 各表查询频率 |

**SQL 查询耗时分布 Buckets**:
- 0.001s, 0.005s, 0.01s, 0.05s, 0.1s, 0.5s, 1s, 2s, 5s

**SQL 指标 Labels**:
- `service`: auction-service, product-service
- `operation`: query, create, update, delete, row
- `table`: auctions, bids, products, orders 等

#### 5. 数据库连接监控

- 连接数: `SHOW PROCESSLIST`
- 慢查询数: MySQL slow_log 表
- 缓存命中率: Redis `hit_rate`

#### 6. Redis监控

- 内存使用
- 命令统计
- 键空间
- 连接数

### 告警通知配置

```yaml
# alertmanager.yml
global:
  smtp_smarthost: 'smtp.example.com:587'
  smtp_from: 'alert@example.com'
  smtp_auth_username: 'alert@example.com'
  smtp_auth_password: 'password'

route:
  receiver: 'team-auction'
  routes:
    - match:
        severity: critical
      receiver: 'team-auction-critical'
    - match:
        severity: warning
      receiver: 'team-auction-warning'

receivers:
  - name: 'team-auction'
    email_configs:
      - to: 'team@example.com'
  
  - name: 'team-auction-critical'
    email_configs:
      - to: 'oncall@example.com'
    webhook_configs:
      - url: 'https://hooks.slack.com/services/xxx'
```

---

## 性能优化

### 数据库优化

#### 1. 慢查询优化

```sql
-- 查看慢查询
SELECT * FROM mysql.slow_log 
WHERE start_time > DATE_SUB(NOW(), INTERVAL 1 DAY) 
ORDER BY query_time DESC 
LIMIT 10;

-- 分析执行计划
EXPLAIN SELECT * FROM auctions WHERE status = 1 AND end_time > NOW();

-- 添加缺失索引
CREATE INDEX idx_status_endtime ON auctions(status, end_time);
```

#### 2. 表优化

```sql
-- 优化表
OPTIMIZE TABLE auctions;
OPTIMIZE TABLE bids;

-- 分析表
ANALYZE TABLE auctions;
ANALYZE TABLE bids;
```

#### 3. 连接池调优

```go
// 增加连接池配置
sqlDB.SetMaxIdleConns(20)    // 从10增加到20
sqlDB.SetMaxOpenConns(200)   // 从100增加到200
sqlDB.SetConnMaxLifetime(30 * time.Minute) // 设置连接生命周期
```

### Redis优化

#### 1. 内存优化

```bash
# 查看内存使用
redis-cli info memory

# 查看大键
redis-cli --bigkeys

# 设置内存上限
redis-cli CONFIG SET maxmemory 2gb
redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

#### 2. 持久化优化

```bash
# RDB配置
save 900 1
save 300 10
save 60 10000

# AOF配置
appendonly yes
appendfsync everysec
```

### 应用优化

#### 1. 缓存策略

```go
// 添加缓存层
func (s *AuctionService) GetAuction(ctx context.Context, id int64) (*model.Auction, error) {
    // 1. 尝试从Redis获取
    cacheKey := fmt.Sprintf("auction:%d", id)
    cached, err := s.redis.Get(ctx, cacheKey).Result()
    if err == nil {
        var auction model.Auction
        json.Unmarshal([]byte(cached), &auction)
        return &auction, nil
    }
    
    // 2. 从数据库获取
    auction, err := s.auctionDAO.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // 3. 写入缓存
    data, _ := json.Marshal(auction)
    s.redis.Set(ctx, cacheKey, data, 5*time.Minute)
    
    return auction, nil
}
```

#### 2. 批量处理

```go
// 批量获取竞拍
func (s *AuctionService) GetAuctions(ctx context.Context, ids []int64) ([]*model.Auction, error) {
    // 使用IN查询
    return s.auctionDAO.GetByIDs(ctx, ids)
}
```

#### 3. 连接复用

```go
// 使用连接池
var httpClient = &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
    Timeout: 10 * time.Second,
}
```

---

## 故障处理

### 故障处理流程

```
发现故障 → 初步评估 → 紧急止损 → 根因分析 → 彻底修复 → 复盘总结
```

### 应急预案

#### 1. 服务宕机

```bash
# 1. 确认服务状态
docker-compose ps

# 2. 查看日志
docker-compose logs --tail=100 auction

# 3. 重启服务
docker-compose restart auction

# 4. 验证恢复
curl http://localhost:8082/health
```

#### 2. 数据库连接耗尽

```bash
# 1. 查看连接数
mysql -u root -p -e "SHOW PROCESSLIST"

# 2. 杀掉空闲连接
mysql -u root -p -e "SELECT CONCAT('KILL ',id,';') FROM information_schema.processlist WHERE Time > 60 AND Command = 'Sleep'" | mysql -u root -p

# 3. 增加最大连接数
mysql -u root -p -e "SET GLOBAL max_connections = 500"
```

#### 3. Redis内存不足

```bash
# 1. 查看内存使用
redis-cli info memory

# 2. 清理过期键
redis-cli --scan --pattern "auction:*" | xargs redis-cli DEL

# 3. 执行内存整理
redis-cli MEMORY PURGE
```

#### 4. 磁盘空间不足

```bash
# 1. 查看磁盘使用
df -h

# 2. 清理日志
find /var/log/auction -name "*.log" -mtime +7 -delete

# 3. 清理Docker
docker system prune -a

# 4. 清理未使用的镜像
docker image prune -a
```

#### 5. 高并发导致服务不可用

```bash
# 1. 限流
# 在Gateway中启用更严格的限流策略

# 2. 降级
# 关闭非核心功能

# 3. 扩容
docker-compose up -d --scale auction=3
```

### 故障复盘模板

```markdown
## 故障复盘报告

### 基本信息
- 故障时间：YYYY-MM-DD HH:MM - HH:MM
- 故障等级：P0/P1/P2/P3
- 影响范围：
- 处理人员：

### 故障描述
[详细描述故障现象]

### 时间线
- HH:MM 发现故障
- HH:MM 开始排查
- HH:MM 定位原因
- HH:MM 实施修复
- HH:MM 服务恢复

### 根本原因
[详细分析根本原因]

### 解决方案
[临时方案 + 长期方案]

### 改进措施
1. [改进项1]
2. [改进项2]
3. [改进项3]

### 经验教训
[总结经验教训]
```

---

## 备份恢复

### 自动备份脚本

```bash
#!/bin/bash
# scripts/backup.sh

BACKUP_DIR="/backup"
DATE=$(date +%Y%m%d_%H%M%S)

echo "开始备份 - $DATE"

# 1. 数据库备份
echo "备份数据库..."
mysqldump -u root -proot auction > $BACKUP_DIR/db_$DATE.sql
gzip $BACKUP_DIR/db_$DATE.sql

# 2. Redis备份
echo "备份Redis..."
redis-cli BGSAVE
sleep 5
cp /var/lib/redis/dump.rdb $BACKUP_DIR/redis_$DATE.rdb

# 3. 配置备份
echo "备份配置..."
tar -czf $BACKUP_DIR/config_$DATE.tar.gz \
    /etc/systemd/system/auction-*.service \
    .env \
    docker-compose.yml

# 4. 清理旧备份（保留7天）
find $BACKUP_DIR -type f -mtime +7 -delete

echo "备份完成"
```

### 定时备份配置

```bash
# crontab -e
# 每天凌晨2点执行备份
0 2 * * * /opt/auction/scripts/backup.sh >> /var/log/auction/backup.log 2>&1
```

### 恢复流程

#### 数据库恢复

```bash
# 1. 停止服务
docker-compose stop product auction

# 2. 恢复数据库
gunzip -c /backup/db_20260521.sql.gz | mysql -u root -p auction

# 3. 重启服务
docker-compose start product auction

# 4. 验证数据
mysql -u root -p -e "SELECT COUNT(*) FROM auction.auctions"
```

#### Redis恢复

```bash
# 1. 停止Redis
docker-compose stop redis

# 2. 恢复数据
cp /backup/redis_20260521.rdb /var/lib/redis/dump.rdb

# 3. 启动Redis
docker-compose start redis

# 4. 验证数据
redis-cli ping
```

---

## 安全加固

### 1. 系统安全

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 配置防火墙
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 8080/tcp
sudo ufw allow 8083/tcp
sudo ufw enable

# 禁用root登录
sudo sed -i 's/PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config
sudo systemctl restart sshd
```

### 2. 应用安全

```bash
# 使用非root用户运行
sudo useradd -r -s /bin/false auction
sudo chown -R auction:auction /opt/auction

# 设置文件权限
chmod 600 .env
chmod 700 scripts/
```

### 3. 数据库安全

```sql
-- 删除匿名用户
DELETE FROM mysql.user WHERE User='';

-- 禁止root远程登录
DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');

-- 删除测试数据库
DROP DATABASE IF EXISTS test;

-- 刷新权限
FLUSH PRIVILEGES;
```

### 4. Redis安全

```bash
# 设置密码
echo "requirepass $(openssl rand -base64 32)" >> /etc/redis/redis.conf

# 禁用危险命令
echo "rename-command FLUSHDB \"\"" >> /etc/redis/redis.conf
echo "rename-command FLUSHALL \"\"" >> /etc/redis/redis.conf
echo "rename-command CONFIG \"\"" >> /etc/redis/redis.conf

# 绑定本地地址
echo "bind 127.0.0.1" >> /etc/redis/redis.conf
```

---

## 容量规划

### 资源使用评估

#### 当前配置

| 服务 | CPU | 内存 | 存储 |
|------|-----|------|------|
| Gateway | 1核 | 1GB | - |
| Product | 1核 | 1GB | - |
| Auction | 2核 | 2GB | - |
| MySQL | 2核 | 4GB | 50GB |
| Redis | 1核 | 2GB | 10GB |

#### 性能基线

| 指标 | 当前值 | 目标值 |
|------|--------|--------|
| API QPS | 2000 | 5000 |
| WebSocket连接数 | 1000 | 5000 |
| 数据库连接数 | 50 | 200 |
| Redis内存使用 | 1GB | 2GB |

### 扩容方案

#### 垂直扩容

```bash
# 增加资源限制
# docker-compose.yml
services:
  auction:
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 4G
```

#### 水平扩容

```yaml
# 增加实例数
docker-compose up -d --scale auction=3

# 配置负载均衡
nginx:
  image: nginx:latest
  ports:
    - "8082:80"
  volumes:
    - ./nginx.conf:/etc/nginx/nginx.conf
```

### 容量预警

```yaml
# 容量告警规则
- alert: HighCPUUsage
  expr: 100 - (avg by(instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
  for: 5m
  annotations:
    summary: "CPU使用率过高"

- alert: HighMemoryUsage
  expr: (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 > 85
  for: 5m
  annotations:
    summary: "内存使用率过高"

- alert: DiskSpaceLow
  expr: (node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"}) * 100 < 20
  for: 5m
  annotations:
    summary: "磁盘空间不足"
```

---

## 运维工具集

### 常用脚本

```bash
# scripts/restart_all.sh - 重启所有服务
#!/bin/bash
docker-compose restart

# scripts/clear_cache.sh - 清理缓存
#!/bin/bash
redis-cli FLUSHDB

# scripts/show_status.sh - 查看状态
#!/bin/bash
docker-compose ps
docker stats --no-stream
```

### 诊断工具

```bash
# 网络诊断
netstat -tunlp | grep -E "(8080|8081|8082|8083)"

# 进程诊断
ps aux | grep auction

# 资源诊断
top -p $(pgrep -d',' -f auction)

# 日志诊断
tail -f /var/log/auction/*.log
```

---

## GrowthBook A/B 测试平台

### GrowthBook 服务部署

GrowthBook 服务运行在端口 3200：

```bash
# 启动 GrowthBook 服务
docker-compose up -d growthbook growthbook-db

# 访问 Dashboard
open http://localhost:3200

# 检查服务状态
curl http://localhost:3200/api/health
```

### 实验指标监控

**Prometheus 指标端点**：http://localhost:9090/metrics

**实验相关指标**：

| 指标名称 | 说明 | PromQL 查询 |
|----------|------|-------------|
| `experiment_assigned_total` | 实验分配总数 | `rate(experiment_assigned_total[1h])` |
| `experiment_viewed_total` | 实验查看总数 | `rate(experiment_viewed_total[1h])` |
| `experiment_completed_total` | 实验完成总数 | `rate(experiment_completed_total[1h])` |

**实验转化率查询**：
```promql
# 实验完成率（分配 → 完成）
sum(rate(experiment_completed_total[24h])) 
/ sum(rate(experiment_assigned_total[24h])) * 100
```

**按实验分组的指标**：
```promql
# 各实验的分配速率
sum by (experiment) (rate(experiment_assigned_total[1h]))

# 各变体的分配速率
sum by (experiment, variation) (rate(experiment_assigned_total[1h]))
```

### SQL 查询耗时监控

**SQL 查询指标**：

| 指标名称 | 说明 | PromQL 查询 |
|----------|------|-------------|
| `sql_query_duration_seconds` | SQL 查询耗时分布 | `rate(sql_query_duration_seconds_sum[5m]) / rate(sql_query_duration_seconds_count[5m])` |
| `sql_query_total` | SQL 查询总数 | `rate(sql_query_total[1m])` |
| `sql_query_errors_total` | SQL 查询错误总数 | `rate(sql_query_errors_total[5m])` |

**各服务 SQL 查询速率**：
```promql
# 按服务分组的查询速率
sum by (service) (rate(sql_query_total[1m]))

# 按操作类型分组的查询速率
sum by (service, operation) (rate(sql_query_total[1m]))
```

**SQL 查询 P99 响应时间**：
```promql
histogram_quantile(0.99, 
  sum by (le, service) (rate(sql_query_duration_seconds_bucket[5m]))
)
```

**慢查询检测**：
```promql
# 查询耗时超过 1 秒的数量
sum(rate(sql_query_duration_seconds_bucket{le="1", service="product-service"}[5m]))
```

### 实验管理

**创建新实验**：

1. 访问 GrowthBook Dashboard (http://localhost:3200)
2. 创建新 Feature Flag
3. 配置实验变体和流量分配
4. 设置 Layer（避免实验碰撞）

**父子实验配置**：

```
UI Layer (ui-layer):
- new-auction-ui-theme
- bid-button-color
- admin-ui-style

Business Layer (business-layer):
- new-bidding-algorithm
- price-suggestion-strategy
- auction-sorting
```

### 前端集成验证

```javascript
// 检查 GrowthBook SDK 是否正常初始化
console.log(window.GrowthBook?.getAttributes());

// 手动检查特性开关
const isOn = useFeatureIsOnByKey('new-auction-ui-theme');
console.log(`Feature status: ${isOn}`);
```

---

## 联系方式

- **运维团队**: ops@example.com
- **值班电话**: +86-xxx-xxxx-xxxx
- **工单系统**: https://ticket.example.com
- **知识库**: https://wiki.example.com/auction

---

**最后更新**: 2026-05-28
**维护人员**: DevOps Team

	#### 直播竞拍核心业务指标

	以下是直播竞拍平台新增的关键业务指标：

	| 指标名称 | 类型 | 说明 | 建议阈值 |
	|----------|------|------|----------|
	| `auction_bid_latency_seconds` | Histogram | 出价响应延迟 | P95 < 100ms |
	| `auction_delay_triggered_total` | Counter | 延时触发次数 | 监控频率 |
	| `auction_duration_seconds` | Histogram | 竞拍时长分布 | 监控异常值 |
	| `auction_premium_rate` | Gauge | 竞拍溢价率（成交价/起拍价） | 监控趋势 |
	| `auction_concurrent_bids_peak` | Gauge | 并发出价峰值 | 监控容量 |
	| `gmv_total` | Gauge | GMV（成交总额） | 核心业务指标 |
	| `bid_user_count_total` | Counter | 出价用户数 | 参与度指标 |
	| `watch_user_count` | Gauge | 观看用户数 | 用户活跃度 |
	| `websocket_message_latency_seconds` | Histogram | WebSocket消息推送延迟 | P95 < 200ms |

	**直播竞拍核心指标 PromQL 查询**：

	```promql
	# 出价响应延迟 P95（核心体验指标）
	histogram_quantile(0.95, sum(rate(auction_bid_latency_seconds_bucket[5m])) by (le))

	# 竞拍参与率（出价用户数 / 观看用户数）
	sum(increase(bid_user_count_total[1h])) / watch_user_count

	# 竞拍成功率（成交场次 / 总场次）
	sum(increase(auction_completed_total{has_winner="true"}[1h])) / sum(increase(auction_completed_total[1h]))

	# 平均竞拍时长（分钟）
	histogram_quantile(0.50, sum(rate(auction_duration_seconds_bucket[1h])) by (le)) / 60

	# 竞拍溢价率
	auction_premium_rate

	# GMV 总额
	sum(gmv_total)

	# 延时触发频率
	rate(auction_delay_triggered_total[5m])

	# 并发出价峰值（容量监控）
	auction_concurrent_bids_peak

	# WebSocket 消息推送延迟 P95
	histogram_quantile(0.95, sum(rate(websocket_message_latency_seconds_bucket[5m])) by (le))
	```
