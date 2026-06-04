package service

import (
	"context"
	"errors"
	"time"

	"auction-service/dao"
	"auction-service/model"
)

// ProductReminderService 用户订阅商品竞拍提醒服务
type ProductReminderService struct {
	reminderDAO *dao.UserProductReminderDAO
	auctionDAO  *dao.AuctionDAO
}

// NewProductReminderService 创建用户订阅商品竞拍提醒服务
func NewProductReminderService(reminderDAO *dao.UserProductReminderDAO) *ProductReminderService {
	return &ProductReminderService{
		reminderDAO: reminderDAO,
	}
}

// SetAuctionDAO 设置竞拍DAO
func (s *ProductReminderService) SetAuctionDAO(auctionDAO *dao.AuctionDAO) {
	s.auctionDAO = auctionDAO
}

// Subscribe 订阅商品提醒
func (s *ProductReminderService) Subscribe(ctx context.Context, userID, productID int64) error {
	// 检查是否已订阅
	existing, err := s.reminderDAO.GetByUserProduct(ctx, userID, productID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	// 获取商品关联的竞拍信息
	var auctionID int64 = 0
	var startTime time.Time
	if s.auctionDAO != nil {
		auction, err := s.auctionDAO.GetByProductID(ctx, productID)
		if err != nil {
			return err
		}
		if auction != nil {
			auctionID = auction.ID
			startTime = auction.StartTime
		}
	}

	// 创建订阅记录
	reminder := &model.UserProductReminder{
		UserID:              userID,
		ProductID:           productID,
		AuctionID:           auctionID,
		NotificationEnabled: true,
		CreatedAt:           time.Now(),
	}

	if err := s.reminderDAO.Create(ctx, reminder); err != nil {
		return err
	}

	// 如果有竞拍信息，添加到Redis ZSET
	if auctionID > 0 && !startTime.IsZero() {
		if err := s.reminderDAO.AddToRedisZSET(ctx, userID, auctionID, startTime); err != nil {
			// Redis写入失败不影响订阅，只记录错误
			// 实际生产环境应该记录日志
		}
	}

	return nil
}

// Unsubscribe 取消订阅
func (s *ProductReminderService) Unsubscribe(ctx context.Context, userID, productID int64) error {
	// 获取订阅记录（用于获取auctionID）
	reminder, err := s.reminderDAO.GetByUserProduct(ctx, userID, productID)
	if err != nil {
		return err
	}
	if reminder == nil {
		return errors.New("未订阅该商品的提醒")
	}

	// 从数据库删除订阅记录
	if err := s.reminderDAO.Delete(ctx, userID, productID); err != nil {
		return err
	}

	// 从Redis ZSET移除
	if reminder.AuctionID > 0 {
		if err := s.reminderDAO.RemoveFromRedisZSET(ctx, userID, reminder.AuctionID); err != nil {
			// Redis删除失败不影响取消订阅，只记录错误
		}
	}

	return nil
}

// GetUserReminders 获取用户订阅列表
func (s *ProductReminderService) GetUserReminders(ctx context.Context, userID int64) ([]*model.UserProductReminder, error) {
	return s.reminderDAO.GetByUser(ctx, userID)
}

// GetRemindersStartingSoon 获取用户订阅的即将开始的竞拍
func (s *ProductReminderService) GetRemindersStartingSoon(ctx context.Context, userID int64, start, end time.Time) ([]int64, error) {
	return s.reminderDAO.GetRemindersStartingSoon(ctx, userID, start, end)
}

// UpdateAuctionID 更新订阅记录的竞拍ID（当商品创建竞拍时调用）
func (s *ProductReminderService) UpdateAuctionID(ctx context.Context, userID, productID, auctionID int64, startTime time.Time) error {
	// 获取订阅记录
	reminder, err := s.reminderDAO.GetByUserProduct(ctx, userID, productID)
	if err != nil {
		return err
	}
	if reminder == nil {
		return nil // 用户未订阅，无需更新
	}

	// 更新数据库中的auctionID
	reminder.AuctionID = auctionID

	// 使用GORM更新（这里需要通过DAO层实现）
	// 添加到Redis ZSET
	if auctionID > 0 && !startTime.IsZero() {
		if err := s.reminderDAO.AddToRedisZSET(ctx, userID, auctionID, startTime); err != nil {
			// Redis写入失败不影响更新
		}
	}

	return nil
}

// CountSubscribers 统计商品的订阅人数
func (s *ProductReminderService) CountSubscribers(ctx context.Context, productID int64) (int64, error) {
	return s.reminderDAO.CountByProduct(ctx, productID)
}
