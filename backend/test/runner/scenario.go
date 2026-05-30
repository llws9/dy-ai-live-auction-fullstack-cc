package runner

import (
	"context"
	"encoding/json"
)

// ProgressEmitter 用于场景上报进度
type ProgressEmitter interface {
	// Emit 上报当前进度（0-100）、当前步骤描述、可选指标
	Emit(progress int, step string, metrics map[string]any)
}

// Scenario 测试场景统一接口
type Scenario interface {
	Type() string
	Run(ctx context.Context, cfg json.RawMessage, p ProgressEmitter) (any, error)
}
