package chaos

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	tchaos "test-service/chaos"
)

type stubProgress struct{ events int }

func (s *stubProgress) Emit(progress int, step string, metrics map[string]any) { s.events++ }

func TestChaosScenario_RunWithErrorRateInjection(t *testing.T) {
	tchaos.Default().RecoverAll()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := Config{
		ProbeURL:    srv.URL,
		ProbeQPS:    10,
		BaselineSec: 1,
		InjectSec:   1,
		RecoverSec:  1,
		FaultType:   string(tchaos.FaultErrorRate),
		ErrorRate:   1.0,
	}
	raw, _ := json.Marshal(cfg)

	scn := NewScenario(srv.URL)
	p := &stubProgress{}
	out, err := scn.Run(context.Background(), raw, p)
	if err != nil {
		t.Fatalf("run err: %v", err)
	}
	rep, ok := out.(*Report)
	if !ok || rep == nil {
		t.Fatalf("expected *Report, got %T", out)
	}

	if rep.BaselineErrorRate > 0.2 {
		t.Errorf("baseline should be near 0, got %.2f", rep.BaselineErrorRate)
	}
	if rep.InjectErrorRate < 0.8 {
		t.Errorf("inject error rate should be high (≥0.8) under 100%% rate, got %.2f", rep.InjectErrorRate)
	}
	if rep.RecoverErrorRate > 0.2 {
		t.Errorf("recover should drop, got %.2f", rep.RecoverErrorRate)
	}
	if !rep.AllOK {
		t.Errorf("expected all_ok=true under correct injection, got false (inject=%.2f baseline=%.2f recover=%.2f)",
			rep.InjectErrorRate, rep.BaselineErrorRate, rep.RecoverErrorRate)
	}
	if p.events == 0 {
		t.Errorf("expected progress emits, got 0")
	}
	if got := len(rep.Buckets); got != 3 {
		t.Errorf("expected 3 buckets, got %d", got)
	}
}

func TestChaosScenario_UnknownFaultType(t *testing.T) {
	scn := NewScenario("http://example/health")
	cfg := Config{FaultType: "no_such_fault", BaselineSec: 1, InjectSec: 1, RecoverSec: 1, ProbeQPS: 1}
	raw, _ := json.Marshal(cfg)
	if _, err := scn.Run(context.Background(), raw, nil); err == nil {
		t.Fatalf("expected error for unknown fault_type")
	}
}
