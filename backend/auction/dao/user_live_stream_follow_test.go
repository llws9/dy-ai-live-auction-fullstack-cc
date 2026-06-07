package dao

import (
	"context"
	"testing"

	"auction-service/model"

	"github.com/stretchr/testify/require"
)

func TestUserLiveStreamFollowDAOGetFollowStatsReturnsFrontendCountAliases(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.UserLiveStreamFollow{}))

	dao := NewUserLiveStreamFollowDAO(db)
	ctx := context.Background()
	require.NoError(t, dao.Create(ctx, &model.UserLiveStreamFollow{
		UserID:              1,
		LiveStreamID:        10,
		NotificationEnabled: true,
	}))

	stats, err := dao.GetFollowStats(ctx, 10)

	require.NoError(t, err)
	require.Equal(t, int64(1), stats["total_count"])
	require.Equal(t, int64(1), stats["followers_count"])
	require.Equal(t, int64(1), stats["count"])
}

func TestUserLiveStreamFollowDAOCountFollowersByLiveStreamIDs(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.UserLiveStreamFollow{}))

	dao := NewUserLiveStreamFollowDAO(db)
	ctx := context.Background()
	require.NoError(t, dao.Create(ctx, &model.UserLiveStreamFollow{UserID: 1, LiveStreamID: 10}))
	require.NoError(t, dao.Create(ctx, &model.UserLiveStreamFollow{UserID: 2, LiveStreamID: 10}))
	require.NoError(t, dao.Create(ctx, &model.UserLiveStreamFollow{UserID: 3, LiveStreamID: 11}))

	counts, err := dao.CountFollowersByLiveStreamIDs(ctx, []int64{10, 11, 12})

	require.NoError(t, err)
	require.Equal(t, int64(2), counts[10])
	require.Equal(t, int64(1), counts[11])
	require.Equal(t, int64(0), counts[12])
}
