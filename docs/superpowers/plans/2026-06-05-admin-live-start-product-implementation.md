# Admin Live Start Product Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将管理端开播入口从工作台手输直播间 ID 改为商家在直播间详情页对当前直播间执行“开始直播”，并用一期说明文案澄清 PC 触发的是直播业务状态。

**Architecture:** 本期不新增后端接口，继续使用 `POST /api/v1/live-streams/:id/start`。前端只调整 Admin 端页面职责：`Dashboard` 不再承担开播入口，`LiveDetail` 根据登录角色展示商家经营动作或管理员治理动作，并在商家开播前展示确认文案。

**Tech Stack:** React 18, TypeScript, React Router, Jest, Testing Library, existing `liveStreamApi`, existing `AuthProvider`.

---

## Source Documents

- Spec: `docs/superpowers/specs/2026-06-05-admin-live-start-product-design.md`
- Tasks: `docs/superpowers/plans/2026-06-05-admin-live-start-product-tasks.md`
- State: `docs/superpowers/sdd/runs/2026-06-05-admin-live-start-product-state.md`

## File Scope

- Modify: `frontend/admin/src/pages-new/LiveDetail.tsx`
  - Add merchant-only `开始直播` action for the current `liveStreamId`.
  - Add phase-one copy explaining PC-triggered business live state.
  - Keep admin-only governance actions (`封禁直播间`, `关闭直播`) separated from merchant operations.
- Modify: `frontend/admin/src/pages-new/Dashboard.tsx`
  - Remove Dashboard `window.prompt` live start flow and unused imports/API dependency.
  - Keep `发布商品` for merchants.
- Modify: `frontend/admin/src/pages-new/__tests__/Dashboard.roleVisibility.test.tsx`
  - Change merchant expectation: Dashboard no longer shows `开启直播` or `开始直播`.
- Create: `frontend/admin/src/pages-new/__tests__/LiveDetail.startLive.test.tsx`
  - Cover merchant start-live flow and admin non-visibility.
- Modify: `frontend/admin/src/shared/api/types.ts`
  - Complete `LiveStream.status` comment with `3=已封禁`.

## Current State Note

This plan was created after the first RED cycle had already started. Current uncommitted implementation state must be preserved:

- `LiveDetail.startLive.test.tsx` has been added.
- `Dashboard.roleVisibility.test.tsx` has been changed to expect no Dashboard start-live button.
- The focused test command has been run and failed for the expected reasons:
  - Dashboard still showed `开启直播`.
  - `LiveDetail` did not yet show phase-one copy or `开始直播`.
- `LiveDetail.tsx` has a partial minimal implementation in progress.

Do not discard these changes. Continue from the state file.

## Task T001: Move Start Live Entry To LiveDetail

**Files:**

- Modify: `frontend/admin/src/pages-new/LiveDetail.tsx`
- Create: `frontend/admin/src/pages-new/__tests__/LiveDetail.startLive.test.tsx`

- [ ] **Step 1: Write failing LiveDetail tests**

Create `frontend/admin/src/pages-new/__tests__/LiveDetail.startLive.test.tsx` with two behaviors:

```tsx
it('lets merchants start the current live stream from detail page with explicit phase-one copy', async () => {
  jest.spyOn(window, 'confirm').mockReturnValue(true);
  (liveStreamApi.start as jest.Mock).mockResolvedValue({ success: true });

  renderLiveDetailAs(1, {
    id: 501,
    name: '商家直播间',
    streamer_id: 1002,
    streamer_name: '商家用户',
    status: 0,
    viewer_count: 0,
    auction_count: 2,
    created_at: '2026-06-05T00:00:00Z',
  });

  expect(await screen.findByRole('heading', { name: '直播间控制台' })).toBeInTheDocument();
  expect(screen.getByText(/当前版本支持通过 PC 管理端发起直播状态/)).toBeInTheDocument();

  fireEvent.click(screen.getByRole('button', { name: /开始直播/ }));

  expect(window.confirm).toHaveBeenCalledWith(expect.stringContaining('确认开始直播'));
  await waitFor(() => expect(liveStreamApi.start).toHaveBeenCalledWith(501));
  expect(screen.getByText('直播中')).toBeInTheDocument();
});

it('does not show merchant start action for admins', async () => {
  renderLiveDetailAs(2, {
    id: 501,
    name: '平台巡检直播间',
    streamer_id: 1002,
    streamer_name: '商家用户',
    status: 0,
    viewer_count: 0,
    auction_count: 2,
    created_at: '2026-06-05T00:00:00Z',
  });

  expect(await screen.findByRole('heading', { name: '直播间控制台' })).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /开始直播/ })).not.toBeInTheDocument();
});
```

