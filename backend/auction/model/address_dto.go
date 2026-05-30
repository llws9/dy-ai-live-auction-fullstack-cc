package model

import "time"

// AddressView 是对外稳定 shape（spec A / F-A3），跨 handler/dao 共用，避免 import cycle。
type AddressView struct {
	ID            int64     `json:"id"`
	RecipientName string    `json:"recipient_name"`
	Phone         string    `json:"phone"`
	Province      string    `json:"province"`
	City          string    `json:"city"`
	District      string    `json:"district"`
	Detail        string    `json:"detail"`
	IsDefault     bool      `json:"is_default"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AddressMutation 是 handler→dao 的写入快照。
type AddressMutation struct {
	ID            int64
	UserID        int64
	RecipientName string
	Phone         string
	Province      string
	City          string
	District      string
	Detail        string
	IsDefault     bool
}
