package model

import "time"

// CategoryStatus 类别状态
type CategoryStatus int

const (
	CategoryStatusDisabled CategoryStatus = 0 // 禁用
	CategoryStatusActive   CategoryStatus = 1 // 启用
)

// Category 商品类别模型
type Category struct {
	ID          int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string         `json:"name" gorm:"type:varchar(64);not null"`
	Code        string         `json:"code" gorm:"type:varchar(32);uniqueIndex;not null"`
	Description string         `json:"description" gorm:"type:text"`
	SortOrder   int            `json:"sort_order" gorm:"default:0"`
	Status      CategoryStatus `json:"status" gorm:"type:tinyint;default:1"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (Category) TableName() string {
	return "categories"
}

// IsActive 判断类别是否启用
func (c *Category) IsActive() bool {
	return c.Status == CategoryStatusActive
}