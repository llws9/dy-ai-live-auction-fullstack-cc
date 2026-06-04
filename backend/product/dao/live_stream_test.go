package dao

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/model"
)

func setupLiveStreamDAOTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.LiveStream{}))
	require.NoError(t, db.Exec("DELETE FROM live_streams").Error)
	return db
}

func TestLiveStreamDAOListAdminScopedMerchantOnlyOwnStreams(t *testing.T) {
	db := setupLiveStreamDAOTestDB(t)
	dao := NewLiveStreamDAO(db)
	ctx := context.Background()
	require.NoError(t, dao.Create(ctx, &model.LiveStream{CreatorID: 1001, Name: "A", Status: model.LiveStreamStatusLive}))
	require.NoError(t, dao.Create(ctx, &model.LiveStream{CreatorID: 1002, Name: "B", Status: model.LiveStreamStatusLive}))

	items, total, err := dao.ListAdminScoped(ctx, 0, 20, nil, ptrInt64(1001))

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "A", items[0].Name)
}

func TestLiveStreamDAOGetByIDAndCreatorIDRejectsOtherOwner(t *testing.T) {
	db := setupLiveStreamDAOTestDB(t)
	dao := NewLiveStreamDAO(db)
	ctx := context.Background()
	item := &model.LiveStream{CreatorID: 1001, Name: "A", Status: model.LiveStreamStatusLive}
	require.NoError(t, dao.Create(ctx, item))

	got, err := dao.GetByIDAndCreatorID(ctx, item.ID, 1002)

	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.Nil(t, got)
}

func ptrInt64(v int64) *int64 {
	return &v
}
