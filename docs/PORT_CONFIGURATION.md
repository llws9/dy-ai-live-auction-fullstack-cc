# ⚠️ 前端端口配置说明

## 📋 正确的端口分配

| 应用 | 正确端口 | 配置文件 | 用途 |
|------|---------|---------|------|
| **H5 用户端** | 5173 | `frontend/h5/vite.config.ts` | 用户参与竞拍 |
| **Admin 管理后台** | 5175 | `frontend/admin/vite.config.ts` | 管理员后台 |

---

## 🔧 端口配置详情

### H5 用户端配置
**文件**: `frontend/h5/vite.config.ts`

```typescript
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,  // ✅ H5用户端端口
    proxy: {
      '/api': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
    },
  },
})
```

### Admin 管理后台配置
**文件**: `frontend/admin/vite.config.ts`

```typescript
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5175,  // ✅ Admin管理后台端口
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
    },
  },
})
```

---

## ⚠️ 常见问题

### 问题1: 端口被占用
**现象**: 启动时端口被占用，自动切换到其他端口

**解决方案**:
```bash
# 查看端口占用
lsof -i :5173
lsof -i :5175

# 停止占用端口的进程
kill -9 <PID>
```

### 问题2: 启动错误的目录
**现象**: 在错误的目录下运行启动命令

**正确启动方式**:
```bash
# 回到项目根目录
cd /path/to/dy-ai-live-auction-fullstack-cc

# 启动H5用户端
cd frontend/h5
npm run dev

# 新开终端，启动Admin后台
cd frontend/admin
npm run dev
```

### 问题3: 端口配置不一致
**现象**: vite.config.ts中配置的端口与实际需要的不一致

**检查步骤**:
1. 检查 `frontend/h5/vite.config.ts` - 应该是 5173
2. 检查 `frontend/admin/vite.config.ts` - 应该是 5175

---

## 🚀 标准启动流程

### 方式一：手动启动（推荐）

```bash
# 1. 确保在项目根目录
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc

# 2. 启动H5用户端（端口5173）
cd frontend/h5
npm run dev &

# 3. 启动Admin后台（端口5175）
cd ../admin
npm run dev &
```

### 方式二：使用启动脚本

创建启动脚本 `scripts/start-frontend.sh`:

```bash
#!/bin/bash

echo "🚀 启动前端服务..."

# 获取项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# 启动H5用户端
echo "📱 启动H5用户端 (端口5173)..."
cd "$PROJECT_ROOT/frontend/h5"
npm run dev > /tmp/h5-auction.log 2>&1 &
H5_PID=$!
echo "H5用户端 PID: $H5_PID"

# 等待2秒
sleep 2

# 启动Admin后台
echo "💼 启动Admin管理后台 (端口5175)..."
cd "$PROJECT_ROOT/frontend/admin"
npm run dev > /tmp/admin-auction.log 2>&1 &
ADMIN_PID=$!
echo "Admin后台 PID: $ADMIN_PID"

echo ""
echo "✅ 前端服务启动完成!"
echo "📱 H5用户端: http://localhost:5173"
echo "💼 Admin后台: http://localhost:5175"
echo ""
echo "查看日志:"
echo "  H5: tail -f /tmp/h5-auction.log"
echo "  Admin: tail -f /tmp/admin-auction.log"
```

---

## 📊 端口规划总览

### 前端端口（5173-5175）
- **5173**: H5用户端（用户参与竞拍）
- **5175**: Admin管理后台（管理员操作）

### 后端端口（8080-8083）
- **8080**: API Gateway（网关）
- **8081**: Product Service（商品服务）
- **8082**: Auction Service HTTP（竞拍HTTP接口）
- **8083**: Auction Service WebSocket（竞拍实时推送）

### 基础设施端口
- **3307**: MySQL（映射自容器内的3306）
- **6379**: Redis

---

## ✅ 验证清单

启动后验证步骤：

1. **检查端口监听**
   ```bash
   lsof -i :5173 | grep LISTEN
   lsof -i :5175 | grep LISTEN
   ```

2. **访问页面验证**
   ```bash
   # H5用户端
   curl http://localhost:5173 | grep "直播竞拍"

   # Admin后台
   curl http://localhost:5175 | grep "竞拍管理后台"
   ```

3. **检查日志**
   ```bash
   # 查看H5日志
   tail -f /tmp/h5-auction.log

   # 查看Admin日志
   tail -f /tmp/admin-auction.log
   ```

---

## 🔍 故障排查

### 检查端口是否正确启动

```bash
# 检查H5用户端
curl -s http://localhost:5173 | grep "title"
# 应该看到: <title>直播竞拍</title>

# 检查Admin后台
curl -s http://localhost:5175 | grep "title"
# 应该看到: <title>竞拍管理后台</title>
```

### 如果端口不对

1. 停止所有node进程
   ```bash
   pkill -f "vite"
   ```

2. 检查配置文件
   ```bash
   cat frontend/h5/vite.config.ts | grep port
   cat frontend/admin/vite.config.ts | grep port
   ```

3. 重新启动
   ```bash
   cd frontend/h5 && npm run dev &
   cd frontend/admin && npm run dev &
   ```

---

## 📝 更新记录

**2026-05-23**:
- 修正Admin后台端口从5174改为5175
- 添加端口配置说明文档
- 创建标准启动流程
- 添加故障排查指南

---

## 🎯 记住这个！

**永远记住**:
- **5173** = H5用户端 = "直播竞拍"
- **5175** = Admin后台 = "竞拍管理后台"

不要搞混！不要搞混！不要搞混！
