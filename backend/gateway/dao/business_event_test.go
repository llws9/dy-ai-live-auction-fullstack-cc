package dao

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gateway-service/model"
)

func TestBusinessEventDAOCreatePersistsEvent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:business_event_create?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.BusinessEvent{}))

	dao := NewBusinessEventDAO(db)
	err = dao.Create(context.Background(), &model.BusinessEvent{
		UserID:        42,
		EventType:     "live_room_enter",
		Source:        "live_reminder",
		LiveStreamID:  1001,
		ClientEventID: "evt-create-1",
		Metadata:      `{"source":"test"}`,
	})
	require.NoError(t, err)

	var saved model.BusinessEvent
	require.NoError(t, db.Where("client_event_id = ?", "evt-create-1").First(&saved).Error)
	require.Equal(t, int64(42), saved.UserID)
	require.Equal(t, "live_room_enter", saved.EventType)
}

func TestBusinessEventDAOCreateIsIdempotentByClientEventID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:business_event_idempotent?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.BusinessEvent{}))

	dao := NewBusinessEventDAO(db)
	event := &model.BusinessEvent{
		UserID:        42,
		EventType:     "reminder_click",
		Source:        "notification_center",
		ClientEventID: "evt-dup-1",
		Metadata:      `{}`,
	}

	require.NoError(t, dao.Create(context.Background(), event))
	require.NoError(t, dao.Create(context.Background(), event))

	var count int64
	require.NoError(t, db.Model(&model.BusinessEvent{}).Where("client_event_id = ?", "evt-dup-1").Count(&count).Error)
	require.Equal(t, int64(1), count)
}
