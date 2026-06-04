package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/dao"
	"product-service/model"
)

func setupLiveStreamServiceTest(t *testing.T) *LiveStreamService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.LiveStream{}))
	require.NoError(t, db.Exec("DELETE FROM live_streams").Error)
	return NewLiveStreamService(dao.NewLiveStreamDAO(db))
}

func TestLiveStreamServiceListAdminScopedMerchantOnlyOwnStreams(t *testing.T) {
	svc := setupLiveStreamServiceTest(t)
	ctx := context.Background()
	_, err := svc.CreateForCreator(ctx, 1001, AdminLiveStreamRequest{Name: "A"})
	require.NoError(t, err)
	_, err = svc.CreateForCreator(ctx, 1002, AdminLiveStreamRequest{Name: "B"})
	require.NoError(t, err)
	creatorID := int64(1001)

	items, total, err := svc.ListAdminScoped(ctx, 1, 20, nil, &creatorID)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "A", items[0].Name)
}

func TestLiveStreamServiceGetAdminDetailMerchantRejectsOtherOwner(t *testing.T) {
	svc := setupLiveStreamServiceTest(t)
	ctx := context.Background()
	item, err := svc.CreateForCreator(ctx, 1001, AdminLiveStreamRequest{Name: "A"})
	require.NoError(t, err)

	got, err := svc.GetAdminDetail(ctx, "merchant", 1002, item.ID)

	require.Error(t, err)
	require.Nil(t, got)
}
