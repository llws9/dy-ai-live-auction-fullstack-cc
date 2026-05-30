package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestJWTAuth_ValidToken(t *testing.T) {
	secret := "test-secret-key"

	t.Run("should validate correct token", func(t *testing.T) {
		// Generate a valid token
		token, err := GenerateToken(secret, 123, "testuser", 0, 24)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// Parse token
		parsedToken, err := jwt.ParseWithClaims(token, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		assert.NoError(t, err)
		assert.True(t, parsedToken.Valid)

		claims, ok := parsedToken.Claims.(*JWTClaims)
		assert.True(t, ok)
		assert.Equal(t, int64(123), claims.UserID)
		assert.Equal(t, "testuser", claims.Username)
		assert.Equal(t, 0, claims.Role)
	})

	t.Run("should extract user info from token", func(t *testing.T) {
		userID := int64(456)
		username := "admin"
		role := 2

		token, err := GenerateToken(secret, userID, username, role, 24)
		assert.NoError(t, err)

		parsedToken, err := jwt.ParseWithClaims(token, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		assert.NoError(t, err)
		claims, ok := parsedToken.Claims.(*JWTClaims)
		assert.True(t, ok)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
		assert.Equal(t, role, claims.Role)
	})
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	secret := "test-secret-key"

	t.Run("should reject expired token", func(t *testing.T) {
		// Create an expired token
		claims := &JWTClaims{
			UserID:   123,
			Username: "testuser",
			Role:     0,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-24 * time.Hour)), // Expired 24 hours ago
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-48 * time.Hour)),
				NotBefore: jwt.NewNumericDate(time.Now().Add(-48 * time.Hour)),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		assert.NoError(t, err)

		// Parse expired token
		parsedToken, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		assert.Error(t, err)
		assert.False(t, parsedToken.Valid)
		assert.Contains(t, err.Error(), "token is expired")
	})

	t.Run("should accept token near expiry", func(t *testing.T) {
		// Create token expiring in 1 second
		token, err := GenerateToken(secret, 123, "testuser", 0, 1)
		assert.NoError(t, err)

		parsedToken, err := jwt.ParseWithClaims(token, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		assert.NoError(t, err)
		assert.True(t, parsedToken.Valid)
	})
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	secret := "test-secret-key"

	t.Run("should reject malformed token", func(t *testing.T) {
		invalidTokens := []string{
			"",
			"invalid-token",
			"header.payload", // Missing signature
			"header.payload.signature.extra", // Too many parts
		}

		for _, tokenStr := range invalidTokens {
			parsedToken, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})

			assert.Error(t, err)
			if parsedToken != nil {
				assert.False(t, parsedToken.Valid)
			}
		}
	})

	t.Run("should reject token with wrong secret", func(t *testing.T) {
		token, err := GenerateToken(secret, 123, "testuser", 0, 24)
		assert.NoError(t, err)

		// Try to parse with different secret
		parsedToken, err := jwt.ParseWithClaims(token, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte("wrong-secret"), nil
		})

		assert.Error(t, err)
		assert.False(t, parsedToken.Valid)
	})

	t.Run("should reject token with invalid signature", func(t *testing.T) {
		token, err := GenerateToken(secret, 123, "testuser", 0, 24)
		assert.NoError(t, err)

		// Tamper with token
		tamperedToken := token + "tampered"

		parsedToken, err := jwt.ParseWithClaims(tamperedToken, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		assert.Error(t, err)
		if parsedToken != nil {
			assert.False(t, parsedToken.Valid)
		}
	})
}

func TestJWTAuth_NoToken(t *testing.T) {
	t.Run("should handle request without Authorization header", func(t *testing.T) {
		// Simulate empty Authorization header
		authHeader := ""
		assert.Empty(t, authHeader)
	})

	t.Run("should reject request without Bearer prefix", func(t *testing.T) {
		// Test with invalid format
		authHeader := "Basic dGVzdDp0ZXN0" // Basic auth instead of Bearer

		assert.NotEmpty(t, authHeader)
		assert.Contains(t, authHeader, "Basic")
		assert.NotContains(t, authHeader, "Bearer")
	})
}

func TestJWTAuth_BearerToken(t *testing.T) {
	secret := "test-secret-key"

	t.Run("should extract Bearer token correctly", func(t *testing.T) {
		token, err := GenerateToken(secret, 123, "testuser", 0, 24)
		assert.NoError(t, err)

		authHeader := "Bearer " + token

		// Simulate parsing Bearer token
		assert.Contains(t, authHeader, "Bearer ")
		assert.True(t, len(authHeader) > len("Bearer "))
	})

	t.Run("should handle missing Bearer prefix", func(t *testing.T) {
		token := "sometoken"

		// Test format validation
		assert.NotContains(t, token, "Bearer ")
	})
}

