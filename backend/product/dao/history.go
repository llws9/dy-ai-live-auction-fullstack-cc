package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AuctionStatus 竞拍状态常量
const (
	AuctionStatusEnded = 3 // 已结束
)

// HistoryDAO 历史记录DAO
type HistoryDAO struct {
	db *gorm.DB
}

// NewHistoryDAO 创建历史记录DAO
func NewHistoryDAO(db *gorm.DB) *HistoryDAO {
	return &HistoryDAO{db: db}
}

// UserHistoryItem 用户历史记录项
type UserHistoryItem struct {
	AuctionID   int64   `json:"auction_id" gorm:"column:auction_id"`
	ProductName string  `json:"product_name" gorm:"column:product_name"`
	FinalPrice  float64 `json:"final_price" gorm:"column:final_price"`
	IsWinner    bool    `json:"is_winner" gorm:"column:is_winner"`
	BidCount    int     `json:"bid_count" gorm:"column:bid_count"`
	CreatedAt   string  `json:"created_at" gorm:"column:created_at"`
}

// QueryUserHistory 查询用户竞拍历史
func (d *HistoryDAO) QueryUserHistory(ctx context.Context, userID int64, page, pageSize int) ([]UserHistoryItem, int64, error) {
	var items []UserHistoryItem
	var total int64

	offset := (page - 1) * pageSize

	// 查询用户参与的竞拍历史
	// 使用原生SQL进行复杂联表查询
	query := `
		SELECT
			a.id as auction_id,
			p.name as product_name,
			COALESCE(o.final_price, 0) as final_price,
			CASE WHEN o.winner_id = ? THEN 1 ELSE 0 END as is_winner,
			(SELECT COUNT(*) FROM bids b WHERE b.auction_id = a.id AND b.user_id = ?) as bid_count,
			DATE_FORMAT(a.created_at, '%Y-%m-%dT%H:%i:%sZ') as created_at
		FROM auctions a
		JOIN products p ON a.product_id = p.id
		LEFT JOIN orders o ON a.id = o.auction_id
		WHERE a.status = 3
		  AND EXISTS (SELECT 1 FROM bids b WHERE b.auction_id = a.id AND b.user_id = ?)
		ORDER BY a.created_at DESC
		LIMIT ? OFFSET ?
	`

	if err := d.db.WithContext(ctx).Raw(query, userID, userID, userID, pageSize, offset).Scan(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("query user history failed: %w", err)
	}

	// 查询总数
	countQuery := `
		SELECT COUNT(DISTINCT a.id)
		FROM auctions a
		JOIN bids b ON a.id = b.auction_id
		WHERE a.status = 3 AND b.user_id = ?
	`

	if err := d.db.WithContext(ctx).Raw(countQuery, userID).Scan(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count user history failed: %w", err)
	}

	return items, total, nil
}

// QueryUserHistoryGORM 使用GORM查询用户竞拍历史（备用方案）
func (d *HistoryDAO) QueryUserHistoryGORM(ctx context.Context, userID int64, page, pageSize int) ([]UserHistoryItem, int64, error) {
	var items []UserHistoryItem
	var total int64

	offset := (page - 1) * pageSize

	// 分步查询（简化版，避免复杂SQL）
	// 1. 获取用户参与过的竞拍ID
	var auctionIDs []int64
	if err := d.db.WithContext(ctx).
		Table("bids").
		Select("DISTINCT auction_id").
		Where("user_id = ?", userID).
		Pluck("auction_id", &auctionIDs).Error; err != nil {
		return nil, 0, err
	}

	if len(auctionIDs) == 0 {
		return []UserHistoryItem{}, 0, nil
	}

	// 2. 获取已结束的竞拍
	var auctions []struct {
		ID        int64
		ProductID int64
		CreatedAt time.Time
	}
	if err := d.db.WithContext(ctx).
		Table("auctions").
		Select("id, product_id, created_at").
		Where("id IN ? AND status = ?", auctionIDs, AuctionStatusEnded).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&auctions).Error; err != nil {
		return nil, 0, err
	}

	// 3. 构建结果
	for _, a := range auctions {
		item := UserHistoryItem{
			AuctionID: a.ID,
			CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		// 获取商品名
		var productName string
		if err := d.db.WithContext(ctx).
			Table("products").
			Select("name").
			Where("id = ?", a.ProductID).
			Scan(&productName).Error; err == nil {
			item.ProductName = productName
		}

		// 获取订单信息
		var order struct {
			FinalPrice float64
			WinnerID   int64
		}
		if err := d.db.WithContext(ctx).
			Table("orders").
			Select("final_price, winner_id").
			Where("auction_id = ?", a.ID).
			Scan(&order).Error; err == nil {
			item.FinalPrice = order.FinalPrice
			item.IsWinner = order.WinnerID == userID
		}

		// 获取出价次数
		var bidCount int64
		d.db.WithContext(ctx).
			Table("bids").
			Where("auction_id = ? AND user_id = ?", a.ID, userID).
			Count(&bidCount)
		item.BidCount = int(bidCount)

		items = append(items, item)
	}

	// 获取总数
	d.db.WithContext(ctx).
		Table("auctions").
		Where("id IN ? AND status = ?", auctionIDs, AuctionStatusEnded).
		Count(&total)

	return items, total, nil
}
