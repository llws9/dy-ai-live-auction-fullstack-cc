package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"auction-service/dao"
	"auction-service/model"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGenerateTokenIncludesRoleClaim(t *testing.T) {
	h := &AuthHandler{
		jwtSecret: "test-secret",
		jwtExpire: 24,
	}

	tokenString, err := h.generateToken(999, "admin", 2)
	require.NoError(t, err)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	require.Equal(t, float64(999), claims["user_id"])
	require.Equal(t, "admin", claims["username"])
	require.Equal(t, float64(2), claims["role"])
}

func TestGetCurrentUserReturnsUnauthorizedWhenTokenUserMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))

	authHandler := NewAuthHandler(dao.NewUserDAO(db), "test-secret", 24)
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	h.GET("/api/v1/users/me", func(ctx context.Context, c *app.RequestContext) {
		c.Set("user_id", int64(404))
		authHandler.GetCurrentUser(ctx, c)
	})

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/users/me", nil)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, http.StatusUnauthorized, resp.Code)
	require.Equal(t, "登录已失效，请重新登录", resp.Message)
}
