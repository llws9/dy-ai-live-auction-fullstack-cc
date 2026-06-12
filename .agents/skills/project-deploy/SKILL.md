---
name: project-deploy
description: Deploys this repository locally or to demo prod. Invoke for /dp-dev, /dp-prod, local restart, local Docker Compose deployment, service connectivity checks, or demo deployment. Use this project-level skill whenever the user asks to restart local services, verify local deployment, push the demo environment, or diagnose deployment failures in this repository.
---

# Project Deploy Skill

## Scope

This is a project-level skill for the current repository root. Keep it under `.agents/skills/project-deploy` so deployment behavior, safety rules, and repository scripts evolve with this codebase.

It supports two commands:

- `/dp-dev`: strong local restart from `origin/main`.
- `/dp-prod`: demo production deployment to `14.103.53.55`, plan first, confirm before apply.

It also covers follow-up checks such as "本地服务是否都通", "重新部署本地服务", "看一下部署为什么失败", and "根据部署问题优化部署流程".

## Mandatory First Step

Always read:

- `AGENTS.md`
- `docs/superpowers/specs/2026-06-04-project-deploy-skill-design.md`
- `deploy/demo/MAIN_DEPLOY_QUICKSTART.md`

Then run these read-only commands:

```bash
git fetch origin main
git status --short --branch
git rev-parse HEAD
git rev-parse origin/main
```

Do not claim anything is synced until the command output confirms it.

Treat repository scripts as the execution SSOT. The skill explains, sequences, and verifies; it should not hand-roll deployment logic that already exists in:

- `scripts/deploy-dev.sh`
- `scripts/deploy-prod.sh`
- `scripts/test-deploy-dev-scripts.sh`

## `/dp-dev` Workflow

Use this when the user enters `/dp-dev`.

1. Explain that `/dp-dev` will force restart local project services.
2. Run read-only status:

```bash
scripts/deploy-dev.sh status
```

3. Treat `scripts/deploy-dev.sh status` as read-only display only. If status shows `HEAD != origin/main`, stop and ask the user to sync or use an isolated worktree.
4. Otherwise run:

```bash
scripts/deploy-dev.sh restart
```

5. Let the restart script precheck report ignored-local changes or block non-ignored local changes. Do not duplicate worktree classification in the skill.
6. Verify with:

```bash
scripts/deploy-dev.sh verify
```

7. If the user asks whether services are reachable, or if verification output is ambiguous, run a connectivity sweep:

```bash
INTERNAL_API_TOKEN=dev docker compose ps --format 'table {{.Service}}\t{{.Status}}\t{{.Ports}}'
for p in 3000 3001 3002 3003 3100 3200 3306 5672 6379 8080 8848 9091 15672; do nc -z localhost "$p" >/dev/null 2>&1 && echo "PORT $p OK" || echo "PORT $p FAIL"; done
curl -s -o /dev/null -w 'gateway /health %{http_code}\n' http://localhost:8080/health
curl -s -o /dev/null -w 'gateway /api/v1/live-streams %{http_code}\n' http://localhost:8080/api/v1/live-streams
curl -s -o /dev/null -w 'gateway /api/v1/auctions %{http_code}\n' 'http://localhost:8080/api/v1/auctions?page=1&page_size=1'
redis-cli -h localhost -p 6379 ping
INTERNAL_API_TOKEN=dev docker compose exec -T mysql mysqladmin ping -h 127.0.0.1 -uroot -proot
INTERNAL_API_TOKEN=dev docker compose exec -T rabbitmq rabbitmq-diagnostics -q ping
```

8. Report exact URLs:

- H5: `http://localhost:3000`
- Admin: `http://localhost:3001`
- Gateway API: `http://localhost:8080/api/v1`
- Auction WS through H5/Gateway path: `ws://localhost:3000/api/v1/ws`
- Direct Auction WS: `ws://localhost:8083/ws`
- Test Dashboard: `http://localhost:3003`
- Grafana: `http://localhost:3002`
- GrowthBook: `http://localhost:3200`
- Nacos: `http://localhost:8848/nacos`
- RabbitMQ Management: `http://localhost:15672`

Never stop a dev server after giving the user a preview URL unless the user explicitly asks to stop it.

## Local Deployment Invariants

These are lessons from real failures in this project. Apply them during `/dp-dev`, local restart, and local troubleshooting:

- Local deployment is Docker Compose based. Prefer `INTERNAL_API_TOKEN=dev docker compose ...` and `scripts/deploy-dev.sh` over ad hoc per-service starts.
- Frontend traffic must enter through the H5/Admin Nginx or `gateway-service` `/api/v1`; do not make the H5 frontend call backend sub-services directly.
- Do not change committed `localhost` or service-name configuration to work around local IPv6, stale processes, or port conflicts. Fix the local runtime state instead.
- Port cleanup must exclude Docker-owned listeners such as macOS `com.docker`/`Docker` processes. Killing these can disconnect Docker Desktop and make the entire stack unavailable.
- `docker-compose.yml` business containers should wait for infra `service_healthy` where relevant, especially MySQL and RabbitMQ. If routes are missing after startup, suspect startup races before changing route code.
- After `docker compose up -d`, remember that startup is asynchronous. Wait for infra readiness, key schema readiness such as the `users` table, and HTTP readiness before running seed scripts or declaring deployment healthy.
- H5 Docker/Nginx must proxy `/api/v1/ws` to the auction WebSocket service before generic `/api` handling. A `200 text/html` response on `/api/v1/ws` is a failed WebSocket proxy, not a healthy result.
- H5 production Nginx must not rewrite stale Vite dev-client requests such as `/@vite/client` to `index.html`; it should return `404`. `/` and `/index.html` should be `no-cache` so local rebuilds do not leave the browser running stale chunks.
- Historical dirty data can survive deployment. If the UI still shows impossible state after a backend fix, check whether the database already contains old duplicate records before assuming rebuild failed.

