package dao

import (
	"context"
	"errors"
	"time"

	"auction-service/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrAlreadyClaimed indicates the (user, date, tier) treasure claim already exists.
var ErrAlreadyClaimed = errors.New("treasure already claimed")

// TreasureDAO manages watch duration, entertainment coin balance, and treasure claims.
type TreasureDAO struct {
	db *gorm.DB
}

func NewTreasureDAO(db *gorm.DB) *TreasureDAO {
	return &TreasureDAO{db: db}
}

// AddWatchSeconds accumulates watch duration in the (user_id, stat_date) bucket.
func (d *TreasureDAO) AddWatchSeconds(ctx context.Context, userID int64, statDate string, delta int) (int, error) {
	now := time.Now()
	row := model.UserWatchDuration{
		UserID:       userID,
		StatDate:     statDate,
		TotalSeconds: delta,
		UpdatedAt:    now,
	}
	err := d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "stat_date"}},
		DoUpdates: clause.Assignments(map[string]any{
			"total_seconds": gorm.Expr("total_seconds + ?", delta),
			"updated_at":    now,
		}),
	}).Create(&row).Error
	if err != nil {
		return 0, err
	}
	return d.GetWatchSeconds(ctx, userID, statDate)
}

// GetWatchSeconds returns the accumulated watch seconds, or 0 when no row exists.
func (d *TreasureDAO) GetWatchSeconds(ctx context.Context, userID int64, statDate string) (int, error) {
	var row model.UserWatchDuration
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND stat_date = ?", userID, statDate).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return row.TotalSeconds, nil
}

// GetCoinBalance returns the user's coin balance, or 0 when no row exists.
func (d *TreasureDAO) GetCoinBalance(ctx context.Context, userID int64) (int64, error) {
	var row model.UserCoin
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return row.Balance, nil
}

// ListClaimedTiers returns the claimed tier set for a user on a business date.
func (d *TreasureDAO) ListClaimedTiers(ctx context.Context, userID int64, statDate string) (map[int8]bool, error) {
	var rows []model.TreasureClaim
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND stat_date = ?", userID, statDate).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	claimed := make(map[int8]bool, len(rows))
	for _, row := range rows {
		claimed[row.Tier] = true
	}
	return claimed, nil
}

// ClaimTx inserts the claim and increments coin balance in a single transaction.
func (d *TreasureDAO) ClaimTx(ctx context.Context, userID int64, statDate string, tier int8, coins int64) (int64, error) {
	var newBalance int64
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		claim := model.TreasureClaim{
			UserID:    userID,
			StatDate:  statDate,
			Tier:      tier,
			Coins:     coins,
			ClaimedAt: now,
		}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&claim)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrAlreadyClaimed
		}

		coin := model.UserCoin{UserID: userID, Balance: coins, UpdatedAt: now}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"balance":    gorm.Expr("balance + ?", coins),
				"updated_at": now,
			}),
		}).Create(&coin).Error; err != nil {
			return err
		}

		var updated model.UserCoin
		if err := tx.Where("user_id = ?", userID).First(&updated).Error; err != nil {
			return err
		}
		newBalance = updated.Balance
		return nil
	})
	if err != nil {
		return 0, err
	}
	return newBalance, nil
}
