package dao

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"test-service/model"
)

// newTestDB 启动一个 in-memory SQLite 用于 DAO 单测
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1) // :memory: 每连接独立空间，串行化避免跨连接看不到表
	require.NoError(t, db.AutoMigrate(&model.TestResult{}, &model.TestSeedData{}))
	return db
}

func TestResultDAO_SaveAndGet(t *testing.T) {
	db := newTestDB(t)
	dao := NewResultDAO(db)

	r := &model.TestResult{
		ID:         uuid.NewString(),
		TestType:   model.TypeDummy,
		Status:     model.StatusRunning,
		ConfigJSON: `{"x":1}`,
		CreatedAt:  time.Now(),
	}

	require.NoError(t, dao.Save(context.Background(), r))

	got, err := dao.GetByID(context.Background(), r.ID)
	require.NoError(t, err)
	assert.Equal(t, r.ID, got.ID)
	assert.Equal(t, model.StatusRunning, got.Status)
	assert.Equal(t, `{"x":1}`, got.ConfigJSON)
}

func TestResultDAO_UpdateStatus(t *testing.T) {
	db := newTestDB(t)
	dao := NewResultDAO(db)

	r := &model.TestResult{
		ID:         uuid.NewString(),
		TestType:   model.TypeDummy,
		Status:     model.StatusRunning,
		ConfigJSON: "{}",
		CreatedAt:  time.Now(),
	}
	require.NoError(t, dao.Save(context.Background(), r))

	now := time.Now()
	require.NoError(t, dao.UpdateStatus(context.Background(), r.ID, model.StatusCompleted, `{"ok":true}`, "", &now))

	got, err := dao.GetByID(context.Background(), r.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusCompleted, got.Status)
	assert.Equal(t, `{"ok":true}`, got.ResultJSON)
	require.NotNil(t, got.CompletedAt)
}

func TestResultDAO_GetHistory_Filter(t *testing.T) {
	db := newTestDB(t)
	dao := NewResultDAO(db)

	now := time.Now()
	for i := 0; i < 3; i++ {
		require.NoError(t, dao.Save(context.Background(), &model.TestResult{
			ID:         uuid.NewString(),
			TestType:   model.TypeDummy,
			Status:     model.StatusCompleted,
			ConfigJSON: "{}",
			CreatedAt:  now.Add(-time.Duration(i) * time.Minute),
		}))
	}
	require.NoError(t, dao.Save(context.Background(), &model.TestResult{
		ID:         uuid.NewString(),
		TestType:   model.TypePressure,
		Status:     model.StatusFailed,
		ConfigJSON: "{}",
		CreatedAt:  now,
	}))

	// 只查 dummy
	list, total, err := dao.GetHistory(context.Background(), HistoryFilters{TestType: model.TypeDummy, Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, list, 3)

	// 全部
	_, total, err = dao.GetHistory(context.Background(), HistoryFilters{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
}

func TestSeedDAO_AddListDelete(t *testing.T) {
	db := newTestDB(t)
	dao := NewSeedDAO(db)

	tid := uuid.NewString()
	require.NoError(t, dao.Add(context.Background(), tid, "product", 100))
	require.NoError(t, dao.Add(context.Background(), tid, "auction", 200))

	list, err := dao.ListByTestID(context.Background(), tid)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	require.NoError(t, dao.DeleteByTestID(context.Background(), tid))
	list, err = dao.ListByTestID(context.Background(), tid)
	require.NoError(t, err)
	assert.Len(t, list, 0)
}
