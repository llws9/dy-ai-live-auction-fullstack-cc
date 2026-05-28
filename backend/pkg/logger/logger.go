package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// OperationType 操作类型
type OperationType string

const (
	OperationCreate  OperationType = "create"
	OperationUpdate  OperationType = "update"
	OperationDelete  OperationType = "delete"
	OperationQuery   OperationType = "query"
	OperationLogin   OperationType = "login"
	OperationLogout  OperationType = "logout"
	OperationBid     OperationType = "bid"
	OperationPay     OperationType = "pay"
	OperationShip    OperationType = "ship"
	OperationComplete OperationType = "complete"
)

// ObjectType 操作对象类型
type ObjectType string

const (
	ObjectProduct  ObjectType = "product"
	ObjectAuction  ObjectType = "auction"
	ObjectBid      ObjectType = "bid"
	ObjectOrder    ObjectType = "order"
	ObjectUser     ObjectType = "user"
	ObjectAuth     ObjectType = "auth"
)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	ServiceName   string                 `json:"service_name"`
	RequestID     string                 `json:"request_id,omitempty"`
	OperationType OperationType          `json:"operation_type"`
	ObjectType    ObjectType             `json:"object_type"`
	ObjectID      string                 `json:"object_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	UserName      string                 `json:"user_name,omitempty"`
	Success       bool                   `json:"success"`
	Duration      time.Duration          `json:"duration,omitempty"`
	ErrorMsg      string                 `json:"error_msg,omitempty"`
	RequestData   map[string]interface{} `json:"request_data,omitempty"`
	ResponseData  interface{}            `json:"response_data,omitempty"`
	ClientIP      string                 `json:"client_ip,omitempty"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
}

// Logger 操作日志记录器
type Logger struct {
	serviceName string
	output      *os.File
	fileOutput  *os.File
	mu          sync.Mutex
}

// NewLogger 创建日志记录器
func NewLogger(serviceName string) *Logger {
	return &Logger{
		serviceName: serviceName,
		output:      os.Stdout,
	}
}

// NewLoggerWithFile 创建日志记录器（带文件输出）
func NewLoggerWithFile(serviceName, logDir string) *Logger {
	l := &Logger{
		serviceName: serviceName,
		output:      os.Stdout,
	}

	// 创建日志目录
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		return l
	}

	// 打开日志文件（按日期）
	logFile := filepath.Join(logDir, fmt.Sprintf("%s-%s.log", serviceName, time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		return l
	}

	l.fileOutput = file
	return l
}

// Close 关闭日志文件
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.fileOutput != nil {
		return l.fileOutput.Close()
	}
	return nil
}

// Log 记录操作日志
func (l *Logger) Log(ctx context.Context, entry LogEntry) {
	entry.Timestamp = time.Now()
	entry.ServiceName = l.serviceName

	// 从上下文获取请求ID
	if reqID := ctx.Value("request_id"); reqID != nil {
		entry.RequestID = reqID.(string)
	}

	// 序列化为JSON
	logJSON, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 输出到 stdout
	fmt.Fprintln(l.output, string(logJSON))

	// 输出到文件
	if l.fileOutput != nil {
		fmt.Fprintln(l.fileOutput, string(logJSON))
	}
}

// LogOperation 记录操作日志（简化版）
func (l *Logger) LogOperation(ctx context.Context, opType OperationType, objType ObjectType, objectID string, success bool, err error) {
	entry := LogEntry{
		OperationType: opType,
		ObjectType:    objType,
		ObjectID:      objectID,
		Success:       success,
	}

	if err != nil {
		entry.ErrorMsg = err.Error()
	}

	l.Log(ctx, entry)
}

// LogOperationWithData 记录操作日志（带数据）
func (l *Logger) LogOperationWithData(ctx context.Context, opType OperationType, objType ObjectType, objectID string, success bool, err error, requestData map[string]interface{}, responseData interface{}) {
	entry := LogEntry{
		OperationType: opType,
		ObjectType:    objType,
		ObjectID:      objectID,
		Success:       success,
		RequestData:   sanitizeData(requestData),
		ResponseData:  sanitizeResponseData(responseData),
	}

	if err != nil {
		entry.ErrorMsg = err.Error()
	}

	l.Log(ctx, entry)
}

// LogUserOperation 记录用户操作日志
func (l *Logger) LogUserOperation(ctx context.Context, userID, userName string, opType OperationType, objType ObjectType, objectID string, success bool, err error) {
	entry := LogEntry{
		UserID:        userID,
		UserName:      userName,
		OperationType: opType,
		ObjectType:    objType,
		ObjectID:      objectID,
		Success:       success,
	}

	if err != nil {
		entry.ErrorMsg = err.Error()
	}

	l.Log(ctx, entry)
}

// LogHTTPRequest 记录HTTP请求日志
func (l *Logger) LogHTTPRequest(ctx context.Context, c *app.RequestContext, opType OperationType, objType ObjectType, objectID string, success bool, err error, duration time.Duration) {
	entry := LogEntry{
		RequestID:     string(c.GetHeader("X-Request-ID")),
		UserID:        string(c.GetHeader("X-User-ID")),
		UserName:      string(c.GetHeader("X-User-Name")),
		OperationType: opType,
		ObjectType:    objType,
		ObjectID:      objectID,
		Success:       success,
		Duration:      duration,
		ClientIP:      c.ClientIP(),
	}

	if err != nil {
		entry.ErrorMsg = err.Error()
	}

	l.Log(ctx, entry)
}

// sanitizeData 脱敏处理请求数据
func sanitizeData(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	sanitized := make(map[string]interface{})
	for k, v := range data {
		// 敏感字段脱敏
		if k == "password" || k == "token" || k == "secret" || k == "api_key" {
			sanitized[k] = "***"
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}

// sanitizeResponseData 处理响应数据
func sanitizeResponseData(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	// 简单处理：只保留基本类型
	switch v := data.(type) {
	case string, int, int64, float64, bool:
		return v
	default:
		// 其他类型返回类型名
		return fmt.Sprintf("%T", data)
	}
}
