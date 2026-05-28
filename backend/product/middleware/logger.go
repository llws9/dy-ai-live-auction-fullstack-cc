package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// LoggerConfig 日志配置
type LoggerConfig struct {
	ServiceName string
}

// OperationLog 操作日志
type OperationLog struct {
	Timestamp     time.Time              `json:"timestamp"`
	ServiceName   string                 `json:"service_name"`
	RequestID     string                 `json:"request_id"`
	OperationType string                 `json:"operation_type"` // create, update, delete, query
	ObjectType    string                 `json:"object_type"`    // product, auction, order, bid
	ObjectID      string                 `json:"object_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	UserName      string                 `json:"user_name,omitempty"`
	Method        string                 `json:"method"`
	Path          string                 `json:"path"`
	Status        int                    `json:"status"`
	Latency       time.Duration          `json:"latency"`
	ClientIP      string                 `json:"client_ip"`
	Success       bool                   `json:"success"`
	ErrorMsg      string                 `json:"error_msg,omitempty"`
	RequestData   map[string]interface{} `json:"request_data,omitempty"`
	ResponseData  interface{}            `json:"response_data,omitempty"`
}

// RequestLogger 请求日志中间件
func RequestLogger(config LoggerConfig) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()

		// 生成请求ID
		requestID := string(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = uuid.New().String()
			c.Response.Header.Set("X-Request-ID", requestID)
		}

		// 获取用户信息
		userID := string(c.GetHeader("X-User-ID"))
		userName := string(c.GetHeader("X-User-Name"))

		// 处理请求
		c.Next(ctx)

		// 记录日志
		latency := time.Since(start)
		method := string(c.Method())
		path := string(c.URI().Path())
		status := c.Response.StatusCode()
		clientIP := c.ClientIP()

		// 确定操作类型
		operationType := getOperationType(method, path)

		// 创建日志记录
		log := OperationLog{
			Timestamp:     start,
			ServiceName:   config.ServiceName,
			RequestID:     requestID,
			OperationType: operationType,
			Method:        method,
			Path:          path,
			Status:        status,
			Latency:       latency,
			ClientIP:      clientIP,
			Success:       status >= 200 && status < 400,
			UserID:        userID,
			UserName:      userName,
		}

		// 序列化日志
		logJSON, _ := json.Marshal(log)
		fmt.Printf("%s\n", string(logJSON))
	}
}

// getOperationType 根据方法和路径确定操作类型
func getOperationType(method, path string) string {
	switch method {
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	case "GET":
		return "query"
	default:
		return "unknown"
	}
}
