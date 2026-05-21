package apitest

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadEnv parses the fixed apitest .env format without external YAML modules.
// Supported syntax is intentionally small: top-level `key: value`, optional
// leading `- ` for legacy single-item lists, and an indented `test_account:` map.
func LoadEnv(path string) (*EnvConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read env file %q: %w", path, err)
	}
	defer file.Close()

	cfg := &EnvConfig{TestAccount: map[string]string{}}
	section := ""
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		rawLine := scanner.Text()
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		}
		key, value, ok := splitEnvLine(trimmed)
		if !ok {
			return nil, fmt.Errorf("parse env file %q line %d: expected key: value", path, lineNo)
		}
		if key == "test_account" {
			section = "test_account"
			if cfg.TestAccount == nil {
				cfg.TestAccount = map[string]string{}
			}
			continue
		}
		if section == "test_account" && isIndented(rawLine) {
			cfg.TestAccount[key] = cleanEnvValue(value)
			continue
		}
		section = ""
		assignEnvField(cfg, key, cleanEnvValue(value))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read env file %q: %w", path, err)
	}
	applyEnvDefaults(cfg)
	return cfg, nil
}

func splitEnvLine(line string) (string, string, bool) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}

func isIndented(line string) bool {
	return strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
}

func cleanEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "{}" {
		return ""
	}
	if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
		return value[1 : len(value)-1]
	}
	return value
}

func assignEnvField(cfg *EnvConfig, key, value string) {
	switch key {
	case "psm":
		cfg.PSM = value
	case "host":
		cfg.Host = value
	case "env":
		cfg.Env = value
	case "branch":
		cfg.Branch = value
	case "zone":
		cfg.Zone = value
	case "idc":
		cfg.IDC = value
	case "cluster":
		cfg.Cluster = value
	}
}

func applyEnvDefaults(c *EnvConfig) {
	if c.Env == "" {
		c.Env = "prod"
	}
	if c.Cluster == "" {
		c.Cluster = "default"
	}
	if c.Branch == "" {
		c.Branch = "master"
	}
}

// Validate enforces the minimal field set required by the gateway client.
func (c *EnvConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("env config is nil")
	}
	if c.PSM == "" {
		return fmt.Errorf("env config: psm is required")
	}
	return nil
}
