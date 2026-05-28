# 故障排查指南

## 🔍 登录问题

### 问题1: 登录显示"网络错误，请重试"

**原因**: Admin前端API代理配置错误

**检查步骤**:
```bash
# 1. 检查vite.config.ts配置
cat frontend/admin/vite.config.ts | grep "target"

# 应该显示:
# target: 'http://localhost:8080',  // Gateway端口

# 2. 如果不是8080，需要修改
```

**解决方案**:
```bash
# 修改配置文件
vim frontend/admin/vite.config.ts

# 将 target 改为 'http://localhost:8080'

# 重启Admin前端
pkill -f "frontend/admin"
cd frontend/admin && npm run dev
```

---

### 问题2: 登录成功但显示"非管理员账号"

**原因**: 用户role不是2（平台管理员）

**检查步骤**:
```bash
# 查询用户role
docker exec -i mysql mysql -u root -proot auction <<EOF
SELECT id, name, email, role FROM users WHERE email = 'admin@example.com';
EOF

# role应该是2
```

**解决方案**:
```bash
# 更新用户role
docker exec -i mysql mysql -u root -proot auction <<EOF
UPDATE users SET role = 2 WHERE email = 'admin@example.com';
EOF
```

---

## 🔍 服务启动问题

### 问题3: 端口被占用

**检查**:
```bash
# 检查端口占用
lsof -i :5175  # Admin前端
lsof -i :8080  # Gateway
lsof -i :8082  # Auction Service
```

**解决方案**:
```bash
# 停止占用端口的进程
kill -9 <PID>

# 或停止所有node进程
pkill -f node

# 或停止所有go进程
pkill -f "go run"
```

---

### 问题4: 数据库连接失败

**检查**:
```bash
# 检查MySQL容器状态
docker ps | grep mysql

# 应该显示容器运行中
```

**解决方案**:
```bash
# 启动MySQL
docker-compose up -d mysql

# 或重启
docker-compose restart mysql
```

---

### 问题5: Auction Service连接错误的数据库

**症状**: 登录API返回的数据role不正确

**原因**: Auction Service默认连接"auction"数据库，但实际应该连接"auction"

**解决方案**:
```bash
# 停止auction服务
pkill -f "auction"

# 用正确的环境变量启动
cd backend/auction
DB_NAME=auction go run main.go
```

---

## 🔍 权限问题

### 问题6: 无法访问统计API

**症状**: 访问 `/api/v1/statistics/*` 返回403

**原因**: 统计API需要管理员权限（role=2）

**检查**:
```bash
# 获取token后解码查看role
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}' | jq '.data.user.role'

# 应该返回 2
```

---

## 🔍 前端问题

### 问题7: 页面空白或加载不出来

**检查步骤**:
```bash
# 1. 检查前端服务是否运行
lsof -i :5175

# 2. 检查控制台错误
# 打开浏览器开发者工具 (F12) -> Console

# 3. 检查网络请求
# 开发者工具 -> Network
```

**解决方案**:
```bash
# 清除依赖重新安装
cd frontend/admin
rm -rf node_modules package-lock.json
npm install
npm run dev
```

---

### 问题8: WebSocket连接失败

**症状**: 竞拍页面无法实时更新

**检查**:
```bash
# 检查WebSocket服务
lsof -i :8083
```

**解决方案**:
```bash
# 确保Auction Service的WebSocket端口启动
cd backend/auction
go run main.go

# 检查日志中是否有:
# "Auction service starting on :8082 (HTTP) and :8083 (WebSocket)"
```

---

## 🔍 数据问题

### 问题9: 用户名显示乱码

**原因**: 数据库字符编码问题

**解决方案**:
```bash
# 更新用户名
docker exec -i mysql mysql -u root -proot auction <<EOF
UPDATE users SET name = '系统管理员' WHERE id = 999;
EOF
```

---

### 问题10: 管理员密码错误

**症状**: 登录返回"密码错误"

**解决方案**:
```bash
# 重置密码为 admin123
docker exec -i mysql mysql -u root -proot auction <<EOF
UPDATE users
SET password = '\$2a\$10\$P767XYZO9ntqyLwbxTfwnOM0HEvWp0b/RJdojahFCvGojW4joj3vS'
WHERE id = 999;
EOF
```

---

## 📚 快速诊断命令

### 检查所有服务状态
```bash
# 前端服务
lsof -i :5173 && echo "✅ H5用户端运行中" || echo "❌ H5用户端未运行"
lsof -i :5175 && echo "✅ Admin后台运行中" || echo "❌ Admin后台未运行"

# 后端服务
lsof -i :8080 && echo "✅ Gateway运行中" || echo "❌ Gateway未运行"
lsof -i :8081 && echo "✅ Product Service运行中" || echo "❌ Product Service未运行"
lsof -i :8082 && echo "✅ Auction Service运行中" || echo "❌ Auction Service未运行"

# 基础设施
docker ps | grep mysql && echo "✅ MySQL运行中" || echo "❌ MySQL未运行"
docker ps | grep redis && echo "✅ Redis运行中" || echo "❌ Redis未运行"
```

### 测试API端点
```bash
# 测试Gateway
curl -s http://localhost:8080/health | jq '.'

# 测试登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}' | jq '.code, .message'
```

---

## 🆘 还是不行？

1. **完全重启所有服务**:
```bash
# 停止所有
pkill -f node
pkill -f "go run"
docker-compose down

# 重启
docker-compose up -d mysql redis
cd backend/auction && DB_NAME=auction go run main.go &
cd backend/gateway && go run main.go &
cd frontend/admin && npm run dev &
cd frontend/h5 && npm run dev &
```

2. **查看日志**:
```bash
tail -f /tmp/auction.log
tail -f /tmp/admin-auction.log
tail -f /tmp/h5-auction.log
```

3. **检查配置**:
   - Admin前端API代理：`frontend/admin/vite.config.ts` -> target: 'http://localhost:8080'
   - H5前端API代理：`frontend/h5/vite.config.ts` -> target: 'http://localhost:8080'
   - Auction Service数据库：环境变量 `DB_NAME=auction`

---

**最后更新**: 2026-05-28
