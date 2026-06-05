# Test Dashboard User Journey Tasks

- [ ] T1 Extend test-service business client and result contract
  - Scope: `backend/test/client/auction/client.go`, `backend/test/client/auction/client_test.go`, `backend/test/model/test.go`
  - Dependencies: T0 completed on `main`
  - Expected tests:
    - `cd backend/test && go test ./client/auction -run 'Test(DoSetsMerchantIdentityHeaders|TopUpUserBalanceCallsInternalEndpoint|PurchaseFixedPriceIncludesIdempotencyKey|FollowAndFollowStatusUseBuyerIdentity)' -count=1`
    - `cd backend/test && go test ./client/auction -count=1`
  - Acceptance:
    - business client supports `user` and `merchant` actors
    - user journey required API methods are available
    - `user_journey` test type constant is added

- [ ] T2 Add internal buyer balance top-up endpoint
  - Scope: `backend/auction/handler/user_balance_http.go`, `backend/auction/handler/user_balance_http_test.go`, `backend/auction/main.go`
  - Dependencies: T1
  - Expected tests:
    - `cd backend/auction && go test ./handler -run 'TestTopUpUserBalanceInternal' -count=1`
    - `cd backend/auction && go test ./handler -run 'Test(TopUpUserBalanceInternal|GetUserBalance)' -count=1`
  - Acceptance:
    - `/internal/test/user-balance` exists under `InternalAuthMiddleware`
    - amount uses decimal string and rejects invalid/non-positive input

- [ ] T3 Implement backend test-service user_journey scenario
  - Scope: `backend/test/scenario/user_journey/*`, `backend/test/handler/test.go`, `backend/test/main.go`
  - Dependencies: T1, T2
  - Expected tests:
    - `cd backend/test && go test ./scenario/user_journey -run 'Test(RunHappyPathProducesEvidenceReport|PrepareFailsClosedWhenTopUpFails|PrepareSkipsCleanupAndStillRecordsSeedRefs|ReminderStepUsesFollowAndFollowStatusOnly)' -count=1`
    - `cd backend/test && go test ./scenario/user_journey ./handler -run 'Test(UserJourney|PostUserJourney)' -count=1`
  - Acceptance:
    - `/api/test/user-journey` can start a scenario
    - report structure contains required ids / balances / stock / warnings
    - scenario keeps evidence and does not cleanup by default

- [ ] T4 Add frontend test-dashboard user journey page
  - Scope: `frontend/test-dashboard/src/api/test.ts`, `frontend/test-dashboard/src/App.tsx`, `frontend/test-dashboard/src/components/Layout.tsx`, `frontend/test-dashboard/src/components/StepTimeline.tsx`, `frontend/test-dashboard/src/pages/UserJourney.tsx`, `frontend/test-dashboard/src/pages/Report.tsx`
  - Dependencies: T3
  - Expected tests:
    - `cd frontend/test-dashboard && npm run build`
  - Acceptance:
    - `/test/user-journey` route exists
    - page can start the scenario, show WS progress, timeline, evidence cards, and report link
    - report page gives a compact user_journey summary

- [ ] T5 Cross-module verification and documentation sync
  - Scope: affected files from T1-T4
  - Dependencies: T1, T2, T3, T4
  - Expected tests:
    - `cd backend/test && go test ./client/auction ./scenario/user_journey ./handler -count=1`
    - `cd backend/auction && go test ./handler -run 'Test(TopUpUserBalanceInternal|GetUserBalance|.*StartLive)' -count=1`
    - `cd frontend/test-dashboard && npm run build`
    - `rg -n "user-journey|user_journey|/test/user-journey|/api/test/user-journey|/internal/test/user-balance" backend frontend docs`
  - Acceptance:
    - introduced routes and contracts are wired consistently
    - SDD state contains full red/green/verification evidence
