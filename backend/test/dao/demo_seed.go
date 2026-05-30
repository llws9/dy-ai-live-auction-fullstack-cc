package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/google/uuid"

	"test-service/model"
)

// SeedDemoHistory 当 test_results 表为空时，预置一批多状态的历史记录用于演示
//
// 用途：让前端 History/Report 页面在没有真实跑过任务时也能看到内容。
// 触发：仅在表中 count==0 时写入；不会重复写入。
// 调用方建议在启动阶段且 ENABLE_DEMO_SEED=true 时执行。
func SeedDemoHistory(ctx context.Context, d *ResultDAO) error {
	// 已有数据则跳过，避免重复污染
	_, total, err := d.GetHistory(ctx, HistoryFilters{Page: 1, PageSize: 1})
	if err != nil {
		return fmt.Errorf("seed demo: count failed: %w", err)
	}
	if total > 0 {
		hlog.Infof("[seed] history not empty (total=%d), skip demo seed", total)
		return nil
	}

	now := time.Now()
	mustJSON := func(v any) string {
		b, _ := json.Marshal(v)
		return string(b)
	}

	// 5 条覆盖：completed × 2、failed、cancelled、running
	completed1 := now.Add(-2 * time.Hour)
	completed2 := now.Add(-30 * time.Minute)
	failedAt := now.Add(-90 * time.Minute)
	cancelledAt := now.Add(-15 * time.Minute)

	demos := []model.TestResult{
		{
			ID:         uuid.NewString(),
			TestType:   model.TypeDummy,
			Status:     model.StatusCompleted,
			ConfigJSON: mustJSON(map[string]any{"duration_sec": 5}),
			ResultJSON: mustJSON(map[string]any{
				"ok": true, "steps": 10, "bids_total": 6800, "errors_total": 12, "duration_ms": 5012,
			}),
			ReplayToken: "demo-replay-001",
			ScriptName:  "dummy.demo",
			CreatedAt:   now.Add(-2*time.Hour - 5*time.Second),
			CompletedAt: &completed1,
		},
		{
			ID:         uuid.NewString(),
			TestType:   model.TypePressure,
			Status:     model.StatusCompleted,
			ConfigJSON: mustJSON(map[string]any{"concurrency": 500, "duration_sec": 60}),
			ResultJSON: mustJSON(map[string]any{
				"qps_peak": 1842, "p99_ms": 287, "error_rate": 0.0021, "total_requests": 110520,
			}),
			ReplayToken: "demo-replay-002",
			ScriptName:  "pressure.normal",
			CreatedAt:   now.Add(-31 * time.Minute),
			CompletedAt: &completed2,
		},
		{
			ID:         uuid.NewString(),
			TestType:   model.TypeAntiSnipe,
			Status:     model.StatusFailed,
			ConfigJSON: mustJSON(map[string]any{"snipe_window_ms": 50}),
			ResultJSON: "{}",
			ErrorMsg:   "依赖 redis 连接失败：dial tcp 127.0.0.1:6379: connect: connection refused",
			ScriptName: "antisnipe.window",
			CreatedAt:  now.Add(-91 * time.Minute),
			CompletedAt: &failedAt,
		},
		{
			ID:         uuid.NewString(),
			TestType:   model.TypeCallback,
			Status:     model.StatusCancelled,
			ConfigJSON: mustJSON(map[string]any{"callback_targets": 3}),
			ErrorMsg:   "cancelled",
			ScriptName: "callback.recovery",
			CreatedAt:  now.Add(-16 * time.Minute),
			CompletedAt: &cancelledAt,
		},
		{
			ID:         uuid.NewString(),
			TestType:   model.TypeChaos,
			Status:     model.StatusRunning,
			ConfigJSON: mustJSON(map[string]any{"toxic": "redis_blackout", "duration_sec": 120}),
			ScriptName: "chaos.redis.blackout",
			CreatedAt:  now.Add(-30 * time.Second),
		},
	}

	for i := range demos {
		if err := d.Save(ctx, &demos[i]); err != nil {
			return fmt.Errorf("seed demo: save[%d] failed: %w", i, err)
		}
	}

	hlog.Infof("[seed] inserted %d demo history records", len(demos))
	return nil
}