- [ ] **Step 2: Verify RED**

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath src/pages-new/__tests__/LiveDetail.startLive.test.tsx --runInBand
```

Expected: FAIL because `LiveDetail` does not show phase-one copy or merchant `开始直播`.

- [ ] **Step 3: Minimal implementation**

In `LiveDetail.tsx`:

- Use `useAuth()` to read `user.role`.
- Define `isMerchant = user?.role === MERCHANT_ROLE`.
- Define `isPlatformAdmin = user?.role === ADMIN_ROLE`.
- Add `starting` state.
- Add `handleStart()`:
  - Return early when no `liveStreamId`, request already pending, or status is live.
  - Confirm with text containing `确认开始直播`.
  - Call `liveStreamApi.start(Number(liveStreamId))`.
  - Update local `liveStream.status` to `1`.
  - Show failure via `alert("开始直播失败")`.
- In operation center:
  - Merchants see phase-one copy and `开始直播` / `开始中...` / `直播中`.
  - Admins see `封禁直播间` and `关闭直播`.

- [ ] **Step 4: Verify GREEN**

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath src/pages-new/__tests__/LiveDetail.startLive.test.tsx --runInBand
```

Expected: PASS.

## Task T002: Remove Dashboard Prompt Start Live

**Files:**

- Modify: `frontend/admin/src/pages-new/Dashboard.tsx`
- Modify: `frontend/admin/src/pages-new/__tests__/Dashboard.roleVisibility.test.tsx`

- [ ] **Step 1: Write failing Dashboard test update**

Change merchant Dashboard expectation:

```tsx
expect(screen.queryByRole('button', { name: /开启直播|开始直播/ })).not.toBeInTheDocument();
```

- [ ] **Step 2: Verify RED**

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath src/pages-new/__tests__/Dashboard.roleVisibility.test.tsx --runInBand
```

Expected: FAIL because Dashboard still renders `开启直播`.

- [ ] **Step 3: Minimal implementation**

In `Dashboard.tsx`:

- Remove `Video` from `lucide-react` imports.
- Remove `liveStreamApi` from `@/shared/api` imports.
- Delete `handleStartLive`.
- Delete the `开启直播` button from the merchant welcome action area.
- Keep `发布商品`.

- [ ] **Step 4: Verify GREEN**

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath src/pages-new/__tests__/Dashboard.roleVisibility.test.tsx --runInBand
```

Expected: PASS.

## Task T003: Align Status Comment And Focused Regression

**Files:**

- Modify: `frontend/admin/src/shared/api/types.ts`
- Verify: files modified by T001 and T002.

- [ ] **Step 1: Update status comment**

Change `LiveStream.status` comment from:

```ts
status: number; // 0=未开播, 1=直播中, 2=已结束
```

to:

```ts
status: number; // 0=未开播, 1=直播中, 2=已结束, 3=已封禁
```

- [ ] **Step 2: Run focused tests**

Run:

```bash
cd frontend/admin
npm test -- --runTestsByPath src/pages-new/__tests__/LiveDetail.startLive.test.tsx src/pages-new/__tests__/Dashboard.roleVisibility.test.tsx --runInBand
```

Expected: PASS.

- [ ] **Step 3: Run Admin build**

Run:

```bash
cd frontend/admin
npm run build
```

Expected: PASS. If build fails due to unrelated existing issues, record the exact failure in the state file before stopping.

- [ ] **Step 4: Commit**

Run:

```bash
git add frontend/admin/src/pages-new/LiveDetail.tsx frontend/admin/src/pages-new/Dashboard.tsx frontend/admin/src/pages-new/__tests__/LiveDetail.startLive.test.tsx frontend/admin/src/pages-new/__tests__/Dashboard.roleVisibility.test.tsx frontend/admin/src/shared/api/types.ts docs/superpowers/plans/2026-06-05-admin-live-start-product-implementation.md docs/superpowers/plans/2026-06-05-admin-live-start-product-tasks.md docs/superpowers/sdd/runs/2026-06-05-admin-live-start-product-state.md
git commit -m "feat: move admin live start to detail page"
```

Expected: commit succeeds and worktree is clean.
