// Package chaos 场景 G：故障注入与恢复观测。
//
// 编排：basline → inject → during → recover → after，每阶段以固定 QPS 请求
// gateway 健康端点（默认）或自定义 URL；记录每秒成功率/延迟，最终输出：
//   - 错误率时序（per second buckets）
//   - 注入到第一个错误的延迟（detection_latency_ms）
//   - 恢复到第一个稳定成功秒的延迟（recovery_latency_ms）
package chaos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"test-service/chaos"
	"test-service/runner"
)

// Phase 阶段名
const (
	PhaseBaseline = "baseline"
	PhaseInject   = "inject"
	PhaseRecover  = "recover"
)

// Config 场景参数
type Config struct {
	ProbeURL      string  `json:"probe_url"`        // 默认 gateway /health
	ProbeQPS      int     `json:"probe_qps"`        // 默认 20
	BaselineSec   int     `json:"baseline_sec"`     // 默认 3
	InjectSec     int     `json:"inject_sec"`       // 默认 8
	RecoverSec    int     `json:"recover_sec"`      // 默认 5
	FaultType     string  `json:"fault_type"`       // latency|jitter|error_rate|disconnect|redis_flap|mq_pause
	LatencyMs     int     `json:"latency_ms"`
	JitterMs      int     `json:"jitter_ms"`
	ErrorRate     float64 `json:"error_rate"`
}

// Bucket 一秒桶
type Bucket struct {
	TS         time.Time `json:"ts"`
	Phase      string    `json:"phase"`
	OKCount    int       `json:"ok_count"`
	FailCount  int       `json:"fail_count"`
	AvgLatency int64     `json:"avg_latency_ms"`
}

// Report 输出
type Report struct {
	Profile             chaos.Profile `json:"profile"`
	Buckets             []Bucket      `json:"buckets"`
	BaselineErrorRate   float64       `json:"baseline_error_rate"`
	InjectErrorRate     float64       `json:"inject_error_rate"`
	RecoverErrorRate    float64       `json:"recover_error_rate"`
	DetectionLatencyMs  int64         `json:"detection_latency_ms"`
	RecoveryLatencyMs   int64         `json:"recovery_latency_ms"`
	AllOK               bool          `json:"all_ok"`
}

// Scenario 实现 runner.Scenario
type Scenario struct {
	defaultProbeURL string
	hc              *http.Client
}

// NewScenario 构造；defaultProbeURL 通常是 gateway:8080/health
func NewScenario(defaultProbeURL string) *Scenario {
	return &Scenario{
		defaultProbeURL: defaultProbeURL,
		hc: &http.Client{
			Transport: chaos.NewTransport(nil),
			Timeout:   2 * time.Second,
		},
	}
}

// Type 场景标识
func (s *Scenario) Type() string { return "chaos" }

// Run 跑一轮
func (s *Scenario) Run(ctx context.Context, raw json.RawMessage, p runner.ProgressEmitter) (any, error) {
	cfg := Config{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("invalid chaos config: %w", err)
		}
	}
	applyDefaults(&cfg, s.defaultProbeURL)

	profile, err := buildProfile(cfg)
	if err != nil {
		return nil, err
	}

	rep := &Report{Profile: profile, Buckets: make([]Bucket, 0, 64)}

	totalSec := cfg.BaselineSec + cfg.InjectSec + cfg.RecoverSec

	// baseline
	if err := s.runPhase(ctx, cfg, PhaseBaseline, cfg.BaselineSec, rep, p, 0, totalSec); err != nil {
		return rep, err
	}

	// inject
	chaos.Default().Inject(profile)
	injectStart := time.Now()
	if err := s.runPhase(ctx, cfg, PhaseInject, cfg.InjectSec, rep, p, cfg.BaselineSec, totalSec); err != nil {
		chaos.Default().Recover(profile.ID)
		return rep, err
	}

	// recover
	chaos.Default().Recover(profile.ID)
	recoverStart := time.Now()
	if err := s.runPhase(ctx, cfg, PhaseRecover, cfg.RecoverSec, rep, p, cfg.BaselineSec+cfg.InjectSec, totalSec); err != nil {
		return rep, err
	}

	finalize(rep, injectStart, recoverStart)
	return rep, nil
}

// runPhase 跑一个阶段
func (s *Scenario) runPhase(
	ctx context.Context, cfg Config, phase string, seconds int,
	rep *Report, p runner.ProgressEmitter, secOffset, total int,
) error {
	for i := 0; i < seconds; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		bucket := s.runOneSecond(ctx, cfg, phase)
		rep.Buckets = append(rep.Buckets, bucket)
		if p != nil {
			progress := (secOffset + i + 1) * 100 / total
			p.Emit(progress, phase, map[string]any{
				"ok":   bucket.OKCount,
				"fail": bucket.FailCount,
				"avg_latency_ms": bucket.AvgLatency,
			})
		}
	}
	return nil
}

