// Package docs Swagger API文档
package docs

import (
	"github.com/cloudwego/hertz/pkg/app/server"
)

// SwaggerInfo API文档基本信息
var SwaggerInfo = struct {
	Version     string
	Title       string
	Description string
}{
	Version:     "1.0",
	Title:       "直播竞拍系统 API",
	Description: "直播竞拍系统后端API文档",
}

// Register 注册Swagger文档路由
// Swagger文档将由 swag init 生成，此文件提供占位符
func Register(h *server.Hertz) {
	// 占位符实现
	// 实际的Swagger路由将在router中配置
}
