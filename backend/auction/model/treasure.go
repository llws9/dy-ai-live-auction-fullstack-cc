package model

import "time"

// UserCoin stores the entertainment coin balance, isolated from cash balance.
type UserCoin struct {
	UserID    int64     `json:"user_id" gorm:"primaryKey;column:user_id"`
	Balance   int64     `json:"balance" gorm:"not null;default:0"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UserCoin) TableName() string { return "user_coins" }

// UserWatchDuration stores the daily watch duration bucketed by business date.
type UserWatchDuration struct {
	UserID       int64     `json:"user_id" gorm:"primaryKey;column:user_id"`
	StatDate     string    `json:"stat_date" gorm:"primaryKey;column:stat_date;type:varchar(10)"`
	TotalSeconds int       `json:"total_seconds" gorm:"not null;default:0"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (UserWatchDuration) TableName() string { return "user_watch_duration" }

// TreasureClaim records claimed treasure tiers; the composite primary key is the idempotency guard.
type TreasureClaim struct {
	UserID    int64     `json:"user_id" gorm:"primaryKey;column:user_id"`
	StatDate  string    `json:"stat_date" gorm:"primaryKey;column:stat_date;type:varchar(10)"`
	Tier      int8      `json:"tier" gorm:"primaryKey;column:tier"`
	Coins     int64     `json:"coins" gorm:"not null"`
	ClaimedAt time.Time `json:"claimed_at"`
}

func (TreasureClaim) TableName() string { return "treasure_claims" }
