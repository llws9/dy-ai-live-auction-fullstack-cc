# Merchant Live Start Authorization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修正开播鉴权：商家只能开启自己拥有的直播间，平台管理员不能代商家开播，管理员仅保留中断/封禁等治理动作。

**Architecture:** Gateway 将 `POST /api/v1/live-streams/:id/start` 从 admin-only 改为 merchant-only，并继续通过 `LiveStartHandler` 透传 `X-User-ID`、`X-User-Role` 和 `X-Internal-Token` 到 auction-service。auction-service 的内部开播 handler 从“仅 admin”改为“仅 merchant 且 live_stream.creator_id == X-User-ID”，owner 校验复用已有 `client.LiveStreamClient` 调 product-service `/internal/live-streams/batch` 获取直播间摘要。

**Tech Stack:** Go 1.24+, Hertz, testify, internal service token, gateway JWT-derived `X-User-ID` / `X-User-Role`.

---

## File Scope

- Modify: `backend/gateway/router/router.go`
- Modify: `backend/gateway/router/live_stream_start_route_test.go`
- Modify: `backend/auction/handler/live_stream_stats.go`
- Modify: `backend/auction/handler/live_reminder_flow_test.go`
- Modify: `backend/auction/main.go`
- Test-only helper changes may be added in the two test files above.

## Task T0.1: Gateway Route Authorization

**Files:**
- Modify: `backend/gateway/router/router.go`
- Modify: `backend/gateway/router/live_stream_start_route_test.go`

- [ ] **Step 1: Write failing gateway route tests**

Update `backend/gateway/router/live_stream_start_route_test.go`:

```go
func TestStartLiveRouteAllowsMerchantAndForwardsInternalHeaders(t *testing.T) {
	var called atomic.Int32
	var capturedPath atomic.Value
	var capturedToken atomic.Value
	var capturedUserID atomic.Value
	var capturedRole atomic.Value
	capturedPath.Store("")
	capturedToken.Store("")
	capturedUserID.Store("")
	capturedRole.Store("")

	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		capturedPath.Store(r.URL.Path)
		capturedToken.Store(r.Header.Get("X-Internal-Token"))
		capturedUserID.Store(r.Header.Get("X-User-ID"))
		capturedRole.Store(r.Header.Get("X-User-Role"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"success":true}}`))
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			AuctionURL:    auctionMock.URL,
			ProductURL:    "http://127.0.0.1:0",
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "start-live-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	merchantToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9002, "merchant", 1, 24)
	require.NoError(t, err)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/v1/live-streams/123/start",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken},
	)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int32(1), called.Load())
	assert.Equal(t, "/internal/live-streams/123/start", capturedPath.Load().(string))
	assert.Equal(t, "internal-secret", capturedToken.Load().(string))
	assert.Equal(t, "9002", capturedUserID.Load().(string))
	assert.Equal(t, "merchant", capturedRole.Load().(string))
}

func TestStartLiveRouteRejectsAdminBeforeAuction(t *testing.T) {
	var called atomic.Int32
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			AuctionURL:    auctionMock.URL,
			ProductURL:    "http://127.0.0.1:0",
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "start-live-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	adminToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9001, "admin", 2, 24)
	require.NoError(t, err)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/v1/live-streams/123/start",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + adminToken},
	)

	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
	assert.Equal(t, int32(0), called.Load())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd backend/gateway
go test ./router -run 'TestStartLiveRoute(AllowsMerchant|RejectsAdmin|RejectsNonAdmin)' -count=1
```

Expected: FAIL because current route uses `RequireAdmin`, so merchant is rejected and admin is accepted.

- [ ] **Step 3: Implement minimal gateway change**

In `backend/gateway/router/router.go`, change:

```go
authGroup.POST("/live-streams/:id/start", middleware.RequireAdmin(), liveStartHandler.StartLive)
```

to:

```go
authGroup.POST("/live-streams/:id/start", middleware.RequireMerchantOnly(), liveStartHandler.StartLive)
```

- [ ] **Step 4: Run gateway tests to verify pass**

Run:

```bash
cd backend/gateway
go test ./router -run 'TestStartLiveRoute(AllowsMerchant|RejectsAdmin|RejectsNonAdmin|AdminLiveStreamControl)' -count=1
```

Expected: PASS. Admin `end`/`ban` control routes remain admin-only.

## Task T0.2: Auction Internal Owner Authorization

**Files:**
- Modify: `backend/auction/handler/live_stream_stats.go`
- Modify: `backend/auction/handler/live_reminder_flow_test.go`
- Modify: `backend/auction/main.go`

- [ ] **Step 1: Write failing auction handler tests**

Update `backend/auction/handler/live_reminder_flow_test.go` with a fake owner checker and three tests:

```go
type fakeLiveStreamOwnerChecker struct {
	owners map[int64]int64
	err    error
}

