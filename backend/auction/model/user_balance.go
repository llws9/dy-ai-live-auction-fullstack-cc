package model

import "time"

// UserBalance 用户余额（T3.1 / spec A F-A2）。
//
// 设计说明：
//   - user_id 作为主键，1 用户 1 行。无记录由 handler 兜底返回零余额。
//   - decimal(10,2) 与现有 Auction.CurrentPrice 一致，避免 float 精度漂移在跨字段计算时放大。
//   - currency 以字符串保存（如 "CNY"），不引入枚举表，本期只读、不限制取值。
//   - 当前阶段仅作为只读视图：写入由后续支付/退款链路负责，本期不开放写接口。
type UserBalance struct {
	UserID          int64     `json:"user_id" gorm:"primaryKey;column:user_id"`
	AvailableAmount float64   `json:"available_amount" gorm:"type:decimal(10,2);default:0"`
	FrozenAmount    float64   `json:"frozen_amount" gorm:"type:decimal(10,2);default:0"`
	Currency        string    `json:"currency" gorm:"type:varchar(8);default:CNY"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName 显式指定，避免 GORM 复数推断（"user_balances" 已是期望值，但显式更稳）。
func (UserBalance) TableName() string {
	return "user_balances"
}
