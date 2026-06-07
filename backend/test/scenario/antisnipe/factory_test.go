package antisnipe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"test-service/client/auction"
)

func TestSDKAuctionFactory_PrepareUsesMerchantRuleThenAuction(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPost && r.Header.Get("X-User-Role") != auction.RoleMerchant {
			t.Fatalf("X-User-Role: want merchant, got %q for %s", r.Header.Get("X-User-Role"), r.URL.Path)
		}

		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/admin/products":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"code":201,"data":{"id":42}}`))
		case "POST /api/v1/products/42/publish":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":200,"data":{"id":42,"status":1}}`))
		case "POST /api/v1/products/42/rules":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"code":201}`))
		case "POST /api/v1/auctions":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":77}`))
		case "GET /api/v1/auctions/77":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":77,"status":1}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	cli := auction.NewClient(srv.URL, 3*time.Second)
	fac := NewSDKAuctionFactory(cli, 9001, 30)

	auctionID, err := fac.Prepare(context.Background(), CaseLastSecond)
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	if auctionID != 77 {
		t.Fatalf("auctionID: want 77, got %d", auctionID)
	}

	want := []string{
		"POST /api/v1/admin/products",
		"POST /api/v1/products/42/publish",
		"POST /api/v1/products/42/rules",
		"POST /api/v1/auctions",
		"GET /api/v1/auctions/77",
	}
	if len(calls) != len(want) {
		t.Fatalf("calls: want %v, got %v", want, calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("call[%d]: want %s, got %s", i, want[i], calls[i])
		}
	}
}