func (f *fakeLiveStreamOwnerChecker) OwnerID(ctx context.Context, liveStreamID int64) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	owner, ok := f.owners[liveStreamID]
	if !ok {
		return 0, nil
	}
	return owner, nil
}

func TestStartLiveTransitionAllowsMerchantOwner(t *testing.T) {
	ctx := context.Background()
	_, err := dao.InitRedis("localhost:6379", "")
	require.NoError(t, err)

	ownerID := int64(10001)
	liveStreamID := time.Now().UnixNano()%1_000_000_000 + 2_000_000_000
	statsService := service.NewLiveStreamStatsService()
	require.NoError(t, statsService.SetScheduledStartTime(ctx, liveStreamID, time.Now().Add(time.Hour), 80))

	startHandler := NewLiveStreamStatsHandler(statsService)
	startHandler.SetOwnerChecker(&fakeLiveStreamOwnerChecker{owners: map[int64]int64{liveStreamID: ownerID}})
	c := app.NewContext(1)
	c.Params = append(c.Params, param.Param{Key: "id", Value: strconv.FormatInt(liveStreamID, 10)})
	c.Set("user_id", ownerID)
	c.Set("user_role", 1)

	startHandler.StartLive(ctx, c)

	require.Equal(t, http.StatusOK, c.Response.StatusCode())
	stats, err := statsService.GetStats(ctx, liveStreamID)
	require.NoError(t, err)
	require.Equal(t, "live", stats.Status)
	require.NotNil(t, stats.StartedAt)
}

func TestStartLiveTransitionRejectsMerchantNonOwner(t *testing.T) {
	ctx := context.Background()
	_, err := dao.InitRedis("localhost:6379", "")
	require.NoError(t, err)

	ownerID := int64(10001)
	otherMerchantID := int64(10002)
	liveStreamID := time.Now().UnixNano()%1_000_000_000 + 3_000_000_000
	statsService := service.NewLiveStreamStatsService()
	require.NoError(t, statsService.SetScheduledStartTime(ctx, liveStreamID, time.Now().Add(time.Hour), 80))

	startHandler := NewLiveStreamStatsHandler(statsService)
	startHandler.SetOwnerChecker(&fakeLiveStreamOwnerChecker{owners: map[int64]int64{liveStreamID: ownerID}})
	c := app.NewContext(1)
	c.Params = append(c.Params, param.Param{Key: "id", Value: strconv.FormatInt(liveStreamID, 10)})
	c.Set("user_id", otherMerchantID)
	c.Set("user_role", 1)

	startHandler.StartLive(ctx, c)

	require.Equal(t, http.StatusForbidden, c.Response.StatusCode())
	stats, err := statsService.GetStats(ctx, liveStreamID)
	require.NoError(t, err)
	require.Equal(t, "pending", stats.Status)
	require.Nil(t, stats.StartedAt)
}

func TestStartLiveTransitionRejectsAdminOperator(t *testing.T) {
	ctx := context.Background()
	_, err := dao.InitRedis("localhost:6379", "")
	require.NoError(t, err)

	liveStreamID := time.Now().UnixNano()%1_000_000_000 + 4_000_000_000
	statsService := service.NewLiveStreamStatsService()
	require.NoError(t, statsService.SetScheduledStartTime(ctx, liveStreamID, time.Now().Add(time.Hour), 80))

	startHandler := NewLiveStreamStatsHandler(statsService)
	startHandler.SetOwnerChecker(&fakeLiveStreamOwnerChecker{owners: map[int64]int64{liveStreamID: 10001}})
	c := app.NewContext(1)
	c.Params = append(c.Params, param.Param{Key: "id", Value: strconv.FormatInt(liveStreamID, 10)})
	c.Set("user_id", int64(9001))
	c.Set("user_role", 2)

	startHandler.StartLive(ctx, c)

	require.Equal(t, http.StatusForbidden, c.Response.StatusCode())
}
```

Adjust the existing `TestProductionStartLiveTransitionFeedsPendingReminderOnce` to use role `1` and a matching owner checker.

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd backend/auction
go test ./handler -run 'Test(StartLiveTransitionAllowsMerchantOwner|StartLiveTransitionRejectsMerchantNonOwner|StartLiveTransitionRejectsAdminOperator|ProductionStartLiveTransitionFeedsPendingReminderOnce)' -count=1
```

