package dao

import (
	"context"

	"auction-service/model"
	"gorm.io/gorm"
)

// UserLiveStreamFollowDAO 用户关注直播间DAO
type UserLiveStreamFollowDAO struct {
	db *gorm.DB
}

// NewUserLiveStreamFollowDAO 创建用户关注直播间DAO
func NewUserLiveStreamFollowDAO(db *gorm.DB) *UserLiveStreamFollowDAO {
	return &UserLiveStreamFollowDAO{db: db}
}

// Create 创建关注记录
func (d *UserLiveStreamFollowDAO) Create(ctx context.Context, follow *model.UserLiveStreamFollow) error {
	return d.db.WithContext(ctx).Create(follow).Error
}

// Delete 取消关注
func (d *UserLiveStreamFollowDAO) Delete(ctx context.Context, userID, liveStreamID int64) error {
	return d.db.WithContext(ctx).
		Where("user_id = ? AND live_stream_id = ?", userID, liveStreamID).
		Delete(&model.UserLiveStreamFollow{}).Error
}

// GetFollowers 获取直播间的关注用户列表（分页）
func (d *UserLiveStreamFollowDAO) GetFollowers(ctx context.Context, liveStreamID int64, offset, limit int) ([]model.UserLiveStreamFollow, error) {
	var follows []model.UserLiveStreamFollow
	err := d.db.WithContext(ctx).
		Where("live_stream_id = ? AND notification_enabled = ?", liveStreamID, true).
		Offset(offset).
		Limit(limit).
		Order("created_at ASC").
		Find(&follows).Error
	return follows, err
}

// CountByLiveStream 统计直播间关注人数（仅统计开启通知的）
func (d *UserLiveStreamFollowDAO) CountByLiveStream(ctx context.Context, liveStreamID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.UserLiveStreamFollow{}).
		Where("live_stream_id = ? AND notification_enabled = ?", liveStreamID, true).
		Count(&count).Error
	return count, err
}

// GetByUserAndLiveStream 获取用户的关注记录
func (d *UserLiveStreamFollowDAO) GetByUserAndLiveStream(ctx context.Context, userID, liveStreamID int64) (*model.UserLiveStreamFollow, error) {
	var follow model.UserLiveStreamFollow
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND live_stream_id = ?", userID, liveStreamID).
		First(&follow).Error
	if err != nil {
		return nil, err
	}
	return &follow, nil
}

// GetUserFollows 获取用户关注的所有直播间
func (d *UserLiveStreamFollowDAO) GetUserFollows(ctx context.Context, userID int64, offset, limit int) ([]model.UserLiveStreamFollow, error) {
	var follows []model.UserLiveStreamFollow
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&follows).Error
	return follows, err
}

// CountUserFollows 统计用户关注的直播间数量
func (d *UserLiveStreamFollowDAO) CountUserFollows(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.UserLiveStreamFollow{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// UpdateNotificationEnabled 更新通知状态
func (d *UserLiveStreamFollowDAO) UpdateNotificationEnabled(ctx context.Context, userID, liveStreamID int64, enabled bool) error {
	return d.db.WithContext(ctx).
		Model(&model.UserLiveStreamFollow{}).
		Where("user_id = ? AND live_stream_id = ?", userID, liveStreamID).
		Update("notification_enabled", enabled).Error
}

// GetFollowStats 获取直播间关注统计
func (d *UserLiveStreamFollowDAO) GetFollowStats(ctx context.Context, liveStreamID int64) (map[string]int64, error) {
	stats := make(map[string]int64)

	// 总关注数
	var totalCount int64
	if err := d.db.WithContext(ctx).
		Model(&model.UserLiveStreamFollow{}).
		Where("live_stream_id = ?", liveStreamID).
		Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats["total_count"] = totalCount

	// 开启通知的关注数
	var notificationCount int64
	if err := d.db.WithContext(ctx).
		Model(&model.UserLiveStreamFollow{}).
		Where("live_stream_id = ? AND notification_enabled = ?", liveStreamID, true).
		Count(&notificationCount).Error; err != nil {
		return nil, err
	}
	stats["notification_enabled_count"] = notificationCount

	return stats, nil
}
