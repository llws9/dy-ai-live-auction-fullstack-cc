# 前端端口配置说明

## 文档状态

- **状态**: 当前端口配置说明
- **Gateway 本地端口 SSOT**: `8080`
- **最后校准**: 2026-05-30

## 正确的端口分配

| 应用 | 本地端口 | 配置文件 | 代理目标 | 用途 |
|------|---------|---------|---------|------|
| H5 用户端 | `5173` | `frontend/h5/vite.config.ts` | `http://localhost:8080` | 用户参与竞拍 |
| Test Dashboard | `5174` | `frontend/test-dashboard/vite.config.ts` | `http://localhost:8080` | 测试平台控制台 |
| Admin 管理后台 | `5175` | `frontend/admin/vite.config.ts` | `http://localhost:8080` | 管理员后台 |

## 端口配置详情

### H5 用户端

```typescript
server: {
  port: 5173,
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    },
  },
}
```

### Test Dashboard

```typescript
server: {
  port: 5174,
  host: '0.0.0.0',
  proxy: {
    '/api': { target: 'http://localhost:8080', changeOrigin: true },
    '/ws': { target: 'http://localhost:8080', changeOrigin: true },
  },
}
```

### Admin 管理后台

```typescript
server: {
  port: 5175,
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    },
  },
}
```

## 后端端口规划

| 服务 | 端口 | 说明 |
|------|------|------|
| Gateway | `8080` | 前端统一 HTTP/WS 入口，本地 SSOT |
| Product Service | `8081` | 商品服务 |
| Auction Service HTTP | `8082` | 竞拍 HTTP 接口 |
| Auction Service WebSocket | `8083` | 竞拍实时推送 |
| Test Service HTTP | `18090` | 测试平台 HTTP 接口 |
| Test Service WebSocket | `18092` | 测试平台 WS 进度推送 |

## 常见问题

### 端口被占用

```bash
lsof -i :5173
lsof -i :5174
lsof -i :5175
lsof -i :8080
kill -9 <PID>
```

### 前端能打开但接口失败

优先检查 Gateway 是否运行：

```bash
curl http://localhost:8080/health
```

如果 Gateway 未启动，H5/Admin/Test Dashboard 的 `/api` 请求都会失败。

### Test Dashboard WS 连接失败

当前网关对 `/ws/test/progress` 采用 endpoint discovery：前端先请求 Gateway，Gateway 返回真实 `test-service` WebSocket 地址，随后浏览器直连 `ws://localhost:18092/ws/test/progress?test_id=...`。

检查顺序：
1. `curl http://localhost:8080/health`
2. `curl http://localhost:18090/health`
3. 确认 `backend/test` 的 WS 服务监听 `18092`

## 标准启动流程

```bash
# H5
cd frontend/h5
npm run dev

# Test Dashboard
cd frontend/test-dashboard
npm run dev

# Admin
cd frontend/admin
npm run dev
```

## 更新记录

**2026-05-30**:
- 统一 Gateway 本地 SSOT 为 `8080`
- 修正 H5/Admin 代理目标为 Gateway，而不是直连 Product/Auction
- 纳入 Test Dashboard `5174`、Test Service `18090/18092`
- 明确测试平台 WS 是 endpoint discovery，不是 Gateway WS 反代

**2026-05-23**:
- 修正 Admin 后台端口从 `5174` 改为 `5175`
- 添加端口配置说明文档
