# Quick Start: 直播竞拍全栈系统

## 环境要求

- Docker & Docker Compose
- Go 1.21+
- Node.js 18+
- MySQL 8.0+
- Redis 7.0+

## 快速启动

### 1. 克隆项目

```bash
git clone <repo-url>
cd dy-ai-live-auction-fullstack-cc
```

### 2. 启动基础设施

```bash
docker-compose up -d redis mysql
```

等待 MySQL 和 Redis 启动完成（约 30 秒）。

### 3. 初始化数据库

```bash
# 连接 MySQL 执行建表脚本
mysql -h 127.0.0.1 -u root -proot auction < scripts/init.sql
```

### 4. 启动后端服务

```bash
# 启动 Gateway
cd backend/gateway && go run main.go

# 启动 Product Service (新终端)
cd backend/product && go run main.go

# 启动 Auction Service (新终端)
cd backend/auction && go run main.go
```

### 5. 启动前端

```bash
# H5 用户端
cd frontend/h5
npm install
npm run dev

# 管理后台 (新终端)
cd frontend/admin
npm install
npm run dev
```

### 6. 访问服务

| 服务 | 地址 |
|------|------|
| API Gateway | http://localhost:8080 |
| H5 用户端 | http://localhost:3000 |
| 管理后台 | http://localhost:3001 |
| WebSocket | ws://localhost:8083/ws |

## 开发工作流

### 创建商品并开始竞拍

```bash
# 1. 创建商品
curl -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{"name": "稀有珠宝", "description": "限量版珠宝"}'

# 2. 配置竞拍规则
curl -X POST http://localhost:8080/api/v1/products/1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "start_price": 0,
    "increment": 10,
    "cap_price": 1000,
    "duration": 300,
    "delay_duration": 30,
    "max_delay_time": 180
  }'

# 3. 用户出价
curl -X POST http://localhost:8080/api/v1/auctions/1/bids \
  -H "Content-Type: application/json" \
  -d '{"amount": 100}'
```

### WebSocket 连接示例

```javascript
const ws = new WebSocket('ws://localhost:8083/ws?auction_id=1');

ws.onopen = () => {
  console.log('WebSocket connected');
  // 发送心跳
  setInterval(() => ws.send(JSON.stringify({type: 'ping'})), 30000);
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  switch (data.type) {
    case 'bid_placed':
      console.log('New bid:', data.amount);
      break;
    case 'rank_update':
      console.log('Ranking:', data.ranking);
      break;
    case 'delay_triggered':
      console.log('New end time:', data.new_end_time);
      break;
    case 'auction_ended':
      console.log('Auction ended, winner:', data.winner);
      break;
  }
};
```

## 测试

### 单元测试

```bash
# 后端单元测试
cd backend && go test ./...

# 前端单元测试
cd frontend/h5 && npm test
```

### 集成测试

```bash
# 启动测试环境
docker-compose -f docker-compose.test.yml up -d

# 运行集成测试
go test -tags=integration ./tests/integration/...
```

## 常见问题

### Q: WebSocket 连接失败？
A: 检查 auction-service 是否启动，端口 8083 是否开放。

### Q: 出价失败提示"金额不足"？
A: 出价金额必须 >= 当前价格 + 加价幅度。

### Q: 竞拍状态不更新？
A: 检查 Redis 连接是否正常，分布式锁是否释放。

## 项目结构

```
.
├── backend/
│   ├── gateway/         # API 网关
│   ├── product/         # 商品服务
│   └── auction/         # 竞拍服务
├── frontend/
│   ├── h5/              # H5 用户端
│   └── admin/           # 管理后台
├── specs/               # 规格文档
└── docker-compose.yml
```
