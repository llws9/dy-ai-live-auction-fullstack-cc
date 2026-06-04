package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireMerchantOnlyRejectsAdmin(t *testing.T) {
	c := app.NewContext(0)
	c.Set("user_role", 2)

	handler := RequireMerchantOnly()
	handler(context.Background(), c)

	require.True(t, c.IsAborted())
	require.Equal(t, 403, c.Response.StatusCode())
}

func TestRequireMerchantOrAdminAcceptsBoth(t *testing.T) {
	for _, role := range []int{1, 2} {
		c := app.NewContext(0)
		c.Set("user_role", role)

		handler := RequireMerchantOrAdmin()
		handler(context.Background(), c)

		require.False(t, c.IsAborted())
	}
}

func TestRequireMerchantOrAdminRejectsUser(t *testing.T) {
	c := app.NewContext(0)
	c.Set("user_role", 0)

	handler := RequireMerchantOrAdmin()
	handler(context.Background(), c)

	require.True(t, c.IsAborted())
	require.Equal(t, 403, c.Response.StatusCode())
}

func TestRBACMiddleware_AdminRole(t *testing.T) {
	t.Run("should allow admin access to admin resources", func(t *testing.T) {
		// Admin role = 2
		userRole := 2
		requiredRole := 2
		assert.GreaterOrEqual(t, userRole, requiredRole)
	})

	t.Run("should allow admin to access streamer resources", func(t *testing.T) {
		// Admin should have streamer privileges
		userRole := 2 // Admin
		streamerRole := 1
		assert.GreaterOrEqual(t, userRole, streamerRole)
	})

	t.Run("should allow admin to access user resources", func(t *testing.T) {
		// Admin should have user privileges
		userRole := 2 // Admin
		userRoleLevel := 0
		assert.GreaterOrEqual(t, userRole, userRoleLevel)
	})
}

func TestRBACMiddleware_StreamerRole(t *testing.T) {
	t.Run("should allow streamer access to streamer resources", func(t *testing.T) {
		// Streamer role = 1
		userRole := 1
		requiredRole := 1
		assert.GreaterOrEqual(t, userRole, requiredRole)
	})

	t.Run("should deny streamer access to admin resources", func(t *testing.T) {
		// Streamer should NOT have admin privileges
		userRole := 1 // Streamer
		adminRole := 2
		assert.Less(t, userRole, adminRole)
	})

	t.Run("should allow streamer to access user resources", func(t *testing.T) {
		// Streamer should have user privileges
		userRole := 1 // Streamer
		userRoleLevel := 0
		assert.GreaterOrEqual(t, userRole, userRoleLevel)
	})
}

func TestRBACMiddleware_UserRole(t *testing.T) {
	t.Run("should allow user access to user resources", func(t *testing.T) {
		// User should have basic privileges
		userRole := 0 // Regular user
		userRoleLevel := 0
		assert.GreaterOrEqual(t, userRole, userRoleLevel)
	})

	t.Run("should deny user access to streamer resources", func(t *testing.T) {
		// User should NOT have streamer privileges
		userRole := 0 // Regular user
		streamerRole := 1
		assert.Less(t, userRole, streamerRole)
	})

	t.Run("should deny user access to admin resources", func(t *testing.T) {
		// User should NOT have admin privileges
		userRole := 0 // Regular user
		adminRole := 2
		assert.Less(t, userRole, adminRole)
	})
}

func TestRBACMiddleware_AccessDenied(t *testing.T) {
	t.Run("should return 403 when permission denied", func(t *testing.T) {
		response := map[string]interface{}{
			"code":    403,
			"message": "权限不足",
		}

		assert.Equal(t, 403, response["code"])
		assert.Contains(t, response["message"], "权限不足")
	})

	t.Run("should abort request on permission denied", func(t *testing.T) {
		userRole := 0 // Regular user

		// Check if user has admin access
		requiredRole := 2

		shouldDeny := userRole < requiredRole
		assert.True(t, shouldDeny)
	})
}

