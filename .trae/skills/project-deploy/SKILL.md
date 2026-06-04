---
name: project-deploy
description: Deploys this project locally or to demo prod. Invoke for /dp-dev, /dp-prod, local restart, or demo deployment.
---

# Project Deploy Skill

## Scope

This skill is only for `/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc`.

It supports two commands:

- `/dp-dev`: strong local restart from `origin/main`.
- `/dp-prod`: demo production deployment to `14.103.53.55`, plan first, confirm before apply.

## Mandatory First Step

Always read:

- `AGENTS.md`
- `docs/superpowers/specs/2026-06-04-project-deploy-skill-design.md`
- `deploy/demo/MAIN_DEPLOY_QUICKSTART.md`

Then run:

```bash
git fetch origin main
git status --short --branch
git rev-parse HEAD
git rev-parse origin/main
```

Do not claim anything is synced until the command output confirms it.

## `/dp-dev` Workflow

Use this when the user enters `/dp-dev`.

1. Explain that `/dp-dev` will force restart local project services.
2. Run read-only status:

```bash
scripts/deploy-dev.sh status
```

3. If status shows the working tree is not clean or HEAD is not `origin/main`, stop and ask whether to create an isolated worktree or sync the current tree.
4. If safe, run:

```bash
scripts/deploy-dev.sh restart
```

5. Verify with:

```bash
scripts/deploy-dev.sh verify
```

6. Report exact URLs:

- H5: `http://localhost:5173`
- Admin: `http://localhost:5175`
- Gateway API: `http://localhost:8080/api/v1`
- Auction WS: `ws://localhost:8083/ws`

Never stop a dev server after giving the user a preview URL unless the user explicitly asks to stop it.

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

## Safety Rules

- `/dp-prod` must never run `apply` before explicit confirmation.
- `/dp-prod` must never print `.env.demo`, `ARK_API_KEY`, `JWT_SECRET`, or `INTERNAL_API_TOKEN`.
- `/dp-prod` must not use `/api/v1/health` as the only health check.
- `/dp-dev` must not change source config to work around localhost, IPv6, or port conflicts.
- Do not use `git reset --hard`, `git checkout --`, or destructive cleanup unless the user explicitly approves.
- Do not silently discard local changes.

## Failure Handling

If a command fails:

1. Read the error output.
2. Identify the failing layer: Git, local ports, SSH, build, rsync, Docker, Nginx, or HTTP verification.
3. Report the exact failing command and root cause.
4. Do not continue to the next phase after a failed phase.

## Completion Report

Final response must include:

- current branch and worktree first line, matching `AGENTS.md`
- command invoked: `/dp-dev` or `/dp-prod`
- target commit
- commands executed
- verification evidence
- remaining risks or follow-up actions
