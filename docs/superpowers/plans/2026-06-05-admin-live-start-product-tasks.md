# Admin Live Start Product Tasks

- [ ] T001 Move Start Live Entry To LiveDetail
  - Scope:
    - `frontend/admin/src/pages-new/LiveDetail.tsx`
    - `frontend/admin/src/pages-new/__tests__/LiveDetail.startLive.test.tsx`
  - Expected tests:
    - `cd frontend/admin && npm test -- --runTestsByPath src/pages-new/__tests__/LiveDetail.startLive.test.tsx --runInBand`
  - Acceptance:
    - 商家在直播间详情页看到一期说明文案。
    - 商家点击 `开始直播` 前出现确认文案。
    - 确认后调用 `liveStreamApi.start(当前页面 liveStreamId)`，测试用例中为 `liveStreamApi.start(501)`，不再要求手输 ID。
    - 成功后当前页面状态更新为 `直播中`。
    - 管理员不展示 `开始直播`。

- [ ] T002 Remove Dashboard Prompt Start Live
  - Scope:
    - `frontend/admin/src/pages-new/Dashboard.tsx`
    - `frontend/admin/src/pages-new/__tests__/Dashboard.roleVisibility.test.tsx`
  - Expected tests:
    - `cd frontend/admin && npm test -- --runTestsByPath src/pages-new/__tests__/Dashboard.roleVisibility.test.tsx --runInBand`
  - Acceptance:
    - Dashboard 不再展示 `开启直播` 或 `开始直播`。
    - Dashboard 不再使用 `window.prompt` 获取直播间 ID。
    - 商家仍可在 Dashboard 看到 `发布商品`。
    - 管理员仍看不到商家经营按钮。

- [ ] T003 Align Status Comment And Focused Regression
  - Scope:
    - `frontend/admin/src/shared/api/types.ts`
    - all files modified by T001 and T002
  - Expected tests:
    - `cd frontend/admin && npm test -- --runTestsByPath src/pages-new/__tests__/LiveDetail.startLive.test.tsx src/pages-new/__tests__/Dashboard.roleVisibility.test.tsx --runInBand`
    - `cd frontend/admin && npm run build`
  - Acceptance:
    - `LiveStream.status` 注释补全 `3=已封禁`。
    - 两个聚焦测试文件通过。
    - Admin build 通过，或状态文件记录阻塞根因与完整错误。
    - SDD 状态文件记录 RED/GREEN、验证命令、修改文件和提交信息。
