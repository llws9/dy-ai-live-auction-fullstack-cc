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
		{ID: 10, ProductID: 100, LiveStreamID: &ls3, Status: model.AuctionStatusOngoing, StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(time.Hour)},
		{ID: 11, ProductID: 101, LiveStreamID: &ls3, Status: model.AuctionStatusDelayed, StartTime: now.Add(-1 * time.Hour), EndTime: now.Add(time.Hour)},
		{ID: 12, ProductID: 102, LiveStreamID: &ls3, Status: model.AuctionStatusEnded, StartTime: now.Add(-1 * time.Hour), EndTime: now.Add(time.Hour)},
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

func TestGetCurrentByLiveStreamIDsSkipsExpiredActiveAuction(t *testing.T) {
	db := newCurrentTestDB(t)
	dao := NewAuctionDAO(db)
	ctx := context.Background()

	ls3 := int64(3)
	now := time.Now()
	rows := []model.Auction{
		{ID: 20, ProductID: 200, LiveStreamID: &ls3, Status: model.AuctionStatusOngoing, StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-time.Minute)},
		{ID: 21, ProductID: 201, LiveStreamID: &ls3, Status: model.AuctionStatusOngoing, StartTime: now.Add(-3 * time.Hour), EndTime: now.Add(time.Hour)},
	}
	for i := range rows {
		require.NoError(t, db.Create(&rows[i]).Error)
	}

	got, err := dao.GetCurrentByLiveStreamIDs(ctx, []int64{3})
	require.NoError(t, err)

	cur, ok := got[3]
	require.True(t, ok, "应回退到尚未过 end_time 的竞拍")
	require.Equal(t, int64(21), cur.ID)
}

func TestListOrdersByLiveUpcomingEndedPriority(t *testing.T) {
	db := newCurrentTestDB(t)
	dao := NewAuctionDAO(db)
	ctx := context.Background()

	now := time.Now()
	rows := []model.Auction{
		{ID: 100, ProductID: 100, Status: model.AuctionStatusEnded, StartTime: now.Add(-3 * time.Hour), EndTime: now.Add(-2 * time.Hour)},
		{ID: 101, ProductID: 101, Status: model.AuctionStatusPending, StartTime: now.Add(time.Hour), EndTime: now.Add(2 * time.Hour)},
		{ID: 102, ProductID: 102, Status: model.AuctionStatusOngoing, StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)},
		{ID: 103, ProductID: 103, Status: model.AuctionStatusOngoing, StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-time.Minute)},
	}
	for i := range rows {
		require.NoError(t, db.Create(&rows[i]).Error)
	}

	got, total, err := dao.List(ctx, nil, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(4), total)
	require.Len(t, got, 4)

	require.Equal(t, int64(102), got[0].ID, "未过 end_time 的 ongoing 应排在最前")
	require.Equal(t, int64(101), got[1].ID, "pending 应排在直播中之后、已结束之前")
	require.Equal(t, int64(103), got[2].ID, "已过 end_time 的 active 状态应按已结束处理")
	require.Equal(t, int64(100), got[3].ID)
}
