# Demo Deployment Readme

## Goal

Deploy the MVP to a single Volcengine ECS instance using one public IP.

- H5 is served from `http://<PUBLIC_IP>/`
- Admin is served from `http://<PUBLIC_IP>/admin/`
- API stays behind `http://<PUBLIC_IP>/api/v1`
- WebSocket stays behind `ws://<PUBLIC_IP>/api/v1/ws`

## Server Info

- Public IP: `14.103.53.55`
- SSH user: `root`
- SSH private key: `/Users/bytedance/Downloads/dy-auction.pem`
- Domain: none; access the demo through the public IP.
- Repository changes: allowed to generate or update demo deployment files.
- Not deployed for now: `test-service`, `grafana`, `prometheus`, `growthbook`.

## Why This Layout

- `frontend/h5` and `frontend/admin` do not have production Dockerfiles in the repository.
- `frontend/admin` uses `HashRouter`, so it can safely live under `/admin/`.
- `backend/gateway` does not terminate WebSocket upgrades for `/api/v1/ws`, so Nginx forwards that path to `auction`'s dedicated WS port while keeping the public path unchanged.

## Files

- `docker-compose.demo.yml`: demo-only backend and middleware stack.
- `.env.demo.example`: runtime variables template.
- `deploy/demo/nginx-ip.conf`: Nginx config for the public-IP deployment.
- `backend/product/Dockerfile`: missing product image build entry.
- `backend/auction/Dockerfile`: missing auction image build entry.

## Local Build And Upload

Run these commands on the local machine:

```bash
cd /Users/bytedance/.config/superpowers/worktrees/dy-ai-live-auction-fullstack-cc/feat/demo-deploy
cp .env.demo.example .env.demo
# edit .env.demo before continuing
set -a
source ./.env.demo
set +a

mkdir -p /tmp/auction-demo-build/h5 /tmp/auction-demo-build/admin

cd frontend/h5
npm ci
VITE_API_BASE_URL="$APP_BASE_URL" \
VITE_GROWTHBOOK_API_HOST="$VITE_GROWTHBOOK_API_HOST" \
VITE_GROWTHBOOK_CLIENT_KEY="$VITE_GROWTHBOOK_CLIENT_KEY" \
npm run build
rsync -av --delete dist/ /tmp/auction-demo-build/h5/

cd ../admin
npm ci
VITE_GROWTHBOOK_API_HOST="$VITE_GROWTHBOOK_API_HOST" \
VITE_GROWTHBOOK_CLIENT_KEY="$VITE_GROWTHBOOK_CLIENT_KEY" \
npx vite build --base=/admin/
rsync -av --delete dist/ /tmp/auction-demo-build/admin/
```

`frontend/admin` currently contains pre-existing TypeScript test noise that blocks `npm run build`.
For this demo path, use `vite build --base=/admin/` directly; this bundle has been verified locally.

## Server Bootstrap

Run these commands on the ECS instance:

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

## Sync Project To Server

Run these commands on the local machine:

```bash
cd /Users/bytedance/.config/superpowers/worktrees/dy-ai-live-auction-fullstack-cc/feat/demo-deploy
rsync -av --delete \
  --exclude '.git' \
  --exclude 'node_modules' \
  --exclude '.DS_Store' \
  ./ root@14.103.53.55:/srv/auction/app/

rsync -av --delete /tmp/auction-demo-build/h5/ root@14.103.53.55:/var/www/auction-h5/
rsync -av --delete /tmp/auction-demo-build/admin/ root@14.103.53.55:/var/www/auction-admin/
scp .env.demo root@14.103.53.55:/srv/auction/env/.env.demo
scp deploy/demo/nginx-ip.conf root@14.103.53.55:/etc/nginx/sites-available/auction-demo.conf
```

## Start Services

Run these commands on the ECS instance:

```bash
ln -sf /etc/nginx/sites-available/auction-demo.conf /etc/nginx/sites-enabled/auction-demo.conf
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl reload nginx

cd /srv/auction/app
docker compose --env-file /srv/auction/env/.env.demo -f docker-compose.demo.yml up -d --build
docker compose -f docker-compose.demo.yml ps
```

## Smoke Checks

Run these commands on the ECS instance:

```bash
curl -I http://127.0.0.1:8080/health
curl -I http://127.0.0.1/
curl -I http://127.0.0.1/admin/
curl http://127.0.0.1:8080/health
docker compose -f /srv/auction/app/docker-compose.demo.yml logs --tail=100 gateway product auction
```

Run these commands on the local machine:

```bash
curl -I http://14.103.53.55/
curl -I http://14.103.53.55/admin/
curl http://14.103.53.55/api/v1/health
```
