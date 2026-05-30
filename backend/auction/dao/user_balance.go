package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"auction-service/model"
)

// UserBalanceDAO 余额数据访问层（T3.1 / spec A F-A2）。
//
// 仅暴露读接口；写入路径由后续支付/退款链路负责（本期 out of scope）。
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
func (d *UserBalanceDAO) GetByUserID(ctx context.Context, userID int64) (available, frozen float64, currency string, hit bool, err error) {
	var b model.UserBalance
	err = d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&b).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, "", false, nil
		}
		return 0, 0, "", false, err
	}
	return b.AvailableAmount, b.FrozenAmount, b.Currency, true, nil
}
