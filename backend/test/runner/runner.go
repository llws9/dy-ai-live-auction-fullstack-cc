package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/google/uuid"

	"test-service/dao"
	"test-service/model"
)

// ErrUnknownScenario 未注册场景
var ErrUnknownScenario = errors.New("unknown scenario")

// ErrTestNotFound 测试不存在
var ErrTestNotFound = errors.New("test not found")

// EmitFunc 进度上报回调（runner 通过 Broker 注入）
type EmitFunc func(testID string, progress int, step string, metrics map[string]any)

// Runner 任务调度器
type Runner struct {
	resultDAO *dao.ResultDAO

	scenarios sync.Map // type → Scenario
	tasks     sync.Map // testID → *task
	emit      EmitFunc
}

type task struct {
	id     string
	cancel context.CancelFunc
}

// New 构造 Runner
func New(d *dao.ResultDAO) *Runner {
	return &Runner{resultDAO: d, emit: func(string, int, string, map[string]any) {}}
}

// SetEmitter 注入外部 emit（如 WS broker），不设置时为 noop
func (r *Runner) SetEmitter(f EmitFunc) {
	if f != nil {
		r.emit = f
	}
}

// Register 注册一个场景
func (r *Runner) Register(s Scenario) {
	r.scenarios.Store(s.Type(), s)
}

// Get 按类型获取已注册场景（供 script 等组合场景按名取子场景）
func (r *Runner) Get(scenarioType string) (Scenario, bool) {
	v, ok := r.scenarios.Load(scenarioType)
	if !ok {
		return nil, false
	}
	return v.(Scenario), true
}

// Submit 提交一个测试任务，返回 test_id
func (r *Runner) Submit(ctx context.Context, scenarioType string, cfg json.RawMessage) (string, error) {
	v, ok := r.scenarios.Load(scenarioType)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownScenario, scenarioType)
	}
	s := v.(Scenario)

	id := uuid.NewString()
	cfgStr := string(cfg)
	if cfgStr == "" {
		cfgStr = "{}"
	}
	rec := &model.TestResult{
		ID:         id,
		TestType:   s.Type(),
		Status:     model.StatusRunning,
		ConfigJSON: cfgStr,
		CreatedAt:  time.Now(),
	}
	if err := r.resultDAO.Save(ctx, rec); err != nil {
		hlog.CtxErrorf(ctx, "[runner] save record failed type=%s err=%v", scenarioType, err)
		return "", err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	runCtx = WithTestID(runCtx, id)
	r.tasks.Store(id, &task{id: id, cancel: cancel})

	hlog.CtxInfof(ctx, "[runner] submitted test_id=%s type=%s", id, scenarioType)
	go r.run(runCtx, id, s, cfg)
	return id, nil
}

// ctxKeyTestID 用于在 ctx 中传递 test_id（场景内部可读取以做 seed 记录）
type ctxKeyTestID struct{}

// WithTestID 把 testID 注入 ctx
func WithTestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyTestID{}, id)
}

// TestIDFromContext 从 ctx 取 testID（找不到返回 ""）
func TestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyTestID{}).(string); ok {
		return v
	}
	return ""
}

func (r *Runner) run(ctx context.Context, id string, s Scenario, cfg json.RawMessage) {
	defer r.tasks.Delete(id)
	startedAt := time.Now()
	hlog.Infof("[runner] start test_id=%s type=%s", id, s.Type())

	emitter := emitterFunc(func(p int, step string, metrics map[string]any) {
		hlog.Debugf("[runner] emit test_id=%s progress=%d step=%q", id, p, step)
		r.emit(id, p, step, metrics)
	})

	res, err := s.Run(ctx, cfg, emitter)
	now := time.Now()
	cost := now.Sub(startedAt)

	if errors.Is(err, context.Canceled) {
		hlog.Infof("[runner] cancelled test_id=%s cost=%s", id, cost)
		_ = r.resultDAO.UpdateStatus(context.Background(), id, model.StatusCancelled, "", "cancelled", &now)
		return
	}
	if err != nil {
		hlog.Errorf("[runner] failed test_id=%s cost=%s err=%v", id, cost, err)
		r.emit(id, 100, "failed", map[string]any{
			"error":  err.Error(),
			"status": model.StatusFailed,
		})
		_ = r.resultDAO.UpdateStatus(context.Background(), id, model.StatusFailed, "", err.Error(), &now)
		return
	}

	resultJSON := "{}"
	if res != nil {
		if b, mErr := json.Marshal(res); mErr == nil {
			resultJSON = string(b)
		}
	}
	hlog.Infof("[runner] completed test_id=%s cost=%s", id, cost)
	_ = r.resultDAO.UpdateStatus(context.Background(), id, model.StatusCompleted, resultJSON, "", &now)
}

// Cancel 取消一个正在执行的任务
func (r *Runner) Cancel(id string) error {
	v, ok := r.tasks.Load(id)
	if !ok {
		hlog.Warnf("[runner] cancel: not found test_id=%s", id)
		return ErrTestNotFound
	}
	v.(*task).cancel()
	hlog.Infof("[runner] cancel signal sent test_id=%s", id)
	return nil
}

// emitterFunc 适配 ProgressEmitter
type emitterFunc func(progress int, step string, metrics map[string]any)

func (f emitterFunc) Emit(p int, step string, metrics map[string]any) {
	f(p, step, metrics)
}
