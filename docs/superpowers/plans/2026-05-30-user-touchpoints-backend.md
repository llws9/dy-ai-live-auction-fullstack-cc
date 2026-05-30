# User Touchpoints Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace H5 touchpoint Mock data with real backend data for badge dots, login-time live reminder, and Toast-triggering notification events.

**Architecture:** Use `gateway-service` as the only frontend-facing entry point. Keep domain data inside its owning service: notification counts and read state in `auction-service`, order counts in `product-service`, and cross-service aggregation in `gateway-service`. Reuse the existing auction WebSocket `notification` message instead of adding a new user-events socket.

**Tech Stack:** Go 1.24+, Hertz, GORM, MySQL/SQLite tests, existing `gateway-service`, `auction-service`, `product-service`, React H5 integration tests.

---

## Scope

This plan implements the adapted backend design in `docs/superpowers/specs/2026-05-30-user-touchpoints-backend-design-adapted.md`.

### In Scope

- `GET /api/v1/notifications/summary`
- `POST /api/v1/notifications/read-category`
- `GET /api/v1/orders/summary`
- `GET /api/v1/live/pending-reminder`
- Frontend hook switch from Mock to backend API
- Frontend Toast mapping from existing `notification` WebSocket messages

### Out of Scope

- New `/ws/user-events` endpoint
- New Kafka/RabbitMQ dependency
- Full live streaming lifecycle redesign
- Theme switching UI
- Payment workflow changes

---

## File Map

### Auction Service

- Modify `backend/auction/model/notification.go`
  - Add summary/read-category request and response DTOs.
- Modify `backend/auction/dao/notification.go`
  - Add count by notification type.
  - Add mark unread notifications by type as read.
- Modify `backend/auction/service/notification.go`
  - Add `GetSummary`.
  - Add `MarkCategoryAsRead`.
- Modify `backend/auction/handler/notification.go`
  - Add `GetSummary`.
  - Add `MarkCategoryAsRead`.
- Modify `backend/auction/main.go`
  - Register `/api/v1/notifications/summary`.
  - Register `/api/v1/notifications/read-category`.
- Create `backend/auction/migration/002_create_live_stream_reminder_receipts.sql`
  - Add receipt table for one-time login reminders.
- Create `backend/auction/model/live_stream_reminder_receipt.go`
  - GORM model for reminder receipts.
- Create `backend/auction/dao/live_stream_reminder_receipt.go`
  - Receipt existence check and insert.
- Create `backend/auction/service/live_reminder.go`
  - Query pending reminder and claim the receipt atomically with the unique key.
- Modify `backend/auction/service/live_stream_stats.go`
  - Preserve real live session `StartedAt` when a stream enters `live`.
- Create `backend/auction/handler/live_reminder.go`
  - Expose pending reminder endpoint.
- Modify `backend/auction/main.go`
  - AutoMigrate receipt model.
  - Wire DAO/service/handler.

### Product Service

- Modify `backend/product/model/order.go`
  - Add order summary DTO.
- Modify `backend/product/dao/order.go`
  - Add count by winner and status.
- Modify `backend/product/service/order.go`
  - Add `GetSummary`.
- Modify `backend/product/handler/order.go`
  - Add `Summary`.
- Modify `backend/product/main.go`
  - Register `/api/v1/orders/summary`.

### Gateway Service

- Create `backend/gateway/handler/touchpoint.go`
  - Aggregate auction summary and product order summary.
- Create `backend/gateway/handler/touchpoint_test.go`
  - Test aggregation, partial upstream failure fallback, and auth error propagation.
- Modify `backend/gateway/router/router.go`
  - Register frontend-facing `/api/v1/notifications/summary`.
  - Register `/api/v1/live/pending-reminder`.
  - Proxy `/api/v1/notifications/read-category`.

### Frontend H5

- Modify `frontend/h5/src/services/notification.ts`
  - Add `getTouchpointSummary`.
  - Add `markCategoryAsRead`.
- Modify `frontend/h5/src/hooks/useTouchpointNotifications.ts`
  - Replace constants with API-backed state and zero fallback.
- Modify `frontend/h5/src/components/MobileShell/MobileContainer.tsx`
  - Replace local `pending_live_reminder` marker with API call.
- Modify `frontend/h5/src/services/websocket.ts`
  - Ensure `notification` messages are emitted to registered handlers.
- Modify `frontend/h5/src/pages/Live/index.tsx`
  - Map notification messages to global Toast.
  - Remove development-only Toast Demo after real WS mapping lands.

---

## Task 0: Commit Adapted Design And Plan

**Files:**
- Add: `docs/superpowers/specs/2026-05-30-user-touchpoints-backend-design-adapted.md`
- Add: `docs/superpowers/plans/2026-05-30-user-touchpoints-backend.md`

