package dao

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"auction-service/model"
)

// ErrAlreadyBought 表示用户已购买过该一口价商品（命中唯一键 uniq_item_user）。
var ErrAlreadyBought = errors.New("user already bought this fixed price item")

// FixedPricePurchaseDAO 一口价购买记录数据访问层（方案③ purchase 自成闭环）。
type FixedPricePurchaseDAO struct {
	db *gorm.DB
}

func NewFixedPricePurchaseDAO(db *gorm.DB) *FixedPricePurchaseDAO {
	return &FixedPricePurchaseDAO{db: db}
}

// InsertWithTx 在给定事务中写入购买记录；命中唯一键时返回 ErrAlreadyBought。
func (d *FixedPricePurchaseDAO) InsertWithTx(ctx context.Context, tx *gorm.DB, p *model.FixedPricePurchase) error {
	if err := tx.WithContext(ctx).Create(p).Error; err != nil {
		if isDuplicateKey(err) {
			return ErrAlreadyBought
		}
		return err
	}
	return nil
}

// Insert 使用默认连接写入购买记录。
func (d *FixedPricePurchaseDAO) Insert(ctx context.Context, p *model.FixedPricePurchase) error {
	return d.InsertWithTx(ctx, d.db, p)
}

// GetByItemAndUser 按 item_id + user_id 查询购买记录；未命中返回 gorm.ErrRecordNotFound。
func (d *FixedPricePurchaseDAO) GetByItemAndUser(ctx context.Context, itemID, userID int64) (*model.FixedPricePurchase, error) {
	var p model.FixedPricePurchase
	if err := d.db.WithContext(ctx).
		Where("item_id = ? AND user_id = ?", itemID, userID).
		First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// isDuplicateKey 判断错误是否为唯一键冲突。
// 同时兼容 MySQL（"Duplicate entry"）与 sqlite（"UNIQUE constraint failed"）。
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "UNIQUE constraint failed")
}
