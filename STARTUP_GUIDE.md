# MVP版本启动指南

## 📋 服务端口总览

### 前端服务 ⚠️ 重要
- **H5用户端**: 端口 **5173** - http://localhost:5173
- **Admin管理后台**: 端口 **5175** - http://localhost:5175

### 后端服务
- **API Gateway**: 端口 **8080** - http://localhost:8080
- **Product Service**: 端口 **8081** - http://localhost:8081
- **Auction Service (HTTP)**: 端口 **8082** - http://localhost:8082
- **Auction Service (WebSocket)**: 端口 **8083** - ws://localhost:8083

### 基础设施
- **MySQL (主数据库)**: 端口 **3306** - 主业务数据库
- **MySQL (Nacos)**: 端口 **3307** - Nacos 配置中心数据库
- **Redis**: 端口 **6379**
- **Nacos**: 端口 **8848** - 配置中心 Dashboard
- **GrowthBook**: 端口 **3200** - A/B 测试平台

### 日志/监控平台
- **Grafana**: 端口 **3002** - http://localhost:3002
- **Prometheus**: 端口 **9090** - http://localhost:9090
- **Loki**: 端口 **3100** - http://localhost:3100

---

## 🚀 快速启动

### 必填运行时密钥

后端本地启动前必须先注入 `INTERNAL_API_TOKEN`。这是 Gateway 调用 Auction `/internal/*` 接口的服务间凭证，`gateway-service` 与 `auction-service` 必须使用同一个值。

```bash
export INTERNAL_API_TOKEN="$(openssl rand -hex 32)"
```

注意：
- 不要把真实 token 写入 `configs/nacos/*.yaml`、`docker-compose.yml`、README 或前端环境变量。
- 如果未设置该变量，开播提醒相关的内部转发会按 fail closed 处理。

### 使用启动脚本（推荐）

```bash
# 回到项目根目录
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc

# 注入 Gateway/Auction 共享的内部服务 token
export INTERNAL_API_TOKEN="$(openssl rand -hex 32)"

# 运行启动脚本
./scripts/start-frontend.sh
```

脚本会自动：
1. 检查端口占用
2. 启动H5用户端（5173）
3. 启动Admin后台（5175）
4. 验证启动成功
5. 显示访问地址

---

### 手动启动

```bash
# 1. 启动H5用户端
cd frontend/h5
npm run dev &
# 访问: http://localhost:5173

# 2. 启动Admin后台（新终端）
cd frontend/admin
npm run dev &
# 访问: http://localhost:5175
```

---

## 📊 监控平台

### 启动监控平台

```bash
# 如果同时启动应用服务，先注入 Gateway/Auction 共享 token
export INTERNAL_API_TOKEN="$(openssl rand -hex 32)"

# 方式一：使用启动脚本
cd observability
./start.sh start

# 方式二：随应用服务一起启动（推荐）
docker compose up -d
```

### 访问 Grafana

- **URL**: http://localhost:3002
- **用户名**: `admin`
- **密码**: `admin`

### 预置仪表板

1. **业务监控仪表板** - 直播间进入、成交次数、支付统计等
2. **微服务日志仪表板** - 全链路日志查询

### 全链路日志查询

在 Grafana 的 Explore 中使用 LogQL：

```logql
# 按 request_id 查询全链路日志
{service_name=~".+"} |= "your-request-id"

# 查看错误日志
{service_name=~".+", success="false"}
```

### 业务指标查询 (PromQL)

```promql
# 1小时内的直播间进入次数
sum(increase(live_room_enter_total[1h]))

# 成交次数（1小时）
sum(increase(order_completed_total[1h]))

# 请求成功率
sum(rate(http_requests_total{status=~"2.."}[5m]))
  / sum(rate(http_requests_total[5m]))
```

### 前端埋点

```typescript
import { initTracking, getTracker } from '@/shared/tracking';

// 初始化
initTracking({ endpoint: '/api/track', debug: true });

// 使用
getTracker().trackLiveRoomEnter('room-123', 'vip');
getTracker().trackBidClick('auction-456', 999);
```

### 管理命令

```bash
cd observability

./start.sh start     # 启动
./start.sh stop      # 停止
./start.sh restart   # 重启
./start.sh logs      # 查看日志
./start.sh status    # 查看状态
```

详细文档：`observability/README.md`

---

## 🎫 管理员账户

**邮箱**: `admin@example.com`  
**密码**: `admin123`

**注意**: 
- 管理员账户role=2（平台管理员）
- 如遇登录问题，请确保前端API代理配置正确（指向Gateway 8080端口）

---

## ⚠️ 重要提醒

### 记住正确的端口！
- **5173** = H5用户端 = "直播竞拍"
- **5175** = Admin后台 = "竞拍管理后台"

### 常见错误
❌ 错误：以为5174是Admin后台
✅ 正确：5175才是Admin后台

### 验证方法
```bash
# 验证H5用户端
curl http://localhost:5173 | grep "title"
# 应该看到: <title>直播竞拍</title>

# 验证Admin后台
curl http://localhost:5175 | grep "title"
# 应该看到: <title>竞拍管理后台</title>
```

---

## 📚 相关文档

- **端口配置详细说明**: `docs/PORT_CONFIGURATION.md`
- **访问指南**: `MVP_ACCESS_GUIDE.md`
- **端口问题记录**: 本次问题已记录到 `docs/PORT_CONFIGURATION.md`

---

## 🔧 故障排查

### 端口被占用
```bash
# 查看端口占用
lsof -i :5173
lsof -i :5175

# 停止占用进程
kill -9 <PID>
```

### 启动失败
```bash
# 查看日志
tail -f /tmp/h5-auction.log
tail -f /tmp/admin-auction.log
```

### 重启服务
```bash
# 停止所有前端服务
pkill -f "vite"

# 重新启动
./scripts/start-frontend.sh
```

---

**记住**: H5用户端5173，Admin后台5175！
