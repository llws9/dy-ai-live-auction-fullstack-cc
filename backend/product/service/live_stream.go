package service

import (
	"context"
	"product-service/dao"
	"product-service/model"
)

// LiveStreamService 直播间服务
type LiveStreamService struct {
	liveStreamDAO *dao.LiveStreamDAO
	viewerCounter LiveViewerCounter
}

// NewLiveStreamService 创建直播间服务
func NewLiveStreamService(liveStreamDAO *dao.LiveStreamDAO) *LiveStreamService {
	return NewLiveStreamServiceWithMetrics(liveStreamDAO, ZeroLiveViewerCounter{})
}

// LiveViewerCounter abstracts realtime viewer counts, backed by Redis in production.
type LiveViewerCounter interface {
	Count(ctx context.Context, liveStreamID int64) (int64, error)
}

// ZeroLiveViewerCounter is the safe default when realtime metrics are unavailable.
type ZeroLiveViewerCounter struct{}

func (ZeroLiveViewerCounter) Count(context.Context, int64) (int64, error) {
	return 0, nil
}

// StaticLiveViewerCounter is used by tests to model Redis live:viewer:{id} values.
type StaticLiveViewerCounter map[int64]int64

func (c StaticLiveViewerCounter) Count(_ context.Context, liveStreamID int64) (int64, error) {
	return c[liveStreamID], nil
}

func NewLiveStreamServiceWithMetrics(liveStreamDAO *dao.LiveStreamDAO, viewerCounter LiveViewerCounter) *LiveStreamService {
	if viewerCounter == nil {
		viewerCounter = ZeroLiveViewerCounter{}
	}
	return &LiveStreamService{
		liveStreamDAO: liveStreamDAO,
		viewerCounter: viewerCounter,
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

func (s *LiveStreamService) End(ctx context.Context, id int64) (*model.LiveStream, error) {
	if err := s.liveStreamDAO.UpdateStatus(ctx, id, model.LiveStreamStatusEnded); err != nil {
		return nil, err
	}
	return s.liveStreamDAO.GetByID(ctx, id)
}

func (s *LiveStreamService) Ban(ctx context.Context, id int64, reason string) (*model.LiveStream, error) {
	if err := s.liveStreamDAO.Ban(ctx, id, reason); err != nil {
		return nil, err
	}
	return s.liveStreamDAO.GetByID(ctx, id)
}

func (s *LiveStreamService) ViewerCount(ctx context.Context, id int64) int64 {
	count, err := s.viewerCounter.Count(ctx, id)
	if err != nil || count < 0 {
		return 0
	}
	return count
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
