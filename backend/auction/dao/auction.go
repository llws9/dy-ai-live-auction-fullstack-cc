package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"auction-service/model"
	"github.com/shopspring/decimal"
)

// ErrVersionConflict 乐观锁版本冲突
var ErrVersionConflict = errors.New("竞拍版本不匹配，可能已被其他请求更新")

// AuctionDAO 竞拍数据访问层
type AuctionDAO struct {
	db *gorm.DB
}

// NewAuctionDAO 创建竞拍 DAO
func NewAuctionDAO(db *gorm.DB) *AuctionDAO {
	return &AuctionDAO{db: db}
}

// DB 获取底层数据库连接
func (d *AuctionDAO) DB() *gorm.DB {
	return d.db
}

// WithTx 使用事务创建新的 DAO 实例
func (d *AuctionDAO) WithTx(tx *gorm.DB) *AuctionDAO {
	return &AuctionDAO{db: tx}
}

// Create 创建竞拍
func (d *AuctionDAO) Create(ctx context.Context, auction *model.Auction) error {
	return d.db.WithContext(ctx).Create(auction).Error
}

// GetByID 根据 ID 获取竞拍
func (d *AuctionDAO) GetByID(ctx context.Context, id int64) (*model.Auction, error) {
	var auction model.Auction
	err := d.db.WithContext(ctx).First(&auction, id).Error
	if err != nil {
		return nil, err
	}
	return &auction, nil
}

// GetByProductID 根据商品 ID 获取竞拍
func (d *AuctionDAO) GetByProductID(ctx context.Context, productID int64) (*model.Auction, error) {
	var auction model.Auction
	err := d.db.WithContext(ctx).Where("product_id = ?", productID).First(&auction).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &auction, nil
}

// Update 更新竞拍
func (d *AuctionDAO) Update(ctx context.Context, auction *model.Auction) error {
	return d.db.WithContext(ctx).Save(auction).Error
}