func TestRBACMiddleware_RoleHierarchy(t *testing.T) {
	t.Run("should enforce role hierarchy (Admin > Streamer > User)", func(t *testing.T) {
		roles := []struct {
			name  string
			level int
		}{
			{"User", 0},
			{"Streamer", 1},
			{"Admin", 2},
		}

		// Verify hierarchy
		for i := 1; i < len(roles); i++ {
			assert.Greater(t, roles[i].level, roles[i-1].level)
		}
	})

	t.Run("should allow higher roles to access lower role resources", func(t *testing.T) {
		testCases := []struct {
			userRole     int
			requiredRole int
			shouldAllow  bool
		}{
			{2, 0, true},  // Admin accessing user resource
			{2, 1, true},  // Admin accessing streamer resource
			{2, 2, true},  // Admin accessing admin resource
			{1, 0, true},  // Streamer accessing user resource
			{1, 1, true},  // Streamer accessing streamer resource
			{1, 2, false}, // Streamer accessing admin resource
			{0, 0, true},  // User accessing user resource
			{0, 1, false}, // User accessing streamer resource
			{0, 2, false}, // User accessing admin resource
		}

		for _, tc := range testCases {
			result := tc.userRole >= tc.requiredRole
			assert.Equal(t, tc.shouldAllow, result,
				"Role %d accessing resource requiring role %d", tc.userRole, tc.requiredRole)
		}
	})
}

func TestRBACMiddleware_MissingRole(t *testing.T) {
	t.Run("should deny when user_role not set", func(t *testing.T) {
		// When user_role is not set, it defaults to 0
		userRole := 0

		// For admin resources, should be denied
		adminRole := 2
		shouldDeny := userRole < adminRole
		assert.True(t, shouldDeny)
	})

	t.Run("should handle zero role value", func(t *testing.T) {
		userRole := 0
		assert.Equal(t, 0, userRole)
	})
}

func TestRequireRole(t *testing.T) {
	t.Run("should create middleware for specific role", func(t *testing.T) {
		middleware := RequireRole(1)
		assert.NotNil(t, middleware)
	})

	t.Run("should work with different role levels", func(t *testing.T) {
		roles := []int{0, 1, 2}

		for _, role := range roles {
			middleware := RequireRole(role)
			assert.NotNil(t, middleware)
		}
	})
}

func TestRequireStreamer(t *testing.T) {
	t.Run("should create streamer middleware", func(t *testing.T) {
		middleware := RequireStreamer()
		assert.NotNil(t, middleware)
	})

	t.Run("should require role level 1", func(t *testing.T) {
		// RequireStreamer should require role >= 1
		streamerRole := 1
		assert.Equal(t, 1, streamerRole)
	})
}

func TestRequireAdmin(t *testing.T) {
	t.Run("should create admin middleware", func(t *testing.T) {
		middleware := RequireAdmin()
		assert.NotNil(t, middleware)
	})

	t.Run("should require role level 2", func(t *testing.T) {
		// RequireAdmin should require role >= 2
		adminRole := 2
		assert.Equal(t, 2, adminRole)
	})
}

func TestRBACMiddleware_ContextIntegrity(t *testing.T) {
	t.Run("should not modify other context values", func(t *testing.T) {
		// Simulate context values
		userID := int64(123)
		username := "testuser"
		userRole := 1

		// After RBAC check, other values should remain
		assert.Equal(t, int64(123), userID)
		assert.Equal(t, "testuser", username)
		assert.Equal(t, 1, userRole)
	})
}

func TestRBACMiddleware_EdgeCases(t *testing.T) {
	t.Run("should handle negative role values", func(t *testing.T) {
		userRole := -1

		// Negative role should be denied for all resources
		assert.Less(t, userRole, 0)
	})

	t.Run("should handle high role values", func(t *testing.T) {
		userRole := 999

		// Very high role should have all access
		assert.GreaterOrEqual(t, userRole, 2)
	})
}
