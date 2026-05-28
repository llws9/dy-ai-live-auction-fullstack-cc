package dbmetrics

import (
	"time"

	"gorm.io/gorm"
)

// MetricsRecorder 指标记录器接口
type MetricsRecorder interface {
	RecordSQLQuery(service, operation, table string, duration float64, err error)
}

// GormMetricsPlugin GORM 指标插件
type GormMetricsPlugin struct {
	serviceName string
	recorder    MetricsRecorder
}

// NewGormMetricsPlugin 创建 GORM 指标插件
func NewGormMetricsPlugin(serviceName string, recorder MetricsRecorder) *GormMetricsPlugin {
	return &GormMetricsPlugin{
		serviceName: serviceName,
		recorder:    recorder,
	}
}

// Name 返回插件名称
func (p *GormMetricsPlugin) Name() string {
	return "gorm:metrics"
}

// Initialize 初始化插件，注册回调
func (p *GormMetricsPlugin) Initialize(db *gorm.DB) error {
	// 注册查询前的回调
	err := db.Callback().Query().Before("gorm:query").Register("metrics:before_query", p.beforeQuery)
	if err != nil {
		return err
	}

	// 注册查询后的回调
	err = db.Callback().Query().After("gorm:query").Register("metrics:after_query", p.afterQuery)
	if err != nil {
		return err
	}

	// 注册创建前的回调
	err = db.Callback().Create().Before("gorm:create").Register("metrics:before_create", p.beforeCreate)
	if err != nil {
		return err
	}

	// 注册创建后的回调
	err = db.Callback().Create().After("gorm:create").Register("metrics:after_create", p.afterCreate)
	if err != nil {
		return err
	}

	// 注册更新前的回调
	err = db.Callback().Update().Before("gorm:update").Register("metrics:before_update", p.beforeUpdate)
	if err != nil {
		return err
	}

	// 注册更新后的回调
	err = db.Callback().Update().After("gorm:update").Register("metrics:after_update", p.afterUpdate)
	if err != nil {
		return err
	}

	// 注册删除前的回调
	err = db.Callback().Delete().Before("gorm:delete").Register("metrics:before_delete", p.beforeDelete)
	if err != nil {
		return err
	}

	// 注册删除后的回调
	err = db.Callback().Delete().After("gorm:delete").Register("metrics:after_delete", p.afterDelete)
	if err != nil {
		return err
	}

	// 注册事务前的回调
	err = db.Callback().Row().Before("gorm:row").Register("metrics:before_row", p.beforeRow)
	if err != nil {
		return err
	}

	// 注册事务后的回调
	err = db.Callback().Row().After("gorm:row").Register("metrics:after_row", p.afterRow)
	if err != nil {
		return err
	}

	return nil
}

// beforeQuery 查询前回调
func (p *GormMetricsPlugin) beforeQuery(db *gorm.DB) {
	db.Set("metrics:start_time", time.Now())
}

// afterQuery 查询后回调
func (p *GormMetricsPlugin) afterQuery(db *gorm.DB) {
	p.recordMetrics(db, "query")
}

// beforeCreate 创建前回调
func (p *GormMetricsPlugin) beforeCreate(db *gorm.DB) {
	db.Set("metrics:start_time", time.Now())
}

// afterCreate 创建后回调
func (p *GormMetricsPlugin) afterCreate(db *gorm.DB) {
	p.recordMetrics(db, "create")
}

// beforeUpdate 更新前回调
func (p *GormMetricsPlugin) beforeUpdate(db *gorm.DB) {
	db.Set("metrics:start_time", time.Now())
}

// afterUpdate 更新后回调
func (p *GormMetricsPlugin) afterUpdate(db *gorm.DB) {
	p.recordMetrics(db, "update")
}

// beforeDelete 删除前回调
func (p *GormMetricsPlugin) beforeDelete(db *gorm.DB) {
	db.Set("metrics:start_time", time.Now())
}

// afterDelete 删除后回调
func (p *GormMetricsPlugin) afterDelete(db *gorm.DB) {
	p.recordMetrics(db, "delete")
}

// beforeRow 行操作前回调
func (p *GormMetricsPlugin) beforeRow(db *gorm.DB) {
	db.Set("metrics:start_time", time.Now())
}

// afterRow 行操作后回调
func (p *GormMetricsPlugin) afterRow(db *gorm.DB) {
	p.recordMetrics(db, "row")
}

// recordMetrics 记录指标
func (p *GormMetricsPlugin) recordMetrics(db *gorm.DB, operation string) {
	startTime, ok := db.Get("metrics:start_time")
	if !ok {
		return
	}

	duration := time.Since(startTime.(time.Time)).Seconds()

	// 获取表名
	table := db.Statement.Table
	if table == "" {
		table = "unknown"
	}

	// 记录指标
	if p.recorder != nil {
		p.recorder.RecordSQLQuery(p.serviceName, operation, table, duration, db.Error)
	}
}