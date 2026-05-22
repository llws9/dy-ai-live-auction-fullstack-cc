package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT 声明
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     int    `json:"role"` // 用户角色: 0=普通用户, 1=主播, 2=管理员
	jwt.RegisteredClaims
}

// JWTConfig JWT 中间件配置
type JWTConfig struct {
	Secret     string
	ExpireTime int // 小时
}

// JWTAuth JWT 认证中间件
func JWTAuth(secret string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 从 Header 获取 Token
		authHeader := string(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.JSON(401, map[string]interface{}{
				"code":    401,
				"message": "未提供认证令牌",
			})
			c.Abort()
			return
		}

		// 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(401, map[string]interface{}{
				"code":    401,
				"message": "认证令牌格式错误",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 解析 Token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, map[string]interface{}{
				"code":    401,
				"message": "认证令牌无效或已过期",
			})
			c.Abort()
			return
		}

		// 提取用户信息
		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			c.JSON(401, map[string]interface{}{
				"code":    401,
				"message": "认证令牌解析失败",
			})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_role", claims.Role)

		c.Next(ctx)
	}
}

// OptionalJWTAuth 可选的 JWT 认证中间件（不强制要求认证）
func OptionalJWTAuth(secret string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		authHeader := string(c.GetHeader("Authorization"))
		if authHeader == "" {
			// 没有提供 Token，继续处理但不设置用户信息
			c.Next(ctx)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next(ctx)
			return
		}

		tokenString := parts[1]
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.Next(ctx)
			return
		}

		if claims, ok := token.Claims.(*JWTClaims); ok {
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("user_role", claims.Role)
		}

		c.Next(ctx)
	}
}

// GenerateToken 生成 JWT Token
func GenerateToken(secret string, userID int64, username string, role int, expireHours int) (string, error) {
	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
