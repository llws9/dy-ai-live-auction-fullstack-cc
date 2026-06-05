package pressure

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

// TestClient_Success 成功路径：返回 2xx 与 latency
func TestClient_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer t-token" {
			t.Errorf("auth header: want 'Bearer t-token', got %q", got)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["amount"] == nil {
			t.Errorf("amount missing in body")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "Bearer t-token", 2*time.Second)
	res := c.PlaceBid(context.Background(), 100, 8888, 99)
	if !res.OK {
		t.Fatalf("expected OK, got err=%v code=%d", res.Err, res.StatusCode)
	}
	if res.StatusCode != 200 {
		t.Fatalf("StatusCode: want 200, got %d", res.StatusCode)
	}
	if res.Latency <= 0 {
		t.Fatalf("Latency should be > 0")
	}
}

// TestClient_JWTPerUser 确保压测请求按 userID 注入合法 JWT。
func TestClient_JWTPerUser(t *testing.T) {
	const secret = "jwt-secret"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if got == "" {
			t.Fatalf("Authorization header missing")
		}
		tokenText := strings.TrimPrefix(got, "Bearer ")
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenText, claims, func(token *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			t.Fatalf("invalid JWT: token=%v err=%v", token, err)
		}
		if claims["user_id"] != float64(100123) {
			t.Fatalf("user_id claim: want 100123, got %v", claims["user_id"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	c := NewJWTClient(srv.URL, secret, 2*time.Second)
	res := c.PlaceBid(context.Background(), 100, 8888, 100123)
	if !res.OK {
		t.Fatalf("expected OK, got err=%v code=%d", res.Err, res.StatusCode)
	}
}

// TestClient_HTTPError 5xx 必须标记失败 + 携带 code
func TestClient_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 2*time.Second)
	res := c.PlaceBid(context.Background(), 100, 1, 1)
	if res.OK {
		t.Fatalf("expected failure on 500")
	}
	if res.StatusCode != 500 {
		t.Fatalf("StatusCode: want 500, got %d", res.StatusCode)
	}
}

// TestClient_BusinessError code != 0 && != 200 视为业务失败
func TestClient_BusinessError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":40001,"message":"price too low"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 2*time.Second)
	res := c.PlaceBid(context.Background(), 100, 1, 1)
	if res.OK {
		t.Fatalf("expected business failure")
	}
	if res.StatusCode != 40001 {
		t.Fatalf("StatusCode (业务码): want 40001, got %d", res.StatusCode)
	}
}

// TestClient_Timeout 客户端超时必须返回失败 + ctx 错误
func TestClient_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 50*time.Millisecond)
	res := c.PlaceBid(context.Background(), 100, 1, 1)
	if res.OK {
		t.Fatalf("expected timeout failure")
	}
	if res.Err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// TestClient_ConcurrentSafe 并发调用安全
func TestClient_ConcurrentSafe(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 2*time.Second)
	const N = 100
	done := make(chan struct{}, N)
	for i := 0; i < N; i++ {
		go func() {
			_ = c.PlaceBid(context.Background(), 100, 1, 1)
			done <- struct{}{}
		}()
	}
	for i := 0; i < N; i++ {
		<-done
	}
	if got := atomic.LoadInt32(&hits); got != N {
		t.Fatalf("hits: want %d, got %d", N, got)
	}
}
