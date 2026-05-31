package dao

import (
	"context"
	"time"

	"auction-service/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LiveStreamReminderReceiptDAO struct {
	db *gorm.DB
}

func NewLiveStreamReminderReceiptDAO(db *gorm.DB) *LiveStreamReminderReceiptDAO {
	return &LiveStreamReminderReceiptDAO{db: db}
}

func (d *LiveStreamReminderReceiptDAO) Claim(ctx context.Context, userID, liveStreamID, startedAt int64) (bool, error) {
	receipt := &model.LiveStreamReminderReceipt{
		UserID:        userID,
		LiveStreamID:  liveStreamID,
		LiveStartedAt: startedAt,
		RemindedAt:    time.Now(),
	}
	result := d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(receipt)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}
