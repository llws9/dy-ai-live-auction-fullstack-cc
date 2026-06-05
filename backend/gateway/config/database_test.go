package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadIncludesDatabaseConfigFromEnvironment(t *testing.T) {
	t.Setenv("DB_HOST", "db.local")
	t.Setenv("DB_PORT", "3307")
	t.Setenv("DB_USER", "metrics")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "auction_metrics")

	cfg := Load()

	assert.Equal(t, "db.local", cfg.Database.Host)
	assert.Equal(t, "3307", cfg.Database.Port)
	assert.Equal(t, "metrics", cfg.Database.User)
	assert.Equal(t, "secret", cfg.Database.Password)
	assert.Equal(t, "auction_metrics", cfg.Database.Name)
}