// runOneSecond 在 1 秒内以 cfg.ProbeQPS 速率打 ProbeURL
func (s *Scenario) runOneSecond(ctx context.Context, cfg Config, phase string) Bucket {
	bucket := Bucket{TS: time.Now(), Phase: phase}
	interval := time.Second / time.Duration(cfg.ProbeQPS)
	deadline := time.Now().Add(time.Second)

	var (
		mu        sync.Mutex
		latencies []int64
		wg        sync.WaitGroup
	)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, cfg.ProbeURL, nil)
			resp, err := s.hc.Do(req)
			elapsed := time.Since(start).Milliseconds()
			ok := err == nil && resp != nil && resp.StatusCode < 400
			if resp != nil {
				resp.Body.Close()
			}
			mu.Lock()
			if ok {
				bucket.OKCount++
			} else {
				bucket.FailCount++
			}
			latencies = append(latencies, elapsed)
			mu.Unlock()
		}()
		time.Sleep(interval)
	}
	wg.Wait()

	if n := len(latencies); n > 0 {
		var sum int64
		for _, v := range latencies {
			sum += v
		}
		bucket.AvgLatency = sum / int64(n)
	}
	return bucket
}

// finalize 计算汇总指标
func finalize(rep *Report, injectStart, recoverStart time.Time) {
	rep.BaselineErrorRate = phaseErrRate(rep.Buckets, PhaseBaseline)
	rep.InjectErrorRate = phaseErrRate(rep.Buckets, PhaseInject)
	rep.RecoverErrorRate = phaseErrRate(rep.Buckets, PhaseRecover)

	// detection_latency: 注入后第一个出现 fail 的桶时间 - injectStart
	for _, b := range rep.Buckets {
		if b.Phase == PhaseInject && b.FailCount > 0 {
			rep.DetectionLatencyMs = b.TS.Sub(injectStart).Milliseconds()
			break
		}
	}

	// recovery_latency: recover 阶段第一个 fail==0 的桶时间 - recoverStart
	for _, b := range rep.Buckets {
		if b.Phase == PhaseRecover && b.FailCount == 0 && b.OKCount > 0 {
			rep.RecoveryLatencyMs = b.TS.Sub(recoverStart).Milliseconds()
			break
		}
	}

	// 通过条件：注入阶段错误率高于基线，恢复阶段错误率回落
	rep.AllOK = rep.InjectErrorRate > rep.BaselineErrorRate &&
		rep.RecoverErrorRate <= rep.InjectErrorRate
}

func phaseErrRate(bs []Bucket, phase string) float64 {
	var ok, fail int
	for _, b := range bs {
		if b.Phase != phase {
			continue
		}
		ok += b.OKCount
		fail += b.FailCount
	}
	if ok+fail == 0 {
		return 0
	}
	return float64(fail) / float64(ok+fail)
}

func applyDefaults(c *Config, defaultURL string) {
	if c.ProbeURL == "" {
		c.ProbeURL = defaultURL
		if c.ProbeURL == "" {
			c.ProbeURL = "http://localhost:8080/health"
		}
	}
	if c.ProbeQPS <= 0 {
		c.ProbeQPS = 20
	}
	if c.BaselineSec <= 0 {
		c.BaselineSec = 3
	}
	if c.InjectSec <= 0 {
		c.InjectSec = 8
	}
	if c.RecoverSec <= 0 {
		c.RecoverSec = 5
	}
	if c.FaultType == "" {
		c.FaultType = string(chaos.FaultErrorRate)
	}
	if c.ErrorRate == 0 && c.FaultType == string(chaos.FaultErrorRate) {
		c.ErrorRate = 0.5
	}
	if c.LatencyMs == 0 && c.FaultType == string(chaos.FaultLatency) {
		c.LatencyMs = 500
	}
	if c.JitterMs == 0 && c.FaultType == string(chaos.FaultJitter) {
		c.JitterMs = 800
	}
}

func buildProfile(cfg Config) (chaos.Profile, error) {
	t := chaos.FaultType(cfg.FaultType)
	switch t {
	case chaos.FaultLatency, chaos.FaultJitter, chaos.FaultErrorRate,
		chaos.FaultDisconnect, chaos.FaultRedisFlap, chaos.FaultMQPause:
	default:
		return chaos.Profile{}, errors.New("unknown fault_type: " + cfg.FaultType)
	}
	return chaos.Profile{
		ID:        fmt.Sprintf("chaos-%s-%d", cfg.FaultType, time.Now().UnixNano()),
		Type:      t,
		LatencyMs: cfg.LatencyMs,
		JitterMs:  cfg.JitterMs,
		ErrorRate: cfg.ErrorRate,
	}, nil
}
