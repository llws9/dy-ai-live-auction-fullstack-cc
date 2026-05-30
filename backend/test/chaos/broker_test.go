package chaos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestBroker_InjectAndRecover(t *testing.T) {
	b := Default()
	b.RecoverAll()
	b.Inject(Profile{ID: "p1", Type: FaultLatency, LatencyMs: 10})
	if got := len(b.List()); got != 1 {
		t.Fatalf("expected 1 profile, got %d", got)
	}
	b.Recover("p1")
	if got := len(b.List()); got != 0 {
		t.Fatalf("expected 0 after recover, got %d", got)
	}
}

func TestBroker_Expiry(t *testing.T) {
	b := Default()
	b.RecoverAll()
	b.Inject(Profile{ID: "p2", Type: FaultLatency, LatencyMs: 1, Duration: 20 * time.Millisecond})
	time.Sleep(40 * time.Millisecond)
	if got := len(b.List()); got != 0 {
		t.Fatalf("expected expired removed, got %d", got)
	}
}

func TestChaosTransport_DenyOnDisconnect(t *testing.T) {
	b := Default()
	b.RecoverAll()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cli := &http.Client{Transport: NewTransport(nil), Timeout: time.Second}
	// 先无 chaos：应成功
	resp, err := cli.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	resp.Body.Close()

	// 注入断连：应失败
	b.Inject(Profile{ID: "deny", Type: FaultDisconnect})
	defer b.RecoverAll()
	if _, err := cli.Get(srv.URL); err == nil {
		t.Fatalf("expected ErrInjected, got nil")
	}
}

func TestChaosTransport_LatencyRespectsContext(t *testing.T) {
	b := Default()
	b.RecoverAll()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	b.Inject(Profile{ID: "slow", Type: FaultLatency, LatencyMs: 500})
	defer b.RecoverAll()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	cli := &http.Client{Transport: NewTransport(nil)}
	if _, err := cli.Do(req); err == nil {
		t.Fatalf("expected ctx timeout, got nil")
	}
}

func TestChaosTransport_ErrorRateNonZero(t *testing.T) {
	b := Default()
	b.RecoverAll()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	b.Inject(Profile{ID: "rate", Type: FaultErrorRate, ErrorRate: 1.0})
	defer b.RecoverAll()

	cli := &http.Client{Transport: NewTransport(nil), Timeout: time.Second}
	var ok int32
	for i := 0; i < 10; i++ {
		if _, err := cli.Get(srv.URL); err == nil {
			atomic.AddInt32(&ok, 1)
		}
	}
	if ok != 0 {
		t.Fatalf("expected 0 success at 100%% error rate, got %d", ok)
	}
}
