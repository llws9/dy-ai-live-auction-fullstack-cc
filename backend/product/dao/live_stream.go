package dao

import (
	"context"
	"product-service/model"

	"gorm.io/gorm"
)

// LiveStreamDAO 直播间数据访问对象
type LiveStreamDAO struct {
	db *gorm.DB
}

// NewLiveStreamDAO 创建直播间DAO
func NewLiveStreamDAO(db *gorm.DB) *LiveStreamDAO {
	return &LiveStreamDAO{db: db}
}

// GetByCreatorID 根据创建者ID获取直播间
func (d *LiveStreamDAO) GetByCreatorID(ctx context.Context, creatorID int64) (*model.LiveStream, error) {
	var liveStream model.LiveStream
	err := d.db.WithContext(ctx).
		Where("creator_id = ?", creatorID).
		First(&liveStream).Error
	if err != nil {
		return nil, err
	}
	return &liveStream, nil
}

// GetByID 根据ID获取直播间
func (d *LiveStreamDAO) GetByID(ctx context.Context, id int64) (*model.LiveStream, error) {
	var liveStream model.LiveStream
	err := d.db.WithContext(ctx).
		First(&liveStream, id).Error
	if err != nil {
		return nil, err
	}
	return &liveStream, nil
}

// GetByIDs 批量根据 ID 获取直播间，返回 id -> *LiveStream 映射
// （T3.3 / spec B §4.1：/internal/live-streams/batch 内部接口）。
func (d *LiveStreamDAO) GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.LiveStream, error) {
	if len(ids) == 0 {
		return map[int64]*model.LiveStream{}, nil
	}
	var items []model.LiveStream
	err := d.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]*model.LiveStream, len(items))
	for i := range items {
		result[items[i].ID] = &items[i]
	}
	return result, nil
}

// Create 创建直播间
func (d *LiveStreamDAO) Create(ctx context.Context, liveStream *model.LiveStream) error {
	return d.db.WithContext(ctx).Create(liveStream).Error
}

// Update 更新直播间
func (d *LiveStreamDAO) Update(ctx context.Context, liveStream *model.LiveStream) error {
	return d.db.WithContext(ctx).Save(liveStream).Error
}

// UpdateStatus 更新直播间状态
func (d *LiveStreamDAO) UpdateStatus(ctx context.Context, id int64, status model.LiveStreamStatus) error {
	return d.db.WithContext(ctx).
		Model(&model.LiveStream{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// GetAll 获取所有直播间（管理员用）
func (d *LiveStreamDAO) GetAll(ctx context.Context, offset, limit int) ([]model.LiveStream, int64, error) {
	var liveStreams []model.LiveStream
	var total int64

	// 获取总数
	if err := d.db.WithContext(ctx).Model(&model.LiveStream{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	err := d.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&liveStreams).Error
	if err != nil {
		return nil, 0, err
	}

	return liveStreams, total, nil
}

// GetOrCreateByCreatorID 获取或创建直播间
func (d *LiveStreamDAO) GetOrCreateByCreatorID(ctx context.Context, creatorID int64, creatorName string) (*model.LiveStream, error) {
	// 尝试获取
	liveStream, err := d.GetByCreatorID(ctx, creatorID)
	if err == nil {
		return liveStream, nil
	}

	// 如果不存在，创建新的
	if err == gorm.ErrRecordNotFound {
		liveStream = &model.LiveStream{
			CreatorID:   creatorID,
			Name:        creatorName + "的直播间",
			Description: creatorName + "的个人直播间",
			Status:      model.LiveStreamStatusActive,
		}
		if err := d.Create(ctx, liveStream); err != nil {
			return nil, err
		}
		return liveStream, nil
	}

	return nil, err
}

// ListAdmin 管理端直播间列表 (T013)
func (d *LiveStreamDAO) ListAdmin(ctx context.Context, offset, limit int, statusFilter *int) ([]model.LiveStream, int64, error) {
	var liveStreams []model.LiveStream
	var total int64

	query := d.db.WithContext(ctx).Model(&model.LiveStream{})

	if statusFilter != nil {
		query = query.Where("status = ?", *statusFilter)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&liveStreams).Error
	if err != nil {
		return nil, 0, err
	}

	return liveStreams, total, nil
}

func (d *LiveStreamDAO) GetAvatarsByIDs(ctx context.Context, ids []int64) (map[int64]string, error) {
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}
	type row struct {
		ID     int64
		Avatar string
	}
	var rows []row
	err := d.db.WithContext(ctx).
		Table("users").
		Select("id, avatar").
		Where("id IN ?", ids).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]string, len(rows))
	for _, r := range rows {
		result[r.ID] = r.Avatar
	}
	return result, nil
}

func (d *LiveStreamDAO) CountActiveByLiveStreamIDs(ctx context.Context, liveStreamIDs []int64) (map[int64]int, error) {
	if len(liveStreamIDs) == 0 {
		return map[int64]int{}, nil
	}
	type row struct {
		LiveStreamID int64
		Cnt          int
	}
	var rows []row
	err := d.db.WithContext(ctx).
		Table("auctions").
		Select("live_stream_id, COUNT(*) AS cnt").
		Where("live_stream_id IN ?", liveStreamIDs).
		Where("status IN ?", []int{1, 2}).
		Group("live_stream_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]int, len(rows))
	for _, r := range rows {
		result[r.LiveStreamID] = r.Cnt
	}
	return result, nil
}
