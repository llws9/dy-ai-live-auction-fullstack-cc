package handler

import "github.com/cloudwego/hertz/pkg/app"

const adminRoleHeader = "admin"

func requireAdminRole(c *app.RequestContext) bool {
	if string(c.GetHeader("X-User-Role")) == adminRoleHeader {
		return true
	}

	c.JSON(403, map[string]interface{}{
		"code":    403,
		"message": "权限不足：需要管理员权限",
	})
	return false
}