- [ ] **Step 1: Verify documentation files exist**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc
test -f docs/superpowers/specs/2026-05-30-user-touchpoints-backend-design-adapted.md
test -f docs/superpowers/plans/2026-05-30-user-touchpoints-backend.md
```

Expected: exit code `0`.

- [ ] **Step 2: Check no unresolved placeholders**

Run:

```bash
python3 - <<'PY'
from pathlib import Path
needles = ["TB" + "D", "TO" + "DO", "待" + "补", "待" + "定"]
paths = [
    Path("docs/superpowers/specs/2026-05-30-user-touchpoints-backend-design-adapted.md"),
    Path("docs/superpowers/plans/2026-05-30-user-touchpoints-backend.md"),
]
for path in paths:
    for no, line in enumerate(path.read_text().splitlines(), 1):
        if any(needle in line for needle in needles):
            print(f"{path}:{no}:{line}")
PY
```

Expected: no matches.

- [ ] **Step 3: Commit documentation**

Run:

```bash
git add docs/superpowers/specs/2026-05-30-user-touchpoints-backend-design-adapted.md docs/superpowers/plans/2026-05-30-user-touchpoints-backend.md
git commit -m "docs: adapt touchpoints backend design"
```

Expected: commit succeeds.

---

## Task 1: Auction Notification Summary And Read Category

**Files:**
- Modify: `backend/auction/model/notification.go`
- Modify: `backend/auction/dao/notification.go`
- Modify: `backend/auction/service/notification.go`
- Modify: `backend/auction/handler/notification.go`
- Modify: `backend/auction/main.go`
- Test: `backend/auction/service/notification_test.go`

- [ ] **Step 1: Add service tests first**

Append to `backend/auction/service/notification_test.go`:

```go
func TestNotificationCategoryTypes(t *testing.T) {
	tests := []struct {
		category string
		want     []model.NotificationType
		wantErr  bool
	}{
		{category: "outbid", want: []model.NotificationType{model.NotificationTypeBidOutbid}},
		{category: "endingSoon", want: nil},
		{category: "pendingPayment", want: nil},
		{category: "all", want: nil},
		{category: "unknown", wantErr: true},
	}

	for _, tt := range tests {
		got, err := notificationTypesForCategory(tt.category)
		if tt.wantErr {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, tt.want, got)
	}
}
```

Also add this import:

```go
import "auction-service/model"
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction
go test ./service -run TestNotificationCategoryTypes -v
```

Expected: FAIL because `notificationTypesForCategory` is undefined.

- [ ] **Step 3: Add model DTOs**

Add to `backend/auction/model/notification.go`:

```go
type NotificationSummaryResponse struct {
	UnreadTotal int64 `json:"unreadTotal"`
	Outbid      int64 `json:"outbid"`
	EndingSoon  int64 `json:"endingSoon"`
}

type MarkCategoryReadRequest struct {
	Category string `json:"category"`
}
```

- [ ] **Step 4: Add DAO count/read helpers**

Add to `backend/auction/dao/notification.go`:

```go
func (d *NotificationDAO) CountUnreadByTypes(ctx context.Context, userID int64, types []model.NotificationType) (int64, error) {
	var count int64
	query := d.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID)
	if len(types) > 0 {
		query = query.Where("type IN ?", types)
	}
	return count, query.Count(&count).Error
}

func (d *NotificationDAO) MarkUnreadByTypesAsRead(ctx context.Context, userID int64, types []model.NotificationType) error {
	now := time.Now()
	query := d.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID)
	if len(types) > 0 {
		query = query.Where("type IN ?", types)
	}
	return query.Update("read_at", now).Error
}
```

- [ ] **Step 5: Add service logic**

Add to `backend/auction/service/notification.go`:

```go
func notificationTypesForCategory(category string) ([]model.NotificationType, error) {
	switch category {
	case "outbid":
		return []model.NotificationType{model.NotificationTypeBidOutbid}, nil
	case "pendingPayment", "endingSoon", "all":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported notification category: %s", category)
	}
}

func (s *NotificationService) GetSummary(ctx context.Context, userID int64) (*model.NotificationSummaryResponse, error) {
	unreadTotal, err := s.notificationDAO.CountUnreadByTypes(ctx, userID, nil)
	if err != nil {
		return nil, err
	}
	outbid, err := s.notificationDAO.CountUnreadByTypes(ctx, userID, []model.NotificationType{model.NotificationTypeBidOutbid})
	if err != nil {
		return nil, err
	}
	return &model.NotificationSummaryResponse{
		UnreadTotal: unreadTotal,
		Outbid:      outbid,
		EndingSoon:  0,
	}, nil
}

func (s *NotificationService) MarkCategoryAsRead(ctx context.Context, userID int64, category string) error {
	if category == "pendingPayment" || category == "endingSoon" {
		return nil
	}
	if category == "all" {
		return s.MarkAllAsRead(ctx, userID)
	}
	types, err := notificationTypesForCategory(category)
	if err != nil {
		return err
	}
	return s.notificationDAO.MarkUnreadByTypesAsRead(ctx, userID, types)
}
```

- [ ] **Step 6: Add handler methods**

Add to `backend/auction/handler/notification.go`:

```go
func (h *NotificationHandler) GetSummary(ctx context.Context, c *app.RequestContext) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
		return
	}
	userID := userIDInterface.(int64)

	summary, err := h.notificationService.GetSummary(ctx, userID)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取通知汇总失败: " + err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{"code": 0, "message": "success", "data": summary})
}