## Local Failure Triage

When `/dp-dev` or local verification fails, classify the failure before changing code:

1. Docker daemon/Desktop unavailable or killed.
   - Check `docker ps`.
   - If Docker is down, restart Docker Desktop; do not edit application config.
2. Port conflict.
   - Run `scripts/deploy-dev.sh status`.
   - If the listener is Docker-owned, do not kill it manually.
3. Container starts but route returns `404`.
   - Check `docker compose ps`.
   - Check route initialization and startup race logs.
   - Verify MySQL/RabbitMQ health and service dependency ordering.
4. H5 returns ErrorBoundary after rebuild.
   - Check browser/HTTP requests for stale `/@vite/client` or stale asset chunks.
   - Verify Nginx headers with `curl -I http://localhost:3000/` and `curl -I http://localhost:3000/@vite/client`.
5. API call through H5 fails.
   - Verify the same API through Gateway: `http://localhost:8080/api/v1/...`.
   - If Gateway works but H5 fails, inspect `frontend/h5/nginx/default.conf`.
6. WebSocket fails.
   - Verify H5 Nginx `/api/v1/ws` proxy and direct auction WS separately.

If a root cause matches one of these known classes, fix that layer. Do not patch around symptoms in unrelated code.

## `/dp-prod` Workflow

Use this when the user enters `/dp-prod`.

1. Run:

```bash
scripts/deploy-prod.sh plan
```

2. Summarize:

- target commit
- remote current commit
- changed areas
- expected actions
- verification commands
- rollback point

3. Ask the user for explicit confirmation before any online mutation.

Use this exact confirmation prompt:

```text
确认执行线上部署吗？回复“确认部署”后我才会执行 apply。
```

4. Only if the user replies exactly `确认部署`, run:

```bash
scripts/deploy-prod.sh apply
```

5. Run fresh verification:

```bash
scripts/deploy-prod.sh verify
```

6. Report success only if the apply and verify commands both exit with code `0`.

Production deployment should reuse the same root-cause discipline as local deployment. Do not use a single homepage `200` as proof of full deployment health; verify H5, Admin, API, Nginx, and backend containers.

## Safety Rules

- `/dp-prod` must never run `apply` before explicit confirmation.
- `/dp-prod` must never print `.env.demo`, `ARK_API_KEY`, `JWT_SECRET`, or `INTERNAL_API_TOKEN`.
- `/dp-prod` must not use `/api/v1/health` as the only health check.
- `/dp-dev` must not change source config to work around localhost, IPv6, or port conflicts.
- `/dp-dev` must not kill Docker-owned listeners while clearing ports.
- `/dp-dev` must not bypass Docker Compose health ordering by starting dependent business services manually.
- Do not use `git reset --hard`, `git checkout --`, or destructive cleanup unless the user explicitly approves.
- Do not silently discard local changes.
- Local changes whose paths match `.gitignore` are allowed for `/dp-dev` and `/dp-prod`; deploy scripts must report them as ignored-local changes, and the skill must not delete, reset, stash, or overwrite them.
- Local changes that do not match `.gitignore` must still block deployment in deploy script prechecks.
- `/dp-prod` backend source sync follows `.gitignore` filters so ignored-local files are not rsynced to the remote app; frontend `dist/` is synchronized only through explicit frontend sync steps.

## Failure Handling

If a command fails:

1. Read the error output.
2. Identify the failing layer: Git, local ports, Docker daemon, Compose health, seed data, SSH, build, rsync, Docker image build, Nginx, WebSocket proxy, or HTTP verification.
3. Report the exact failing command and root cause.
4. Do not continue to the next phase after a failed phase.
5. Prefer evidence from logs and direct probes over speculation:

```bash
INTERNAL_API_TOKEN=dev docker compose ps
INTERNAL_API_TOKEN=dev docker compose logs --tail=120 <service>
curl -I http://localhost:3000/
curl -I http://localhost:3000/@vite/client
curl -s http://localhost:8080/api/v1/live-streams | head -c 300
```

Use the minimum relevant subset; do not dump unrelated logs into the final answer.

## Completion Report

Final response must include:

- current branch and worktree first line, matching `AGENTS.md`
- command invoked: `/dp-dev` or `/dp-prod`
- target commit
- commands executed
- verification evidence
- remaining risks or follow-up actions

For local service reachability checks, include a compact table or bullet list with:

- service URL or port
- observed status such as HTTP code, `PONG`, `mysqld is alive`, or `Ping succeeded`
- any known caveat, for example Nacos root `/` may return `404` while `/nacos` is healthy