// UpdateStatus 更新竞拍状态
func (d *AuctionDAO) UpdateStatus(ctx context.Context, id int64, status model.AuctionStatus) error {
	return d.db.WithContext(ctx).
		Model(&model.Auction{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdatePrice 更新当前价格和中标者（使用乐观锁）
func (d *AuctionDAO) UpdatePrice(ctx context.Context, id int64, price decimal.Decimal, winnerID int64, expectedVersion int) error {
	result := d.db.WithContext(ctx).
		Model(&model.Auction{}).
		Where("id = ? AND version = ?", id, expectedVersion).
		Updates(map[string]interface{}{
			"current_price": price,
			"winner_id":     winnerID,
			"version":       gorm.Expr("version + 1"),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrVersionConflict
	}

	return nil
}

// ExtendEndTime 延长结束时间
func (d *AuctionDAO) ExtendEndTime(ctx context.Context, id int64, additionalSeconds int) error {
	return d.db.WithContext(ctx).
		Model(&model.Auction{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"end_time":   gorm.Expr("DATE_ADD(end_time, INTERVAL ? SECOND)", additionalSeconds),
			"delay_used": gorm.Expr("delay_used + ?", additionalSeconds),
		}).Error
}

// ListByStatus 根据状态获取竞拍列表
func (d *AuctionDAO) ListByStatus(ctx context.Context, status model.AuctionStatus) ([]model.Auction, error) {
	var auctions []model.Auction
	err := d.db.WithContext(ctx).
		Where("status = ?", status).
		Find(&auctions).Error
	return auctions, err
}

// GetExpiredAuctions 获取已过期但未结束的竞拍
func (d *AuctionDAO) GetExpiredAuctions(ctx context.Context) ([]model.Auction, error) {
	var auctions []model.Auction
	err := d.db.WithContext(ctx).
		Where("status IN ?", []model.AuctionStatus{
			model.AuctionStatusOngoing,
			model.AuctionStatusDelayed,
		}).
		Where("end_time <= NOW()").
		Find(&auctions).Error
	return auctions, err
}

// GetPendingAuctionsToStart 获取待开始且已到开始时间的竞拍
func (d *AuctionDAO) GetPendingAuctionsToStart(ctx context.Context) ([]model.Auction, error) {
	var auctions []model.Auction
	err := d.db.WithContext(ctx).
		Where("status = ?", model.AuctionStatusPending).
		Where("start_time <= NOW()").
		Find(&auctions).Error
	return auctions, err
}

// List 获取竞拍列表（支持分页和状态筛选）
func (d *AuctionDAO) List(ctx context.Context, status *model.AuctionStatus, page, pageSize int) ([]model.Auction, int64, error) {
	var auctions []model.Auction
	var total int64

	query := d.db.WithContext(ctx).Model(&model.Auction{})

	// 状态筛选
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&auctions).Error

	return auctions, total, err
}

// ListAll 获取所有竞拍（不分页）
func (d *AuctionDAO) ListAll(ctx context.Context) ([]model.Auction, error) {
	var auctions []model.Auction
	err := d.db.WithContext(ctx).
		Order("id DESC").
		Find(&auctions).Error
	return auctions, err
}

// ListWithFilters 获取竞拍列表（支持多条件筛选）
func (d *AuctionDAO) ListWithFilters(ctx context.Context, filters *AuctionFilters, page, pageSize int) ([]model.Auction, int64, error) {
	var auctions []model.Auction
	var total int64

	query := d.db.WithContext(ctx).Model(&model.Auction{})

	// 状态筛选
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	// product_id IN (...) 过滤（来自分类筛选编排，见 AuctionFilters.ProductIDs 注释）
	if filters.ProductIDs != nil {
		if len(filters.ProductIDs) == 0 {
			// 显式空切片视为"无命中"，直接返回空结果，避免空 IN 触发不可预期 SQL。
			return []model.Auction{}, 0, nil
		}
		query = query.Where("auctions.product_id IN ?", filters.ProductIDs)
	}

	// 直播间ID筛选
	if filters.LiveStreamID != nil {
		query = query.Where("live_stream_id = ?", *filters.LiveStreamID)
	}

	// 直播间名称搜索（需要JOIN live_streams表）
	if filters.LiveStreamName != "" {
		query = query.Joins("JOIN live_streams ON live_streams.id = auctions.live_stream_id").
			Where("live_streams.name LIKE ?", "%"+filters.LiveStreamName+"%")
	}

	// 关键词搜索（商品名称或直播间名称）
	if filters.Search != "" {
		query = query.Joins("JOIN products ON products.id = auctions.product_id").
			Joins("LEFT JOIN live_streams ON live_streams.id = auctions.live_stream_id").
			Where("products.name LIKE ? OR live_streams.name LIKE ?", "%"+filters.Search+"%", "%"+filters.Search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&auctions).Error

	return auctions, total, err
}

// CountActiveByLiveStreamIDs 按 live_stream_id 批量统计 "进行中" 竞拍数量
// （T3.3 / spec B §2.3：F-B3 列表项 auction_count 字段）。
//
// "进行中" 定义：status IN (Ongoing=1, Delayed=2)。
// 返回 map[live_stream_id]count；无记录的 id 不会出现在 map 中（调用方按 0 处理）。
func (d *AuctionDAO) CountActiveByLiveStreamIDs(ctx context.Context, liveStreamIDs []int64) (map[int64]int64, error) {
	if len(liveStreamIDs) == 0 {
		return map[int64]int64{}, nil
	}
	type row struct {
		LiveStreamID int64
		Cnt          int64
	}
	var rows []row
	err := d.db.WithContext(ctx).
		Model(&model.Auction{}).
		Select("live_stream_id, COUNT(*) AS cnt").
		Where("live_stream_id IN ?", liveStreamIDs).
		Where("status IN ?", []model.AuctionStatus{model.AuctionStatusOngoing, model.AuctionStatusDelayed}).
		Where("end_time > ?", time.Now()).
		Group("live_stream_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]int64, len(rows))
	for _, r := range rows {
		result[r.LiveStreamID] = r.Cnt
	}
	return result, nil
}

// AuctionFilters 竞拍筛选条件
type AuctionFilters struct {
	Status         *model.AuctionStatus
	LiveStreamID   *int64
	LiveStreamName string
	Search         string
	// ProductIDs 仅在 category_id 过滤时由 handler 层装填，
	// 来自 product-service /internal/products?category_id= 的 id 列表（spec C §5.2）。
	// 为空切片表示无命中（应由调用方提前短路），nil 表示未过滤。
	ProductIDs []int64
}

// GetByLiveStreamID 根据直播间ID获取竞拍列表
func (d *AuctionDAO) GetByLiveStreamID(ctx context.Context, liveStreamID int64, page, pageSize int) ([]model.Auction, int64, error) {
	var auctions []model.Auction
	var total int64

	query := d.db.WithContext(ctx).Model(&model.Auction{}).Where("live_stream_id = ?", liveStreamID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&auctions).Error

	return auctions, total, err
}

// GetCurrentByLiveStreamIDs 为每个 live_stream 取"当前竞拍"。
// 规则：仅在 ongoing/delayed 中，按 start_time DESC, id DESC 每组取第一条。
func (d *AuctionDAO) GetCurrentByLiveStreamIDs(ctx context.Context, liveStreamIDs []int64) (map[int64]*model.Auction, error) {
	result := make(map[int64]*model.Auction)
	if len(liveStreamIDs) == 0 {
		return result, nil
	}
	var rows []model.Auction
	err := d.db.WithContext(ctx).
		Where("live_stream_id IN ?", liveStreamIDs).
		Where("status IN ?", []model.AuctionStatus{model.AuctionStatusOngoing, model.AuctionStatusDelayed}).
		Where("end_time > ?", time.Now()).
		Order("live_stream_id ASC, start_time DESC, id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for i := range rows {
		ls := rows[i].LiveStreamID
		if ls == nil {
			continue
		}
		if _, ok := result[*ls]; !ok {
			result[*ls] = &rows[i]
		}
	}
	return result, nil
}