func (h *NotificationHandler) MarkCategoryAsRead(ctx context.Context, c *app.RequestContext) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
		return
	}
	userID := userIDInterface.(int64)

	var req model.MarkCategoryReadRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误"})
		return
	}

	if err := h.notificationService.MarkCategoryAsRead(ctx, userID, req.Category); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    0,
		"message": "success",
		"data":    map[string]interface{}{"success": true},
	})
}
```

Also add `auction-service/model` to handler imports.

- [ ] **Step 7: Register auction routes**

Modify notification routes in `backend/auction/main.go`:

```go
v1.GET("/notifications/summary", notificationHandler.GetSummary)
v1.POST("/notifications/read-category", notificationHandler.MarkCategoryAsRead)
```

- [ ] **Step 8: Verify auction service**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction
gofmt -w model/notification.go dao/notification.go service/notification.go handler/notification.go main.go service/notification_test.go
go test ./dao ./service ./handler
```

Expected: PASS.

- [ ] **Step 9: Commit**

Run:

```bash
git add backend/auction/model/notification.go backend/auction/dao/notification.go backend/auction/service/notification.go backend/auction/handler/notification.go backend/auction/main.go backend/auction/service/notification_test.go
git commit -m "feat(auction): add notification summary endpoints"
```

Expected: commit succeeds.

---

## Task 2: Product Order Summary

**Files:**
- Modify: `backend/product/model/order.go`
- Modify: `backend/product/dao/order.go`
- Modify: `backend/product/service/order.go`
- Modify: `backend/product/handler/order.go`
- Modify: `backend/product/main.go`
- Test: `backend/product/service/order_test.go`

- [ ] **Step 1: Add failing service test**

Append to `backend/product/service/order_test.go`:

```go
func (suite *OrderTestSuite) TestGetSummary() {
	ctx := context.Background()
	userID := int64(100)

	_, err := suite.service.CreateOrder(ctx, 101, 1, userID, 500.0)
	suite.NoError(err)
	paid, err := suite.service.CreateOrder(ctx, 102, 1, userID, 600.0)
	suite.NoError(err)
	_, err = suite.service.PayOrder(ctx, paid.ID)
	suite.NoError(err)
	_, err = suite.service.CreateOrder(ctx, 103, 1, 200, 700.0)
	suite.NoError(err)

	summary, err := suite.service.GetSummary(ctx, userID)

	suite.NoError(err)
	suite.Equal(int64(1), summary.PendingPayment)
	suite.Equal(int64(1), summary.WonNotPaid)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product
go test ./service -run TestOrderTestSuite/TestGetSummary -v
```

Expected: FAIL because `GetSummary` is undefined.

- [ ] **Step 3: Add order summary DTO**

Add to `backend/product/model/order.go`:

```go
type OrderSummaryResponse struct {
	PendingPayment int64 `json:"pendingPayment"`
	WonNotPaid     int64 `json:"wonNotPaid"`
}
```

- [ ] **Step 4: Add DAO count method**

Add to `backend/product/dao/order.go`:

```go
func (d *OrderDAO) CountByWinnerAndStatus(ctx context.Context, winnerID int64, status model.OrderStatus) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.Order{}).
		Where("winner_id = ? AND status = ?", winnerID, status).
		Count(&count).Error
	return count, err
}
```

- [ ] **Step 5: Add service method**

Add to `backend/product/service/order.go`:

```go
func (s *OrderService) GetSummary(ctx context.Context, userID int64) (*model.OrderSummaryResponse, error) {
	pending, err := s.orderDAO.CountByWinnerAndStatus(ctx, userID, model.OrderStatusPending)
	if err != nil {
		return nil, err
	}
	return &model.OrderSummaryResponse{
		PendingPayment: pending,
		WonNotPaid:     pending,
	}, nil
}
```

- [ ] **Step 6: Add handler**

Add to `backend/product/handler/order.go`:

```go
func (h *OrderHandler) Summary(ctx context.Context, c *app.RequestContext) {
	userIDStr := string(c.Request.Header.Peek("X-User-ID"))
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
		return
	}

	summary, err := h.orderService.GetSummary(ctx, userID)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取订单汇总失败: " + err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{"code": 0, "message": "success", "data": summary})
}
```

- [ ] **Step 7: Register product route**

Add in `backend/product/main.go` order routes:

```go
v1.GET("/orders/summary", orderHandler.Summary)
```

Place it before `v1.GET("/orders/:id", orderHandler.Get)` to avoid `summary` being parsed as `:id`.

