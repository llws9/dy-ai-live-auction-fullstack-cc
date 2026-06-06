package dao

import (
	"context"
	"encoding/json"
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
	AuctionID         int64   `json:"auction_id" gorm:"column:auction_id"`
	ProductName       string  `json:"product_name" gorm:"column:product_name"`
	ProductImage      string  `json:"product_image" gorm:"-"`
	ProductImagesJSON string  `json:"-" gorm:"column:product_images"`
	FinalPrice        float64 `json:"final_price" gorm:"column:final_price"`
	IsWinner          bool    `json:"is_winner" gorm:"column:is_winner"`
	Status            int     `json:"status" gorm:"column:status"`
	BidCount          int     `json:"bid_count" gorm:"column:bid_count"`
	CreatedAt         string  `json:"created_at" gorm:"column:created_at"`
}

// QueryUserHistory 查询用户竞拍历史
func (d *HistoryDAO) QueryUserHistory(ctx context.Context, userID int64, page, pageSize int) ([]UserHistoryItem, int64, error) {
	var items []UserHistoryItem
	var total int64

	offset := (page - 1) * pageSize

	bidCountExpr := "0"
	args := make([]interface{}, 0, 4)
	if d.db.Migrator().HasTable("bids") {
		bidCountExpr = "(SELECT COUNT(*) FROM bids b WHERE b.auction_id = o.auction_id AND b.user_id = ?)"
		args = append(args, userID)
	}

	// 中标记录以 orders 为事实源：订单由 auction-service 结算后写入 product-service。
	// 不依赖 product-service 本地存在 auctions/bids 镜像，避免跨服务数据镜像缺失导致漏单。
	query := fmt.Sprintf(`
		SELECT
			o.auction_id as auction_id,
			p.name as product_name,
			p.images as product_images,
			o.final_price as final_price,
			1 as is_winner,
			o.status as status,
			%s as bid_count,
			o.created_at as created_at
		FROM orders o
		JOIN products p ON o.product_id = p.id
		WHERE o.winner_id = ?
		ORDER BY o.created_at DESC
		LIMIT ? OFFSET ?
	`, bidCountExpr)
	args = append(args, userID, pageSize, offset)

	if err := d.db.WithContext(ctx).Raw(query, args...).Scan(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("query user history failed: %w", err)
	}
	fillProductImages(items)

	// 查询总数
	countQuery := `
		SELECT COUNT(*)
		FROM orders o
		WHERE o.winner_id = ?
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

	var orders []struct {
		AuctionID  int64
		ProductID  int64
		FinalPrice float64
		Status     int
		CreatedAt  time.Time
	}
	if err := d.db.WithContext(ctx).
		Table("orders").
		Select("auction_id, product_id, final_price, status, created_at").
		Where("winner_id = ?", userID).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	for _, order := range orders {
		item := UserHistoryItem{
			AuctionID:  order.AuctionID,
			FinalPrice: order.FinalPrice,
			IsWinner:   true,
			Status:     order.Status,
			CreatedAt:  order.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		// 获取商品名和首图来源
		var productInfo struct {
			Name   string `gorm:"column:name"`
			Images string `gorm:"column:images"`
		}
		if err := d.db.WithContext(ctx).
			Table("products").
			Select("name, images").
			Where("id = ?", order.ProductID).
			Scan(&productInfo).Error; err == nil {
			item.ProductName = productInfo.Name
			item.ProductImagesJSON = productInfo.Images
		}

		// 获取出价次数
		var bidCount int64
		if d.db.Migrator().HasTable("bids") {
			d.db.WithContext(ctx).
				Table("bids").
				Where("auction_id = ? AND user_id = ?", order.AuctionID, userID).
				Count(&bidCount)
		}
		item.BidCount = int(bidCount)

		items = append(items, item)
	}
	fillProductImages(items)

	// 获取总数
	d.db.WithContext(ctx).
		Table("orders").
		Where("winner_id = ?", userID).
		Count(&total)

	return items, total, nil
}

func fillProductImages(items []UserHistoryItem) {
	for i := range items {
		items[i].ProductImage = firstHistoryProductImage(items[i].ProductImagesJSON)
	}
}

func firstHistoryProductImage(raw string) string {
	if raw == "" {
		return ""
	}
	var images []string
	if err := json.Unmarshal([]byte(raw), &images); err != nil || len(images) == 0 {
		return ""
	}
	return images[0]
}
