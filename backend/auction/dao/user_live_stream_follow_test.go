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
