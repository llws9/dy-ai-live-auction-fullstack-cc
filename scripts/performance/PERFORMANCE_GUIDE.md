# 性能压测完整指南

## 目录

1. [概述](#概述)
2. [测试类型](#测试类型)
3. [测试场景](#测试场景)
4. [性能指标](#性能指标)
5. [运行测试](#运行测试)
6. [结果分析](#结果分析)
7. [瓶颈识别](#瓶颈识别)
8. [优化建议](#优化建议)

## 概述

本性能压测方案针对直播竞拍系统设计,使用k6作为主要压测工具,支持wrk作为辅助工具。

### 测试目标

- 验证系统在高并发下的稳定性
- 识别系统性能瓶颈
- 为容量规划提供数据支持
- 确保系统满足业务性能指标

### 系统要求

- k6 >= 0.40.0
- Docker >= 20.10 (可选)
- wrk >= 4.2.0 (可选)
- jq >= 1.6 (可选,用于结果分析)

## 测试类型

### 1. 负载测试 (Load Test)

**目的**: 验证系统在预期负载下的性能表现

**特点**:
- 逐步增加负载
- 持续时间较长(10-30分钟)
- 关注系统稳定性

**运行命令**:
```bash
k6 run scripts/performance/load_test.js
```

**负载模式**:
- 0-2分钟: 0 → 100 用户
- 2-7分钟: 100 用户保持
- 7-9分钟: 100 → 200 用户
- 9-14分钟: 200 用户保持
- 14-16分钟: 200 → 300 用户
- 16-21分钟: 300 用户保持
- 21-24分钟: 降至 0

### 2. 压力测试 (Stress Test)

**目的**: 找到系统的性能极限和崩溃点

**特点**:
- 负载持续增加直到系统极限
- 持续时间较长(30-40分钟)
- 关注系统容量

**运行命令**:
```bash
k6 run scripts/performance/stress_test.js
```

**负载模式**:
- 0-5分钟: 0 → 100 用户
- 5-10分钟: 100 → 500 用户
- 10-15分钟: 500 → 1000 用户
- 15-20分钟: 1000 → 2000 用户
- 20-30分钟: 2000 → 3000 用户(系统极限)
- 30-35分钟: 3000 用户保持
- 35-40分钟: 降至 0

### 3. 峰值测试 (Spike Test)

**目的**: 验证系统处理突发流量的能力

**特点**:
- 快速增加负载(突发)
- 多次峰值测试
- 关注系统弹性

**运行命令**:
```bash
k6 run scripts/performance/spike_test.js
```

**负载模式**:
- 基线: 50 用户
- 峰值1: 50 → 1000 用户(30秒内)
- 峰值2: 50 → 2000 用户(30秒内)
- 峰值3: 50 → 3000 用户(30秒内)

## 测试场景

### 场景1: 用户认证

#### 并发登录 (100/s)

**目标**: 测试用户登录接口的并发处理能力

**指标**:
- 目标QPS: 100/s
- P50延迟: < 100ms
- P99延迟: < 200ms
- 错误率: < 1%

**运行**:
```bash
k6 run scripts/performance/scenarios/concurrent_login.js
```

#### 并发注册 (50/s)

**目标**: 测试用户注册接口的并发处理能力

**指标**:
- 目标QPS: 50/s
- P50延迟: < 150ms
- P99延迟: < 300ms
- 错误率: < 5%

**运行**:
```bash
k6 run scripts/performance/scenarios/concurrent_registration.js
```

### 场景2: 竞拍核心

#### 并发出价 (1000/s)

**目标**: 测试竞拍出价接口的高并发处理能力

**指标**:
- 目标QPS: 1000/s
- P50延迟: < 100ms
- P99延迟: < 500ms
- 出价成功率: > 95%
- 错误率: < 1%

**运行**:
```bash
k6 run scripts/performance/scenarios/concurrent_bid.js
```

#### 竞拍列表查询 (500/s)

**目标**: 测试竞拍列表查询的性能

**指标**:
- 目标QPS: 500/s
- P50延迟: < 100ms
- P99延迟: < 200ms
- 错误率: < 1%

#### 竞拍详情查询 (800/s)

**目标**: 测试竞拍详情查询的性能

**指标**:
- 目标QPS: 800/s
- P50延迟: < 100ms
- P99延迟: < 200ms
- 错误率: < 1%

**运行**:
```bash
k6 run scripts/performance/scenarios/auction_query.js
```

### 场景3: WebSocket连接

**目标**: 测试WebSocket并发连接和消息推送能力

**指标**:
- 目标并发连接: 1000个
- 消息推送: 10000条/s
- 连接延迟 P99: < 500ms
- 消息延迟 P99: < 200ms
- 错误率: < 5%

**运行**:
```bash
k6 run scripts/performance/scenarios/websocket_test.js
```

### 场景4: 商品管理

#### 商品列表查询 (500/s)

**目标**: 测试商品列表查询的性能

**指标**:
- 目标QPS: 500/s
- P50延迟: < 100ms
- P99延迟: < 200ms

#### 商品创建 (100/s)

**目标**: 测试商品创建的性能

**指标**:
- 目标QPS: 100/s
- P50延迟: < 150ms
- P99延迟: < 300ms

**运行**:
```bash
k6 run scripts/performance/scenarios/product_management.js
```

## 性能指标

### 关键指标说明

| 指标 | 说明 | 目标值 |
|------|------|--------|
| QPS | 每秒查询数 | 竞拍出价 > 1000 |
| P50 | 50%请求的响应时间 | < 100ms |
| P99 | 99%请求的响应时间 | < 200ms |
| 错误率 | 失败请求占比 | < 0.1% |
| 并发用户 | 同时在线用户数 | 支持 1000+ |

### 性能基准

| 场景 | 目标QPS | P50 | P99 | 错误率 | 并发用户 |
|------|---------|-----|-----|--------|----------|
| 用户登录 | 100/s | 100ms | 200ms | < 1% | 200 |
| 用户注册 | 50/s | 150ms | 300ms | < 5% | 100 |
| 竞拍出价 | 1000/s | 100ms | 500ms | < 1% | 2000 |
| 竞拍列表 | 500/s | 100ms | 200ms | < 1% | 500 |
| 竞拍详情 | 800/s | 100ms | 200ms | < 1% | 800 |
| 商品列表 | 500/s | 100ms | 200ms | < 1% | 500 |
| 商品创建 | 100/s | 150ms | 300ms | < 1% | 200 |
| WebSocket | 10000/s | - | 200ms | < 5% | 1000连接 |

## 运行测试

### 快速开始

```bash
# 1. 安装k6
brew install k6  # macOS
# 或
sudo apt-get install k6  # Ubuntu

# 2. 启动后端服务
cd backend
go run cmd/gateway/main.go

# 3. 运行测试
cd ../scripts/performance
./run_performance_test.sh load
```

### Docker方式运行

```bash
# 启动完整的监控栈
docker-compose up -d

# 运行测试(会自动发送数据到InfluxDB)
docker-compose run k6 run /scripts/load_test.js

# 查看Grafana仪表板
open http://localhost:3000
# 用户名: admin, 密码: admin123
```

### 运行所有测试

```bash
# 运行所有主要测试
./run_performance_test.sh all

# 运行所有测试场景
./run_performance_test.sh scenarios
```

### 自定义参数

```bash
# 设置目标URL
export BASE_URL=http://your-server:8080

# 自定义VU数量和持续时间
k6 run --vus 500 --duration 10m scripts/performance/load_test.js

# 自定义阶段
k6 run --stage 2m:100,5m:500,2m:0 scripts/performance/load_test.js
```

## 结果分析

### 查看测试报告

```bash
# JSON格式报告
cat reports/load_test_summary.json | jq .

# HTML格式报告
open reports/load_test_report.html
```

### 分析脚本

```bash
# 查看所有可用报告
./analyze_results.sh

# 分析特定报告
./analyze_results.sh reports/load_test_summary.json
```

### 关键指标解读

#### 响应时间

- **P50**: 表示50%的请求响应时间低于此值,反映系统典型性能
- **P95**: 表示95%的请求响应时间低于此值,反映系统大部分情况下的性能
- **P99**: 表示99%的请求响应时间低于此值,反映系统在大部分情况下的最差性能

#### 错误率

- **< 1%**: 优秀,系统稳定
- **1% - 5%**: 警告,需要关注
- **> 5%**: 严重,系统不稳定

#### 吞吐量

- **QPS**: 每秒处理的请求数量,反映系统处理能力
- **RPS**: 每秒接收的请求数量,反映系统负载

## 瓶颈识别

### 常见瓶颈

#### 1. 数据库瓶颈

**症状**:
- 响应时间随并发增加而显著增长
- 数据库CPU使用率过高
- 慢查询增加

**诊断**:
```bash
# 查看数据库连接数
SHOW PROCESSLIST;

# 查看慢查询日志
tail -f /var/log/mysql/slow.log

# 查看数据库性能指标
SHOW STATUS LIKE 'Threads%';
SHOW STATUS LIKE 'Connections';
```

**解决方案**:
- 添加数据库索引
- 优化查询语句
- 增加数据库连接池大小
- 考虑读写分离

#### 2. 应用瓶颈

**症状**:
- 应用服务器CPU或内存使用率过高
- GC频繁
- 响应时间长但数据库负载低

**诊断**:
```bash
# 查看应用日志
tail -f logs/app.log

# 查看GC日志
jstat -gcutil <pid> 1000

# 性能分析
go tool pprof http://localhost:6060/debug/pprof/profile
```

**解决方案**:
- 优化代码逻辑
- 使用缓存减少数据库访问
- 异步处理耗时操作
- 增加应用实例

#### 3. 网络瓶颈

**症状**:
- 连接超时增加
- 网络带宽饱和
- 传输速率下降

**诊断**:
```bash
# 查看网络连接状态
netstat -an | grep ESTABLISHED | wc -l

# 查看网络带宽使用
iftop

# 查看端口占用
lsof -i :8080
```

**解决方案**:
- 增加网络带宽
- 使用CDN加速
- 优化数据传输格式
- 连接复用

#### 4. 内存瓶颈

**症状**:
- 内存使用率接近100%
- 频繁Full GC
- OOM错误

**诊断**:
```bash
# 查看内存使用
free -h
top

# 查看应用内存
jmap -heap <pid>
```

**解决方案**:
- 增加系统内存
- 优化内存使用
- 调整JVM参数
- 内存泄漏排查

## 优化建议

### 数据库优化

1. **索引优化**
   ```sql
   -- 为常用查询字段添加索引
   CREATE INDEX idx_auction_status ON auctions(status, created_at);
   CREATE INDEX idx_user_username ON users(username);
   ```

2. **查询优化**
   - 避免SELECT *
   - 使用EXPLAIN分析查询计划
   - 合理使用JOIN

3. **连接池配置**
   ```yaml
   database:
     max_open_conns: 100
     max_idle_conns: 20
     conn_max_lifetime: 300s
   ```

### 应用优化

1. **缓存策略**
   - 使用Redis缓存热点数据
   - 实现多级缓存
   - 合理设置缓存过期时间

2. **异步处理**
   - 使用消息队列处理耗时操作
   - 异步更新统计数据
   - 批量处理提高效率

3. **代码优化**
   - 避免N+1查询
   - 减少不必要的对象创建
   - 使用连接池

### 架构优化

1. **水平扩展**
   - 负载均衡
   - 微服务拆分
   - 容器化部署

2. **读写分离**
   - 主从复制
   - 读写路由
   - 数据同步

3. **限流熔断**
   - 实现限流保护
   - 熔断降级机制
   - 服务降级策略

## 最佳实践

### 测试前准备

1. **环境检查**
   - 确保测试环境与生产环境一致
   - 检查系统资源和配置
   - 清理测试数据

2. **基线测试**
   - 先运行小规模测试建立基线
   - 记录系统初始性能指标
   - 确保系统处于健康状态

### 测试中监控

1. **系统监控**
   - 监控CPU、内存、网络、磁盘
   - 关注异常指标
   - 记录关键时间点

2. **应用监控**
   - 监控应用日志
   - 关注错误率变化
   - 记录性能拐点

### 测试后分析

1. **数据分析**
   - 对比多次测试结果
   - 分析性能趋势
   - 识别性能瓶颈

2. **报告撰写**
   - 总结测试结果
   - 提出优化建议
   - 制定改进计划

## 附录

### 测试数据准备

```sql
-- 创建测试用户
INSERT INTO users (username, email, password)
SELECT
    CONCAT('test_user_', n) as username,
    CONCAT('test_', n, '@example.com') as email,
    '$2a$10$...' as password
FROM (
    SELECT @row := @row + 1 as n
    FROM information_schema.tables, (SELECT @row := 0) r
    LIMIT 10000
) numbers;

-- 创建测试竞拍
INSERT INTO auctions (title, description, start_price, current_price, status)
SELECT
    CONCAT('测试竞拍商品 ', n) as title,
    CONCAT('这是测试竞拍商品 ', n, ' 的描述') as description,
    FLOOR(100 + RAND() * 1000) as start_price,
    FLOOR(100 + RAND() * 1000) as current_price,
    'active' as status
FROM (
    SELECT @row := @row + 1 as n
    FROM information_schema.tables, (SELECT @row := 0) r
    LIMIT 1000
) numbers;
```

### 监控命令

```bash
# 系统资源监控
htop
iotop
iftop

# 数据库监控
mysqladmin -u root -p processlist
mysqladmin -u root -p status

# 应用监控
curl http://localhost:6060/debug/pprof/
```

### 故障排查

```bash
# 查看端口占用
lsof -i :8080
netstat -tulpn | grep 8080

# 查看进程信息
ps aux | grep app
top -p <pid>

# 查看系统日志
tail -f /var/log/syslog
journalctl -u app -f
```

## 联系支持

如有问题或建议,请联系开发团队或提交Issue。