Expected: FAIL because `LiveStreamStatsHandler` currently requires role >= 2 and has no owner checker.

- [ ] **Step 3: Implement owner checker interface and handler authorization**

In `backend/auction/handler/live_stream_stats.go`, add:

```go
type LiveStreamOwnerChecker interface {
	OwnerID(ctx context.Context, liveStreamID int64) (int64, error)
}
```

Add field and setter:

```go
type LiveStreamStatsHandler struct {
	service      LiveStarter
	ownerChecker LiveStreamOwnerChecker
}

func (h *LiveStreamStatsHandler) SetOwnerChecker(checker LiveStreamOwnerChecker) {
	h.ownerChecker = checker
}
```

Change `StartLive` authorization:

```go
userIDRaw, exists := c.Get("user_id")
if !exists {
	c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
	return
}
userID, ok := userIDRaw.(int64)
if !ok || userID <= 0 {
	c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
	return
}
if role := c.GetInt("user_role"); role != 1 {
	c.JSON(403, map[string]interface{}{"code": 403, "message": "无权限操作直播间"})
	return
}
```

After parsing `liveStreamID`, add owner check before `service.StartLive`:

```go
if h.ownerChecker == nil {
	c.JSON(500, map[string]interface{}{"code": 500, "message": "直播间归属校验未配置"})
	return
}
ownerID, err := h.ownerChecker.OwnerID(ctx, liveStreamID)
if err != nil {
	log.Printf("StartLive owner check failed: liveStreamID=%d userID=%d err=%v", liveStreamID, userID, err)
	c.JSON(500, map[string]interface{}{"code": 500, "message": "开始直播失败"})
	return
}
if ownerID == 0 || ownerID != userID {
	c.JSON(403, map[string]interface{}{"code": 403, "message": "无权限操作直播间"})
	return
}
```

- [ ] **Step 4: Add production owner checker adapter**

In `backend/auction/main.go`, add an adapter near the existing `liveStreamOwnerChecker`:

```go
type liveStreamStartOwnerChecker struct {
	client client.LiveStreamClient
}

func (c *liveStreamStartOwnerChecker) OwnerID(ctx context.Context, liveStreamID int64) (int64, error) {
	items, err := c.client.BatchGetLiveStreams(ctx, []int64{liveStreamID})
	if err != nil {
		return 0, err
	}
	item, ok := items[liveStreamID]
	if !ok {
		return 0, nil
	}
	return item.CreatorID, nil
}
```

After constructing `liveStreamStatsHandler`, wire:

```go
liveStreamStatsHandler.SetOwnerChecker(&liveStreamStartOwnerChecker{client: liveStreamClient})
```

- [ ] **Step 5: Run auction tests**

Run:

```bash
cd backend/auction
go test ./handler -run 'Test(StartLiveTransitionAllowsMerchantOwner|StartLiveTransitionRejectsMerchantNonOwner|StartLiveTransitionRejectsAdminOperator|ProductionStartLiveTransitionFeedsPendingReminderOnce|StartLiveTransitionRejectsNonAdminOperator)' -count=1
```

Expected: PASS. If the old non-admin test remains, update its name/assertion to reflect user role rejection instead of merchant rejection.

## Task T0.3: Full Verification

**Files:**
- No additional files unless formatting changes are needed.

- [ ] **Step 1: Format touched Go files**

Run:

```bash
gofmt -w backend/gateway/router/router.go backend/gateway/router/live_stream_start_route_test.go backend/auction/handler/live_stream_stats.go backend/auction/handler/live_reminder_flow_test.go backend/auction/main.go
```

- [ ] **Step 2: Run focused regression**

Run:

```bash
cd backend/gateway && go test ./router -run 'TestStartLiveRoute|TestAdminLiveStreamControlRoutes' -count=1
cd ../auction && go test ./handler -run 'Test.*StartLive' -count=1
```

Expected: PASS.

- [ ] **Step 3: Run affected package tests**

Run:

```bash
cd backend/gateway && go test ./router -count=1
cd ../auction && go test ./handler -count=1
```

Expected: PASS or document unrelated pre-existing failures with logs.

- [ ] **Step 4: Update SDD state**

Record modified files, test commands, actual results, and residual risks in the SDD state file before reporting completion.
