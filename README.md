# 直播竞拍系统 (Live Auction System)

一个完整的直播竞拍全栈系统，支持实时竞价、商品管理、用户认证、A/B 测试等功能。

## 技术栈

### 后端
- **Go 1.21+** - Hertz 框架
- **MySQL 8.0** - 主数据库
- **Redis 7.0** - 缓存和分布式锁
- **RabbitMQ** - 消息队列（通知系统）
- **Nacos** - 配置中心
- **GrowthBook** - A/B 测试平台

### 前端
- **React 18** - TypeScript
- **Vite** - 构建工具
- **Semi Design** - UI 组件库

### 监控
- **Prometheus** - 指标收集
- **Grafana** - 可视化面板
- **Loki** - 日志存储

## 目录结构

```
dy-ai-live-auction-fullstack-cc/
├── backend/
│   ├── gateway/          # API 网关 (8080)
│   ├── product/          # 商品服务 (8081)
│   ├── auction/          # 竞拍服务 (8082/8083)
│   ├── pkg/              # 共享包
│   └── .env              # 环境变量配置
├── frontend/
│   ├── h5/               # 用户端 H5 (5173/3000)
│   └── admin/            # 管理后台 (5175/3001)
├── configs/
│   └── nacos/            # Nacos 配置模板
├── docs/
│   ├── deployment.md     # 部署文档
│   ├── operations_manual.md # 运维手册
│   └── api_documentation.md # API 文档
├── observability/
│   ├── prometheus/       # Prometheus 配置
│   ├── grafana/          # Grafana 配置
│   └── loki/             # Loki 配置
├── scripts/
│   ├── init.sql          # 数据库初始化
│   ├── nacos-init.sql    # Nacos 初始化
│   └── start-all.sh      # 启动脚本
├── docker-compose.yml    # Docker 编排
└── Makefile              # 构建命令
```

## 快速启动

### Docker Compose（推荐）

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 本地开发

```bash
# 1. 启动基础设施
docker-compose up -d mysql redis nacos

# 2. 启动后端服务
cd backend/gateway && go run main.go &
cd backend/product && go run main.go &
cd backend/auction && go run main.go &

# 3. 启动前端
cd frontend/h5 && npm run dev &
cd frontend/admin && npm run dev &
```

## 数据初始化

### 种子数据生成
项目提供种子数据生成工具，用于快速创建测试数据：

```bash
cd backend/seed
go run main.go --size medium
```

### 数据规模
| 规模 | 数据量 | 说明 |
|------|--------|------|
| small | ~100条 | 快速测试 |
| medium | ~150条 | 开发演示（默认） |
| large | ~500条 | 性能测试 |

### 生成数据类型
- 用户数据（管理员、主播、普通用户）
- 商品分类和商品数据
- 直播间数据
- 竞拍规则数据
- 订单数据

更多详情请参阅 [种子数据指南](docs/SEED_DATA_GUIDE.md)。

## 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Gateway | 8080 | API 网关 |
| Product | 8081 | 商品服务 |
| Auction HTTP | 8082 | 竞拍 HTTP |
| Auction WebSocket | 8083 | 竞拍 WebSocket |
| H5 Frontend | 5173/3000 | 用户端 |
| Admin Frontend | 5175/3001 | 管理后台 |
| MySQL | 3306 | 主数据库 |
| Redis | 6379 | 缓存 |
| Nacos | 8848 | 配置中心 |
| GrowthBook | 3200 | A/B 测试 |
| Grafana | 3002 | 监控面板 |
| Prometheus | 9090 | 指标 |

## 核心功能

- **商品管理**: 商品创建、编辑、发布、分类
- **竞拍系统**: 实时竞价、倒计时、状态管理
- **订单管理**: 订单创建、支付、发货
- **用户认证**: JWT 认证、角色权限
- **A/B 测试**: GrowthBook 集成、父子实验
- **监控告警**: Prometheus + Grafana
- **配置中心**: Nacos 多环境配置

## 文档索引

- [部署文档](docs/deployment.md)
- [运维手册](docs/operations_manual.md)
- [API 文档](docs/api_documentation.md)
- [端口配置](docs/PORT_CONFIGURATION.md)
- [故障排查](docs/TROUBLESHOOTING.md)
- [启动指南](STARTUP_GUIDE.md)

## 开发指南

```bash
# 运行测试
go test ./backend/... -v

# 构建
go build ./backend/...

# Swagger 文档
# Gateway: http://localhost:8080/swagger/index.html
```

## 许可证

MIT License