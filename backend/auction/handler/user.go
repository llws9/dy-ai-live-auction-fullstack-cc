package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/dao"
	"auction-service/model"
)

// UserHandler 用户 Handler
type UserHandler struct {
	userDAO *dao.UserDAO
}

// NewUserHandler 创建用户 Handler
func NewUserHandler(userDAO *dao.UserDAO) *UserHandler {
	return &UserHandler{
		userDAO: userDAO,
	}
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	ID     int64  `json:"id"`
	Name   string `json:"name" binding:"required"`
	Avatar string `json:"avatar"`
}

// CreateUser 创建用户
func (h *UserHandler) CreateUser(ctx context.Context, c *app.RequestContext) {
	var req CreateUserRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	user := &model.User{
		Name:   req.Name,
		Avatar: req.Avatar,
	}

	// 如果指定了 ID，设置 ID（用于测试）
	if req.ID > 0 {
		user.ID = req.ID
	}

	if err := h.userDAO.CreateIfNotExists(ctx, user); err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "创建用户失败: " + err.Error(),
		})
		return
	}

	c.JSON(201, map[string]interface{}{
		"code":    201,
		"message": "用户创建成功",
		"data":    user,
	})
}

// BatchCreateUsersRequest 批量创建用户请求
type BatchCreateUsersRequest struct {
	StartID int64 `json:"start_id" binding:"required"`
	Count   int   `json:"count" binding:"required,gt=0"`
}

// BatchCreateUsers 批量创建测试用户
func (h *UserHandler) BatchCreateUsers(ctx context.Context, c *app.RequestContext) {
	var req BatchCreateUsersRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 限制批量创建数量
	if req.Count > 1000 {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "单次最多创建1000个用户",
		})
		return
	}

	users := make([]*model.User, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		users = append(users, &model.User{
			ID:   req.StartID + int64(i),
			Name: "测试用户",
		})
	}

	created := 0
	for _, user := range users {
		if err := h.userDAO.CreateIfNotExists(ctx, user); err == nil {
			created++
		}
	}

	c.JSON(201, map[string]interface{}{
		"code":    201,
		"message": "批量创建用户成功",
		"data": map[string]interface{}{
			"total_requested": req.Count,
			"total_created":   created,
			"start_id":        req.StartID,
			"end_id":          req.StartID + int64(req.Count-1),
		},
	})
}
