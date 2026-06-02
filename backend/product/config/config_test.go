package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad_LLM_DefaultsAndEnvOverride(t *testing.T) {
	t.Setenv("ARK_API_KEY", "secret-from-env")

	cfg := Load()

	if cfg.LLM.Doubao.APIKey != "secret-from-env" {
		t.Fatalf("want APIKey overridden by env, got %q", cfg.LLM.Doubao.APIKey)
	}
	if cfg.LLM.Provider != "doubao" {
		t.Fatalf("default provider want=doubao got=%q", cfg.LLM.Provider)
	}
	if cfg.LLM.TimeoutMs <= 0 {
		t.Fatalf("default TimeoutMs must be >0, got %d", cfg.LLM.TimeoutMs)
	}
	if !strings.HasPrefix(cfg.LLM.Doubao.BaseURL, "https://ark.cn-beijing.volces.com") {
		t.Fatalf("default BaseURL unexpected: %q", cfg.LLM.Doubao.BaseURL)
	}
}

func TestLoadFromYAML_LLM_PlaceholderResolved(t *testing.T) {
	_ = os.Setenv("ARK_API_KEY", "yaml-env-key")
	defer os.Unsetenv("ARK_API_KEY")
	yaml := `
llm:
  provider: doubao
  timeout_ms: 5000
  doubao:
    base_url: https://ark.cn-beijing.volces.com/api/v3
    api_key: ${ARK_API_KEY}
    model: doubao-1.5-vision-pro
`
	cfg, err := LoadFromYAML(yaml)
	if err != nil {
		t.Fatalf("LoadFromYAML err: %v", err)
	}

	ResolveLLMSecrets(cfg)

	if cfg.LLM.Doubao.APIKey != "yaml-env-key" {
		t.Fatalf("placeholder must be resolved from env, got %q", cfg.LLM.Doubao.APIKey)
	}
}
