package auction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestSDK_CreateProduct 创建拍品 → 200 + 返回 ID
func TestSDK_CreateProduct(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/products" || r.Method != http.MethodPost {
			t.Errorf("path/method: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-User-ID") != "100001" {
			t.Errorf("X-User-ID missing")
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":42,"name":"iPhone","status":1}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	step := c.CreateProduct(context.Background(), 100001, CreateProductReq{Name: "iPhone"})
	if !step.OK {
		t.Fatalf("CreateProduct failed: %s err=%v", step.Message, step.Err)
	}
	if step.RefID != 42 {
		t.Fatalf("RefID: want 42, got %d", step.RefID)
	}
	if step.Step != "create_product" {
		t.Fatalf("Step name: %s", step.Step)
	}
	if step.DurationMs <= 0 {
		t.Fatalf("DurationMs should be > 0")
	}
}

// TestSDK_CreateAuction 创建拍卖 → 201 + 返回 ID
func TestSDK_CreateAuction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":7,"product_id":42,"status":0,"current_price":100}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	step := c.CreateAuction(context.Background(), 100001, CreateAuctionReq{
		ProductID: 42, StartPrice: 100, Increment: 10, Duration: 30,
	})
	if !step.OK {
		t.Fatalf("CreateAuction failed: %s", step.Message)
	}
	if step.RefID != 7 {
		t.Fatalf("RefID: want 7, got %d", step.RefID)
	}
}

// TestSDK_PlaceBid 出价（测试模式：body 注入 user_id）
func TestSDK_PlaceBid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/auctions/7/bids") {
			t.Errorf("path: %s", r.URL.Path)
		}
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b["user_id"] == nil {
			t.Errorf("user_id should be in body")
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"success":true,"current_price":110}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	step := c.PlaceBid(context.Background(), 100002, 7, 110)
	if !step.OK {
		t.Fatalf("PlaceBid failed: %s", step.Message)
	}
}

// TestSDK_GetAuction 查询拍卖详情
func TestSDK_GetAuction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":7,"status":2,"current_price":150,"winner_id":100002}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	a, step := c.GetAuction(context.Background(), 7)
	if !step.OK {
		t.Fatalf("GetAuction failed: %s", step.Message)
	}
	if a.ID != 7 || a.Status != 2 || a.WinnerID != 100002 {
		t.Fatalf("auction parse wrong: %+v", a)
	}
}

