package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"auction-service/model"
)

// UserAddressDAO 收货地址数据访问层（T3.2 / spec A F-A3）。
//
// 实现 handler.AddressStore：所有写操作以 (id, user_id) 复合定位，越权访问表现为 hit=false。
// SetDefault 内部以事务保证同 user_id 的 is_default 互斥。
type UserAddressDAO struct {
	db *gorm.DB
}

func NewUserAddressDAO(db *gorm.DB) *UserAddressDAO {
	return &UserAddressDAO{db: db}
}

func toView(a *model.UserAddress) model.AddressView {
	return model.AddressView{
		ID:            a.ID,
		RecipientName: a.RecipientName,
		Phone:         a.Phone,
		Province:      a.Province,
		City:          a.City,
		District:      a.District,
		Detail:        a.Detail,
		IsDefault:     a.IsDefault,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

func (d *UserAddressDAO) List(ctx context.Context, userID int64) ([]model.AddressView, error) {
	var rows []model.UserAddress
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_default DESC, updated_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]model.AddressView, 0, len(rows))
	for i := range rows {
		out = append(out, toView(&rows[i]))
	}
	return out, nil
}

func (d *UserAddressDAO) Count(ctx context.Context, userID int64) (int64, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&model.UserAddress{}).
		Where("user_id = ?", userID).
		Count(&n).Error
	return n, err
}

func (d *UserAddressDAO) Get(ctx context.Context, id, userID int64) (*model.AddressView, bool, error) {
	var a model.UserAddress
	err := d.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	v := toView(&a)
	return &v, true, nil
}

// Create 在事务中处理首次/显式 is_default=true 的互斥语义。
// 编排层已保证 count<20 与首条强默认。
func (d *UserAddressDAO) Create(ctx context.Context, m model.AddressMutation) (*model.AddressView, error) {
	row := model.UserAddress{
		UserID:        m.UserID,
		RecipientName: m.RecipientName,
		Phone:         m.Phone,
		Province:      m.Province,
		City:          m.City,
		District:      m.District,
		Detail:        m.Detail,
		IsDefault:     m.IsDefault,
	}
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if m.IsDefault {
			if err := tx.Model(&model.UserAddress{}).
				Where("user_id = ? AND is_default = ?", m.UserID, true).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		return nil, err
	}
	v := toView(&row)
	return &v, nil
}

func (d *UserAddressDAO) Update(ctx context.Context, id, userID int64, m model.AddressMutation) (bool, error) {
	var hit bool
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.UserAddress
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		hit = true
		if m.IsDefault && !existing.IsDefault {
			if err := tx.Model(&model.UserAddress{}).
				Where("user_id = ? AND is_default = ? AND id <> ?", userID, true, id).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		updates := map[string]interface{}{
			"recipient_name": m.RecipientName,
			"phone":          m.Phone,
			"province":       m.Province,
			"city":           m.City,
			"district":       m.District,
			"detail":         m.Detail,
			"is_default":     m.IsDefault,
		}
		return tx.Model(&model.UserAddress{}).
			Where("id = ? AND user_id = ?", id, userID).
			Updates(updates).Error
	})
	return hit, err
}

func (d *UserAddressDAO) Delete(ctx context.Context, id, userID int64) (bool, error) {
	res := d.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.UserAddress{})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// SetDefault 事务：定位 (id, user_id) → 清零同用户其它 is_default → 置位本行。
func (d *UserAddressDAO) SetDefault(ctx context.Context, id, userID int64) (bool, error) {
	var hit bool
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.UserAddress
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		hit = true
		if err := tx.Model(&model.UserAddress{}).
			Where("user_id = ? AND id <> ?", userID, id).
			Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&existing).Update("is_default", true).Error
	})
	return hit, err
}
