package dao

import (
	"context"

	"gorm.io/gorm"

	"auction-service/model"
)

// BidDAO 出价数据访问层
type BidDAO struct {
	db *gorm.DB
}

// NewBidDAO 创建出价 DAO
func NewBidDAO(db *gorm.DB) *BidDAO {
	return &BidDAO{db: db}
}

// DB 返回底层数据库连接（用于事务）
func (d *BidDAO) DB() *gorm.DB {
	return d.db
}

// WithTx 使用事务创建 DAO 实例
func (d *BidDAO) WithTx(tx *gorm.DB) *BidDAO {
	return &BidDAO{db: tx}
}

// Create 创建出价记录
func (d *BidDAO) Create(ctx context.Context, bid *model.Bid) error {
	return d.db.WithContext(ctx).Create(bid).Error
}

// GetByID 根据 ID 获取出价记录
func (d *BidDAO) GetByID(ctx context.Context, id int64) (*model.Bid, error) {
	var bid model.Bid
	err := d.db.WithContext(ctx).First(&bid, id).Error
	if err != nil {
		return nil, err
	}
	return &bid, nil
}

// ListByAuctionID 获取竞拍的所有出价记录
func (d *BidDAO) ListByAuctionID(ctx context.Context, auctionID int64, limit int) ([]model.Bid, error) {
	var bids []model.Bid
	query := d.db.WithContext(ctx).
		Where("auction_id = ?", auctionID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&bids).Error
	return bids, err
}

// GetRanking 获取出价排名
func (d *BidDAO) GetRanking(ctx context.Context, auctionID int64, limit int) ([]model.Bid, error) {
	var bids []model.Bid

	// 使用子查询获取每个用户的最高出价
	subQuery := d.db.WithContext(ctx).
		Model(&model.Bid{}).
		Select("MIN(id) as id, auction_id, user_id, MAX(amount) as amount").
		Where("auction_id = ?", auctionID).
		Group("auction_id, user_id")

	err := d.db.WithContext(ctx).
		Table("(?) as max_bids", subQuery).
		Order("amount DESC").
		Limit(limit).
		Find(&bids).Error

	return bids, err
}

// GetWinnerBid returns the authoritative winning bid for an auction.
// It preserves the original bid row so result pages can show bid time.
func (d *BidDAO) GetWinnerBid(ctx context.Context, auctionID int64) (*model.Bid, error) {
	var bid model.Bid
	err := d.db.WithContext(ctx).
		Where("auction_id = ?", auctionID).
		Order("amount DESC, created_at ASC, id ASC").
		First(&bid).Error
	if err != nil {
		return nil, err
	}
	return &bid, nil
}

// GetUserHighestBid 获取用户在某个竞拍中的最高出价
func (d *BidDAO) GetUserHighestBid(ctx context.Context, auctionID, userID int64) (*model.Bid, error) {
	var bid model.Bid
	err := d.db.WithContext(ctx).
		Where("auction_id = ? AND user_id = ?", auctionID, userID).
		Order("amount DESC").
		First(&bid).Error

	if err != nil {
		return nil, err
	}
	return &bid, nil
}

// CountByAuctionID 统计竞拍的出价次数
func (d *BidDAO) CountByAuctionID(ctx context.Context, auctionID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.Bid{}).
		Where("auction_id = ?", auctionID).
		Count(&count).Error
	return count, err
}

// CountByUserIDAndAuctionID 统计用户在竞拍中的出价次数
func (d *BidDAO) CountByUserIDAndAuctionID(ctx context.Context, auctionID, userID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.Bid{}).
		Where("auction_id = ? AND user_id = ?", auctionID, userID).
		Count(&count).Error
	return count, err
}
