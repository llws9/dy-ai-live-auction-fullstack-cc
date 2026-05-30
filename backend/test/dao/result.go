package dao

import (
	"context"
	"time"

	"gorm.io/gorm"

	"test-service/model"
)

// ResultDAO test_results 表读写
type ResultDAO struct {
	db *gorm.DB
}

// NewResultDAO 构造
func NewResultDAO(db *gorm.DB) *ResultDAO {
	return &ResultDAO{db: db}
}

// HistoryFilters 历史查询过滤参数
type HistoryFilters struct {
	TestType string
	Status   string
	Page     int
	PageSize int
}

// Save 写入一条测试记录
func (d *ResultDAO) Save(ctx context.Context, r *model.TestResult) error {
	return d.db.WithContext(ctx).Create(r).Error
}

// GetByID 按 ID 查询
func (d *ResultDAO) GetByID(ctx context.Context, id string) (*model.TestResult, error) {
	var r model.TestResult
	if err := d.db.WithContext(ctx).First(&r, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// UpdateStatus 更新测试状态/结果/错误信息/完成时间
func (d *ResultDAO) UpdateStatus(ctx context.Context, id, status, resultJSON, errorMsg string, completedAt *time.Time) error {
	upd := map[string]any{
		"status": status,
	}
	if resultJSON != "" {
		upd["result_json"] = resultJSON
	}
	if errorMsg != "" {
		upd["error_msg"] = errorMsg
	}
	if completedAt != nil {
		upd["completed_at"] = *completedAt
	}
	return d.db.WithContext(ctx).Model(&model.TestResult{}).Where("id = ?", id).Updates(upd).Error
}

// GetHistory 历史列表查询
func (d *ResultDAO) GetHistory(ctx context.Context, f HistoryFilters) ([]model.TestResult, int64, error) {
	page := f.Page
	if page < 1 {
		page = 1
	}
	size := f.PageSize
	if size <= 0 {
		size = 20
	}

	q := d.db.WithContext(ctx).Model(&model.TestResult{})
	if f.TestType != "" {
		q = q.Where("test_type = ?", f.TestType)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []model.TestResult
	if err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// DeleteOlderThan 删除早于 cutoff 的 test_results 记录，返回删除条数
func (d *ResultDAO) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res := d.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&model.TestResult{})
	return res.RowsAffected, res.Error
}
