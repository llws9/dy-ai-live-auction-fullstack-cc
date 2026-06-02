package dao

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"auction-service/model"
)

// UserBalanceDAO 余额数据访问层（T3.1 / spec A F-A2）。
//
// 读接口 GetByUserID 供 handler 查询；写接口 DeductWithTx 供一口价抢购
// 在同库单事务内扣减余额（方案③ purchase 自成闭环，不跨服务建单）。
type UserBalanceDAO struct {
	db *gorm.DB
}

func NewUserBalanceDAO(db *gorm.DB) *UserBalanceDAO {
	return &UserBalanceDAO{db: db}
}

// GetByUserID 实现 handler.BalanceProvider：
//   - 命中：返回字段 + hit=true
//   - 未命中：返回 hit=false（不返回 ErrRecordNotFound，避免 handler 把"无记录"误判为故障）
//   - DB 故障：返回 err
func (d *UserBalanceDAO) GetByUserID(ctx context.Context, userID int64) (available, frozen decimal.Decimal, currency string, hit bool, err error) {
	var b model.UserBalance
	err = d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&b).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return decimal.Zero, decimal.Zero, "", false, nil
		}
		return decimal.Zero, decimal.Zero, "", false, err
	}
	return b.AvailableAmount, b.FrozenAmount, b.Currency, true, nil
}

// DeductWithTx 在传入事务内条件扣减用户可用余额。
//
// 防超扣依赖 SQL 条件更新：仅当 available_amount >= amount 时才扣减，
// 通过 RowsAffected 判定结果，避免"读-判断-写"竞态。
//   - affected == 1：扣减成功
//   - affected == 0：余额不足或无记录（由调用方映射为余额不足错误）
//
// 调用方必须在已开启的事务 tx 中调用，与扣库存/写购买记录保持原子。
func (d *UserBalanceDAO) DeductWithTx(ctx context.Context, tx *gorm.DB, userID int64, amount decimal.Decimal) (affected int64, err error) {
	res := tx.WithContext(ctx).
		Model(&model.UserBalance{}).
		Where("user_id = ? AND available_amount >= ?", userID, amount).
		Update("available_amount", gorm.Expr("available_amount - ?", amount))
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
