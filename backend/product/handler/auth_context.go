package handler

import (
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	roleAdmin    = "admin"
	roleMerchant = "merchant"
)

type AdminActor struct {
	UserID int64
	Role   string
}

func (a AdminActor) IsAdmin() bool {
	return a.Role == roleAdmin
}

func (a AdminActor) IsMerchant() bool {
	return a.Role == roleMerchant
}

func readAdminActor(c *app.RequestContext) (AdminActor, bool) {
	role := string(c.GetHeader("X-User-Role"))
	userIDRaw := string(c.GetHeader("X-User-ID"))
	userID, err := strconv.ParseInt(userIDRaw, 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
		return AdminActor{}, false
	}
	if role != roleAdmin && role != roleMerchant {
		c.JSON(403, map[string]interface{}{"code": 403, "message": "权限不足"})
		return AdminActor{}, false
	}
	return AdminActor{UserID: userID, Role: role}, true
}

func requireMerchantActor(c *app.RequestContext) (AdminActor, bool) {
	actor, ok := readAdminActor(c)
	if !ok {
		return AdminActor{}, false
	}
	if !actor.IsMerchant() {
		c.JSON(403, map[string]interface{}{"code": 403, "message": "平台管理员不具备代运营权限"})
		return AdminActor{}, false
	}
	return actor, true
}

func requireAdminActor(c *app.RequestContext) (AdminActor, bool) {
	actor, ok := readAdminActor(c)
	if !ok {
		return AdminActor{}, false
	}
	if !actor.IsAdmin() {
		c.JSON(403, map[string]interface{}{"code": 403, "message": "权限不足：需要管理员权限"})
		return AdminActor{}, false
	}
	return actor, true
}
