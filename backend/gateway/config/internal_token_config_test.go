package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDefaultConfigsDoNotContainUsablePlaintextInternalToken(t *testing.T) {
	forbiddenToken := "dev-" + "internal-" + "api-" + "token"
	gatewayConfigBytes, err := os.ReadFile("../../../configs/nacos/gateway-config.yaml")
	require.NoError(t, err)
	var gatewayCfg Config
	require.NoError(t, yaml.Unmarshal(gatewayConfigBytes, &gatewayCfg))
	require.Empty(t, gatewayCfg.Services.InternalToken)

	auctionConfigBytes, err := os.ReadFile("../../../configs/nacos/auction-config.yaml")
	require.NoError(t, err)
	require.NotContains(t, string(auctionConfigBytes), forbiddenToken)

	dockerComposeBytes, err := os.ReadFile("../../../docker-compose.yml")
	require.NoError(t, err)
	dockerCompose := string(dockerComposeBytes)
	require.NotContains(t, dockerCompose, forbiddenToken)
	require.Contains(t, dockerCompose, "INTERNAL_API_TOKEN=${INTERNAL_API_TOKEN:?set INTERNAL_API_TOKEN}")
	require.GreaterOrEqual(t, strings.Count(dockerCompose, "INTERNAL_API_TOKEN=${INTERNAL_API_TOKEN:?set INTERNAL_API_TOKEN}"), 4)
}

func TestInjectRuntimeSecretsLoadsInternalTokenFromEnvironment(t *testing.T) {
	t.Setenv("INTERNAL_API_TOKEN", "runtime-secret")

	cfg := &Config{}
	injectRuntimeSecrets(cfg)

	require.Equal(t, "runtime-secret", cfg.Services.InternalToken)
}

func TestInjectRuntimeSecretsBackfillsEmptyNacosRuntimeFieldsFromEnvironment(t *testing.T) {
	t.Setenv("GATEWAY_PORT", ":8080")
	t.Setenv("PRODUCT_SERVICE_URL", "http://product:8081")
	t.Setenv("AUCTION_SERVICE_URL", "http://auction:8082")
	t.Setenv("TEST_SERVICE_URL", "http://test-service:18090")
	t.Setenv("TEST_SERVICE_WS_URL", "ws://localhost:18092")
	t.Setenv("JWT_SECRET", "runtime-jwt-secret")

	cfg := &Config{}
	injectRuntimeSecrets(cfg)

	require.Equal(t, ":8080", cfg.Server.Port)
	require.Equal(t, "http://product:8081", cfg.Services.ProductURL)
	require.Equal(t, "http://auction:8082", cfg.Services.AuctionURL)
	require.Equal(t, "http://test-service:18090", cfg.Services.TestURL)
	require.Equal(t, "ws://localhost:18092", cfg.Services.TestWSURL)
	require.Equal(t, "runtime-jwt-secret", cfg.JWT.Secret)
}
