package dao

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gateway-service/model"
)

type BusinessEventDAO struct {
	db *gorm.DB
}

func NewBusinessEventDAO(db *gorm.DB) *BusinessEventDAO {
	return &BusinessEventDAO{db: db}
}

func (d *BusinessEventDAO) Create(ctx context.Context, event *model.BusinessEvent) error {
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "client_event_id"}},
		DoNothing: true,
	}).Create(event).Error
}
