package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"auction-service/model"
)

// UserProductReminderDAO 用户订阅商品竞拍提醒DAO
type UserProductReminderDAO struct {
	db *gorm.DB
}

type ProductReminderCandidate struct {
	UserID    int64
	ProductID int64
	AuctionID int64
	StartTime time.Time
}

// NewUserProductReminderDAO 创建用户订阅商品竞拍提醒DAO
func NewUserProductReminderDAO(db *gorm.DB) *UserProductReminderDAO {
	return &UserProductReminderDAO{db: db}
}

// Create 创建订阅记录
func (d *UserProductReminderDAO) Create(ctx context.Context, reminder *model.UserProductReminder) error {
	return d.db.WithContext(ctx).Create(reminder).Error
}

// GetByUserProduct 根据用户ID和商品ID获取订阅记录
func (d *UserProductReminderDAO) GetByUserProduct(ctx context.Context, userID, productID int64) (*model.UserProductReminder, error) {
	var reminder model.UserProductReminder
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND product_id = ?", userID, productID).
		First(&reminder).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reminder, nil
}

// Delete 删除订阅记录
func (d *UserProductReminderDAO) Delete(ctx context.Context, userID, productID int64) error {
	return d.db.WithContext(ctx).
		Where("user_id = ? AND product_id = ?", userID, productID).
		Delete(&model.UserProductReminder{}).Error
}

// GetByUser 获取用户的所有订阅记录
func (d *UserProductReminderDAO) GetByUser(ctx context.Context, userID int64) ([]*model.UserProductReminder, error) {
	var reminders []*model.UserProductReminder
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&reminders).Error
	return reminders, err
}

// CountByProduct 统计商品的订阅人数
func (d *UserProductReminderDAO) CountByProduct(ctx context.Context, productID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.UserProductReminder{}).
		Where("product_id = ? AND notification_enabled = ?", productID, true).
		Count(&count).Error
	return count, err
}

func (d *UserProductReminderDAO) GetStartingSoonByUser(ctx context.Context, userID int64, start, end time.Time) ([]ProductReminderCandidate, error) {
	var candidates []ProductReminderCandidate
	err := d.db.WithContext(ctx).
		Table("user_product_reminders AS r").
		Select("r.user_id, r.product_id, r.auction_id, a.start_time").
		Joins("JOIN auctions AS a ON a.id = r.auction_id").
		Where("r.user_id = ? AND r.notification_enabled = ? AND r.auction_id > 0", userID, true).
		Where("a.status = ? AND a.start_time >= ? AND a.start_time <= ?", model.AuctionStatusPending, start, end).
		Order("a.start_time ASC").
		Scan(&candidates).Error
	return candidates, err
}

func (d *UserProductReminderDAO) ClaimAndCreateAuctionStartNotification(ctx context.Context, userID, auctionID int64, notification *model.Notification) (bool, error) {
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		receipt := &model.ProductReminderReceipt{
			UserID:     userID,
			AuctionID:  auctionID,
			RemindedAt: time.Now(),
		}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(receipt)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		return tx.Create(notification).Error
	})
	if err != nil {
		return false, err
	}
	return notification.ID > 0, nil
}

// Redis key 常量
const (
	UserProductReminderZSETKey = "user:%d:product_reminders:start_time" // 用户订阅的商品开播时间索引
)

// AddToRedisZSET 添加用户订阅的商品到Redis ZSET（score为开播时间，member为auctionID）
func (d *UserProductReminderDAO) AddToRedisZSET(ctx context.Context, userID, auctionID int64, startTime time.Time) error {
	client := GetRedis()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf(UserProductReminderZSETKey, userID)
	score := float64(startTime.Unix())
	member := auctionID

	return client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

// RemoveFromRedisZSET 从Redis ZSET移除用户订阅的商品
func (d *UserProductReminderDAO) RemoveFromRedisZSET(ctx context.Context, userID, auctionID int64) error {
	client := GetRedis()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf(UserProductReminderZSETKey, userID)
	return client.ZRem(ctx, key, auctionID).Err()
}

// GetRemindersStartingSoon 从Redis获取用户订阅的即将开始的竞拍
func (d *UserProductReminderDAO) GetRemindersStartingSoon(ctx context.Context, userID int64, start, end time.Time) ([]int64, error) {
	client := GetRedis()
	if client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf(UserProductReminderZSETKey, userID)
	min := float64(start.Unix())
	max := float64(end.Unix())

	result, err := client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", min),
		Max: fmt.Sprintf("%f", max),
	}).Result()
	if err != nil {
		return nil, err
	}

	// 转换string到int64
	auctionIDs := make([]int64, 0, len(result))
	for _, s := range result {
		var id int64
		if _, err := fmt.Sscanf(s, "%d", &id); err == nil {
			auctionIDs = append(auctionIDs, id)
		}
	}

	return auctionIDs, nil
}

// GetAllRemindersFromRedis 获取用户所有订阅的竞拍ID（从Redis）
func (d *UserProductReminderDAO) GetAllRemindersFromRedis(ctx context.Context, userID int64) ([]int64, error) {
	client := GetRedis()
	if client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf(UserProductReminderZSETKey, userID)
	result, err := client.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	// 转换string到int64
	auctionIDs := make([]int64, 0, len(result))
	for _, s := range result {
		var id int64
		if _, err := fmt.Sscanf(s, "%d", &id); err == nil {
			auctionIDs = append(auctionIDs, id)
		}
	}

	return auctionIDs, nil
}