// TestSDK_WaitAuctionStarted 轮询直到 status >= 1（Ongoing）
func TestSDK_WaitAuctionStarted(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		// 前两次返回 0（Pending），第三次返回 1（Ongoing）
		status := 0
		if calls >= 3 {
			status = 1
		}
		_, _ = w.Write([]byte(`{"id":7,"status":` + itoa(status) + `}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	step := c.WaitAuctionStarted(context.Background(), 7, 50*time.Millisecond, 500*time.Millisecond)
	if !step.OK {
		t.Fatalf("WaitAuctionStarted failed: %s", step.Message)
	}
	if calls < 3 {
		t.Fatalf("expected at least 3 polls, got %d", calls)
	}
}

// TestSDK_WaitAuctionEnded 轮询直到 status >= 2（Ended）
func TestSDK_WaitAuctionEnded(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		status := 1
		if calls >= 2 {
			status = 2
		}
		_, _ = w.Write([]byte(`{"id":7,"status":` + itoa(status) + `,"winner_id":100002}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	step := c.WaitAuctionEnded(context.Background(), 7, 50*time.Millisecond, 500*time.Millisecond)
	if !step.OK {
		t.Fatalf("WaitAuctionEnded failed: %s", step.Message)
	}
}

// TestSDK_SubscribeSkyLamp 点天灯订阅
func TestSDK_SubscribeSkyLamp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sky-lamp/subscriptions" {
			t.Errorf("path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"code":200,"subscription":{"id":99,"auction_id":7}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	step := c.SubscribeSkyLamp(context.Background(), 100003, 7)
	if !step.OK {
		t.Fatalf("SubscribeSkyLamp failed: %s", step.Message)
	}
	if step.RefID != 99 {
		t.Fatalf("RefID: want 99, got %d", step.RefID)
	}
}

// TestSDK_FindOrderByAuction 用 winner_id 拉订单 + 按 auction_id 过滤
func TestSDK_FindOrderByAuction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "user_id=100002") {
			t.Errorf("expected user_id query, got %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"items":[{"id":1,"auction_id":99,"winner_id":100002},{"id":2,"auction_id":7,"winner_id":100002}],"total":2}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	orders, step := c.FindOrdersByAuction(context.Background(), 100002, 7)
	if !step.OK {
		t.Fatalf("FindOrdersByAuction failed: %s", step.Message)
	}
	if len(orders) != 1 || orders[0].ID != 2 {
		t.Fatalf("filter wrong: got %+v", orders)
	}
}

func TestDoSetsMerchantIdentityHeaders(t *testing.T) {
	var capturedUserID atomic.Value
	var capturedUsername atomic.Value
	var capturedRole atomic.Value
	capturedUserID.Store("")
	capturedUsername.Store("")
	capturedRole.Store("")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID.Store(r.Header.Get("X-User-ID"))
		capturedUsername.Store(r.Header.Get("X-Username"))
		capturedRole.Store(r.Header.Get("X-User-Role"))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":42}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	step := c.CreateProductAs(context.Background(), Actor{
		UserID:   9002,
		Username: "merchant_9002",
		Role:     RoleMerchant,
	}, CreateProductReq{Name: "merchant product"})
	if !step.OK {
		t.Fatalf("CreateProductAs failed: %s err=%v", step.Message, step.Err)
	}
	if capturedUserID.Load().(string) != "9002" {
		t.Fatalf("X-User-ID: want 9002, got %s", capturedUserID.Load().(string))
	}
	if capturedUsername.Load().(string) != "merchant_9002" {
		t.Fatalf("X-Username mismatch: %s", capturedUsername.Load().(string))
	}
	if capturedRole.Load().(string) != "merchant" {
		t.Fatalf("X-User-Role: want merchant, got %s", capturedRole.Load().(string))
	}
}

