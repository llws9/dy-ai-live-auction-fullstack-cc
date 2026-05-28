# 🚀 快速部署指南

## 一键部署（推荐）

```bash
# 1. 进入项目目录
cd frontend/h5

# 2. 给脚本执行权限
chmod +x deploy.sh

# 3. 运行部署脚本
./deploy.sh production yourdomain.com
```

脚本会自动完成：
- ✅ 检查系统依赖
- ✅ 安装项目依赖
- ✅ 配置环境变量
- ✅ 运行测试
- ✅ 构建项目
- ✅ 上传到服务器
- ✅ 配置Nginx

---

## 手动部署（3步）

### 步骤1: 本地构建

```bash
# 安装依赖
npm install

# 构建生产版本
npm run build
```

构建产物位于 `dist/` 目录。

### 步骤2: 上传到服务器

**方式A: 使用rsync（推荐）**
```bash
rsync -avz dist/ user@yourserver:/var/www/auction-h5/
```

**方式B: 使用SCP**
```bash
scp -r dist/* user@yourserver:/var/www/auction-h5/
```

**方式C: 使用FTP/SFTP工具**
- FileZilla
- WinSCP
- Cyberduck

### 步骤3: 配置Nginx

```bash
# SSH到服务器
ssh user@yourserver

# 复制Nginx配置
sudo cp nginx.conf /etc/nginx/sites-available/auction-h5
sudo ln -s /etc/nginx/sites-available/auction-h5 /etc/nginx/sites-enabled/

# 测试并重载配置
sudo nginx -t
sudo systemctl reload nginx
```

---

## Docker部署

### 快速启动

```bash
# 构建镜像
docker build -t auction-h5 .

# 运行容器
docker run -d -p 80:80 auction-h5
```

### 使用Docker Compose

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

---

## 环境变量配置

创建 `.env.production` 文件：

```bash
# API配置
VITE_API_BASE_URL=https://api.yourdomain.com
VITE_WS_BASE_URL=wss://ws.yourdomain.com

# 应用配置
VITE_APP_TITLE=直播竞拍系统
NODE_ENV=production
```

---

## 常用命令

```bash
# 开发模式
npm run dev

# 构建项目
npm run build

# 预览构建
npm run preview

# 运行测试
npm test

# 运行E2E测试
npm run test:e2e
```

---

## 部署检查清单

部署前请确认：
- [ ] 代码已推送到主分支
- [ ] 所有测试通过
- [ ] 环境变量已配置
- [ ] 构建成功
- [ ] 文件已上传到服务器
- [ ] Nginx配置正确
- [ ] SSL证书有效
- [ ] 域名解析正确

部署后请验证：
- [ ] 网站可以正常访问
- [ ] API请求正常
- [ ] WebSocket连接正常
- [ ] 登录功能正常
- [ ] 出价功能正常
- [ ] 关注功能正常

---

## 故障排查

### 页面空白
```bash
# 检查Nginx日志
tail -f /var/log/nginx/error.log

# 检查文件权限
ls -la /var/www/auction-h5/
```

### API请求失败
```bash
# 检查后端服务
curl http://localhost:8080/api/v1/health

# 检查Nginx代理配置
sudo nginx -t
```

### WebSocket连接失败
```bash
# 检查WebSocket代理配置
grep -A 10 "location /ws/" /etc/nginx/sites-available/auction-h5

# 检查防火墙
sudo ufw status
```

---

## 获取帮助

详细部署文档请查看：
- 📖 [完整部署指南](./DEPLOYMENT_GUIDE.md)
- 📖 [项目文档](./README.md)
- 📖 [API文档](./docs/API_DOCUMENTATION.md)

---

**部署愉快！** 🎉
