// Package script 场景剧本：按预定义顺序串行执行多个 sub-scenario。
//
// 5 个内置剧本（spec 演示场景）：
//   - quickstart   ：dummy → e2e（演示骨架联调，不需要业务全栈）
//   - antisnipe    ：e2e → antisnipe（先打通业务，再演示防狙击）
//   - reliability  ：callback（HMAC + Probe + DLQ + 幂等）
//   - chaos        ：chaos（基线 → 注入 → 恢复曲线）
//   - fullshow     ：dummy → pressure → e2e → antisnipe → callback → chaos（一镜到底）
//
// 设计：本场景不是再写 HTTP，而是直接调用已注册到 runner.Runner 的子场景，复用同一个
// resultDAO/Broker/WS 进度上报通道；ProgressEmitter 由 runner 注入；子步骤的进度被
// 重新映射到 [stepStartPct, stepEndPct] 的子区间。
package script

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"test-service/runner"
)

// ScenarioGetter Script 场景调用注册表的能力（避免循环依赖）
type ScenarioGetter interface {
	Get(scenarioType string) (runner.Scenario, bool)
}

// Step 单个剧本步骤
type Step struct {
	Name string          `json:"name"`        // 子场景类型，如 "e2e" / "antisnipe"
	Cfg  json.RawMessage `json:"cfg"`         // 透传给子场景的配置
	Note string          `json:"note,omitempty"`
}

// Definition 剧本定义
type Definition struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Steps []Step `json:"steps"`
}

// Library 内置剧本库
var Library = map[string]Definition{
	"quickstart": {
		Name:  "quickstart",
		Title: "骨架联调（dummy + e2e）",
		Steps: []Step{
			{Name: "dummy", Note: "5s 假任务"},
			{Name: "e2e", Note: "业务全链路"},
		},
	},
	"antisnipe": {
		Name:  "antisnipe",
		Title: "防狙击演示（e2e + antisnipe）",
		Steps: []Step{
			{Name: "e2e", Note: "业务热身"},
			{Name: "antisnipe", Note: "5 用例延时验证"},
		},
	},
	"reliability": {
		Name:  "reliability",
		Title: "可靠投递（callback 全 6 用例）",
		Steps: []Step{
			{Name: "callback"},
		},
	},
	"chaos": {
		Name:  "chaos",
		Title: "故障演练（基线/注入/恢复曲线）",
		Steps: []Step{
			{Name: "chaos"},
		},
	},
	"fullshow": {
		Name:  "fullshow",
		Title: "全场剧本（dummy → pressure → e2e → antisnipe → callback → chaos）",
		Steps: []Step{
			{Name: "dummy"},
			{Name: "pressure"},
			{Name: "e2e"},
			{Name: "antisnipe"},
			{Name: "callback"},
			{Name: "chaos"},
		},
	},
}

// Config 场景入参
type Config struct {
	ScriptName string `json:"script_name"`
	// Steps 显式指定的剧本步骤，优先级高于 ScriptName
	Steps []Step `json:"steps,omitempty"`
}

// StepResult 单步执行结果
type StepResult struct {
	Name      string `json:"name"`
	OK        bool   `json:"ok"`
	StartedAt time.Time `json:"started_at"`
	Cost      string `json:"cost"`
	Result    any    `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Report 整体输出
type Report struct {
	Script string       `json:"script"`
	Title  string       `json:"title"`
	Steps  []StepResult `json:"steps"`
	AllOK  bool         `json:"all_ok"`
}

// Scenario 实现 runner.Scenario
type Scenario struct {
	getter ScenarioGetter
}

// NewScenario 构造
func NewScenario(g ScenarioGetter) *Scenario {
	return &Scenario{getter: g}
}

// Type 标识
func (s *Scenario) Type() string { return "script" }

// Run 顺序跑剧本
func (s *Scenario) Run(ctx context.Context, raw json.RawMessage, p runner.ProgressEmitter) (any, error) {
	cfg := Config{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("invalid script config: %w", err)
		}
	}

	steps := cfg.Steps
	title := ""
	if len(steps) == 0 {
		def, ok := Library[cfg.ScriptName]
		if !ok {
			return nil, errors.New("unknown script: " + cfg.ScriptName)
		}
		steps = def.Steps
		title = def.Title
	}

	rep := Report{Script: cfg.ScriptName, Title: title, Steps: make([]StepResult, 0, len(steps)), AllOK: true}
	total := len(steps)
	for i, st := range steps {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		stepStart := i * 100 / total
		stepEnd := (i + 1) * 100 / total

		sub, ok := s.getter.Get(st.Name)
		if !ok {
			rep.Steps = append(rep.Steps, StepResult{
				Name: st.Name, OK: false, StartedAt: time.Now(),
				Error: "scenario not registered: " + st.Name,
			})
			rep.AllOK = false
			continue
		}

		startedAt := time.Now()
		// 子进度映射到 [stepStart, stepEnd]
		subEmitter := stepProgressMapper(p, st.Name, stepStart, stepEnd, i+1, total)
		out, err := sub.Run(ctx, st.Cfg, subEmitter)

		sr := StepResult{
			Name: st.Name, StartedAt: startedAt,
			Cost: time.Since(startedAt).String(),
		}
		if err != nil {
			sr.OK = false
			sr.Error = err.Error()
			rep.AllOK = false
		} else {
			sr.OK = true
			sr.Result = out
		}
		rep.Steps = append(rep.Steps, sr)

		if p != nil {
			p.Emit(stepEnd, st.Name, map[string]any{
				"step":      i + 1,
				"of":        total,
				"step_ok":   sr.OK,
				"step_cost": sr.Cost,
			})
		}
	}
	return rep, nil
}

// stepProgressMapper 把子场景的 0-100 进度映射到 [outStart, outEnd] 子区间
func stepProgressMapper(parent runner.ProgressEmitter, stepName string, outStart, outEnd, idx, total int) runner.ProgressEmitter {
	return emitterFn(func(progress int, sub string, metrics map[string]any) {
		if parent == nil {
			return
		}
		if progress < 0 {
			progress = 0
		}
		if progress > 100 {
			progress = 100
		}
		mapped := outStart + progress*(outEnd-outStart)/100
		out := map[string]any{
			"sub_step": sub,
			"step_idx": idx,
			"of":       total,
		}
		for k, v := range metrics {
			out[k] = v
		}
		parent.Emit(mapped, stepName, out)
	})
}

type emitterFn func(progress int, step string, metrics map[string]any)

func (f emitterFn) Emit(p int, step string, metrics map[string]any) { f(p, step, metrics) }
