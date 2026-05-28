# 前端API代理配置修复

**修复日期**: 2026-05-23
**问题**: Admin前端登录显示"网络错误"

---

## 🐛 问题原因

### 错误配置

**文件**: `frontend/admin/vite.config.ts`

```typescript
proxy: {
  '/api': {
    target: 'http://localhost:8081',  // ❌ 错误：指向Product Service
    changeOrigin: true,
  },
}
```

### 问题分析

1. **Product Service (8081)**: 没有认证API（/api/v1/auth/login）
2. **Gateway (8080)**: 正确路由认证请求到Auction Service
3. **Admin前端**: 将API请求错误地代理到Product Service

**结果**: 登录请求发送到没有认证接口的服务，返回404或网络错误

---

## ✅ 修复方案

### 正确配置

```typescript
proxy: {
  '/api': {
    target: 'http://localhost:8080',  // ✅ 正确：指向Gateway
    changeOrigin: true,
  },
}
```

### 架构说明

```
Admin前端 (5175)
    ↓ API请求
Gateway (8080)
    ↓ 路由转发
    ├─→ Product Service (8081) - 商品、订单、统计API
    └─→ Auction Service (8082) - 认证、竞拍、通知API
```

---

## 🔧 服务端口总结

| 服务 | 端口 | 功能 |
|------|------|------|
| **H5用户端** | 5173 | 用户参与竞拍 |
| **Admin管理后台** | 5175 | 管理员后台 |
| **API Gateway** | 8080 | API网关、路由转发 |
| **Product Service** | 8081 | 商品、订单、统计服务 |
| **Auction Service** | 8082 | 认证、竞拍、通知服务 |
| **WebSocket** | 8083 | 实时竞拍推送 |

---

## ✅ 验证

### 测试登录API

```bash
# 通过Admin前端代理访问
curl -X POST http://localhost:5175/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}'
```

**预期结果**: 返回JWT token和用户信息

---

## 📝 H5用户端配置检查

H5用户端应该也使用Gateway：

```typescript
// frontend/h5/vite.config.ts
proxy: {
  '/api': {
    target: 'http://localhost:8080',  // 指向Gateway
    changeOrigin: true,
  },
}
```

---

**修复人**: Claude Code
**状态**: ✅ 已完成
