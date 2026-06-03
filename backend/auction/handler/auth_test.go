package handler

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
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
