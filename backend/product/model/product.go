package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// ProductStatus 商品状态
type ProductStatus int

const (
	ProductStatusDraft       ProductStatus = 0 // 草稿
	ProductStatusPublished   ProductStatus = 1 // 已发布
	ProductStatusUnpublished ProductStatus = 2 // 已下架
)

// Product 商品信息模型
type Product struct {
	ID           int64         `json:"id" gorm:"primaryKey;autoIncrement"`
	Name         string        `json:"name" gorm:"type:varchar(128);not null"`
	Description  string        `json:"description" gorm:"type:text"`
	Images       JSONArray     `json:"images" gorm:"type:json"`
	CategoryID   *int64        `json:"category_id" gorm:"index"` // 逻辑外键关联Category
	CategoryName string        `json:"category_name,omitempty" gorm:"column:category_name;->;-:migration"`
	Status       ProductStatus `json:"status" gorm:"type:tinyint;default:0"`
	CreatedAt    time.Time     `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (Product) TableName() string {
	return "products"
}

// JSONArray 用于存储 JSON 数组类型
type JSONArray []string

// Value 实现 driver.Valuer 接口
func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}
