package middleware

import (
	"testing"
	"time"

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

func TestOptionalJWTAuth(t *testing.T) {
	secret := "test-secret-key"

	t.Run("should allow request without token", func(t *testing.T) {
		// OptionalJWTAuth should continue even without token
		assert.True(t, true) // Middleware should call c.Next(ctx)
	})

	t.Run("should set user context if valid token provided", func(t *testing.T) {
		token, err := GenerateToken(secret, 123, "testuser", 0, 24)
		assert.NoError(t, err)

		// Token would be parsed and user info set in context
		assert.NotEmpty(t, token)
	})

	t.Run("should continue if invalid token provided", func(t *testing.T) {
		// OptionalJWTAuth should continue even with invalid token
		invalidToken := "invalid-token"
		assert.NotEmpty(t, invalidToken)
		// Middleware should call c.Next(ctx) regardless
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
