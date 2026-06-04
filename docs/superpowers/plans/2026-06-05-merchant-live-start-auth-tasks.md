# Merchant Live Start Authorization Tasks

- [ ] T0.1 Gateway Route Authorization
  - Scope: `backend/gateway/router/router.go`, `backend/gateway/router/live_stream_start_route_test.go`
  - Expected tests:
    - `cd backend/gateway && go test ./router -run 'TestStartLiveRoute(AllowsMerchant|RejectsAdmin|RejectsNonAdmin|AdminLiveStreamControl)' -count=1`
  - Acceptance:
    - Merchant role can call `POST /api/v1/live-streams/:id/start`.
    - Admin role is rejected for start.
    - Admin `end` and `ban` routes remain admin-only.
    - Gateway still forwards `X-Internal-Token`, `X-User-ID`, and `X-User-Role`.

- [ ] T0.2 Auction Internal Owner Authorization
  - Scope: `backend/auction/handler/live_stream_stats.go`, `backend/auction/handler/live_reminder_flow_test.go`, `backend/auction/main.go`
  - Expected tests:
    - `cd backend/auction && go test ./handler -run 'Test(StartLiveTransitionAllowsMerchantOwner|StartLiveTransitionRejectsMerchantNonOwner|StartLiveTransitionRejectsAdminOperator|ProductionStartLiveTransitionFeedsPendingReminderOnce)' -count=1`
  - Acceptance:
    - Merchant owner can start own live stream.
    - Merchant non-owner is rejected.
    - Admin is rejected for start.
    - Handler uses product live stream summary via existing internal client to check `creator_id`.

- [ ] T0.3 Focused Verification
  - Scope: all files touched by T0.1 and T0.2
  - Expected tests:
    - `cd backend/gateway && go test ./router -run 'TestStartLiveRoute|TestAdminLiveStreamControlRoutes' -count=1`
    - `cd backend/auction && go test ./handler -run 'Test.*StartLive' -count=1`
    - `cd backend/gateway && go test ./router -count=1`
    - `cd backend/auction && go test ./handler -count=1`
  - Acceptance:
    - Focused route and handler tests pass.
    - Affected package tests pass, or unrelated pre-existing failures are recorded with logs.
    - SDD state records modified files, commands, outcomes, and residual risks.
