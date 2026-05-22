package service

import (
	"context"

	"product-service/dao"
)

// HistoryService 历史记录服务
type HistoryService struct {
	historyDAO *dao.HistoryDAO
}

// NewHistoryService 创建历史记录服务
func NewHistoryService(historyDAO *dao.HistoryDAO) *HistoryService {
	return &HistoryService{
		historyDAO: historyDAO,
	}
}

// GetUserHistory 获取用户竞拍历史
func (s *HistoryService) GetUserHistory(ctx context.Context, userID int64, page, pageSize int) ([]dao.UserHistoryItem, int64, error) {
	// 参数校验
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	// 调用DAO查询
	items, total, err := s.historyDAO.QueryUserHistory(ctx, userID, page, pageSize)
	if err != nil {
		// 尝试使用备用方案
		items, total, err = s.historyDAO.QueryUserHistoryGORM(ctx, userID, page, pageSize)
		if err != nil {
			return nil, 0, err
		}
	}

	return items, total, nil
}
