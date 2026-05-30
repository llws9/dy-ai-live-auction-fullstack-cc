package handler

import (
	"context"
	"errors"
)

// BalanceProvider 抽象 BalanceDAO.GetByUserID，方便单测。
//
// 返回签名说明：available/frozen/currency 是命中时的字段；hit 标识是否存在记录；
// err 为 DAO 故障。未命中（hit=false, err=nil）由编排层补默认值，
// 让前端始终拿到 stable shape。
type BalanceProvider interface {
	GetByUserID(ctx context.Context, userID int64) (available, frozen float64, currency string, hit bool, err error)
}

// UserBalanceResponse 是 GET /api/v1/user/balance 的稳定响应数据。
type UserBalanceResponse struct {
	AvailableAmount float64 `json:"available_amount"`
	FrozenAmount    float64 `json:"frozen_amount"`
	Currency        string  `json:"currency"`
}

// BuildUserBalanceResponse 是 T3.1 / spec A F-A2 的纯编排函数：
//   - 校验 userID
//   - 查询 BalanceProvider
//   - 未命中返回零余额 + currency=CNY 默认值
//   - 错误向上冒泡（handler 转 5xx）
func BuildUserBalanceResponse(ctx context.Context, bp BalanceProvider, userID int64) (*UserBalanceResponse, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user_id")
	}
	available, frozen, currency, hit, err := bp.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !hit {
		return &UserBalanceResponse{
			AvailableAmount: 0,
			FrozenAmount:    0,
			Currency:        "CNY",
		}, nil
	}
	if currency == "" {
		currency = "CNY"
	}
	return &UserBalanceResponse{
		AvailableAmount: available,
		FrozenAmount:    frozen,
		Currency:        currency,
	}, nil
}
