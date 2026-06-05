package config

import "testing"

func TestDefaultJWTSecretMatchesGatewayLocalDefault(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Security.JWTSecret != "your-secret-key-change-in-production" {
		t.Fatalf("JWTSecret default must match gateway local default, got %q", cfg.Security.JWTSecret)
	}
	if cfg.Security.InternalToken != "" {
		t.Fatalf("InternalToken default must stay empty for fail-closed internal auth, got %q", cfg.Security.InternalToken)
	}
}

func TestLoadFromEnvReadsSecurityConfig(t *testing.T) {
	t.Setenv("JWT_SECRET", "jwt-secret")
	t.Setenv("INTERNAL_API_TOKEN", "internal-secret")

	cfg := LoadFromEnv()
	if cfg.Security.JWTSecret != "jwt-secret" {
		t.Fatalf("JWTSecret: want jwt-secret, got %q", cfg.Security.JWTSecret)
	}
	if cfg.Security.InternalToken != "internal-secret" {
		t.Fatalf("InternalToken: want internal-secret, got %q", cfg.Security.InternalToken)
	}
}
