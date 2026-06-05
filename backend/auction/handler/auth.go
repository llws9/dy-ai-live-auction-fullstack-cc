package handler

import (
	"context"
	"errors"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"auction-service/dao"
	"auction-service/model"
)

// AuthHandler 认证 Handler
type AuthHandler struct {
	userDAO   *dao.UserDAO
	jwtSecret string
	jwtExpire int
}

// NewAuthHandler 创建认证 Handler
func NewAuthHandler(userDAO *dao.UserDAO, jwtSecret string, jwtExpire int) *AuthHandler {
	return &AuthHandler{
		userDAO:   userDAO,
		jwtSecret: jwtSecret,
		jwtExpire: jwtExpire,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Name     string  `json:"name"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	Password string  `json:"password"`
	Avatar   string  `json:"avatar"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

// Register 用户注册
func (h *AuthHandler) Register(ctx context.Context, c *app.RequestContext) {
	var req RegisterRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 验证必填字段
	if req.Name == "" {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "用户名不能为空",
		})
		return
	}

	if req.Password == "" || len(req.Password) < 6 {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "密码长度至少6位",
		})
		return
	}

	// 验证至少提供邮箱或手机号之一
	if req.Email == nil && req.Phone == nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "至少需要提供邮箱或手机号",
		})
		return
	}

	// 检查邮箱是否已存在
	if req.Email != nil {
		exists, _ := h.userDAO.GetByEmail(ctx, *req.Email)
		if exists != nil {
			c.JSON(400, map[string]interface{}{
				"code":    400,
				"message": "邮箱已被注册",
			})
			return
		}
	}

	// 检查手机号是否已存在
	if req.Phone != nil {
		exists, _ := h.userDAO.GetByPhone(ctx, *req.Phone)
		if exists != nil {
			c.JSON(400, map[string]interface{}{
				"code":    400,
				"message": "手机号已被注册",
			})
			return
		}
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "密码加密失败",
		})
		return
	}

	// 创建用户
	user := &model.User{
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		Password: string(hashedPassword),
		Avatar:   req.Avatar,
		Role:     0, // 默认普通用户
		Status:   1, // 默认激活
	}

	if err := h.userDAO.Create(ctx, user); err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "注册失败: " + err.Error(),
		})
		return
	}

	// 生成Token
	token, err := h.generateToken(user.ID, user.Name, user.Role)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "生成Token失败",
		})
		return
	}

	c.JSON(201, map[string]interface{}{
		"code":    201,
		"message": "注册成功",
		"data": LoginResponse{
			Token: token,
			User:  user,
		},
	})
}

// Login 用户登录
func (h *AuthHandler) Login(ctx context.Context, c *app.RequestContext) {
	var req LoginRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 根据邮箱或手机号查找用户
	var user *model.User
	var err error

	if req.Email != "" {
		user, err = h.userDAO.GetByEmail(ctx, req.Email)
	} else if req.Phone != "" {
		user, err = h.userDAO.GetByPhone(ctx, req.Phone)
	} else {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "请提供邮箱或手机号",
		})
		return
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(401, map[string]interface{}{
				"code":    401,
				"message": "用户不存在",
			})
		} else {
			c.JSON(500, map[string]interface{}{
				"code":    500,
				"message": "查询用户失败",
			})
		}
		return
	}

	// 检查用户状态
	if !user.IsActive() {
		c.JSON(403, map[string]interface{}{
			"code":    403,
			"message": "账号已被禁用",
		})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "密码错误",
		})
		return
	}

	// 更新最后登录时间
	_ = h.userDAO.UpdateLastLogin(ctx, user.ID)

	// 生成Token
	token, err := h.generateToken(user.ID, user.Name, user.Role)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "生成Token失败",
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "登录成功",
		"data": LoginResponse{
			Token: token,
			User:  user,
		},
	})
}

// GetCurrentUser 获取当前用户信息
func (h *AuthHandler) GetCurrentUser(ctx context.Context, c *app.RequestContext) {
	// 从上下文获取用户ID（由JWT中间件设置）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "未认证",
		})
		return
	}

	user, err := h.userDAO.GetByID(ctx, userID.(int64))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(401, map[string]interface{}{
				"code":    401,
				"message": "登录已失效，请重新登录",
			})
		} else {
			c.JSON(500, map[string]interface{}{
				"code":    500,
				"message": "查询用户失败",
			})
		}
		return
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": user,
	})
}

// generateToken 生成JWT Token
func (h *AuthHandler) generateToken(userID int64, username string, role int) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(time.Duration(h.jwtExpire) * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
		"nbf":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}
