package handler

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/require"
)

func TestReadAdminActorMerchant(t *testing.T) {
	c := app.NewContext(0)
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")

	actor, ok := readAdminActor(c)

	require.True(t, ok)
	require.Equal(t, int64(1001), actor.UserID)
	require.Equal(t, "merchant", actor.Role)
	require.True(t, actor.IsMerchant())
	require.False(t, actor.IsAdmin())
}

func TestReadAdminActorRejectsMissingUserID(t *testing.T) {
	c := app.NewContext(0)
	c.Request.Header.Set("X-User-Role", "merchant")

	_, ok := readAdminActor(c)

	require.False(t, ok)
	require.Equal(t, 401, c.Response.StatusCode())
}

func TestReadAdminActorRejectsUnsupportedRole(t *testing.T) {
	c := app.NewContext(0)
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "user")

	_, ok := readAdminActor(c)

	require.False(t, ok)
	require.Equal(t, 403, c.Response.StatusCode())
}

func TestRequireMerchantActorRejectsAdmin(t *testing.T) {
	c := app.NewContext(0)
	c.Request.Header.Set("X-User-ID", "2001")
	c.Request.Header.Set("X-User-Role", "admin")

	_, ok := requireMerchantActor(c)

	require.False(t, ok)
	require.Equal(t, 403, c.Response.StatusCode())
}

func TestRequireAdminActorRejectsMerchant(t *testing.T) {
	c := app.NewContext(0)
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")

	_, ok := requireAdminActor(c)

	require.False(t, ok)
	require.Equal(t, 403, c.Response.StatusCode())
}
