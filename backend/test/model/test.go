package model

import "time"

// 测试状态常量
const (
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// 测试类型常量
const (
	TypeDummy       = "dummy"
	TypePressure    = "pressure"
	TypeE2E         = "e2e"
	TypeAntiSnipe   = "antisnipe"
	TypeCallback    = "callback"
	TypeChaos       = "chaos"
	TypeScript      = "script"
)

// TestResult 测试任务记录
type TestResult struct {
	ID          string     `gorm:"primaryKey;column:id;size:36"`
	TestType    string     `gorm:"column:test_type;size:20;not null;index"`
	Status      string     `gorm:"column:status;size:20;not null;index"`
	ConfigJSON  string     `gorm:"column:config_json;type:text;not null"`
	ResultJSON  string     `gorm:"column:result_json;type:text"`
	ReplayToken string     `gorm:"column:replay_token;size:64;index"`
	ScriptName  string     `gorm:"column:script_name;size:64"`
	ErrorMsg    string     `gorm:"column:error_msg;type:text"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;index"`
	CompletedAt *time.Time `gorm:"column:completed_at"`
}

// TableName 指定表名
func (TestResult) TableName() string { return "test_results" }

// TestSeedData 测试创建的业务数据 ref（用于清理）
type TestSeedData struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	TestID    string    `gorm:"column:test_id;size:36;not null;index"`
	Kind      string    `gorm:"column:kind;size:20;not null"`
	RefID     int64     `gorm:"column:ref_id;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

// TableName 指定表名
func (TestSeedData) TableName() string { return "test_seed_data" }
