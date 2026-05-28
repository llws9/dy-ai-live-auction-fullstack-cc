# 直播竞拍系统 - 前端H5部署指南

## 目录

1. [环境准备](#环境准备)
2. [本地构建](#本地构建)
3. [部署方式](#部署方式)
4. [Nginx部署](#nginx部署)
5. [Docker部署](#docker部署)
6. [CI/CD配置](#cicd配置)
7. [环境变量配置](#环境变量配置)
8. [监控和日志](#监控和日志)
9. [性能优化](#性能优化)
10. [故障排查](#故障排查)

---

## 环境准备

### 系统要求

- **Node.js**: v18.0.0 或更高版本
- **npm**: v9.0.0 或更高版本
- **操作系统**: Linux (推荐 Ubuntu 20.04+), macOS, Windows
- **内存**: 至少 2GB RAM
- **磁盘空间**: 至少 1GB 可用空间

### 安装Node.js

#### Ubuntu/Debian
```bash
# 使用 NodeSource 安装 Node.js 18
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# 验证安装
node --version
npm --version
```

#### macOS
```bash
# 使用 Homebrew 安装
brew install node@18

# 验证安装
node --version
npm --version
```

#### CentOS/RHEL
```bash
# 使用 NodeSource 安装
curl -fsSL https://rpm.nodesource.com/setup_18.x | sudo bash -
sudo yum install -y nodejs

# 验证安装
node --version
npm --version
```

---

## 本地构建

### 1. 克隆代码

```bash
# 克隆仓库
git clone <repository-url>
cd dy-ai-live-auction-fullstack-cc/frontend/h5
```

### 2. 安装依赖

```bash
# 安装项目依赖
npm install

# 或使用 yarn
yarn install
```

### 3. 配置环境变量

创建 `.env.production` 文件：

```bash
# API基础URL
VITE_API_BASE_URL=https://api.yourdomain.com

# WebSocket基础URL
VITE_WS_BASE_URL=wss://ws.yourdomain.com

# 应用标题
VITE_APP_TITLE=直播竞拍系统

# 环境
NODE_ENV=production
```

### 4. 构建项目

```bash
# 执行构建
npm run build

# 构建产物在 dist/ 目录
```

### 5. 预览构建结果

```bash
# 本地预览构建结果
npm run preview

# 访问 http://localhost:4173
```

---

## 部署方式

### 方式一：静态文件部署（推荐）

#### 优点
- 简单快速
- 成本低
- 性能好
- 易于CDN加速

#### 适用场景
- 生产环境
- 流量较大的应用

### 方式二：Docker容器部署

#### 优点
- 环境一致
- 易于扩展
- 便于CI/CD
- 支持Kubernetes

#### 适用场景
- 容器化环境
- 微服务架构
- 云原生应用

### 方式三：对象存储 + CDN

#### 优点
- 全球加速
- 高可用性
- 自动扩展
- 成本优化

#### 适用场景
- 全球用户
- 流量波动大
- 静态资源为主

---

## Nginx部署

### 1. 安装Nginx

#### Ubuntu/Debian
```bash
sudo apt update
sudo apt install nginx
```

#### CentOS/RHEL
```bash
sudo yum install epel-release
sudo yum install nginx
```

### 2. 配置Nginx

创建配置文件 `/etc/nginx/sites-available/auction-h5`:

```nginx
server {
    listen 80;
    server_name yourdomain.com www.yourdomain.com;

    # 重定向到HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com www.yourdomain.com;

    # SSL证书配置
    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;

    # SSL优化配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # 根目录
    root /var/www/auction-h5/dist;
    index index.html;

    # Gzip压缩
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/json application/xml+rss;

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        access_log off;
    }

    # 主页面不缓存
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        add_header Pragma "no-cache";
        add_header Expires 0;
    }

    # API代理
    location /api/ {
        proxy_pass http://backend-server:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;

        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # WebSocket代理
    location /ws/ {
        proxy_pass http://backend-server:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket超时设置
        proxy_connect_timeout 7d;
        proxy_send_timeout 7d;
        proxy_read_timeout 7d;
    }

    # SPA路由支持
    location / {
        try_files $uri $uri/ /index.html;
    }

    # 错误页面
    error_page 404 /index.html;
    error_page 500 502 503 504 /50x.html;
    location = /50x.html {
        root /usr/share/nginx/html;
    }

    # 安全头部
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:;" always;

    # 访问日志
    access_log /var/log/nginx/auction-h5-access.log;
    error_log /var/log/nginx/auction-h5-error.log;
}
```

### 3. 启用配置

```bash
# 创建软链接
sudo ln -s /etc/nginx/sites-available/auction-h5 /etc/nginx/sites-enabled/

# 测试配置
sudo nginx -t

# 重启Nginx
sudo systemctl restart nginx

# 设置开机自启
sudo systemctl enable nginx
```

### 4. 部署静态文件

```bash
# 创建部署目录
sudo mkdir -p /var/www/auction-h5

# 复制构建产物
sudo cp -r dist/* /var/www/auction-h5/

# 设置权限
sudo chown -R www-data:www-data /var/www/auction-h5
sudo chmod -R 755 /var/www/auction-h5
```

---

## Docker部署

### 1. 创建Dockerfile

创建 `Dockerfile`:

```dockerfile
# 构建阶段
FROM node:18-alpine as builder

WORKDIR /app

# 复制package文件
COPY package*.json ./

# 安装依赖
RUN npm ci --only=production

# 复制源代码
COPY . .

# 构建应用
RUN npm run build

# 生产阶段
FROM nginx:alpine

# 复制自定义Nginx配置
COPY nginx.conf /etc/nginx/nginx.conf

# 复制构建产物
COPY --from=builder /app/dist /usr/share/nginx/html

# 暴露端口
EXPOSE 80

# 启动Nginx
CMD ["nginx", "-g", "daemon off;"]
```

### 2. 创建Nginx配置

创建 `nginx.conf`:

```nginx
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;

    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/json application/xml+rss;

    server {
        listen 80;
        server_name localhost;

        root /usr/share/nginx/html;
        index index.html;

        # 静态资源缓存
        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }

        # SPA路由
        location / {
            try_files $uri $uri/ /index.html;
        }

        # API代理（如果需要）
        location /api/ {
            proxy_pass http://backend:8080;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host $host;
            proxy_cache_bypass $http_upgrade;
        }

        # WebSocket代理
        location /ws/ {
            proxy_pass http://backend:8080;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
    }
}
```

### 3. 构建Docker镜像

```bash
# 构建镜像
docker build -t auction-h5:latest .

# 查看镜像
docker images | grep auction-h5
```

### 4. 运行容器

```bash
# 运行容器
docker run -d \
  --name auction-h5 \
  -p 80:80 \
  --restart unless-stopped \
  auction-h5:latest

# 查看容器状态
docker ps

# 查看日志
docker logs auction-h5
```

### 5. Docker Compose部署

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  frontend:
    image: auction-h5:latest
    container_name: auction-h5
    ports:
      - "80:80"
    restart: unless-stopped
    networks:
      - auction-network
    depends_on:
      - backend
    environment:
      - NODE_ENV=production

  backend:
    image: auction-backend:latest
    container_name: auction-backend
    ports:
      - "8080:8080"
    restart: unless-stopped
    networks:
      - auction-network
    environment:
      - DB_HOST=db
      - DB_PORT=5432
      - DB_NAME=auction
      - DB_USER=admin
      - DB_PASSWORD=password

  db:
    image: postgres:13
    container_name: auction-db
    restart: unless-stopped
    networks:
      - auction-network
    environment:
      - POSTGRES_DB=auction
      - POSTGRES_USER=admin
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres-data:/var/lib/postgresql/data

networks:
  auction-network:
    driver: bridge

volumes:
  postgres-data:
```

运行：

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f frontend
```

---

## CI/CD配置

### GitHub Actions

创建 `.github/workflows/deploy.yml`:

```yaml
name: Deploy to Production

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest

    steps:
    - name: Checkout code
      uses: actions/checkout@v3

    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '18'
        cache: 'npm'
        cache-dependency-path: frontend/h5/package-lock.json

    - name: Install dependencies
      working-directory: frontend/h5
      run: npm ci

    - name: Run tests
      working-directory: frontend/h5
      run: npm test

    - name: Build application
      working-directory: frontend/h5
      run: npm run build
      env:
        VITE_API_BASE_URL: ${{ secrets.API_BASE_URL }}
        VITE_WS_BASE_URL: ${{ secrets.WS_BASE_URL }}

    - name: Deploy to server
      if: github.ref == 'refs/heads/main'
      uses: appleboy/scp-action@master
      with:
        host: ${{ secrets.SERVER_HOST }}
        username: ${{ secrets.SERVER_USER }}
        key: ${{ secrets.SSH_PRIVATE_KEY }}
        source: "frontend/h5/dist/*"
        target: "/var/www/auction-h5"
        strip_components: 3

    - name: Restart Nginx
      if: github.ref == 'refs/heads/main'
      uses: appleboy/ssh-action@master
      with:
        host: ${{ secrets.SERVER_HOST }}
        username: ${{ secrets.SERVER_USER }}
        key: ${{ secrets.SSH_PRIVATE_KEY }}
        script: |
          sudo nginx -t
          sudo systemctl reload nginx
```

### GitLab CI/CD

创建 `.gitlab-ci.yml`:

```yaml
stages:
  - install
  - test
  - build
  - deploy

variables:
  NODE_VERSION: "18"

cache:
  paths:
    - frontend/h5/node_modules/

install:
  stage: install
  image: node:18-alpine
  script:
    - cd frontend/h5
    - npm ci
  artifacts:
    paths:
      - frontend/h5/node_modules/
    expire_in: 1 hour

test:
  stage: test
  image: node:18-alpine
  script:
    - cd frontend/h5
    - npm test
  dependencies:
    - install

build:
  stage: build
  image: node:18-alpine
  script:
    - cd frontend/h5
    - npm run build
  artifacts:
    paths:
      - frontend/h5/dist/
    expire_in: 1 week
  dependencies:
    - install

deploy_production:
  stage: deploy
  image: alpine:latest
  script:
    - apk add --no-cache rsync openssh-client
    - eval $(ssh-agent -s)
    - echo "$SSH_PRIVATE_KEY" | tr -d '\r' | ssh-add -
    - mkdir -p ~/.ssh
    - chmod 700 ~/.ssh
    - ssh-keyscan $SERVER_HOST >> ~/.ssh/known_hosts
    - rsync -avz --delete frontend/h5/dist/ $SERVER_USER@$SERVER_HOST:/var/www/auction-h5/
    - ssh $SERVER_USER@$SERVER_HOST "sudo systemctl reload nginx"
  only:
    - main
  environment:
    name: production
    url: https://yourdomain.com
```

---

## 环境变量配置

### 开发环境 (.env.development)

```bash
# API配置
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_BASE_URL=ws://localhost:8080

# 应用配置
VITE_APP_TITLE=直播竞拍系统 - 开发环境
NODE_ENV=development
```

### 测试环境 (.env.staging)

```bash
# API配置
VITE_API_BASE_URL=https://api-staging.yourdomain.com
VITE_WS_BASE_URL=wss://ws-staging.yourdomain.com

# 应用配置
VITE_APP_TITLE=直播竞拍系统 - 测试环境
NODE_ENV=staging
```

### 生产环境 (.env.production)

```bash
# API配置
VITE_API_BASE_URL=https://api.yourdomain.com
VITE_WS_BASE_URL=wss://ws.yourdomain.com

# 应用配置
VITE_APP_TITLE=直播竞拍系统
NODE_ENV=production
```

---

## 监控和日志

### 1. 应用监控

#### 使用PM2（Node.js应用）

```bash
# 安装PM2
npm install -g pm2

# 创建 ecosystem.config.js
module.exports = {
  apps: [{
    name: 'auction-h5',
    script: 'npm',
    args: 'run preview',
    env_production: {
      NODE_ENV: 'production',
      PORT: 4173
    }
  }]
}

# 启动应用
pm2 start ecosystem.config.js --env production

# 查看状态
pm2 status

# 查看日志
pm2 logs auction-h5
```

### 2. 性能监控

#### 使用New Relic或Sentry

```bash
# 安装Sentry
npm install @sentry/react

# 在main.tsx中配置
import * as Sentry from '@sentry/react';

Sentry.init({
  dsn: 'YOUR_SENTRY_DSN',
  environment: process.env.NODE_ENV,
  tracesSampleRate: 1.0,
});
```

### 3. 日志管理

#### Nginx日志轮转

创建 `/etc/logrotate.d/auction-h5`:

```
/var/log/nginx/auction-h5-*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 www-data adm
    sharedscripts
    postrotate
        [ -f /var/run/nginx.pid ] && kill -USR1 `cat /var/run/nginx.pid`
    endscript
}
```

---

## 性能优化

### 1. 启用HTTP/2

```nginx
server {
    listen 443 ssl http2;
    # ... 其他配置
}
```

### 2. 启用Brotli压缩

```nginx
# 安装ngx_brotli模块后
brotli on;
brotli_comp_level 6;
brotli_types text/plain text/css text/xml text/javascript application/javascript application/json;
```

### 3. CDN加速

配置CDN（如CloudFlare、阿里云CDN）：

```bash
# 静态资源上传到CDN
# 修改Vite配置，将publicPath指向CDN
```

### 4. 图片优化

```bash
# 使用WebP格式
# 配置Nginx自动提供WebP
map $http_accept $webp_suffix {
    default "";
    "~*webp" ".webp";
}

server {
    location ~* ^/.+\.(png|jpg|jpeg)$ {
        add_header Vary Accept;
        try_files $uri$webp_suffix $uri =404;
    }
}
```

---

## 故障排查

### 常见问题

#### 1. 页面空白

```bash
# 检查控制台错误
# 查看Nginx错误日志
tail -f /var/log/nginx/auction-h5-error.log

# 检查文件权限
ls -la /var/www/auction-h5/

# 检查路由配置
# 确保Nginx配置了 try_files $uri $uri/ /index.html;
```

#### 2. API请求失败

```bash
# 检查API代理配置
# 测试后端连接
curl http://backend-server:8080/api/v1/health

# 检查CORS配置
# 在Nginx中添加CORS头部
add_header Access-Control-Allow-Origin *;
add_header Access-Control-Allow-Methods 'GET, POST, PUT, DELETE, OPTIONS';
```

#### 3. WebSocket连接失败

```bash
# 检查WebSocket代理配置
# 确保Nginx配置了正确的Upgrade头部
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";

# 检查防火墙规则
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

#### 4. 构建失败

```bash
# 清除缓存重新安装
rm -rf node_modules package-lock.json
npm install

# 检查Node.js版本
node --version  # 应该是18+

# 检查内存限制
npm run build --max-old-space-size=4096
```

### 性能问题

#### 1. 加载慢

```bash
# 启用Gzip压缩
# 使用CDN加速
# 优化图片大小
# 启用浏览器缓存
```

#### 2. 内存占用高

```bash
# 检查是否有内存泄漏
# 使用Chrome DevTools分析
# 优化组件卸载逻辑
```

---

## 安全加固

### 1. SSL/TLS配置

```bash
# 使用Let's Encrypt免费证书
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com

# 自动续期
sudo certbot renew --dry-run
```

### 2. 安全头部

已在Nginx配置中添加：
- X-Frame-Options
- X-Content-Type-Options
- X-XSS-Protection
- Referrer-Policy
- Content-Security-Policy

### 3. 防火墙配置

```bash
# 安装UFW
sudo apt install ufw

# 配置规则
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 'Nginx Full'

# 启用防火墙
sudo ufw enable
```

---

## 备份策略

### 1. 代码备份

```bash
#!/bin/bash
# backup.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/auction-h5"

mkdir -p $BACKUP_DIR

# 备份代码
tar -czf $BACKUP_DIR/auction-h5-$DATE.tar.gz /var/www/auction-h5

# 保留最近30天的备份
find $BACKUP_DIR -name "auction-h5-*.tar.gz" -mtime +30 -delete

echo "Backup completed: auction-h5-$DATE.tar.gz"
```

### 2. 自动备份

```bash
# 添加到crontab
crontab -e

# 每天凌晨2点备份
0 2 * * * /path/to/backup.sh >> /var/log/backup.log 2>&1
```

---

## 部署检查清单

### 部署前
- [ ] 代码已提交到main分支
- [ ] 所有测试通过
- [ ] 环境变量配置正确
- [ ] 依赖版本已锁定
- [ ] 构建产物已验证

### 部署中
- [ ] 备份现有版本
- [ ] 上传新版本文件
- [ ] 更新Nginx配置（如有变更）
- [ ] 重启/重载服务
- [ ] 验证服务状态

### 部署后
- [ ] 功能测试通过
- [ ] 性能监控正常
- [ ] 错误日志无异常
- [ ] SSL证书有效
- [ ] 备份策略就绪

---

## 联系支持

如有部署问题，请检查：

1. **日志文件**
   - Nginx: `/var/log/nginx/auction-h5-error.log`
   - 系统: `/var/log/syslog`

2. **监控面板**
   - 服务器资源使用情况
   - 应用性能指标
   - 错误率统计

3. **文档资源**
   - 项目README.md
   - API文档
   - 架构设计文档

---

**部署指南版本**: v1.0.0  
**最后更新**: 2026-05-23  
**维护者**: 开发团队
