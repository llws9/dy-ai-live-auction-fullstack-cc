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

#### 关键指标

- `http_request_duration_seconds`: HTTP请求延迟
- `http_requests_total`: 总请求数
- `active_websocket_connections`: 活跃WebSocket连接数

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
