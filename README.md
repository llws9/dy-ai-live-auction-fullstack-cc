# 直播竞拍系统 (Live Auction System)

一个完整的直播竞拍全栈系统，支持实时竞价、商品管理、用户认证、A/B 测试等功能。

> 部署入口：当前 `main` 分支对应的精简部署操作文档见 [deploy/demo/MAIN_DEPLOY_QUICKSTART.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/deploy/demo/MAIN_DEPLOY_QUICKSTART.md)

## 线上 Demo 入口

- H5 前台：`http://14.103.53.55/`
- Admin 后台：`http://14.103.53.55/admin/`
- Gateway API：`http://14.103.53.55/api/v1`
- Test Dashboard：`http://14.103.53.55/test-dashboard/`
- Grafana：`http://14.103.53.55/grafana/`
- 直播 WebSocket：`ws://14.103.53.55/api/v1/ws`
- 测试进度 WebSocket：`ws://14.103.53.55:18092`

## 演示访问凭据

- `Test Dashboard` 与 `Grafana` 入口使用 Nginx Basic Auth：`ByteDance` / `ByteDance`。
- `Grafana` 已开启 anonymous viewer，Basic Auth 通过后无需再输入 Grafana 应用账号。
- 业务演示账号由 `scripts/init-demo-users.sh` 统一 seed；README 不再维护明文业务账号密码，请以当前环境初始化脚本或演示控制台为准。
- `Nacos`、`Prometheus`、`Loki`、`GrowthBook` 已部署在线上服务器，但不公网裸开；评委通过 `Grafana`/`Test Dashboard` 查看演示结果。

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
│   ├── start-all.sh      # Docker Compose 统一启动脚本
│   ├── start-local-backend.sh # 本地 Go 后端启动脚本
│   └── start-frontend.sh # 本地前端启动脚本
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

### 本地脚本启动（推荐用于开发联调）

本地脚本启动会复用 Docker 中的基础设施，仅把 Go 后端和前端跑在宿主机，便于调试、打断点和查看日志。

#### 1. 启动基础设施

```bash
# 后端本地脚本依赖 MySQL、Redis、RabbitMQ 三个端口可用
INTERNAL_API_TOKEN=dev docker compose up -d mysql redis rabbitmq
```

说明：
- MySQL 默认连接：`127.0.0.1:3306`，数据库 `auction`；本地账号信息以 `docker-compose.yml` 和本地环境变量为准。
- Redis 默认连接：`127.0.0.1:6379`。
- RabbitMQ 默认连接：`127.0.0.1:5672`；本地账号信息以 `docker-compose.yml` 和本地环境变量为准。
- 本地后端脚本会通过环境变量禁用 Nacos 在线依赖，不需要启动 Nacos。

#### 2. 启动本地后端

```bash
# 启动 gateway/product/auction 三个 Go 服务
./scripts/start-local-backend.sh start

# 查看本地后端状态
./scripts/start-local-backend.sh status

# 重启本地后端
./scripts/start-local-backend.sh restart

# 停止本地后端
./scripts/start-local-backend.sh stop
```

脚本行为：
- 启动服务：`auction`、`product`、`gateway`。
- 监听端口：Gateway `8080`、Product `8081`、Auction HTTP `8082`、Auction WS `8083`。
- 日志目录：`.tmp/local-backend/`，例如 `.tmp/local-backend/gateway.log`。
- 端口冲突时会直接失败并打印占用进程，不会自动 kill 现有进程。
- 启动前会检查 `3306`、`6379`、`5672` 是否已就绪。

#### 3. 启动本地前端

```bash
# 启动 H5 和 Admin 前端
./scripts/start-frontend.sh
```

访问地址：
- H5 用户端：`http://localhost:5173`
- Admin 管理端：`http://localhost:5175`
- Gateway API：`http://localhost:8080/api/v1`
- Auction WebSocket：`ws://localhost:8083/ws`

前端日志：
- H5：`/tmp/h5-auction.log`
- Admin：`/tmp/admin-auction.log`

> 注意：`scripts/start-frontend.sh` 会检查 `5173` 和 `5175` 端口，并尝试停止占用进程；如果你不希望脚本处理端口，可手动进入 `frontend/h5` 和 `frontend/admin` 分别执行 `npm run dev`。

#### 4. 本地测试账号

本地部署时，`scripts/init-demo-users.sh` 会自动初始化演示账号；README 不维护明文账号密码，避免文档与脚本状态漂移。需要确认当前账号时，请查看初始化脚本或使用演示控制台的自动登录能力。

#### 5. 常见问题

- 如果 H5 登录提示“请求过于频繁”，可检查 Redis 中是否存在异常限流 key：`redis-cli ttl ratelimit:127.0.0.1`。
- 如需清理本地网关限流状态，可执行：`redis-cli del ratelimit:127.0.0.1`。
- 如果后端脚本提示端口被占用，先用 `lsof -nP -iTCP:<port> -sTCP:LISTEN` 定位占用进程，再决定是否停止。
- 如果前端无法访问 API，请确认 H5 Vite 代理指向 `http://localhost:8080`，并确认 `gateway` 已运行。

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
