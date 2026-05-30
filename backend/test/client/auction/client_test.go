package auction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
