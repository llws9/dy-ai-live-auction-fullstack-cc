# 性能压测脚本使用指南

## 环境要求

### 安装 k6

#### macOS
```bash
brew install k6
```

#### Linux
```bash
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

#### Windows
```powershell
choco install k6
```

### 安装 wrk (可选)

#### macOS
```bash
brew install wrk
```

#### Linux
```bash
# Ubuntu/Debian
sudo apt-get install wrk

# CentOS/RHEL
sudo yum install wrk
```

## 目录结构

```
scripts/performance/
├── load_test.js                    # 负载测试脚本
├── stress_test.js                  # 压力测试脚本
├── spike_test.js                   # 峰值测试脚本
├── scenarios/                      # 测试场景目录
│   ├── concurrent_login.js         # 并发登录场景 (100/s)
│   ├── concurrent_registration.js  # 并发注册场景 (50/s)
│   ├── concurrent_bid.js           # 并发出价场景 (1000/s)
│   ├── auction_query.js            # 竞拍查询场景 (列表500/s, 详情800/s)
│   ├── websocket_test.js           # WebSocket连接测试 (1000个并发)
│   └── product_management.js       # 商品管理场景 (列表500/s, 创建100/s)
└── reports/                        # 测试报告输出目录
```

## 快速开始

### 1. 启动后端服务

确保后端服务已启动并监听在 `http://localhost:8080`

```bash
# 在项目根目录执行
cd backend
go run cmd/gateway/main.go
```

### 2. 运行负载测试

```bash
# 运行基础负载测试
k6 run scripts/performance/load_test.js

# 运行负载测试并输出到指定目录
k6 run --out json=reports/load_test.json scripts/performance/load_test.js
```

### 3. 运行压力测试

```bash
# 运行压力测试(逐步增加到系统极限)
k6 run scripts/performance/stress_test.js
```

### 4. 运行峰值测试

```bash
# 运行峰值测试(模拟突发流量)
k6 run scripts/performance/spike_test.js
```

### 5. 运行特定测试场景

```bash
# 并发登录测试 (100/s)
k6 run scripts/performance/scenarios/concurrent_login.js

# 并发注册测试 (50/s)
k6 run scripts/performance/scenarios/concurrent_registration.js

# 并发出价测试 (1000/s)
k6 run scripts/performance/scenarios/concurrent_bid.js

# 竞拍查询测试 (列表500/s, 详情800/s)
k6 run scripts/performance/scenarios/auction_query.js

# WebSocket连接测试 (1000个并发)
k6 run scripts/performance/scenarios/websocket_test.js

# 商品管理测试 (列表500/s, 创建100/s)
k6 run scripts/performance/scenarios/product_management.js
```

## 测试场景详解

### 1. 用户认证场景

#### 并发登录测试
- **目标**: 100/s 并发登录
- **指标**: P99延迟 < 200ms
- **命令**: `k6 run scripts/performance/scenarios/concurrent_login.js`

#### 并发注册测试
- **目标**: 50/s 并发注册
- **指标**: P99延迟 < 300ms
- **命令**: `k6 run scripts/performance/scenarios/concurrent_registration.js`

### 2. 竞拍核心场景

#### 并发出价测试
- **目标**: 1000/s 并发出价
- **指标**: P99延迟 < 500ms, 出价成功率 > 95%
- **命令**: `k6 run scripts/performance/scenarios/concurrent_bid.js`

#### 竞拍列表查询测试
- **目标**: 500/s 查询
- **指标**: P99延迟 < 200ms
- **命令**: `k6 run scripts/performance/scenarios/auction_query.js`

#### 竞拍详情查询测试
- **目标**: 800/s 查询
- **指标**: P99延迟 < 200ms
- **命令**: `k6 run scripts/performance/scenarios/auction_query.js`

### 3. WebSocket场景

#### 并发连接测试
- **目标**: 1000个并发WebSocket连接
- **指标**: 连接延迟 P99 < 500ms, 消息延迟 P99 < 200ms
- **命令**: `k6 run scripts/performance/scenarios/websocket_test.js`

### 4. 商品管理场景

