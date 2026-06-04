# 当前 `main` 分支精简部署手册

> 目标：把当前 `main` 分支部署到火山引擎 ECS demo 环境，并支持后续重复发布。

## 1. 部署形态

- 服务器：单台火山引擎 ECS
- 入口：
  - H5：`http://<PUBLIC_IP>/`
  - Admin：`http://<PUBLIC_IP>/admin/`
  - API：`http://<PUBLIC_IP>/api/v1`
  - WebSocket：`ws://<PUBLIC_IP>/api/v1/ws`
- 前端：本地构建静态产物，Nginx 托管
- 后端：`docker-compose.demo.yml`
- 当前 demo 服务器：`14.103.53.55`
- 登录用户：`root`
- SSH 私钥路径：`/Users/bytedance/Downloads/dy-auction.pem`
- 域名：无，当前通过公网 IP 访问
- 部署文件：允许修改仓库生成 demo 部署文件
- 暂不部署：`test-service`、`grafana`、`prometheus`、`growthbook`

## 2. 仓库内关键文件

- 编排文件：`docker-compose.demo.yml`
- Nginx 配置：`deploy/demo/nginx-ip.conf`
- 环境变量模板：`.env.demo.example`
- 后端镜像入口：
  - `backend/gateway/Dockerfile`
  - `backend/product/Dockerfile`
  - `backend/auction/Dockerfile`

## 3. 服务器目录约定

- 代码目录：`/srv/auction/app`
- 环境变量：`/srv/auction/env/.env.demo`
- H5 静态文件：`/var/www/auction-h5`
- Admin 静态文件：`/var/www/auction-admin`
- Nginx 配置：`/etc/nginx/sites-available/auction-demo.conf`

## 4. 首次部署

### 4.1 本地准备 `.env.demo`

在仓库根目录执行：

```bash
cp .env.demo.example .env.demo
```

至少填写这些值：

- `APP_PUBLIC_HOST`
- `APP_BASE_URL`
- `DB_PASSWORD`
- `JWT_SECRET`
- `INTERNAL_API_TOKEN`
- `ARK_API_KEY`

### 4.2 服务器初始化

登录 ECS 后执行：

```bash
apt update && apt upgrade -y
apt install -y git curl ca-certificates gnupg lsb-release nginx rsync
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo $VERSION_CODENAME) stable" \
  | tee /etc/apt/sources.list.d/docker.list >/dev/null
apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
systemctl enable docker --now
systemctl enable nginx --now
mkdir -p /srv/auction/app /srv/auction/env /var/www/auction-h5 /var/www/auction-admin
```

### 4.3 本地构建前端

先导入环境变量：

```bash
set -a
source ./.env.demo
set +a
mkdir -p /tmp/auction-demo-build/h5 /tmp/auction-demo-build/admin
```

构建 H5：

```bash
cd frontend/h5
npm ci
VITE_API_BASE_URL="$APP_BASE_URL" \
VITE_GROWTHBOOK_API_HOST="$VITE_GROWTHBOOK_API_HOST" \
VITE_GROWTHBOOK_CLIENT_KEY="$VITE_GROWTHBOOK_CLIENT_KEY" \
npm run build
rsync -av --delete dist/ /tmp/auction-demo-build/h5/
```

构建 Admin：

```bash
cd ../admin
npm ci
VITE_GROWTHBOOK_API_HOST="$VITE_GROWTHBOOK_API_HOST" \
VITE_GROWTHBOOK_CLIENT_KEY="$VITE_GROWTHBOOK_CLIENT_KEY" \
npx vite build --base=/admin/
rsync -av --delete dist/ /tmp/auction-demo-build/admin/
```

说明：

- `frontend/admin` 当前不要用 `npm run build`，仓库里有存量 TS 噪音。
- demo 路径使用 `npx vite build --base=/admin/`。

### 4.4 同步代码与静态资源

在本地执行：

```bash
rsync -av --delete \
  --exclude '.git' \
  --exclude 'node_modules' \
  --exclude '.DS_Store' \
  ./ root@<PUBLIC_IP>:/srv/auction/app/

rsync -av --delete /tmp/auction-demo-build/h5/ root@<PUBLIC_IP>:/var/www/auction-h5/
rsync -av --delete /tmp/auction-demo-build/admin/ root@<PUBLIC_IP>:/var/www/auction-admin/
scp .env.demo root@<PUBLIC_IP>:/srv/auction/env/.env.demo
scp deploy/demo/nginx-ip.conf root@<PUBLIC_IP>:/etc/nginx/sites-available/auction-demo.conf
```

### 4.5 启动 Nginx 与后端容器

在 ECS 上执行：