func TestDoSetsGatewayJWTAuthorization(t *testing.T) {
	const secret = "test-jwt-secret"
	var capturedAuth atomic.Value
	capturedAuth.Store("")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth.Store(r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":42}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	c.SetJWTSecret(secret)
	step := c.CreateAuctionAs(context.Background(), Actor{
		UserID:   9002,
		Username: "merchant_9002",
		Role:     RoleMerchant,
	}, CreateAuctionReq{ProductID: 42, StartPrice: 100, Increment: 10, Duration: 30})
	if !step.OK {
		t.Fatalf("CreateAuctionAs failed: %s err=%v", step.Message, step.Err)
	}

	auth := capturedAuth.Load().(string)
	if !strings.HasPrefix(auth, "Bearer ") {
		t.Fatalf("Authorization must be Bearer token, got %q", auth)
	}
	token, err := jwt.ParseWithClaims(strings.TrimPrefix(auth, "Bearer "), jwt.MapClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("invalid jwt: token=%v err=%v", token, err)
	}
	claims := token.Claims.(jwt.MapClaims)
	if int64(claims["user_id"].(float64)) != 9002 {
		t.Fatalf("user_id claim mismatch: %v", claims["user_id"])
	}
	if claims["username"] != "merchant_9002" {
		t.Fatalf("username claim mismatch: %v", claims["username"])
	}
	if int(claims["role"].(float64)) != 1 {
		t.Fatalf("role claim mismatch: %v", claims["role"])
	}
}

func TestTopUpUserBalanceCallsInternalEndpoint(t *testing.T) {
	var called atomic.Int32
	var capturedPath atomic.Value
	capturedPath.Store("")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		capturedPath.Store(r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"user_id":1001,"balance":"500.00"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	balance, step := c.TopUpUserBalance(context.Background(), 1001, "500.00")
	if !step.OK {
		t.Fatalf("TopUpUserBalance failed: %s err=%v", step.Message, step.Err)
	}
	if balance != "500.00" {
		t.Fatalf("balance: want 500.00, got %s", balance)
	}
	if capturedPath.Load().(string) != "/internal/test/user-balance" {
		t.Fatalf("path: want /internal/test/user-balance, got %s", capturedPath.Load().(string))
	}
	if called.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", called.Load())
	}
}

func TestTopUpUserBalanceSendsInternalToken(t *testing.T) {
	var capturedToken atomic.Value
	capturedToken.Store("")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedToken.Store(r.Header.Get("X-Internal-Token"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"user_id":1001,"balance":"500.00"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	c.SetInternalToken("internal-secret")
	_, step := c.TopUpUserBalance(context.Background(), 1001, "500.00")
	if !step.OK {
		t.Fatalf("TopUpUserBalance failed: %s err=%v", step.Message, step.Err)
	}
	if capturedToken.Load().(string) != "internal-secret" {
		t.Fatalf("X-Internal-Token: want internal-secret, got %q", capturedToken.Load().(string))
	}
}

func TestPurchaseFixedPriceIncludesIdempotencyKey(t *testing.T) {
	var capturedPath atomic.Value
	var capturedKey atomic.Value
	capturedPath.Store("")
	capturedKey.Store("")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath.Store(r.URL.Path)
		capturedKey.Store(r.Header.Get("X-Idempotency-Key"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"order_id":88}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	orderID, step := c.PurchaseFixedPriceItem(context.Background(), Actor{
		UserID:   1001,
		Username: "buyer_1001",
		Role:     RoleUser,
	}, 77, "idem-77")
	if !step.OK {
		t.Fatalf("PurchaseFixedPriceItem failed: %s err=%v", step.Message, step.Err)
	}
	if orderID != 88 {
		t.Fatalf("orderID: want 88, got %d", orderID)
	}
	if capturedPath.Load().(string) != "/api/v1/fixed-price/items/77/purchase" {
		t.Fatalf("path: got %s", capturedPath.Load().(string))
	}
	if capturedKey.Load().(string) != "idem-77" {
		t.Fatalf("idempotency key mismatch: %s", capturedKey.Load().(string))
	}
}

func TestFollowAndFollowStatusUseBuyerIdentity(t *testing.T) {
	var callCount atomic.Int32
	var firstPath atomic.Value
	var firstRole atomic.Value
	var secondPath atomic.Value
	var secondRole atomic.Value
	firstPath.Store("")
	firstRole.Store("")
	secondPath.Store("")
	secondRole.Store("")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idx := callCount.Add(1)
		switch idx {
		case 1:
			firstPath.Store(r.URL.Path)
			firstRole.Store(r.Header.Get("X-User-Role"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
		case 2:
			secondPath.Store(r.URL.Path)
			secondRole.Store(r.Header.Get("X-User-Role"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"is_following":true}`))
		default:
			t.Fatalf("unexpected extra call %d", idx)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	actor := Actor{UserID: 1001, Username: "buyer_1001", Role: RoleUser}
	if step := c.FollowLiveStream(context.Background(), actor, 66); !step.OK {
		t.Fatalf("FollowLiveStream failed: %s err=%v", step.Message, step.Err)
	}
	ok, step := c.GetFollowStatus(context.Background(), actor, 66)
	if !step.OK {
		t.Fatalf("GetFollowStatus failed: %s err=%v", step.Message, step.Err)
	}
	if !ok {
		t.Fatalf("expected follow status true")
	}
	if firstPath.Load().(string) != "/api/v1/live-streams/66/follow" || firstRole.Load().(string) != "user" {
		t.Fatalf("follow call mismatch: path=%s role=%s", firstPath.Load().(string), firstRole.Load().(string))
	}
	if secondPath.Load().(string) != "/api/v1/live-streams/66/follow-status" || secondRole.Load().(string) != "user" {
		t.Fatalf("follow-status call mismatch: path=%s role=%s", secondPath.Load().(string), secondRole.Load().(string))
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i == 1 {
		return "1"
	}
	if i == 2 {
		return "2"
	}
	return "0"
}
