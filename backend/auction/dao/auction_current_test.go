package dao

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/model"
)

// newCurrentTestDB 启动一个 in-memory SQLite 用于 GetCurrentByLiveStreamIDs 单测
func newCurrentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1) // :memory: 每连接独立空间，串行化避免跨连接看不到表
	require.NoError(t, db.AutoMigrate(&model.Auction{}))
	return db
}

func TestGetCurrentByLiveStreamIDs(t *testing.T) {
	db := newCurrentTestDB(t)
	dao := NewAuctionDAO(db)
	ctx := context.Background()

	ls3 := int64(3)
	now := time.Now()
	rows := []model.Auction{
		{ID: 10, ProductID: 100, LiveStreamID: &ls3, Status: model.AuctionStatusOngoing, StartTime: now.Add(-2 * time.Hour)},
		{ID: 11, ProductID: 101, LiveStreamID: &ls3, Status: model.AuctionStatusDelayed, StartTime: now.Add(-1 * time.Hour)},
		{ID: 12, ProductID: 102, LiveStreamID: &ls3, Status: model.AuctionStatusEnded, StartTime: now.Add(-1 * time.Hour)},
	}
	for i := range rows {
		require.NoError(t, db.Create(&rows[i]).Error)
	}

	got, err := dao.GetCurrentByLiveStreamIDs(ctx, []int64{3, 4})
	require.NoError(t, err)

	cur, ok := got[3]
	require.True(t, ok, "结果应包含 live_stream_id=3")
	require.Equal(t, int64(11), cur.ID, "应取 start_time 更新的 delayed 竞拍 id=11")

	_, ok4 := got[4]
	require.False(t, ok4, "live_stream_id=4 无竞拍，不应出现在结果中")

	require.NotEqual(t, int64(12), cur.ID, "ended 的 id=12 不应被选中")
}
