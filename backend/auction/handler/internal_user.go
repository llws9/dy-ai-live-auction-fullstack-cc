package handler

import (
	"context"
	"errors"

	"auction-service/model"
)

// internalUserBatchMaxIDs 限制单次批量上限，与 spec B §4.1 一致。
const internalUserBatchMaxIDs = 200

// UserBatchProvider 抽象 UserDAO.GetByIDs，便于编排单测。
type UserBatchProvider interface {
	GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.User, error)
}

// UserSummary 是 /internal/users/batch 的返回单元（spec B §4.1 契约）。
type UserSummary struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

// UserSummariesResponse 是 /internal/users/batch 的 data 字段。
type UserSummariesResponse struct {
	Items []UserSummary `json:"items"`
}

// BuildUserSummaries 是 T3.3 / spec B §4.1 的纯编排函数：
//   - 校验 ids（非空、长度 ≤ internalUserBatchMaxIDs）
//   - 查询 UserBatchProvider，按入参顺序回填，缺失记录跳过（不抛错）
//   - 错误向上冒泡
func BuildUserSummaries(ctx context.Context, p UserBatchProvider, ids []int64) (*UserSummariesResponse, error) {
	if len(ids) == 0 {
		return nil, errors.New("ids 不能为空")
	}
	if len(ids) > internalUserBatchMaxIDs {
		return nil, errors.New("ids 超出上限")
	}
	users, err := p.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	items := make([]UserSummary, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		u, ok := users[id]
		if !ok {
			continue
		}
		items = append(items, UserSummary{
			ID:       u.ID,
			Username: u.Name,
			Avatar:   u.Avatar,
		})
	}
	return &UserSummariesResponse{Items: items}, nil
}
