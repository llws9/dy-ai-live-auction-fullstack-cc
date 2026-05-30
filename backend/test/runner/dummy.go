package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// DummyScenario 仅用于 M1 联调：分步上报进度，并产出有视觉价值的 metrics
type DummyScenario struct {
	totalDuration time.Duration
}

// NewDummyScenario 构造（duration 为期望总时长）
func NewDummyScenario(duration time.Duration) *DummyScenario {
	if duration <= 0 {
		duration = 5 * time.Second
	}
	return &DummyScenario{totalDuration: duration}
}

// Type 场景类型标识
func (DummyScenario) Type() string { return "dummy" }

// 步骤名（前端可直接展示）
var dummySteps = []string{
	"初始化测试上下文",
	"加载场景配置",
	"准备测试数据 (seed)",
	"启动并发竞拍线程",
	"出价请求 (warm-up)",
	"出价请求 (peak)",
	"采集业务指标",
	"校验数据一致性",
	"清理测试数据",
	"生成测试报告",
}

// Run 10 步，每步 totalDuration/10，并按步骤输出可视化 metrics
//
// metrics 字段约定（前端 Dashboard 直接展示）：
//   - step           当前步骤序号 (1~10)
//   - qps            实时 QPS（峰值阶段会拉高）
//   - p99_ms         实时 P99 延迟，毫秒
//   - error_rate     错误率 0~1
//   - bids_total     累计出价数
//   - errors_total   累计错误数
//   - elapsed_ms     已用时长
func (d *DummyScenario) Run(ctx context.Context, _ json.RawMessage, p ProgressEmitter) (any, error) {
	steps := len(dummySteps)
	tick := d.totalDuration / time.Duration(steps)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	start := time.Now()
	var bidsTotal, errorsTotal int64

	for i := 1; i <= steps; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(tick):
		}

		// 模拟峰值（步骤 5、6 显著拉高）
		isPeak := i == 5 || i == 6
		baseQPS := 80.0
		if isPeak {
			baseQPS = 1200.0
		}
		qps := baseQPS + rng.Float64()*baseQPS*0.2 // 抖动 ±20%

		// 模拟 P99 延迟（峰值更高）
		p99 := 30.0 + rng.Float64()*40.0
		if isPeak {
			p99 = 180.0 + rng.Float64()*120.0
		}

		// 错误率（在第 6 步轻微抖动一次）
		errRate := 0.0
		if i == 6 {
			errRate = 0.005 + rng.Float64()*0.01
		}

		// 累计指标
		bidsTotal += int64(qps * tick.Seconds())
		errorsTotal += int64(float64(bidsTotal) * errRate * 0.1)

		metrics := map[string]any{
			"step":         i,
			"qps":          math.Round(qps*100) / 100,
			"p99_ms":       math.Round(p99*100) / 100,
			"error_rate":   math.Round(errRate*10000) / 10000,
			"bids_total":   bidsTotal,
			"errors_total": errorsTotal,
			"elapsed_ms":   time.Since(start).Milliseconds(),
		}

		stepLabel := fmt.Sprintf("[%d/%d] %s", i, steps, dummySteps[i-1])
		p.Emit(i*100/steps, stepLabel, metrics)
	}

	return map[string]any{
		"ok":           true,
		"steps":        steps,
		"bids_total":   bidsTotal,
		"errors_total": errorsTotal,
		"duration_ms":  time.Since(start).Milliseconds(),
	}, nil
}
