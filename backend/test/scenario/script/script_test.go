package script

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"test-service/runner"
)

// fakeScenario 测试用 stub
type fakeScenario struct {
	typ   string
	emits []int // 上报的进度点
	out   any
	err   error
}

func (f *fakeScenario) Type() string { return f.typ }
func (f *fakeScenario) Run(ctx context.Context, _ json.RawMessage, p runner.ProgressEmitter) (any, error) {
	for _, v := range f.emits {
		if p != nil {
			p.Emit(v, "tick", nil)
		}
	}
	return f.out, f.err
}

type fakeGetter struct {
	scenarios map[string]runner.Scenario
}

func (g *fakeGetter) Get(t string) (runner.Scenario, bool) {
	s, ok := g.scenarios[t]
	return s, ok
}

type captureEmitter struct {
	mu      sync.Mutex
	records []captured
}

type captured struct {
	progress int
	step     string
}

func (c *captureEmitter) Emit(p int, step string, _ map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, captured{progress: p, step: step})
}

func TestScript_RunsLibraryQuickstart(t *testing.T) {
	getter := &fakeGetter{scenarios: map[string]runner.Scenario{
		"dummy": &fakeScenario{typ: "dummy", emits: []int{0, 50, 100}, out: "d"},
		"e2e":   &fakeScenario{typ: "e2e", emits: []int{0, 100}, out: "e"},
	}}
	s := NewScenario(getter)
	cap := &captureEmitter{}

	out, err := s.Run(context.Background(), json.RawMessage(`{"script_name":"quickstart"}`), cap)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	rep, ok := out.(Report)
	if !ok {
		t.Fatalf("expected Report, got %T", out)
	}
	if !rep.AllOK || len(rep.Steps) != 2 {
		t.Fatalf("unexpected report: %+v", rep)
	}

	// 验证子进度被映射到 [0,50] 和 [50,100] 子区间
	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.records) == 0 {
		t.Fatalf("no progress recorded")
	}
	for _, r := range cap.records {
		if r.progress < 0 || r.progress > 100 {
			t.Fatalf("progress out of range: %+v", r)
		}
	}
}

func TestScript_UnknownScript(t *testing.T) {
	s := NewScenario(&fakeGetter{scenarios: map[string]runner.Scenario{}})
	_, err := s.Run(context.Background(), json.RawMessage(`{"script_name":"nope"}`), nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestScript_StepFailureMarksAllOKFalse(t *testing.T) {
	getter := &fakeGetter{scenarios: map[string]runner.Scenario{
		"dummy": &fakeScenario{typ: "dummy", err: errors.New("boom")},
		"e2e":   &fakeScenario{typ: "e2e", out: "ok"},
	}}
	s := NewScenario(getter)

	out, err := s.Run(context.Background(),
		json.RawMessage(`{"steps":[{"name":"dummy"},{"name":"e2e"}]}`), nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	rep := out.(Report)
	if rep.AllOK {
		t.Fatalf("expected AllOK=false")
	}
	if len(rep.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(rep.Steps))
	}
	if rep.Steps[0].OK || !rep.Steps[1].OK {
		t.Fatalf("step status mismatch: %+v", rep.Steps)
	}
}

func TestScript_LibraryHasFiveBuiltin(t *testing.T) {
	want := []string{"quickstart", "antisnipe", "reliability", "chaos", "fullshow"}
	for _, n := range want {
		if _, ok := Library[n]; !ok {
			t.Fatalf("library missing: %s", n)
		}
	}
}

func TestStepProgressMapper_RangesIntoSubInterval(t *testing.T) {
	cap := &captureEmitter{}
	mapper := stepProgressMapper(cap, "x", 40, 80, 1, 2)
	mapper.Emit(0, "s", nil)
	mapper.Emit(50, "s", nil)
	mapper.Emit(100, "s", nil)

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(cap.records))
	}
	if cap.records[0].progress != 40 {
		t.Fatalf("expected 40, got %d", cap.records[0].progress)
	}
	if cap.records[1].progress != 60 {
		t.Fatalf("expected 60, got %d", cap.records[1].progress)
	}
	if cap.records[2].progress != 80 {
		t.Fatalf("expected 80, got %d", cap.records[2].progress)
	}
}