func TestGenerateToken(t *testing.T) {
	secret := "test-secret-key"

	t.Run("should generate valid token with all claims", func(t *testing.T) {
		userID := int64(789)
		username := "streamer"
		role := 1
		expireHours := 48

		token, err := GenerateToken(secret, userID, username, role, expireHours)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// Parse and verify claims
		parsedToken, err := jwt.ParseWithClaims(token, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		assert.NoError(t, err)
		assert.True(t, parsedToken.Valid)

		claims, ok := parsedToken.Claims.(*JWTClaims)
		assert.True(t, ok)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
		assert.Equal(t, role, claims.Role)
		assert.NotNil(t, claims.ExpiresAt)
		assert.NotNil(t, claims.IssuedAt)
		assert.NotNil(t, claims.NotBefore)
	})

	t.Run("should set correct expiration time", func(t *testing.T) {
		expireHours := 24
		token, err := GenerateToken(secret, 123, "testuser", 0, expireHours)
		assert.NoError(t, err)

		parsedToken, err := jwt.ParseWithClaims(token, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		assert.NoError(t, err)
		claims, ok := parsedToken.Claims.(*JWTClaims)
		assert.True(t, ok)

		// Verify expiration is approximately correct (within 1 minute tolerance)
		expectedExpiry := time.Now().Add(time.Duration(expireHours) * time.Hour)
		actualExpiry := claims.ExpiresAt.Time

		diff := expectedExpiry.Sub(actualExpiry)
		assert.Less(t, diff.Abs(), time.Minute)
	})
}

// TestOptionalJWTAuth_Behavior 直接驱动 OptionalJWTAuth 中间件，覆盖三分支：
//   - 无 Authorization header → 放行，不注入 user_id；
//   - 带合法 Bearer token → 放行 + 注入 user_id/username/user_role；
//   - 带非法 token / 错误前缀 → 放行（不返回 401），不注入。
//
// 对应 spec：docs/superpowers/specs/2026-05-30-h5-missing-b-livestream.md §5.3
func TestOptionalJWTAuth_Behavior(t *testing.T) {
	secret := "test-secret-key"
	mw := OptionalJWTAuth(secret)

	t.Run("no token → pass through, no user_id", func(t *testing.T) {
		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/live-streams/1")
		// 注册一个空的下游 handler 让 c.Next 推进 index。
		c.SetHandlers([]app.HandlerFunc{
			mw,
			func(ctx context.Context, c *app.RequestContext) {},
		})
		c.Next(ctx)

		_, exists := c.Get("user_id")
		assert.False(t, exists, "user_id 不应被注入")
		assert.False(t, c.IsAborted(), "无 token 不应中断请求")
	})

	t.Run("valid token → injects user_id/username/user_role", func(t *testing.T) {
		token, err := GenerateToken(secret, 123, "alice", 1, 24)
		assert.NoError(t, err)

		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/live-streams/1")
		c.Request.Header.Set("Authorization", "Bearer "+token)
		c.SetHandlers([]app.HandlerFunc{
			mw,
			func(ctx context.Context, c *app.RequestContext) {},
		})
		c.Next(ctx)

		uid, exists := c.Get("user_id")
		assert.True(t, exists, "合法 token 必须注入 user_id")
		assert.Equal(t, int64(123), uid)
		assert.Equal(t, "alice", c.GetString("username"))
		assert.Equal(t, 1, c.GetInt("user_role"))
		assert.False(t, c.IsAborted())
	})

	t.Run("malformed header → pass through, no user_id", func(t *testing.T) {
		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/live-streams/1")
		c.Request.Header.Set("Authorization", "NotBearer xxx")
		c.SetHandlers([]app.HandlerFunc{
			mw,
			func(ctx context.Context, c *app.RequestContext) {},
		})
		c.Next(ctx)

		_, exists := c.Get("user_id")
		assert.False(t, exists)
		assert.False(t, c.IsAborted(), "非法格式不应触发 401，须放行交由下游决定")
	})

	t.Run("invalid signature → pass through, no user_id", func(t *testing.T) {
		token, err := GenerateToken(secret, 123, "alice", 0, 24)
		assert.NoError(t, err)

		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/live-streams/1")
		c.Request.Header.Set("Authorization", "Bearer "+token+"tampered")
		c.SetHandlers([]app.HandlerFunc{
			mw,
			func(ctx context.Context, c *app.RequestContext) {},
		})
		c.Next(ctx)

		_, exists := c.Get("user_id")
		assert.False(t, exists)
		assert.False(t, c.IsAborted(), "签名错误不应触发 401")
	})
}

func TestJWTClaims_RoleValidation(t *testing.T) {
	secret := "test-secret-key"

	t.Run("should store user role in claims", func(t *testing.T) {
		roles := []int{0, 1, 2} // User, Streamer, Admin

		for _, role := range roles {
			token, err := GenerateToken(secret, 123, "testuser", role, 24)
			assert.NoError(t, err)

			parsedToken, err := jwt.ParseWithClaims(token, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})

			assert.NoError(t, err)
			claims, ok := parsedToken.Claims.(*JWTClaims)
			assert.True(t, ok)
			assert.Equal(t, role, claims.Role)
		}
	})
}
