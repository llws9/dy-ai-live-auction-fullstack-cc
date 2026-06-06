package dao

import (
	"context"
	"errors"
	"time"

	"auction-service/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuctionSettlementTaskDAO struct {
	db *gorm.DB
}

func NewAuctionSettlementTaskDAO(db *gorm.DB) *AuctionSettlementTaskDAO {
	return &AuctionSettlementTaskDAO{db: db}
}

func (d *AuctionSettlementTaskDAO) WithTx(tx *gorm.DB) *AuctionSettlementTaskDAO {
	return &AuctionSettlementTaskDAO{db: tx}
}

func (d *AuctionSettlementTaskDAO) CreatePendingIfNotExists(ctx context.Context, auctionID int64) error {
	task := &model.AuctionSettlementTask{
		AuctionID: auctionID,
		Status:    model.AuctionSettlementTaskStatusPending,
	}
	return d.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(task).Error
}

func (d *AuctionSettlementTaskDAO) GetByAuctionID(ctx context.Context, auctionID int64) (*model.AuctionSettlementTask, error) {
	var task model.AuctionSettlementTask
	err := d.db.WithContext(ctx).First(&task, "auction_id = ?", auctionID).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (d *AuctionSettlementTaskDAO) EnsurePending(ctx context.Context, auctionID int64) (*model.AuctionSettlementTask, error) {
	task, err := d.GetByAuctionID(ctx, auctionID)
	if err == nil {
		return task, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err := d.CreatePendingIfNotExists(ctx, auctionID); err != nil {
		return nil, err
	}
	return d.GetByAuctionID(ctx, auctionID)
}

func (d *AuctionSettlementTaskDAO) UpdateStatus(ctx context.Context, auctionID int64, status model.AuctionSettlementTaskStatus) error {
	return d.db.WithContext(ctx).
		Model(&model.AuctionSettlementTask{}).
		Where("auction_id = ?", auctionID).
		Updates(map[string]interface{}{
			"status":     status,
			"last_error": "",
		}).Error
}

func (d *AuctionSettlementTaskDAO) AdvanceStatus(ctx context.Context, auctionID int64, from, to model.AuctionSettlementTaskStatus) (bool, error) {
	result := d.db.WithContext(ctx).
		Model(&model.AuctionSettlementTask{}).
		Where("auction_id = ? AND status = ?", auctionID, from).
		Updates(map[string]interface{}{
			"status":     to,
			"last_error": "",
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (d *AuctionSettlementTaskDAO) RecordFailureIfStatus(ctx context.Context, auctionID int64, expectedStatus, rollbackStatus model.AuctionSettlementTaskStatus, err error) (bool, error) {
	lastError := ""
	if err != nil {
		lastError = err.Error()
	}
	query := d.db.WithContext(ctx).
		Model(&model.AuctionSettlementTask{}).
		Where("auction_id = ?", auctionID)
	if expectedStatus != "" {
		query = query.Where("status = ?", expectedStatus)
	}
	result := query.Updates(map[string]interface{}{
		"status":     rollbackStatus,
		"last_error": lastError,
		"updated_at": time.Now(),
	})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (d *AuctionSettlementTaskDAO) ListUnfinished(ctx context.Context, limit int) ([]model.AuctionSettlementTask, error) {
	if limit <= 0 {
		limit = 100
	}
	var tasks []model.AuctionSettlementTask
	err := d.db.WithContext(ctx).
		Where("status IN ?", []model.AuctionSettlementTaskStatus{
			model.AuctionSettlementTaskStatusPending,
			model.AuctionSettlementTaskStatusOrderDone,
			model.AuctionSettlementTaskStatusNotifying,
		}).
		Order("updated_at ASC, auction_id ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}