- [ ] **Step 8: Verify product service**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product
gofmt -w model/order.go dao/order.go service/order.go handler/order.go main.go service/order_test.go
go test ./dao ./service ./handler
```

Expected: PASS.

- [ ] **Step 9: Commit**

Run:

```bash
git add backend/product/model/order.go backend/product/dao/order.go backend/product/service/order.go backend/product/handler/order.go backend/product/main.go backend/product/service/order_test.go
git commit -m "feat(product): add order summary endpoint"
```

Expected: commit succeeds.

---

## Task 3: Gateway Touchpoint Summary Aggregation

**Files:**
- Create: `backend/gateway/handler/touchpoint.go`
- Create: `backend/gateway/handler/touchpoint_test.go`
- Modify: `backend/gateway/router/router.go`

- [ ] **Step 1: Add gateway handler test**

Create `backend/gateway/handler/touchpoint_test.go`:

```go
package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestTouchpointHandlerSummary(t *testing.T) {
	auctionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/notifications/summary", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":0,"message":"success","data":{"unreadTotal":2,"outbid":1,"endingSoon":0}}`))
	}))
	defer auctionServer.Close()

	productServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/orders/summary", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":0,"message":"success","data":{"pendingPayment":1,"wonNotPaid":1}}`))
	}))
	defer productServer.Close()

	h := NewTouchpointHandler(auctionServer.URL, productServer.URL)
	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/notifications/summary")
	c.Set("user_id", int64(123))

	h.GetNotificationSummary(context.Background(), c)

	assert.Equal(t, http.StatusOK, c.Response.StatusCode())
	body := string(c.Response.Body())
	assert.Contains(t, body, `"unreadTotal":2`)
	assert.Contains(t, body, `"pendingPayment":1`)
	assert.Contains(t, body, `"wonNotPaid":1`)
	assert.Contains(t, body, `"outbid":1`)
}

func TestTouchpointHandlerSummaryFallsBackForUpstreamFailure(t *testing.T) {
	auctionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer auctionServer.Close()

	productServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":0,"message":"success","data":{"pendingPayment":1,"wonNotPaid":1}}`))
	}))
	defer productServer.Close()

	h := NewTouchpointHandler(auctionServer.URL, productServer.URL)
	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/notifications/summary")
	c.Set("user_id", int64(123))

	h.GetNotificationSummary(context.Background(), c)

	assert.Equal(t, http.StatusOK, c.Response.StatusCode())
	body := string(c.Response.Body())
	assert.Contains(t, body, `"unreadTotal":0`)
	assert.Contains(t, body, `"outbid":0`)
	assert.Contains(t, body, `"endingSoon":0`)
	assert.Contains(t, body, `"pendingPayment":1`)
}

func TestTouchpointHandlerSummaryPropagatesAuthFailure(t *testing.T) {
	auctionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer auctionServer.Close()

	productServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("product upstream must not be called after auth failure")
	}))
	defer productServer.Close()

	h := NewTouchpointHandler(auctionServer.URL, productServer.URL)
	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/notifications/summary")
	c.Set("user_id", int64(123))

	h.GetNotificationSummary(context.Background(), c)

	assert.Equal(t, http.StatusUnauthorized, c.Response.StatusCode())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway
go test ./handler -run TestTouchpointHandlerSummary -v
```

Expected: FAIL because `NewTouchpointHandler` is undefined.

- [ ] **Step 3: Implement gateway aggregation handler**

Create `backend/gateway/handler/touchpoint.go`:

```go
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

type TouchpointHandler struct {
	auctionURL string
	productURL string
	client     *http.Client
}

type upstreamEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type auctionSummary struct {
	UnreadTotal int64 `json:"unreadTotal"`
	Outbid      int64 `json:"outbid"`
	EndingSoon  int64 `json:"endingSoon"`
}

type orderSummary struct {
	PendingPayment int64 `json:"pendingPayment"`
	WonNotPaid     int64 `json:"wonNotPaid"`
}

type touchpointSummary struct {
	UnreadTotal    int64 `json:"unreadTotal"`
	PendingPayment int64 `json:"pendingPayment"`
	WonNotPaid     int64 `json:"wonNotPaid"`
	Outbid         int64 `json:"outbid"`
	EndingSoon     int64 `json:"endingSoon"`
}

func NewTouchpointHandler(auctionURL, productURL string) *TouchpointHandler {
	return &TouchpointHandler{
		auctionURL: strings.TrimRight(auctionURL, "/"),
		productURL: strings.TrimRight(productURL, "/"),
		client:     &http.Client{Timeout: 2 * time.Second},
	}
}

func (h *TouchpointHandler) GetNotificationSummary(ctx context.Context, c *app.RequestContext) {
	token := string(c.Request.Header.Peek("Authorization"))
	userID := toString(c.GetInt64("user_id"))

	auctionData := auctionSummary{}
	if err := h.fetch(ctx, h.auctionURL+"/api/v1/notifications/summary", token, userID, &auctionData); isAuthUpstreamError(err) {
		writeUpstreamAuthError(c, err)
		return
	}

	orderData := orderSummary{}
	if err := h.fetch(ctx, h.productURL+"/api/v1/orders/summary", token, userID, &orderData); isAuthUpstreamError(err) {
		writeUpstreamAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"code":    0,
		"message": "success",
		"data": touchpointSummary{
			UnreadTotal:    auctionData.UnreadTotal,
			PendingPayment: orderData.PendingPayment,
			WonNotPaid:     orderData.WonNotPaid,
			Outbid:         auctionData.Outbid,
			EndingSoon:     auctionData.EndingSoon,
		},
	})
}

