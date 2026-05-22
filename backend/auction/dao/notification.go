package dao

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"auction-service/model"
)

// NotificationDAO 通知数据访问对象
type NotificationDAO struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewNotificationDAO 创建NotificationDAO
func NewNotificationDAO(db *gorm.DB, redis *redis.Client) *NotificationDAO {
	return &NotificationDAO{db: db, redis: redis}
}

// Create 创建通知
func (d *NotificationDAO) Create(ctx context.Context, notification *model.Notification) error {
	return d.db.WithContext(ctx).Create(notification).Error
}

// CreateBatch 批量创建通知
func (d *NotificationDAO) CreateBatch(ctx context.Context, notifications []*model.Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).CreateInBatches(notifications, 100).Error
}

// GetByID 根据ID获取通知
func (d *NotificationDAO) GetByID(ctx context.Context, id int64) (*model.Notification, error) {
	var notification model.Notification
	err := d.db.WithContext(ctx).First(&notification, id).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// GetByUserID 获取用户通知列表
func (d *NotificationDAO) GetByUserID(ctx context.Context, userID int64, page, pageSize int, unreadOnly bool) (*model.NotificationListResponse, error) {
	var notifications []model.Notification
	var total int64

	query := d.db.WithContext(ctx).Model(&model.Notification{}).Where("user_id = ?", userID)
	if unreadOnly {
		query = query.Where("read_at IS NULL")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&notifications).Error; err != nil {
		return nil, err
	}

	return &model.NotificationListResponse{
		Items:    notifications,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetUnreadCount 获取未读通知数量
func (d *NotificationDAO) GetUnreadCount(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

// MarkAsRead 标记为已读
func (d *NotificationDAO) MarkAsRead(ctx context.Context, id int64, userID int64) error {
	now := time.Now()
	return d.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("read_at", now).Error
}

// MarkAllAsRead 标记所有为已读
func (d *NotificationDAO) MarkAllAsRead(ctx context.Context, userID int64) error {
	now := time.Now()
	return d.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now).Error
}

// GetUnreadByUserID 获取用户未读通知列表（用于WebSocket推送）
func (d *NotificationDAO) GetUnreadByUserID(ctx context.Context, userID int64, limit int) ([]model.Notification, error) {
	var notifications []model.Notification
	err := d.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications).Error
	return notifications, err
}
