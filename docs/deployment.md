# 部署文档

## 目录

1. [环境要求](#环境要求)
2. [快速部署](#快速部署)
3. [Docker部署](#docker部署)
4. [手动部署](#手动部署)
5. [配置说明](#配置说明)
6. [服务管理](#服务管理)
7. [监控与日志](#监控与日志)
8. [故障排查](#故障排查)
9. [GrowthBook A/B 测试平台](#growthbook-ab-测试平台)
10. [Nacos 配置中心](#nacos-配置中心)

---

## 环境要求

### 硬件要求

| 组件 | 最低配置 | 推荐配置 |
|------|---------|---------|
| CPU | 2核 | 4核+ |
| 内存 | 4GB | 8GB+ |
| 磁盘 | 20GB | 50GB+ SSD |

### 软件要求

| 软件 | 版本要求 | 用途 |
|------|---------|------|
| Go | 1.21+ | 后端服务运行环境 |
| Node.js | 18+ | 前端构建工具 |
| MySQL | 8.0+ | 数据库 |
| Redis | 7.0+ | 缓存和分布式锁 |
| Docker | 20.10+ | 容器化部署（可选）|
| Docker Compose | 2.0+ | 多容器编排（可选）|

---

## 快速部署

### 方式一：Docker Compose（推荐）

```bash
# 1. 克隆代码
git clone <repository-url>
cd dy-ai-live-auction-fullstack-cc

# 2. 启动所有服务
docker-compose up -d

# 3. 查看服务状态
docker-compose ps

# 4. 查看日志
docker-compose logs -f
```

### 方式二：本地开发环境

```bash
# 1. 安装依赖
make install

# 2. 初始化数据库
make init-db

# 3. 启动服务
make start

# 4. 验证服务
make test
```

---

## Docker部署

### 1. 构建镜像

```bash
# 构建所有服务镜像
docker-compose build

# 或单独构建
docker build -t auction-product:latest ./backend/product
docker build -t auction-service:latest ./backend/auction
docker build -t auction-gateway:latest ./backend/gateway
```

### 2. 配置环境变量

创建 `.env` 文件：

```env
# 数据库配置
DB_HOST=mysql
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=auction

# Redis配置
REDIS_ADDR=redis:6379

# 服务地址
PRODUCT_SERVICE_ADDR=product:8081
AUCTION_SERVICE_ADDR=auction:8082
```

### 3. 启动服务

```bash
# 启动基础设施（MySQL、Redis）
docker-compose up -d mysql redis

# 等待服务就绪
sleep 30

# 初始化数据库
docker-compose exec mysql mysql -u root -p < scripts/init.sql

# 启动应用服务
docker-compose up -d product auction gateway

# 启动前端
docker-compose up -d frontend-h5 frontend-admin
```

### 4. 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Gateway | 8080 | API网关 |
| Product | 8081 | 商品服务 |
| Auction HTTP | 8082 | 竞拍HTTP服务 |
| Auction WebSocket | 8083 | 竞拍WebSocket服务 |
| H5 Frontend | 3000 | 用户端H5 |
| Admin Frontend | 3001 | 管理后台 |
| MySQL | 3306 | 数据库 |
| Redis | 6379 | 缓存 |
| Prometheus | 9090 | 指标收集 |
| Grafana | 3002 | 监控面板 |
| Loki | 3100 | 日志存储 |
| GrowthBook | 3200 | A/B 测试平台 |
| **Nacos** | **8848** | **配置中心** |
| **Nacos MySQL** | **3307** | **Nacos 数据库** |

---

## GrowthBook A/B 测试平台

### 概述

GrowthBook 是一个开源的 A/B 测试平台，支持：
- 特性开关（Feature Flags）
- 父子实验（Experiment Layering）
- 实验数据分析

### 启动 GrowthBook 服务

```bash
# 启动 GrowthBook 及数据库
docker-compose up -d growthbook growthbook-db

# 等待服务就绪（约10秒）
sleep 10

# 验证服务状态
curl http://localhost:3200/api/health
```

### 访问 Dashboard

```bash
# 在浏览器打开
open http://localhost:3200
```

首次访问需要：
1. 创建管理员账户
2. 配置 API Key（用于 SDK 连接）

### 创建实验步骤

#### 1. 创建 Feature Flag

```
Dashboard → Features → New Feature
```

配置示例：
- **Key**: `new-auction-ui-theme`
- **Name**: `新竞拍界面风格`
- **Type**: Boolean
- **Default Value**: `false`

#### 2. 配置实验变体

```
Feature → Add Experiment Rule
```

流量分配示例：
| 变体 | 流量占比 | 值 |
|------|---------|-----|
| Control | 50% | `false` |
| Treatment | 50% | `true` |

#### 3. 配置父子实验（Layering）

父子实验避免实验碰撞，实现多实验并行：

**父实验**：`new-auction-ui-theme`（UI 层）
**子实验**：`bid-button-color`（按钮颜色）

```
Dashboard → Layers → New Layer
```

配置：
| Layer | Namespace | 包含实验 |
|-------|-----------|---------|
| ui-layer | `ui` | `new-auction-ui-theme`, `bid-button-color` |
| business-layer | `business` | `new-bidding-algorithm`, `price-suggestion-strategy` |

### 前端使用示例

#### React Admin 使用

```tsx
// src/pages-new/AuctionList.tsx
import { useFeatureIsOnByKey, ExperimentLayer } from '@/shared/growthbook';

function AuctionList() {
  // 简单特性开关
  const showNewUI = useFeatureIsOnByKey('new-auction-ui-theme');
  
  // 父子实验
  const { parentVariant, childVariant } = useExperimentVariant(
    'new-auction-ui-theme',  // 父实验
    'bid-button-color'       // 子实验
  );

  return (
    <div className={showNewUI ? 'modern-theme' : 'classic-theme'}>
      {/* 根据实验变体渲染不同按钮 */}
      <button className={`btn-${childVariant || 'blue'}`}>
        立即出价
      </button>
    </div>
  );
}
```

#### React H5 使用

```tsx
// src/pages/Auction/index.tsx
import { useFeatureOn, useExperimentLayer } from '@/hooks/useExperiment';

function AuctionPage() {
  const useNewAlgo = useFeatureOn('new-bidding-algorithm');
  const { parentVariant, childVariant } = useExperimentLayer(
    'new-bidding-algorithm',
    'price-suggestion-strategy'
  );

  // 根据实验变体选择算法
  const placeBid = async () => {
    if (useNewAlgo) {
      // 新出价算法
      const suggestion = childVariant === 'smart' 
        ? await getSmartSuggestion() 
        : await getFixedSuggestion();
      return placeBidV2(suggestion);
    }
    return placeBidOld();
  };

  return (
    <div>
      <BidButton onClick={placeBid} />
    </div>
  );
}
```

### 后端使用示例

#### Gateway 特性开关检查

```go
// backend/gateway/handler/auction.go
func (h *Handler) PlaceBid(ctx context.Context, c *app.RequestContext) {
    // 从 context 获取实验属性
    attrs := middleware.GetExperimentAttributes(c)
    
    // 检查特性开关
    if h.gbClient.IsOn("new-bidding-flow", attrs) {
        // 使用新流程
        h.forwardToNewService(ctx, c)
    } else {
        // 使用旧流程
        h.forwardToOldService(ctx, c)
    }
    
    // 记录实验查看
    h.gbClient.TrackViewed("new-bidding-flow", "treatment")
}
```

#### Product Service 实验上下文

```go
// backend/product/service/product.go
import "product-service/pkg/experiment"

func (s *ProductService) GetProducts(ctx context.Context) {
    // 从请求头获取实验上下文（由 Gateway 转发）
    expCtx := experiment.FromHeaders(c)
    
    // 根据实验变体选择排序策略
    if expCtx.IsFeatureOn("new-sorting-algorithm") {
        return s.getProductsSortedByRecommendation(ctx)
    }
    return s.getProductsSortedByTime(ctx)
}
```

### 实验指标查看

#### Prometheus 指标

```bash
# 查看实验指标
curl http://localhost:9090/metrics | grep experiment
```

| 指标 | 说明 |
|------|------|
| `experiment_assigned_total` | 用户被分配到实验变体的次数 |
| `experiment_viewed_total` | 用户看到实验变体的次数 |
| `experiment_completed_total` | 用户完成实验目标的次数 |

#### Grafana Dashboard

```bash
# 打开 Grafana
open http://localhost:3002
```

添加实验监控面板：
1. 创建新 Dashboard
2. 添加 Prometheus 数据源
3. 配置查询：
```promql
# 实验分配速率
rate(experiment_assigned_total[1h])

# 按变体分组
sum by (experiment, variation) (rate(experiment_assigned_total[1h]))

# 实验转化率
rate(experiment_completed_total[1h]) / rate(experiment_assigned_total[1h])
```

### 环境变量配置

```env
# backend/.env
GROWTHBOOK_API_HOST=http://growthbook:3200
GROWTHBOOK_CLIENT_KEY=your-client-key
GROWTHBOOK_SECRET_KEY=your-secret-key
GROWTHBOOK_ENABLED=true

# frontend/.env
VITE_GROWTHBOOK_API_HOST=http://localhost:3200
VITE_GROWTHBOOK_CLIENT_KEY=your-client-key
```

### 常见实验场景

| 实验类型 | 父实验 | 子实验示例 |
|----------|--------|-----------|
| UI 层实验 | `new-auction-ui-theme` | `bid-button-color`, `auction-card-style` |
| 业务流程实验 | `new-bidding-algorithm` | `price-suggestion-strategy`, `auction-sorting` |
| 推荐算法实验 | `recommendation-engine-v2` | `ranking-factor`, `filter-strategy` |

### 实验生命周期

```
创建 → 配置 → 发布 → 监控 → 分析 → 结论 → 关闭/全量
```

1. **创建**: 在 Dashboard 创建 Feature Flag
2. **配置**: 设置流量分配和变体值
3. **发布**: 启用实验规则
4. **监控**: 通过 Prometheus/Grafana 查看指标
5. **分析**: 比较各变体的转化率等指标
6. **结论**: 决定保留哪个变体
7. **关闭**: 停止实验或全量发布获胜变体

---

## Nacos 配置中心

### 概述

Nacos 是阿里巴巴开源的配置中心和服务发现平台，支持：
- 配置集中管理（动态配置）
- 多环境配置隔离（Namespace）
- 配置热更新（无需重启服务）
- 服务注册与发现（可选）

### 启动 Nacos 服务

```bash
# 启动 Nacos 及数据库
docker-compose up -d nacos nacos-mysql

# 等待服务就绪（约30秒）
sleep 30

# 验证服务状态
curl http://localhost:8848/nacos/v1/ns/service/list?pageNo=1&pageSize=10
```

### 访问 Dashboard

```bash
# 在浏览器打开
open http://localhost:8848/nacos
```

默认账号密码：`nacos / nacos`

### 配置管理步骤

#### 1. 创建命名空间（Namespace）

```
Dashboard → 命名空间 → 新建命名空间
```

配置示例：
- **命名空间ID**: `auction-dev`
- **命名空间名**: `开发环境`
- **描述**: `直播竞拍系统开发环境配置`

#### 2. 创建配置

```
Dashboard → 配置管理 → 配置列表 → 创建配置
```

| 服务 | Group | Data ID | 配置文件 |
|------|-------|---------|----------|
| Gateway | `gateway` | `gateway-config.yaml` | `configs/nacos/gateway-config.yaml` |
| Product | `product` | `product-config.yaml` | `configs/nacos/product-config.yaml` |
| Auction | `auction` | `auction-config.yaml` | `configs/nacos/auction-config.yaml` |

#### 3. 配置内容示例

**Gateway 配置 (gateway-config.yaml)**:
```yaml
server:
  port: ":8080"

services:
  product_url: "http://product:8081"
  auction_url: "http://auction:8082"

jwt:
  secret: "your-secret-key-change-in-production"
  expire_time: "24h"

redis:
  addr: "redis:6379"
  password: ""
  pool_size: 100

growthbook:
  api_host: "http://growthbook:3200"
  client_key: "dev-client-key"
  secret_key: "dev-secret-key"
  enabled: true
```

#### 4. 发布配置

1. 点击配置详情 → 编辑
2. 复制配置内容
3. 点击发布

### 配置热更新

当配置变更时，Nacos 会自动推送到服务端：

```bash
# 修改配置后，查看服务日志
docker-compose logs gateway | grep "Config updated"

# 示例输出
# Config updated from Nacos: [group=gateway, dataId=gateway-config.yaml]
```

### 多环境配置

| 环境 | Namespace ID | 说明 |
|------|-------------|------|
| 开发 | `auction-dev` | 开发环境配置 |
| 测试 | `auction-test` | 测试环境配置 |
| 生产 | `auction-prod` | 生产环境配置 |

切换环境：
```bash
# docker-compose.yml 或 .env 中设置
NACOS_NAMESPACE=auction-test
```

### 环境变量配置

```env
# Nacos 配置中心
NACOS_SERVER_ADDR=nacos:8848
NACOS_NAMESPACE=auction-dev
NACOS_GROUP=gateway
NACOS_DATA_ID=gateway-config.yaml
```

### 配置迁移说明

从环境变量迁移到 Nacos 的配置项：

| 配置项 | 原环境变量 | 新位置 |
|--------|-----------|--------|
| 数据库配置 | `DB_HOST/PORT/USER/PASSWORD` | Nacos YAML `database` |
| Redis配置 | `REDIS_ADDR/PASSWORD` | Nacos YAML `redis` |
| JWT配置 | `JWT_SECRET` | Nacos YAML `jwt` |
| RabbitMQ配置 | `RABBITMQ_*` | Nacos YAML `rabbitmq` |
| GrowthBook配置 | `GROWTHBOOK_*` | Nacos YAML `growthbook` |
| 服务端口 | 硬编码 | Nacos YAML `server.port` |

### 本地开发配置

本地开发时，服务会自动使用环境变量作为后备：

```bash
# 如果 Nacos 不可用，自动使用 .env 配置
# 日志输出：Failed to connect Nacos, falling back to env config
```

### 故障排查

#### Nacos 连接失败

```bash
# 检查 Nacos 服务状态
docker-compose ps nacos

# 查看 Nacos 日志
docker-compose logs nacos | grep ERROR

# 检查网络连接
docker-compose exec gateway ping nacos
```

#### 配置加载失败

```bash
# 检查配置是否存在
curl "http://localhost:8848/nacos/v1/cs/configs?dataId=gateway-config.yaml&group=gateway&tenant=auction-dev"

# 检查服务日志
docker-compose logs gateway | grep "Config"
```

---

## 手动部署

### 1. 安装依赖

#### Go依赖
```bash
cd backend
go mod download
```

#### Node.js依赖
```bash
cd frontend/h5
npm install

cd ../admin
npm install
```

### 2. 配置数据库

```bash
# 创建数据库
mysql -u root -p <<EOF
CREATE DATABASE auction CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
EOF

# 导入表结构
mysql -u root -p auction < scripts/init.sql
```

### 3. 配置Redis

```bash
# 启动Redis
redis-server /etc/redis/redis.conf

# 验证连接
redis-cli ping
```

### 4. 启动后端服务

#### Product Service
```bash
cd backend/product
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=your_password
export DB_NAME=auction
export REDIS_ADDR=localhost:6379

go run main.go
```

#### Auction Service
```bash
cd backend/auction
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=your_password
export DB_NAME=auction
export REDIS_ADDR=localhost:6379

go run main.go
```

#### Gateway Service
```bash
cd backend/gateway
export PRODUCT_SERVICE_ADDR=localhost:8081
export AUCTION_SERVICE_ADDR=localhost:8082
export REDIS_ADDR=localhost:6379

go run main.go
```

### 5. 启动前端服务

#### H5 Frontend
```bash
cd frontend/h5
npm run dev
```

#### Admin Frontend
```bash
cd frontend/admin
npm run dev
```

### 6. 使用Systemd管理服务（生产环境）

创建服务文件 `/etc/systemd/system/auction-product.service`：

```ini
[Unit]
Description=Auction Product Service
After=network.target mysql.service redis.service

[Service]
Type=simple
User=auction
WorkingDirectory=/opt/auction/backend/product
Environment="DB_HOST=localhost"
Environment="DB_PORT=3306"
Environment="DB_USER=root"
Environment="DB_PASSWORD=your_password"
Environment="DB_NAME=auction"
Environment="REDIS_ADDR=localhost:6379"
ExecStart=/opt/auction/backend/product/product-service
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

启动服务：
```bash
sudo systemctl daemon-reload
sudo systemctl enable auction-product
sudo systemctl start auction-product
```

---

## 配置说明

### 环境变量配置

#### Product Service

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| DB_HOST | 是 | localhost | 数据库主机 |
| DB_PORT | 否 | 3306 | 数据库端口 |
| DB_USER | 是 | root | 数据库用户名 |
| DB_PASSWORD | 是 | - | 数据库密码 |
| DB_NAME | 否 | auction | 数据库名称 |
| REDIS_ADDR | 是 | localhost:6379 | Redis地址 |
| AUCTION_SERVICE_URL | 否 | http://localhost:8082 | Auction服务URL |

#### Auction Service

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| DB_HOST | 是 | localhost | 数据库主机 |
| DB_PORT | 否 | 3306 | 数据库端口 |
| DB_USER | 是 | root | 数据库用户名 |
| DB_PASSWORD | 是 | - | 数据库密码 |
| DB_NAME | 否 | auction | 数据库名称 |
| REDIS_ADDR | 是 | localhost:6379 | Redis地址 |

#### Gateway Service

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| PRODUCT_SERVICE_ADDR | 是 | localhost:8081 | Product服务地址 |
| AUCTION_SERVICE_ADDR | 是 | localhost:8082 | Auction服务地址 |
| REDIS_ADDR | 是 | localhost:6379 | Redis地址 |

### 数据库配置

#### 连接池配置

编辑 `backend/*/dao/db.go`：

```go
sqlDB.SetMaxIdleConns(10)    // 最大空闲连接数
sqlDB.SetMaxOpenConns(100)   // 最大打开连接数
sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期
```

#### 索引优化

已在 `scripts/init.sql` 中定义：

```sql
-- 竞拍表索引
INDEX idx_product_id (product_id),
INDEX idx_status (status),
INDEX idx_start_time (start_time)

-- 出价表索引
INDEX idx_auction_id (auction_id),
INDEX idx_user_id (user_id),
INDEX idx_auction_created (auction_id, created_at DESC)
```

### Redis配置

#### 连接池配置

编辑 `backend/*/dao/redis.go`：

```go
rdb := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    PoolSize:     100,     // 连接池大小
    MinIdleConns: 10,      // 最小空闲连接数
    MaxRetries:   3,       // 最大重试次数
})
```

---

## 服务管理

### 启动服务

```bash
# Docker方式
docker-compose start

# Systemd方式
sudo systemctl start auction-product
sudo systemctl start auction-service
sudo systemctl start auction-gateway

# 手动方式
cd backend/product && go run main.go &
cd backend/auction && go run main.go &
cd backend/gateway && go run main.go &
```

### 停止服务

```bash
# Docker方式
docker-compose stop

# Systemd方式
sudo systemctl stop auction-product
sudo systemctl stop auction-service
sudo systemctl stop auction-gateway

# 手动方式
pkill -f "go run main.go"
```

### 重启服务

```bash
# Docker方式
docker-compose restart

# Systemd方式
sudo systemctl restart auction-product
sudo systemctl restart auction-service
sudo systemctl restart auction-gateway
```

### 查看服务状态

```bash
# Docker方式
docker-compose ps

# Systemd方式
sudo systemctl status auction-product
sudo systemctl status auction-service
sudo systemctl status auction-gateway

# 检查端口
netstat -tlnp | grep -E "(8080|8081|8082|8083)"
```

---

## 监控与日志

### 日志管理

#### Docker日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f auction

# 查看最近100行日志
docker-compose logs --tail=100 auction
```

#### Systemd日志

```bash
# 查看服务日志
sudo journalctl -u auction-product -f

# 查看最近100行
sudo journalctl -u auction-product -n 100
```

#### 应用日志

日志输出到标准输出，可通过以下方式收集：

```bash
# 重定向到文件
go run main.go > /var/log/auction/product.log 2>&1 &

# 使用logrotate管理
# /etc/logrotate.d/auction
/var/log/auction/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 auction auction
}
```

### 健康检查

```bash
# Product Service
curl http://localhost:8081/health

# Auction Service
curl http://localhost:8082/health

# WebSocket Service
curl http://localhost:8083/health

# Gateway
curl http://localhost:8080/health
```

### 性能监控

#### Prometheus指标

访问 `http://localhost:8080/metrics`

#### Grafana 监控大盘

```bash
# 打开 Grafana
open http://localhost:3002

# 默认账号: admin / admin
```

**业务监控仪表板** (`http://localhost:3002/d/business-metrics`) 包含以下面板：

##### HTTP 服务监控

| 面板 | PromQL | 说明 |
|------|--------|------|
| 请求速率 (QPS) | `sum by (service) (rate(http_requests_total[5m]))` | 各服务每秒请求数 |
| 请求延迟 P95 | `histogram_quantile(0.95, sum by (le, service) (rate(http_request_duration_seconds_bucket[5m])))` | 95% 请求响应时间 |

##### SQL 查询监控（新增）

| 面板 | PromQL | 说明 |
|------|--------|------|
| SQL 耗时 P50 | `histogram_quantile(0.50, sum by (le, service) (rate(sql_query_duration_seconds_bucket[5m])))` | 50% SQL 查询耗时 |
| SQL 耗时 P95 | `histogram_quantile(0.95, sum by (le, service) (rate(sql_query_duration_seconds_bucket[5m])))` | 95% SQL 查询耗时 |
| SQL QPS | `sum by (service, operation) (rate(sql_query_total[5m]))` | SQL 查询速率 |
| SQL 错误率 | `sum(rate(sql_query_errors_total[5m]))` | SQL 查询错误数 |
| 操作类型分布 | `sum by (operation) (increase(sql_query_total[1h]))` | SELECT/INSERT/UPDATE/DELETE |
| 表热点分布 | `sum by (table) (increase(sql_query_total[1h]))` | 各表查询频率 |

##### 业务指标监控

| 面板 | PromQL | 说明 |
|------|--------|------|
| 直播间进入次数 | `sum(increase(live_room_enter_total[1h]))` | 用户进入直播间次数 |
| 当前观看人数 | `live_room_current_viewers` | 实时观看人数 |
| 出价次数 | `sum(increase(auction_bid_total[5m]))` | 出价活动趋势 |
| WebSocket 连接数 | `websocket_connections` | 实时 WS 连接数 |

##### 直播竞拍核心业务监控（新增）

| 面板 | PromQL | 说明 | 建议阈值 |
|------|--------|------|----------||
| 出价响应延迟 P95 | `histogram_quantile(0.95, sum(rate(auction_bid_latency_seconds_bucket[5m])) by (le))` | 用户出价到系统响应时间 | < 100ms ||
| 出价响应延迟 P50 | `histogram_quantile(0.50, sum(rate(auction_bid_latency_seconds_bucket[5m])) by (le))` | 中位数响应延迟 | < 50ms ||
| 延时触发次数 | `sum(increase(auction_delay_triggered_total[1h]))` | 竞拍延时触发总数 | 监控频率 ||
| 并发出价峰值 | `auction_concurrent_bids_peak` | 当前并发出价请求数 | 监控容量 ||
| 竞拍时长分布 | `histogram_quantile(0.50, sum(rate(auction_duration_seconds_bucket[5m])) by (le))` | 竞拍平均时长 | 监控异常 ||
| 竞拍溢价率 | `auction_premium_rate` | 成交价/起拍价 | 监控趋势 ||
| GMV（成交总额） | `sum(gmv_total)` | 累计成交金额 | 核心业务指标 ||
| 出价用户数 | `sum(increase(bid_user_count_total[1h]))` | 出价用户统计 | 参与度指标 ||
| 观看用户数 | `watch_user_count` | 当前观看人数 | 用户活跃度 ||
| WebSocket 消息延迟 | `histogram_quantile(0.95, sum(rate(websocket_message_latency_seconds_bucket[5m])) by (le))` | 消息推送延迟 | < 200ms ||
| WebSocket 错误数 | `sum(increase(websocket_errors_total[1h]))` | WebSocket 错误统计 | 监控稳定性 ||
| 竞拍成交率分布 | `sum by (has_winner) (increase(auction_completed_total[1h]))` | 成交/未成交占比 | 监控成功率 |

##### 实验监控（GrowthBook）

| 面板 | PromQL | 说明 |
|------|--------|------|
| 实验分配速率 | `rate(experiment_assigned_total[1h])` | 用户分配到实验变体 |
| 实验转化率 | `rate(experiment_completed_total[1h]) / rate(experiment_assigned_total[1h])` | 实验目标完成率 |

#### 关键指标列表

| 指标名称 | 类型 | Labels | 说明 |
|----------|------|--------|------|
| `http_requests_total` | Counter | service, path, status | HTTP 请求总数 |
| `http_request_duration_seconds` | Histogram | service | HTTP 请求耗时 |
| `sql_query_duration_seconds` | Histogram | service, operation, table | SQL 查询耗时 |
| `sql_query_total` | Counter | service, operation, table | SQL 查询总数 |
| `sql_query_errors_total` | Counter | service, operation, table, error | SQL 查询错误数 |
| `websocket_connections` | Gauge | - | WebSocket 连接数 |
| `experiment_assigned_total` | Counter | experiment, variation | 实验分配数 |
| `auction_bid_total` | Counter | status | 出价总数 |

---

## 故障排查

### 常见问题

#### 1. 数据库连接失败

**症状**: 服务启动失败，报错 `failed to connect database`

**排查步骤**:
```bash
# 检查MySQL是否运行
docker-compose ps mysql
# 或
sudo systemctl status mysql

# 检查数据库连接
mysql -h localhost -u root -p -e "SELECT 1"

# 检查环境变量
env | grep DB_
```

**解决方案**:
- 确保MySQL已启动
- 检查数据库用户名密码
- 检查数据库是否存在
- 检查防火墙规则

#### 2. Redis连接失败

**症状**: 出价失败，报错 `redis connection refused`

**排查步骤**:
```bash
# 检查Redis是否运行
docker-compose ps redis
# 或
redis-cli ping

# 检查Redis连接
redis-cli -h localhost -p 6379 ping
```

**解决方案**:
- 确保Redis已启动
- 检查Redis配置文件
- 检查Redis端口是否开放

#### 3. WebSocket连接失败

**症状**: 前端WebSocket连接失败

**排查步骤**:
```bash
# 检查WebSocket服务
curl http://localhost:8083/health

# 检查端口
netstat -tlnp | grep 8083

# 查看日志
docker-compose logs auction | grep WebSocket
```

**解决方案**:
- 确保Auction服务已启动
- 检查WebSocket端口配置
- 检查防火墙规则

#### 4. 出价失败

**症状**: 出价返回500错误

**排查步骤**:
```bash
# 查看竞拍状态
curl http://localhost:8082/api/v1/auctions/:id

# 检查日志
docker-compose logs auction | grep "出价"

# 检查数据库连接
docker-compose exec mysql mysql -u root -p -e "SELECT * FROM auction.auctions WHERE id = :id"
```

**常见原因**:
- 竞拍已结束
- 出价金额不足
- 用户ID不存在（外键约束）
- Redis锁获取失败

#### 5. 性能问题

**症状**: 响应时间过长

**排查步骤**:
```bash
# 检查CPU和内存使用
docker stats

# 检查数据库慢查询
docker-compose exec mysql mysql -u root -p -e "SHOW PROCESSLIST"

# 检查Redis内存使用
redis-cli info memory
```

**优化建议**:
- 增加数据库连接池大小
- 添加Redis缓存层
- 优化数据库索引
- 增加服务器资源

---

## 安全建议

### 1. 数据库安全

```bash
# 使用强密码
mysql -u root -p -e "ALTER USER 'root'@'localhost' IDENTIFIED BY 'strong_password';"

# 创建专用用户
mysql -u root -p -e "CREATE USER 'auction'@'%' IDENTIFIED BY 'strong_password';"
mysql -u root -p -e "GRANT ALL PRIVILEGES ON auction.* TO 'auction'@'%';"
```

### 2. Redis安全

```bash
# 设置密码
echo "requirepass your_strong_password" >> /etc/redis/redis.conf

# 禁用危险命令
echo "rename-command FLUSHDB \"\"" >> /etc/redis/redis.conf
echo "rename-command FLUSHALL \"\"" >> /etc/redis/redis.conf
```

### 3. 网络安全

```bash
# 配置防火墙
ufw allow 8080/tcp  # Gateway
ufw allow 8083/tcp  # WebSocket
ufw enable
```

---

## 备份与恢复

### 数据库备份

```bash
# 备份数据库
docker-compose exec mysql mysqldump -u root -p auction > backup_$(date +%Y%m%d).sql

# 恢复数据库
docker-compose exec -T mysql mysql -u root -p auction < backup_20260521.sql
```

### Redis备份

```bash
# 触发RDB快照
redis-cli BGSAVE

# 复制RDB文件
docker cp auction-redis-1:/data/dump.rdb ./backup/
```

---

## 升级指南

### 滚动升级

```bash
# 1. 拉取最新代码
git pull origin main

# 2. 构建新镜像
docker-compose build

# 3. 逐个服务升级
docker-compose up -d --no-deps --build product
sleep 10
docker-compose up -d --no-deps --build auction
sleep 10
docker-compose up -d --no-deps --build gateway

# 4. 验证服务
curl http://localhost:8080/health
```

---

## 附录

### Makefile命令

```makefile
install:
	cd backend && go mod download
	cd frontend/h5 && npm install
	cd frontend/admin && npm install

init-db:
	mysql -u root -p auction < scripts/init.sql

start:
	cd backend/product && go run main.go &
	cd backend/auction && go run main.go &
	cd backend/gateway && go run main.go &

stop:
	pkill -f "go run main.go"

test:
	go test ./backend/... -v
	cd frontend/h5 && npm test

logs:
	docker-compose logs -f

clean:
	docker-compose down -v
	rm -rf node_modules vendor
```

### 参考链接

- [Go官方文档](https://golang.org/doc/)
- [MySQL官方文档](https://dev.mysql.com/doc/)
- [Redis官方文档](https://redis.io/documentation)
- [Docker官方文档](https://docs.docker.com/)
