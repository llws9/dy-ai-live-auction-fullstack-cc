package service

import (
	"context"
	"product-service/dao"
	"product-service/model"
)

// LiveStreamService 直播间服务
type LiveStreamService struct {
	liveStreamDAO *dao.LiveStreamDAO
}

// NewLiveStreamService 创建直播间服务
func NewLiveStreamService(liveStreamDAO *dao.LiveStreamDAO) *LiveStreamService {
	return &LiveStreamService{
		liveStreamDAO: liveStreamDAO,
	}
}

// GetOrCreateLiveStream 获取或创建直播间
func (s *LiveStreamService) GetOrCreateLiveStream(ctx context.Context, creatorID int64, creatorName string) (*model.LiveStream, error) {
	return s.liveStreamDAO.GetOrCreateByCreatorID(ctx, creatorID, creatorName)
}

// GetByCreatorID 根据创建者ID获取直播间
func (s *LiveStreamService) GetByCreatorID(ctx context.Context, creatorID int64) (*model.LiveStream, error) {
	return s.liveStreamDAO.GetByCreatorID(ctx, creatorID)
}

// GetByID 根据ID获取直播间
func (s *LiveStreamService) GetByID(ctx context.Context, id int64) (*model.LiveStream, error) {
	return s.liveStreamDAO.GetByID(ctx, id)
}

// UpdateStatus 更新直播间状态
func (s *LiveStreamService) UpdateStatus(ctx context.Context, id int64, status model.LiveStreamStatus) error {
	return s.liveStreamDAO.UpdateStatus(ctx, id, status)
}

// List 获取直播间列表（管理员用）
func (s *LiveStreamService) List(ctx context.Context, page, pageSize int) ([]model.LiveStream, int64, error) {
	offset := (page - 1) * pageSize
	return s.liveStreamDAO.GetAll(ctx, offset, pageSize)
}

// ListAdmin 管理端直播间列表 (T011)
func (s *LiveStreamService) ListAdmin(ctx context.Context, page, pageSize int, statusFilter *int) ([]model.LiveStream, int64, error) {
	offset := (page - 1) * pageSize
	return s.liveStreamDAO.ListAdmin(ctx, offset, pageSize, statusFilter)
}

// GetDetail 直播间详情 (T012)
func (s *LiveStreamService) GetDetail(ctx context.Context, id int64) (*model.LiveStream, error) {
	return s.liveStreamDAO.GetByID(ctx, id)
}
