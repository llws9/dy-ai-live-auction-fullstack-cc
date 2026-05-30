package runner

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/glebarez/sqlite"

	"test-service/dao"
	"test-service/model"
)

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// SQLite :memory: 每 connection 独立内存空间，runner 后台 goroutine 跨连接写入会触发 no-such-table。
	// 限制连接池为 1，强制串行，确保所有读写共享同一内存 DB。
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.TestResult{}, &model.TestSeedData{}))
	return db
}

// recordingEmitter 用于断言 progress 单调递增
type recordingEmitter struct {
	mu       sync.Mutex
	progress []int
}

func (r *recordingEmitter) Emit(p int, _ string, _ map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.progress = append(r.progress, p)
}

// fakeScenario：固定 emit 5 次，每次 +20%
type fakeScenario struct{ called int32 }

func (f *fakeScenario) Type() string { return "fake" }
func (f *fakeScenario) Run(ctx context.Context, _ json.RawMessage, p ProgressEmitter) (any, error) {
	atomic.AddInt32(&f.called, 1)
	for i := 1; i <= 5; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		p.Emit(i*20, "step", nil)
		time.Sleep(2 * time.Millisecond)
	}
	return map[string]any{"ok": true}, nil
}

func TestRunner_SubmitAndComplete(t *testing.T) {
	db := newDB(t)
	r := New(dao.NewResultDAO(db))
	r.Register(&fakeScenario{})

	id, err := r.Submit(context.Background(), "fake", json.RawMessage(`{}`))
	require.NoError(t, err)
	require.NotEmpty(t, id)

	// 等待完成
	require.Eventually(t, func() bool {
		got, err := dao.NewResultDAO(db).GetByID(context.Background(), id)
		if err != nil {
			return false
		}
		return got.Status == model.StatusCompleted
	}, 2*time.Second, 20*time.Millisecond)

	got, err := dao.NewResultDAO(db).GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, model.StatusCompleted, got.Status)
	assert.Contains(t, got.ResultJSON, `"ok":true`)
	require.NotNil(t, got.CompletedAt)
}

func TestRunner_UnknownScenario(t *testing.T) {
	db := newDB(t)
	r := New(dao.NewResultDAO(db))

	_, err := r.Submit(context.Background(), "nope", json.RawMessage(`{}`))
	require.Error(t, err)
}

// 取消正在运行的任务
type slowScenario struct{}

func (slowScenario) Type() string { return "slow" }
func (slowScenario) Run(ctx context.Context, _ json.RawMessage, p ProgressEmitter) (any, error) {
	for i := 0; i < 100; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
			p.Emit(i, "tick", nil)
		}
	}
	return nil, nil
}

func TestRunner_Cancel(t *testing.T) {
	db := newDB(t)
	r := New(dao.NewResultDAO(db))
	r.Register(slowScenario{})

	id, err := r.Submit(context.Background(), "slow", nil)
	require.NoError(t, err)

	// 等开跑
	time.Sleep(20 * time.Millisecond)
	require.NoError(t, r.Cancel(id))

	require.Eventually(t, func() bool {
		got, _ := dao.NewResultDAO(db).GetByID(context.Background(), id)
		return got != nil && (got.Status == model.StatusCancelled || got.Status == model.StatusFailed)
	}, 2*time.Second, 20*time.Millisecond)

	got, err := dao.NewResultDAO(db).GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, model.StatusCancelled, got.Status)
}

func TestDummyScenario_EmitsMonotonicProgress(t *testing.T) {
	s := NewDummyScenario(50 * time.Millisecond) // 总时长缩短便于 CI
	rec := &recordingEmitter{}
	_, err := s.Run(context.Background(), nil, rec)
	require.NoError(t, err)
	require.NotEmpty(t, rec.progress)
	// 单调递增 + 终值 100
	for i := 1; i < len(rec.progress); i++ {
		assert.GreaterOrEqual(t, rec.progress[i], rec.progress[i-1])
	}
	assert.Equal(t, 100, rec.progress[len(rec.progress)-1])
}
