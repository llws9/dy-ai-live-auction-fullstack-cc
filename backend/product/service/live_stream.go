package service

import (
	"context"
	"errors"
	"product-service/dao"
	"product-service/model"

	"gorm.io/gorm"
)

var ErrLiveStreamBanned = errors.New("live stream is banned")

// LiveStreamService 直播间服务
type LiveStreamService struct {
	liveStreamDAO *dao.LiveStreamDAO
	viewerCounter LiveViewerCounter
}

type AdminLiveStreamRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	CoverImage     string `json:"cover_image"`
	VideoURL       string `json:"video_url"`
	StreamerName   string `json:"streamer_name"`
	StreamerAvatar string `json:"streamer_avatar"`
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
	liveStream, err := s.liveStreamDAO.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if liveStream.Status == model.LiveStreamStatusBanned {
		return nil, ErrLiveStreamBanned
	}
	if liveStream.Status != model.LiveStreamStatusEnded {
		if err := s.liveStreamDAO.UpdateStatus(ctx, id, model.LiveStreamStatusEnded); err != nil {
			return nil, err
		}
	}
	return s.liveStreamDAO.GetByID(ctx, id)
}

func (s *LiveStreamService) StartForCreator(ctx context.Context, creatorID, id int64) (*model.LiveStream, error) {
	liveStream, err := s.liveStreamDAO.GetByIDAndCreatorID(ctx, id, creatorID)
	if err != nil {
		return nil, err
	}
	if liveStream.Status == model.LiveStreamStatusBanned {
		return nil, ErrLiveStreamBanned
	}
	if liveStream.Status != model.LiveStreamStatusLive {
		if err := s.liveStreamDAO.UpdateStatus(ctx, id, model.LiveStreamStatusLive); err != nil {
			return nil, err
		}
	}
	return s.liveStreamDAO.GetByID(ctx, id)
}

func (s *LiveStreamService) EndForCreator(ctx context.Context, creatorID, id int64) (*model.LiveStream, error) {
	liveStream, err := s.liveStreamDAO.GetByIDAndCreatorID(ctx, id, creatorID)
	if err != nil {
		return nil, err
	}
	if liveStream.Status == model.LiveStreamStatusBanned {
		return nil, ErrLiveStreamBanned
	}
	if liveStream.Status != model.LiveStreamStatusEnded {
		if err := s.liveStreamDAO.UpdateStatus(ctx, id, model.LiveStreamStatusEnded); err != nil {
			return nil, err
		}
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

func (s *LiveStreamService) ViewerCountForLiveStream(ctx context.Context, liveStream *model.LiveStream) int64 {
	if liveStream == nil {
		return 0
	}
	count := s.ViewerCount(ctx, liveStream.ID)
	if count > 0 {
		return count
	}
	if liveStream.ViewerCount > 0 {
		return int64(liveStream.ViewerCount)
	}
	return 0
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

func (s *LiveStreamService) ListPublicCandidates(ctx context.Context, page, pageSize int) ([]model.LiveStream, int64, error) {
	offset := (page - 1) * pageSize
	return s.liveStreamDAO.ListPublicCandidates(ctx, offset, pageSize)
}

func (s *LiveStreamService) ListAdminScoped(ctx context.Context, page, pageSize int, statusFilter *int, creatorID *int64) ([]model.LiveStream, int64, error) {
	offset := (page - 1) * pageSize
	return s.liveStreamDAO.ListAdminScoped(ctx, offset, pageSize, statusFilter, creatorID)
}

func (s *LiveStreamService) GetAdminDetail(ctx context.Context, role string, userID, id int64) (*model.LiveStream, error) {
	if role == "merchant" {
		return s.liveStreamDAO.GetByIDAndCreatorID(ctx, id, userID)
	}
	return s.liveStreamDAO.GetByID(ctx, id)
}

func (s *LiveStreamService) CreateForCreator(ctx context.Context, creatorID int64, req AdminLiveStreamRequest) (*model.LiveStream, bool, error) {
	existing, err := s.liveStreamDAO.GetByCreatorID(ctx, creatorID)
	if err == nil {
		return existing, false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}

	liveStream := &model.LiveStream{
		CreatorID:      creatorID,
		Name:           req.Name,
		Description:    req.Description,
		CoverImage:     req.CoverImage,
		VideoURL:       req.VideoURL,
		Status:         model.LiveStreamStatusNotStarted,
		StreamerName:   req.StreamerName,
		StreamerAvatar: req.StreamerAvatar,
	}
	if err := s.liveStreamDAO.Create(ctx, liveStream); err != nil {
		return nil, false, err
	}
	return liveStream, true, nil
}

func (s *LiveStreamService) UpdateForCreator(ctx context.Context, creatorID, id int64, req AdminLiveStreamRequest) (*model.LiveStream, error) {
	liveStream, err := s.liveStreamDAO.GetByIDAndCreatorID(ctx, id, creatorID)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		liveStream.Name = req.Name
	}
	if req.Description != "" {
		liveStream.Description = req.Description
	}
	if req.CoverImage != "" {
		liveStream.CoverImage = req.CoverImage
	}
	if req.VideoURL != "" {
		liveStream.VideoURL = req.VideoURL
	}
	if req.StreamerName != "" {
		liveStream.StreamerName = req.StreamerName
	}
	if req.StreamerAvatar != "" {
		liveStream.StreamerAvatar = req.StreamerAvatar
	}
	if err := s.liveStreamDAO.Update(ctx, liveStream); err != nil {
		return nil, err
	}
	return liveStream, nil
}

// GetDetail 直播间详情 (T012)
func (s *LiveStreamService) GetDetail(ctx context.Context, id int64) (*model.LiveStream, error) {
	return s.liveStreamDAO.GetByID(ctx, id)
}
