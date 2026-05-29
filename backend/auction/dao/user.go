package dao

import (
	"context"
	"time"

	"gorm.io/gorm"

	"auction-service/model"
)

// UserDAO 用户数据访问层
type UserDAO struct {
	db *gorm.DB
}

// NewUserDAO 创建用户 DAO
func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{db: db}
}

// Exists 检查用户是否存在
func (d *UserDAO) Exists(ctx context.Context, userID int64) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetByID 根据 ID 获取用户
func (d *UserDAO) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	var user model.User
	err := d.db.WithContext(ctx).First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByIDs 批量根据 ID 获取用户，返回 id -> user 映射
// 仅查询 id 和 name 字段，避免无关字段开销（broadcastRanking 等场景使用）
func (d *UserDAO) GetByIDs(ctx context.Context, userIDs []int64) (map[int64]*model.User, error) {
	if len(userIDs) == 0 {
		return map[int64]*model.User{}, nil
	}
	var users []model.User
	err := d.db.WithContext(ctx).
		Select("id", "name").
		Where("id IN ?", userIDs).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]*model.User, len(users))
	for i := range users {
		result[users[i].ID] = &users[i]
	}
	return result, nil
}

// Create 创建用户
func (d *UserDAO) Create(ctx context.Context, user *model.User) error {
	return d.db.WithContext(ctx).Create(user).Error
}

// CreateIfNotExists 如果用户不存在则创建
func (d *UserDAO) CreateIfNotExists(ctx context.Context, user *model.User) error {
	exists, err := d.Exists(ctx, user.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return d.Create(ctx, user)
}

// GetByEmail 根据邮箱获取用户
func (d *UserDAO) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := d.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByPhone 根据手机号获取用户
func (d *UserDAO) GetByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	err := d.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateLastLogin 更新最后登录时间
func (d *UserDAO) UpdateLastLogin(ctx context.Context, userID int64) error {
	return d.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Update("last_login_at", time.Now()).Error
}

// UpdatePassword 更新密码
func (d *UserDAO) UpdatePassword(ctx context.Context, userID int64, hashedPassword string) error {
	return d.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Update("password", hashedPassword).Error
}