#### 商品列表查询
- **目标**: 500/s 查询
- **指标**: P99延迟 < 200ms

#### 商品创建
- **目标**: 100/s 创建
- **指标**: P99延迟 < 300ms
- **命令**: `k6 run scripts/performance/scenarios/product_management.js`

## 性能指标

### 目标指标

| 场景 | 目标QPS | P50延迟 | P99延迟 | 错误率 |
|------|---------|---------|---------|--------|
| 并发登录 | 100/s | < 100ms | < 200ms | < 1% |
| 并发注册 | 50/s | < 150ms | < 300ms | < 5% |
| 并发出价 | 1000/s | < 100ms | < 500ms | < 1% |
| 竞拍列表查询 | 500/s | < 100ms | < 200ms | < 1% |
| 竞拍详情查询 | 800/s | < 100ms | < 200ms | < 1% |
| WebSocket连接 | 1000个 | - | < 500ms | < 5% |
| 商品列表查询 | 500/s | < 100ms | < 200ms | < 1% |
| 商品创建 | 100/s | < 150ms | < 300ms | < 1% |

## 高级配置

### 环境变量

```bash
# 设置后端服务地址
export BASE_URL=http://localhost:8080

# 运行测试
k6 run scripts/performance/load_test.js
```

### 自定义参数

```bash
# 自定义VU数量和持续时间
k6 run --vus 100 --duration 5m scripts/performance/load_test.js

# 自定义阶段
k6 run --stage 1m:10,2m:50,1m:0 scripts/performance/load_test.js
```

### 输出格式

```bash
# 输出JSON格式
k6 run --out json=reports/test.json scripts/performance/load_test.js

# 输出InfluxDB格式
k6 run --out influxdb=http://localhost:8086/k6 scripts/performance/load_test.js
```

## 测试报告

### 报告位置

测试报告会自动生成在 `reports/` 目录下:

- JSON报告: `reports/test_summary.json`
- HTML报告: `reports/test_report.html`

### 查看报告

```bash
# 在浏览器中打开HTML报告
open reports/load_test_report.html
```

## 性能分析

### 瓶颈识别

1. **数据库瓶颈**
   - 检查数据库连接池配置
   - 查看慢查询日志
   - 分析数据库CPU和内存使用

2. **应用瓶颈**
   - 检查应用服务器资源使用
   - 分析应用日志错误
   - 查看GC日志

3. **网络瓶颈**
   - 检查网络带宽使用
   - 分析网络延迟
   - 查看连接池状态

### 优化建议

1. **数据库优化**
   - 添加适当的索引
   - 优化查询语句
   - 使用数据库连接池
   - 考虑读写分离

2. **应用优化**
   - 使用缓存(Redis)
   - 实现异步处理
   - 优化代码逻辑
   - 使用CDN加速

3. **架构优化**
   - 水平扩展服务
   - 负载均衡
   - 微服务拆分
   - 容器化部署

## 持续集成

### GitHub Actions 示例

```yaml
name: Performance Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  performance-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Install k6
        run: |
          sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      
      - name: Run load test
        run: k6 run scripts/performance/load_test.js
      
      - name: Upload reports
        uses: actions/upload-artifact@v2
        with:
          name: performance-reports
          path: reports/
```

## 故障排查

### 常见问题

1. **连接超时**
   - 检查后端服务是否启动
   - 确认端口配置正确
   - 检查防火墙设置

2. **内存不足**
   - 减少并发用户数
   - 增加系统内存
   - 优化测试脚本

3. **错误率过高**
   - 检查后端日志
   - 分析错误类型
   - 调整测试参数

### 日志查看

```bash
# 查看k6详细日志
k6 run --verbose scripts/performance/load_test.js

# 查看后端日志
tail -f backend/logs/app.log
```

## 最佳实践

1. **测试前准备**
   - 确保测试环境与生产环境一致
   - 清理测试数据
   - 检查系统资源

2. **测试中监控**
   - 监控系统资源使用
   - 观察错误日志
   - 记录异常情况

3. **测试后分析**
   - 分析性能指标
   - 识别瓶颈
   - 制定优化方案

## 联系支持

如有问题,请提交 Issue 或联系开发团队。
