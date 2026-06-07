package dao

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/model"
)

func TestListPublicCandidatesExcludesEndedAndBanned(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.LiveStream{}))
	db.Exec("DELETE FROM live_streams")
	require.NoError(t, db.Create(&model.LiveStream{ID: 1, CreatorID: 1, Name: "未开播", Status: model.LiveStreamStatusNotStarted}).Error)
	require.NoError(t, db.Create(&model.LiveStream{ID: 2, CreatorID: 2, Name: "直播中", Status: model.LiveStreamStatusLive}).Error)
	require.NoError(t, db.Create(&model.LiveStream{ID: 3, CreatorID: 3, Name: "已结束", Status: model.LiveStreamStatusEnded}).Error)
	require.NoError(t, db.Create(&model.LiveStream{ID: 4, CreatorID: 4, Name: "封禁", Status: model.LiveStreamStatusBanned}).Error)

	d := NewLiveStreamDAO(db)
	rows, total, err := d.ListPublicCandidates(context.Background(), 0, 20)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	ids := map[int64]bool{}
	for _, r := range rows {
		ids[r.ID] = true
	}
	require.True(t, ids[1])
	require.True(t, ids[2])
	require.False(t, ids[3])
	require.False(t, ids[4])
}
