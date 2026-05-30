# 环境配置检查清单

## 文档状态

- **状态**: 当前环境配置检查入口
- **Gateway 本地端口 SSOT**: `8080`
- **最后校准**: 2026-05-30
- **适用范围**: 本地开发、E2E 自检、测试平台 Dummy 联调

## 端口 SSOT

| 组件 | 端口/地址 | 必须满足 |
|------|----------|----------|
| Gateway | `http://localhost:8080` | 所有前端 HTTP/WS discovery 入口统一走 Gateway |
| Product Service | `http://localhost:8081` | 只允许 Gateway 或后端内部调用直连 |
| Auction HTTP | `http://localhost:8082` | 只允许 Gateway 或后端内部调用直连 |
| Auction WS | `ws://localhost:8083` | 由 Gateway discovery 返回真实地址，前端不硬编码 |
| Test Service HTTP | `http://localhost:18090` | Gateway `/api/test/*` 透传目标 |
| Test Service WS | `ws://localhost:18092` | Gateway `/ws/test/progress` discovery 返回目标 |
| H5 | `http://localhost:5173` | Vite dev server |
| Test Dashboard | `http://localhost:5174` | Vite dev server；不是 Admin |
| Admin | `http://localhost:5175` | Vite dev server |

## 必查文件

### 前端代理

- [ ] `frontend/h5/vite.config.ts`：`server.port = 5173`
- [ ] `frontend/h5/vite.config.ts`：`/api.target = http://localhost:8080`
- [ ] `frontend/admin/vite.config.ts`：`server.port = 5175`
- [ ] `frontend/admin/vite.config.ts`：`/api.target = http://localhost:8080`
- [ ] `frontend/test-dashboard/vite.config.ts`：`server.port = 5174`
- [ ] `frontend/test-dashboard/vite.config.ts`：`/api.target = http://localhost:8080`
- [ ] `frontend/test-dashboard/vite.config.ts`：`/ws.target = http://localhost:8080` 且 `ws: true`
- [ ] `frontend/test-dashboard/.env`：`VITE_API_BASE=/api`
- [ ] `frontend/test-dashboard/.env`：`VITE_WS_BASE=/ws`

### 后端服务配置

- [ ] `backend/gateway/config/config.go`：`GATEWAY_PORT` 默认 `:8080`
- [ ] `backend/gateway/config/config.go`：`TEST_SERVICE_URL` 默认 `http://localhost:18090`
- [ ] `backend/gateway/config/config.go`：`TEST_SERVICE_WS_URL` 默认 `ws://localhost:18092`
- [ ] `backend/test/config/config.go`：`TEST_HTTP_PORT` 默认 `:18090`
- [ ] `backend/test/config/config.go`：`TEST_WS_PORT` 默认 `:18092`
- [ ] `backend/test/config/config.go`：`TEST_GATEWAY_URL` 默认 `http://localhost:8080`
- [ ] `configs/nacos/gateway-config.yaml`：`server.port = ":8080"`
- [ ] `configs/nacos/gateway-config.yaml`：包含 `services.test_url`
- [ ] `configs/nacos/gateway-config.yaml`：包含 `services.test_ws_url`

### 本地环境变量

- [ ] `backend/.env`：`GATEWAY_PORT=:8080`
- [ ] `backend/.env`：`PRODUCT_SERVICE_PORT=:8081`
- [ ] `backend/.env`：`TEST_SERVICE_URL=http://localhost:18090`
- [ ] `backend/.env`：`TEST_SERVICE_WS_URL=ws://localhost:18092`
- [ ] `backend/.env`：`TEST_GATEWAY_URL=http://localhost:8080`
- [ ] `frontend/h5/.env.test`：`E2E_BASE_URL=http://localhost:5173`
- [ ] `frontend/h5/.env.test`：`API_BASE_URL=http://localhost:8080/api/v1`
- [ ] `frontend/admin/.env.test`：`E2E_BASE_URL=http://localhost:5175`
- [ ] `frontend/admin/.env.test`：`API_BASE_URL=http://localhost:8080/api/v1`

### Docker/脚本约束

- [ ] `docker-compose.yml`：Gateway 暴露 `8080:8080`
- [ ] `docker-compose.yml`：Product 暴露 `8081:8081`
- [ ] `docker-compose.yml`：Auction 暴露 `8082:8082` 和 `8083:8083`
- [ ] `scripts/start-frontend.sh`：只启动 H5 `5173` 和 Admin `5175`
- [ ] 若需要 Test Dashboard，一律手动执行 `cd frontend/test-dashboard && npm run dev`
- [ ] 当前主 `docker-compose.yml` 未包含 `test-service` / `test-dashboard`，不要把它当测试平台完整启动方式

## 自动检查命令

### 查找旧 Gateway 端口

```bash
grep -R "18080" backend frontend configs README.md STARTUP_GUIDE.md \
  --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=playwright-report --exclude-dir=test-results
```

期望：当前配置和入口文档不应出现 `18080`。历史修复记录可单独阅读，不纳入阻断项。

### 查找旧前端代理目标

```bash
grep -R "target: 'http://localhost:808[12]'" frontend \
  --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=playwright-report --exclude-dir=test-results
```

期望：无结果。前端开发代理只能指向 Gateway `8080`。

### 检查测试平台构建

```bash
cd frontend/test-dashboard
npm run build
```

期望：`tsc && vite build` 退出码为 0。

### 检查 Gateway/Test Service 健康

```bash
curl http://localhost:8080/health
curl http://localhost:18090/health
```

期望：两个命令均返回 2xx。

## 人工联调顺序

1. 启动基础设施：`docker compose up -d mysql redis nacos rabbitmq`
2. 启动 `backend/test`：`go run main.go`
3. 启动 `backend/gateway`：`go run main.go`
4. 启动 `frontend/test-dashboard`：`npm run dev`
5. 打开 `http://localhost:5174/test`
6. 点击 Dummy 测试，确认进度从 0% 到 100%

## 判定标准

- **通过**: 所有前端代理走 `8080`，测试平台服务端口为 `18090/18092`，构建通过，健康检查通过。
- **警告**: `docker-compose.yml` 未包含测试平台服务，这是当前已知边界，不影响本地手动联调。
- **失败**: 任一当前配置文件仍出现 `18080`、H5/Admin 代理直连 `8081/8082`、Admin 使用 `5174`。