func (h *TouchpointHandler) fetch(ctx context.Context, url, token, userID string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	if userID != "" {
		req.Header.Set("X-User-ID", userID)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return upstreamStatusError{status: resp.StatusCode}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var env upstreamEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return err
	}
	if env.Code != 0 && env.Code != 200 {
		return fmt.Errorf("upstream code %d: %s", env.Code, env.Message)
	}
	if len(env.Data) == 0 {
		return nil
	}
	return json.Unmarshal(env.Data, out)
}

type upstreamStatusError struct {
	status int
}

func (e upstreamStatusError) Error() string {
	return fmt.Sprintf("upstream status %d", e.status)
}

func isAuthUpstreamError(err error) bool {
	statusErr, ok := err.(upstreamStatusError)
	return ok && (statusErr.status == http.StatusUnauthorized || statusErr.status == http.StatusForbidden)
}

func writeUpstreamAuthError(c *app.RequestContext, err error) {
	statusErr := err.(upstreamStatusError)
	c.JSON(statusErr.status, map[string]interface{}{
		"code":    statusErr.status,
		"message": "authentication failed",
	})
}
```

- [ ] **Step 4: Register gateway routes**

Modify `backend/gateway/router/router.go`:

```go
touchpointHandler := handler.NewTouchpointHandler(cfg.Services.AuctionURL, cfg.Services.ProductURL)
```

Then add inside `authGroup` notification routes:

```go
authGroup.GET("/notifications/summary", touchpointHandler.GetNotificationSummary)
authGroup.POST("/notifications/read-category", auctionProxy.Forward)
authGroup.GET("/live/pending-reminder", auctionProxy.Forward)
```

- [ ] **Step 5: Verify gateway**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway
gofmt -w handler/touchpoint.go handler/touchpoint_test.go router/router.go
go test ./handler ./router
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add backend/gateway/handler/touchpoint.go backend/gateway/handler/touchpoint_test.go backend/gateway/router/router.go
git commit -m "feat(gateway): aggregate touchpoint summary"
```

Expected: commit succeeds.

---

## Task 4: Backend Live Pending Reminder

**Files:**
- Create: `backend/auction/migration/002_create_live_stream_reminder_receipts.sql`
- Create: `backend/auction/model/live_stream_reminder_receipt.go`
- Create: `backend/auction/dao/live_stream_reminder_receipt.go`
- Create: `backend/auction/service/live_reminder.go`
- Create: `backend/auction/handler/live_reminder.go`
- Modify: `backend/auction/service/live_stream_stats.go`
- Modify: `backend/auction/main.go`
- Test: `backend/auction/service/live_reminder_test.go`

- [ ] **Step 1: Add migration**

Create `backend/auction/migration/002_create_live_stream_reminder_receipts.sql`:

```sql
CREATE TABLE IF NOT EXISTS live_stream_reminder_receipts (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  live_stream_id BIGINT NOT NULL,
  live_started_at BIGINT NOT NULL,
  reminded_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_user_stream_started (user_id, live_stream_id, live_started_at),
  KEY idx_user_id (user_id)
);
```

- [ ] **Step 2: Add model**

Create `backend/auction/model/live_stream_reminder_receipt.go`:

```go
package model

import "time"

type LiveStreamReminderReceipt struct {
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        int64     `json:"user_id" gorm:"not null;uniqueIndex:uk_user_stream_started,priority:1;index"`
	LiveStreamID  int64     `json:"live_stream_id" gorm:"not null;uniqueIndex:uk_user_stream_started,priority:2"`
	LiveStartedAt int64     `json:"live_started_at" gorm:"not null;uniqueIndex:uk_user_stream_started,priority:3"`
	RemindedAt    time.Time `json:"reminded_at" gorm:"not null"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (LiveStreamReminderReceipt) TableName() string {
	return "live_stream_reminder_receipts"
}

type PendingLiveReminderResponse struct {
	HasReminder bool        `json:"hasReminder"`
	Stream      *StreamInfo `json:"stream"`
}