```bash
ln -sf /etc/nginx/sites-available/auction-demo.conf /etc/nginx/sites-enabled/auction-demo.conf
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl reload nginx

cd /srv/auction/app
docker compose --env-file /srv/auction/env/.env.demo -f docker-compose.demo.yml up -d --build
docker compose --env-file /srv/auction/env/.env.demo -f docker-compose.demo.yml ps
```

## 5. 日常增量发布

### 5.1 仅 H5 前端改动

适用场景：只改了 `frontend/h5`，例如样式、文案、前端逻辑。

```bash
cd frontend/h5
npm run build
rsync -av --delete -e 'ssh -i <SSH_KEY> -o StrictHostKeyChecking=no' \
  dist/ root@<PUBLIC_IP>:/var/www/auction-h5/
```

这是当前 `main` 最近一次发布实际使用的最短路径。

### 5.2 H5 + Admin 前端改动

```bash
set -a
source ./.env.demo
set +a

mkdir -p /tmp/auction-demo-build/h5 /tmp/auction-demo-build/admin

cd frontend/h5
npm run build
rsync -av --delete dist/ /tmp/auction-demo-build/h5/

cd ../admin
VITE_GROWTHBOOK_API_HOST="$VITE_GROWTHBOOK_API_HOST" \
VITE_GROWTHBOOK_CLIENT_KEY="$VITE_GROWTHBOOK_CLIENT_KEY" \
npx vite build --base=/admin/
rsync -av --delete dist/ /tmp/auction-demo-build/admin/

rsync -av --delete /tmp/auction-demo-build/h5/ root@<PUBLIC_IP>:/var/www/auction-h5/
rsync -av --delete /tmp/auction-demo-build/admin/ root@<PUBLIC_IP>:/var/www/auction-admin/
```

### 5.3 后端或配置改动

适用场景：改了 `backend/*`、`docker-compose.demo.yml`、`scripts/init.sql`、`deploy/demo/nginx-ip.conf`。

```bash
rsync -av --delete \
  --exclude '.git' \
  --exclude 'node_modules' \
  --exclude '.DS_Store' \
  ./ root@<PUBLIC_IP>:/srv/auction/app/

scp deploy/demo/nginx-ip.conf root@<PUBLIC_IP>:/etc/nginx/sites-available/auction-demo.conf
scp .env.demo root@<PUBLIC_IP>:/srv/auction/env/.env.demo

ssh -i <SSH_KEY> root@<PUBLIC_IP> '
  nginx -t &&
  systemctl reload nginx &&
  cd /srv/auction/app &&
  docker compose --env-file /srv/auction/env/.env.demo -f docker-compose.demo.yml up -d --build &&
  docker compose --env-file /srv/auction/env/.env.demo -f docker-compose.demo.yml ps
'
```

## 6. 验证清单

本地执行：

```bash
curl -I http://<PUBLIC_IP>/
curl -I http://<PUBLIC_IP>/admin/
curl http://<PUBLIC_IP>/api/v1/products | head -c 300
```

服务器执行：

```bash
cd /srv/auction/app
docker compose --env-file /srv/auction/env/.env.demo -f docker-compose.demo.yml ps
docker compose --env-file /srv/auction/env/.env.demo -f docker-compose.demo.yml logs --tail=100 gateway product auction
```

额外建议：

- 前端发布后检查首页是否已经引用新的资源 hash。
- 不要用 `/api/v1/health` 作为唯一探针，当前线上路由不一定有这条接口。
- 更稳妥的是直接验证业务 API，如 `/api/v1/products`。

## 7. 回滚方法

### 7.1 前端回滚

- 如果你保留了上一版 `dist` 备份，直接重新 `rsync` 回 `/var/www/auction-h5` 或 `/var/www/auction-admin`。
- 如果没有备份，切回上一条 Git commit，重新本地构建并同步静态目录。

### 7.2 后端回滚

```bash
git checkout <old_commit>
rsync -av --delete \
  --exclude '.git' \
  --exclude 'node_modules' \
  --exclude '.DS_Store' \
  ./ root@<PUBLIC_IP>:/srv/auction/app/

ssh -i <SSH_KEY> root@<PUBLIC_IP> '
  cd /srv/auction/app &&
  docker compose --env-file /srv/auction/env/.env.demo -f docker-compose.demo.yml up -d --build
'
```

## 8. 当前复用原则

- 原则 1：前端优先走静态发布，不要为了小改动重建整套服务。
- 原则 2：只要后端代码没变，就不要无谓重启 `mysql/redis/rabbitmq`。
- 原则 3：`Admin` 继续使用 `/admin/` 子路径构建。
- 原则 4：WebSocket 公开路径保持 `/api/v1/ws`，由 Nginx 转发到 `auction` 的 WS 端口。
- 原则 5：真实环境变量只放 `/srv/auction/env/.env.demo`，不要把真实值提交到仓库。
