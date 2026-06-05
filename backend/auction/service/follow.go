package service

import (
	"context"
	"errors"
	"fmt"

	"auction-service/dao"
	"auction-service/model"
)

type FollowDAO interface {
	Create(ctx context.Context, follow *model.UserLiveStreamFollow) error
	Delete(ctx context.Context, userID, liveStreamID int64) error
	GetByUserAndLiveStream(ctx context.Context, userID, liveStreamID int64) (*model.UserLiveStreamFollow, error)
	GetUserFollows(ctx context.Context, userID int64, offset, limit int) ([]model.UserLiveStreamFollow, error)
	CountUserFollows(ctx context.Context, userID int64) (int64, error)
	UpdateNotificationEnabled(ctx context.Context, userID, liveStreamID int64, enabled bool) error
	GetFollowStats(ctx context.Context, liveStreamID int64) (map[string]int64, error)
}

// FollowService 关注服务
type FollowService struct {
	followDAO FollowDAO
}

// NewFollowService 创建关注服务
func NewFollowService(followDAO FollowDAO) *FollowService {
	return &FollowService{
		followDAO: followDAO,
	}
}

// Follow 关注直播间
func (s *FollowService) Follow(ctx context.Context, userID, liveStreamID int64) (*model.UserLiveStreamFollow, error) {
	// 检查是否已关注
	existing, _ := s.followDAO.GetByUserAndLiveStream(ctx, userID, liveStreamID)
	if existing != nil {
		return existing, nil
	}

	// 创建关注记录
	follow := &model.UserLiveStreamFollow{
		UserID:              userID,
		LiveStreamID:        liveStreamID,
		NotificationEnabled: true, // 默认开启通知
	}

	if err := s.followDAO.Create(ctx, follow); err != nil {
		return nil, fmt.Errorf("关注失败: %w", err)
	}

	// 同步到Redis（用于热拉通知过滤）
	if err := dao.AddUserFollowedLiveStream(ctx, userID, liveStreamID); err != nil {
		// Redis同步失败不影响主流程，仅记录日志
		fmt.Printf("Warning: failed to sync follow to Redis: %v\n", err)
	}

	return follow, nil
}

// Unfollow 取消关注直播间
func (s *FollowService) Unfollow(ctx context.Context, userID, liveStreamID int64) error {
	// 删除关注记录
	if err := s.followDAO.Delete(ctx, userID, liveStreamID); err != nil {
		return fmt.Errorf("取消关注失败: %w", err)
	}

	// 同步到Redis（用于热拉通知过滤）
	if err := dao.RemoveUserFollowedLiveStream(ctx, userID, liveStreamID); err != nil {
		// Redis同步失败不影响主流程，仅记录日志
		fmt.Printf("Warning: failed to sync unfollow to Redis: %v\n", err)
	}

	return nil
}

// ToggleNotification 切换通知状态
func (s *FollowService) ToggleNotification(ctx context.Context, userID, liveStreamID int64, enabled bool) error {
	// 检查是否已关注
	_, err := s.followDAO.GetByUserAndLiveStream(ctx, userID, liveStreamID)
	if err != nil {
		return errors.New("未关注该直播间")
	}

	// 更新通知状态
	if err := s.followDAO.UpdateNotificationEnabled(ctx, userID, liveStreamID, enabled); err != nil {
		return fmt.Errorf("更新通知状态失败: %w", err)
	}

	return nil
}

// GetUserFollows 获取用户关注的直播间列表
func (s *FollowService) GetUserFollows(ctx context.Context, userID int64, page, pageSize int) ([]model.UserLiveStreamFollow, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	follows, err := s.followDAO.GetUserFollows(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.followDAO.CountUserFollows(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return follows, total, nil
}

// GetFollowStats 获取直播间关注统计
func (s *FollowService) GetFollowStats(ctx context.Context, liveStreamID int64) (map[string]int64, error) {
	return s.followDAO.GetFollowStats(ctx, liveStreamID)
}

// IsFollowing 检查用户是否关注了直播间
func (s *FollowService) IsFollowing(ctx context.Context, userID, liveStreamID int64) (bool, error) {
	follow, err := s.followDAO.GetByUserAndLiveStream(ctx, userID, liveStreamID)
	if err != nil {
		// 如果记录不存在，返回false
		return false, nil
	}
	return follow != nil, nil
}