type StreamInfo struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatarUrl"`
	StatusText string `json:"statusText"`
	LiveRoomID int64  `json:"liveRoomId"`
	StartedAt  int64  `json:"startedAt"`
}
```

- [ ] **Step 3: Preserve live session start time**

Modify `backend/auction/service/live_stream_stats.go`:

```go
type LiveStreamStats struct {
	LiveStreamID   int64      `json:"live_stream_id"`
	FollowerCount  int        `json:"follower_count"`
	IsHot          bool       `json:"is_hot"`
	ScheduledStart *time.Time `json:"scheduled_start,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	Status         string     `json:"status"` // "pending", "live", "ended"
}
```

In `StartLive`, set a real session start timestamp when the stream enters `live`:

```go
now := time.Now()
stats.Status = "live"
stats.StartedAt = &now
stats.ScheduledStart = nil
```

In `EndLive`, clear the session marker before saving or deleting stats:

```go
stats.Status = "ended"
stats.StartedAt = nil
```

- [ ] **Step 4: Add receipt DAO**

Create `backend/auction/dao/live_stream_reminder_receipt.go`:

```go
package dao

import (
	"context"
	"time"

	"auction-service/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LiveStreamReminderReceiptDAO struct {
	db *gorm.DB
}

func NewLiveStreamReminderReceiptDAO(db *gorm.DB) *LiveStreamReminderReceiptDAO {
	return &LiveStreamReminderReceiptDAO{db: db}
}

func (d *LiveStreamReminderReceiptDAO) Claim(ctx context.Context, userID, liveStreamID, startedAt int64) (bool, error) {
	receipt := &model.LiveStreamReminderReceipt{
		UserID:        userID,
		LiveStreamID:  liveStreamID,
		LiveStartedAt: startedAt,
		RemindedAt:    time.Now(),
	}
	result := d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(receipt)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}
```

- [ ] **Step 5: Add live reminder service**

Create `backend/auction/service/live_reminder.go`:

```go
package service

import (
	"context"

	"auction-service/dao"
	"auction-service/model"
)

type LiveSessionResolver interface {
	GetActiveSession(ctx context.Context, liveStreamID int64) (*model.StreamInfo, error)
}

type LiveStatsSessionResolver struct {
	statsService *LiveStreamStatsService
}

func NewLiveStatsSessionResolver(statsService *LiveStreamStatsService) *LiveStatsSessionResolver {
	return &LiveStatsSessionResolver{statsService: statsService}
}

func (r *LiveStatsSessionResolver) GetActiveSession(ctx context.Context, liveStreamID int64) (*model.StreamInfo, error) {
	stats, err := r.statsService.GetStats(ctx, liveStreamID)
	if err != nil {
		return nil, err
	}
	if stats == nil || stats.Status != "live" || stats.StartedAt == nil {
		return nil, nil
	}
	return &model.StreamInfo{
		ID:         liveStreamID,
		Name:       "关注直播间",
		AvatarURL:  "",
		StatusText: "正在直播",
		LiveRoomID: liveStreamID,
		StartedAt:  stats.StartedAt.UnixMilli(),
	}, nil
}

type LiveReminderService struct {
	followDAO            FollowDAO
	liveSessionResolver LiveSessionResolver
	receiptDAO           *dao.LiveStreamReminderReceiptDAO
}

func NewLiveReminderService(followDAO FollowDAO, liveSessionResolver LiveSessionResolver, receiptDAO *dao.LiveStreamReminderReceiptDAO) *LiveReminderService {
	return &LiveReminderService{followDAO: followDAO, liveSessionResolver: liveSessionResolver, receiptDAO: receiptDAO}
}

func (s *LiveReminderService) GetPendingReminder(ctx context.Context, userID int64) (*model.PendingLiveReminderResponse, error) {
	follows, err := s.followDAO.GetUserFollows(ctx, userID, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(follows) == 0 || !follows[0].NotificationEnabled {
		return &model.PendingLiveReminderResponse{HasReminder: false, Stream: nil}, nil
	}

	liveStreamID := follows[0].LiveStreamID
	session, err := s.liveSessionResolver.GetActiveSession(ctx, liveStreamID)
	if err != nil {
		return nil, err
	}
	if session == nil || session.StartedAt <= 0 {
		return &model.PendingLiveReminderResponse{HasReminder: false, Stream: nil}, nil
	}

	claimed, err := s.receiptDAO.Claim(ctx, userID, liveStreamID, session.StartedAt)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return &model.PendingLiveReminderResponse{HasReminder: false, Stream: nil}, nil
	}

	return &model.PendingLiveReminderResponse{
		HasReminder: true,
		Stream:      session,
	}, nil
}
```

语义要求：`StartedAt` 必须来自 `StartLive` 写入的真实直播 session。不要在 `GetPendingReminder` 中用请求时间、小时桶或登录时间合成 `StartedAt`。

- [ ] **Step 6: Add handler**

Create `backend/auction/handler/live_reminder.go`:

```go
package handler

import (
	"context"

	"auction-service/service"
	"github.com/cloudwego/hertz/pkg/app"
)

type LiveReminderHandler struct {
	service *service.LiveReminderService
}

func NewLiveReminderHandler(service *service.LiveReminderService) *LiveReminderHandler {
	return &LiveReminderHandler{service: service}
}

func (h *LiveReminderHandler) GetPendingReminder(ctx context.Context, c *app.RequestContext) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
		return
	}
	userID := userIDInterface.(int64)

	result, err := h.service.GetPendingReminder(ctx, userID)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取开播提醒失败: " + err.Error()})
		return
	}
	c.JSON(200, map[string]interface{}{"code": 0, "message": "success", "data": result})
}
```

- [ ] **Step 7: Wire main and route**

Modify `backend/auction/main.go`:

```go
liveStreamReminderReceiptDAO := dao.NewLiveStreamReminderReceiptDAO(db)
liveStreamStatsService := service.NewLiveStreamStatsService()
liveSessionResolver := service.NewLiveStatsSessionResolver(liveStreamStatsService)
liveReminderService := service.NewLiveReminderService(userLiveStreamFollowDAO, liveSessionResolver, liveStreamReminderReceiptDAO)
liveReminderHandler := handler.NewLiveReminderHandler(liveReminderService)
```

Add AutoMigrate target if this service currently migrates models:

```go
&model.LiveStreamReminderReceipt{},
```

Extend `registerRoutes` signature with `liveReminderHandler *handler.LiveReminderHandler`, then add:

```go
v1.GET("/live/pending-reminder", liveReminderHandler.GetPendingReminder)
```

- [ ] **Step 8: Verify auction service**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction
gofmt -w model/live_stream_reminder_receipt.go dao/live_stream_reminder_receipt.go service/live_reminder.go service/live_stream_stats.go handler/live_reminder.go main.go
go test ./dao ./service ./handler
```

Expected: PASS.

- [ ] **Step 9: Commit**

Run:

```bash
git add backend/auction/migration/002_create_live_stream_reminder_receipts.sql backend/auction/model/live_stream_reminder_receipt.go backend/auction/dao/live_stream_reminder_receipt.go backend/auction/service/live_reminder.go backend/auction/service/live_stream_stats.go backend/auction/handler/live_reminder.go backend/auction/main.go
git commit -m "feat(auction): add pending live reminder endpoint"
```

Expected: commit succeeds.

---

## Task 5: Frontend Replace Touchpoint Mock With API

**Files:**
- Modify: `frontend/h5/src/services/notification.ts`
- Modify: `frontend/h5/src/hooks/useTouchpointNotifications.ts`
- Modify: `frontend/h5/src/components/MobileShell/MobileContainer.tsx`
- Test: `frontend/h5/src/__tests__/components/MobileShell.test.tsx`
- Test: `frontend/h5/src/pages/User/__tests__/Profile.test.tsx`

- [ ] **Step 1: Add service API contracts**

Add to `frontend/h5/src/services/notification.ts`:

```ts
export interface TouchpointSummary {
  unreadTotal: number;
  pendingPayment: number;
  wonNotPaid: number;
  outbid: number;
  endingSoon: number;
}

export interface PendingLiveReminderResponse {
  hasReminder: boolean;
  stream: {
    id: string | number;
    name: string;
    avatarUrl: string;
    statusText?: string;
    liveRoomId?: string | number;
    startedAt?: number;
  } | null;
}
```

Extend `notificationApi`:

```ts
getTouchpointSummary: (): Promise<TouchpointSummary> => {
  return get<TouchpointSummary>('/notifications/summary');
},
markCategoryAsRead: (category: 'pendingPayment' | 'outbid' | 'endingSoon' | 'all'): Promise<void> => {
  return post<void>('/notifications/read-category', { category });
},
getPendingLiveReminder: (): Promise<PendingLiveReminderResponse> => {
  return get<PendingLiveReminderResponse>('/live/pending-reminder');
},
```

- [ ] **Step 2: Replace hook implementation**

Replace `frontend/h5/src/hooks/useTouchpointNotifications.ts` with:

```ts
import { useEffect, useState } from 'react';
import { notificationApi, TouchpointSummary } from '../services/notification';

export interface TouchpointNotifications {
  pendingPayment: number;
  unreadTotal: number;
}

const EMPTY: TouchpointSummary = {
  unreadTotal: 0,
  pendingPayment: 0,
  wonNotPaid: 0,
  outbid: 0,
  endingSoon: 0,
};

export function useTouchpointNotifications(): TouchpointNotifications {
  const [summary, setSummary] = useState<TouchpointSummary>(EMPTY);

  useEffect(() => {
    let alive = true;

    notificationApi.getTouchpointSummary()
      .then((next) => {
        if (alive) setSummary(next);
      })
      .catch(() => {
        if (alive) setSummary(EMPTY);
      });

    return () => {
      alive = false;
    };
  }, []);

  return {
    pendingPayment: summary.pendingPayment,
    unreadTotal: summary.unreadTotal,
  };
}
```

- [ ] **Step 3: Replace local live reminder marker**

In `frontend/h5/src/components/MobileShell/MobileContainer.tsx`, replace the `localStorage.getItem('pending_live_reminder')` effect with:

```ts
useEffect(() => {
  let alive = true;

  notificationApi.getPendingLiveReminder()
    .then((result) => {
      if (!alive || !result.hasReminder || !result.stream) {
        return;
      }
      setReminderStream(result.stream);
      setIsReminderOpen(true);
    })
    .catch(() => {
      if (localStorage.getItem('pending_live_reminder') === '1') {
        localStorage.removeItem('pending_live_reminder');
        setReminderStream(mockLiveReminderStream);
        setIsReminderOpen(true);
      }
    });

  return () => {
    alive = false;
  };
}, []);
```

Also change modal `stream` prop to:

```tsx
stream={reminderStream}
```

- [ ] **Step 4: Verify frontend tests**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/__tests__/components/MobileShell.test.tsx src/pages/User/__tests__/Profile.test.tsx
npm run build
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add frontend/h5/src/services/notification.ts frontend/h5/src/hooks/useTouchpointNotifications.ts frontend/h5/src/components/MobileShell/MobileContainer.tsx frontend/h5/src/__tests__/components/MobileShell.test.tsx frontend/h5/src/pages/User/__tests__/Profile.test.tsx
git commit -m "feat(h5): load touchpoints from backend"
```

Expected: commit succeeds.

---

## Task 6: Frontend Toast From Existing Notification WS

**Files:**
- Modify: `frontend/h5/src/services/websocket.ts`
- Modify: `frontend/h5/src/pages/Live/index.tsx`
- Test: `frontend/h5/src/components/Toast/__tests__/ToastProvider.test.tsx`

- [ ] **Step 1: Ensure notification messages are observable**

In `frontend/h5/src/services/websocket.ts`, ensure `handleMessage` routes `notification` messages through existing handlers. The final branch should contain:

```ts
if (message.type === 'notification') {
  this.notificationHandlers.forEach((handler) => handler(message.data as NotificationData));
  this.emit('notification', message.data);
  return;
}
```

- [ ] **Step 2: Map notification to Toast in Live page**

In `frontend/h5/src/pages/Live/index.tsx`, add helper:

```ts
function toastPayloadFromNotification(notification: any) {
  switch (notification.type) {
    case 'bid_outbid':
      return {
        type: 'danger' as const,
        title: notification.title || '您已被超价',
        message: notification.content || '当前最高价已更新',
        actionText: '重新出价',
      };
    case 'auction_won':
      return {
        type: 'success' as const,
        title: notification.title || '恭喜中标',
        message: notification.content || '请尽快完成支付',
        actionText: '去支付',
      };
    case 'auction_starting':
      return {
        type: 'warning' as const,
        title: notification.title || '截拍预警',
        message: notification.content || '拍品即将截拍',
      };
    default:
      return null;
  }
}
```

Inside WS setup:

```ts
const shownNotificationIds = new Set<number | string>();

ws.on('notification', (notification: any) => {
  const id = notification.id;
  if (id && shownNotificationIds.has(id)) {
    return;
  }
  if (id) {
    shownNotificationIds.add(id);
  }

  const payload = toastPayloadFromNotification(notification);
  if (!payload) {
    return;
  }

  showGlobalToast({
    ...payload,
    onAction: payload.actionText === '去支付'
      ? () => navigate('/result')
      : () => window.scrollTo({ top: document.body.scrollHeight, behavior: 'smooth' }),
  });
});
```

- [ ] **Step 3: Remove development Toast Demo panel**

Remove the `import.meta.env.DEV` Toast Demo block from `frontend/h5/src/pages/Live/index.tsx`.

Remove `.toastDemoPanel` styles from `frontend/h5/src/pages/Live/Live.module.css`.

- [ ] **Step 4: Verify frontend**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/components/Toast/__tests__/ToastProvider.test.tsx
npm run build
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add frontend/h5/src/services/websocket.ts frontend/h5/src/pages/Live/index.tsx frontend/h5/src/pages/Live/Live.module.css frontend/h5/src/components/Toast/__tests__/ToastProvider.test.tsx
git commit -m "feat(h5): show toast from notification websocket"
```

Expected: commit succeeds.

---

## Final Verification

- [ ] **Backend unit tests**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction
go test ./...
```

Expected: PASS.

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product
go test ./...
```

Expected: PASS.

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway
go test ./...
```

Expected: PASS.

- [ ] **Frontend focused tests**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/components/BadgeDot/__tests__/BadgeDot.test.tsx src/__tests__/components/MobileShell.test.tsx src/pages/User/__tests__/Profile.test.tsx src/components/Toast/__tests__/ToastProvider.test.tsx
npm run build
```

Expected: PASS.

- [ ] **Manual integration smoke**

Run backend services through the existing local workflow, then verify:

```bash
curl -H "Authorization: Bearer ${TOKEN}" http://localhost:8080/api/v1/notifications/summary
curl -H "Authorization: Bearer ${TOKEN}" http://localhost:8080/api/v1/live/pending-reminder
```

Expected:

- `notifications/summary` returns `data.unreadTotal` and `data.pendingPayment`.
- `live/pending-reminder` returns either `hasReminder=false` or a single stream and does not repeat on the second request for the same receipt key.

---

## Risk Notes

- `gateway-service` aggregation must treat upstream failures as zeros for badge data; badge failure must not block H5 render.
- `product-service` route `/orders/summary` must be registered before `/orders/:id`.
- Live reminder receipts must be keyed by a real live session `StartedAt`; `GetPendingReminder` must never synthesize the session key from request time.
- Existing WebSocket auth failure behavior is HTTP 401 before upgrade, while H5 also supports close code `4401`. This plan keeps backend behavior unchanged.
- Worktree has known unrelated changes. During execution, only add files listed in each task.

---

## Implementation Choice

Plan complete and saved to `docs/superpowers/plans/2026-05-30-user-touchpoints-backend.md`. Two execution options:

1. Subagent-Driven (recommended) - Dispatch a fresh subagent per task, review between tasks, fast iteration.
2. Inline Execution - Execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?
