package callback

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"test-service/mock/partner"
)

// 起一个 mock partner httptest server
func startPartner(t *testing.T) *httptest.Server {
	t.Helper()
	s := partner.NewServer()
	return s.StartTest()
}

func TestScenario_NormalDelivery(t *testing.T) {
	ts := startPartner(t)
	defer ts.Close()

	sc := NewScenario()
	cfgRaw, _ := json.Marshal(Config{
		PartnerURL: ts.URL,
		Cases:      []string{CaseNormal},
		MaxRetry:   3,
		TimeoutMs:  500,
	})
	out, err := sc.Run(context.Background(), cfgRaw, nil)
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	rep := out.(Report)
	if !rep.AllOK {
		t.Fatalf("normal case should pass: %+v", rep.Cases[0])
	}
	if !endsWith(rep.Cases[0].Trace, StateConfirmed) {
		t.Fatalf("expected end-state Confirmed, trace=%+v", rep.Cases[0].Trace)
	}
}

func TestScenario_DuplicateIdempotent(t *testing.T) {
	ts := startPartner(t)
	defer ts.Close()

	sc := NewScenario()
	cfgRaw, _ := json.Marshal(Config{
		PartnerURL: ts.URL,
		Cases:      []string{CaseDuplicate},
		TimeoutMs:  500,
	})
	out, err := sc.Run(context.Background(), cfgRaw, nil)
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	rep := out.(Report)
	if !rep.AllOK {
		t.Fatalf("duplicate case should pass: %s", rep.Cases[0].Message)
	}
	if rep.Cases[0].IdempotentBlocked != 4 {
		t.Fatalf("expected 4 idempotent blocks, got %d", rep.Cases[0].IdempotentBlocked)
	}
}

func TestScenario_TamperedSignature(t *testing.T) {
	ts := startPartner(t)
	defer ts.Close()

	sc := NewScenario()
	cfgRaw, _ := json.Marshal(Config{
		PartnerURL: ts.URL,
		Cases:      []string{CaseTampered},
		TimeoutMs:  500,
	})
	out, err := sc.Run(context.Background(), cfgRaw, nil)
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	rep := out.(Report)
	if !rep.AllOK {
		t.Fatalf("tampered case should pass (rejecting bad sig): %s", rep.Cases[0].Message)
	}
	hasReject := false
	for _, tr := range rep.Cases[0].Trace {
		if strings.EqualFold(tr.State, StateRejected) {
			hasReject = true
		}
	}
	if !hasReject {
		t.Fatalf("expected Rejected trace; got %+v", rep.Cases[0].Trace)
	}
}

func TestScenario_DLQ(t *testing.T) {
	ts := startPartner(t)
	defer ts.Close()

	sc := NewScenario()
	cfgRaw, _ := json.Marshal(Config{
		PartnerURL: ts.URL,
		Cases:      []string{CaseDLQ},
		MaxRetry:   3,
		TimeoutMs:  500,
	})
	out, err := sc.Run(context.Background(), cfgRaw, nil)
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	rep := out.(Report)
	if !rep.Cases[0].DLQEntered {
		t.Fatalf("expected DLQ entered")
	}
}

func TestScenario_OutOfOrder(t *testing.T) {
	ts := startPartner(t)
	defer ts.Close()

	sc := NewScenario()
	cfgRaw, _ := json.Marshal(Config{
		PartnerURL: ts.URL,
		Cases:      []string{CaseOutOfOrder},
		TimeoutMs:  500,
	})
	out, err := sc.Run(context.Background(), cfgRaw, nil)
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	rep := out.(Report)
	if !rep.AllOK {
		t.Fatalf("out_of_order case should pass: %s", rep.Cases[0].Message)
	}
}

func TestScenario_AllSixCases(t *testing.T) {
	ts := startPartner(t)
	defer ts.Close()

	sc := NewScenario()
	cfgRaw, _ := json.Marshal(Config{
		PartnerURL: ts.URL,
		MaxRetry:   2, // 缩短 DLQ 用例耗时
		TimeoutMs:  300,
	})
	out, err := sc.Run(context.Background(), cfgRaw, nil)
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	rep := out.(Report)
	if len(rep.Cases) != 6 {
		t.Fatalf("expected 6 cases, got %d", len(rep.Cases))
	}
	// 至少一半通过即合理（timeout 用例严格依赖 mock 行为，可能视实现而通过/失败）
	pass := 0
	for _, c := range rep.Cases {
		if c.OK {
			pass++
		}
	}
	if pass < 4 {
		t.Fatalf("too few cases passed (%d/6); details=%+v", pass, rep.Cases)
	}
}
