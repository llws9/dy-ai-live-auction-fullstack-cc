package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNacosAuctionConfigDoesNotContainUsablePlaintextInternalToken(t *testing.T) {
	forbiddenToken := "dev-" + "internal-" + "api-" + "token"
	configBytes, err := os.ReadFile("../../../configs/nacos/auction-config.yaml")
	require.NoError(t, err)

	var cfg Config
	require.NoError(t, yaml.Unmarshal(configBytes, &cfg))
	require.Empty(t, cfg.Internal.Token)
	require.NotContains(t, string(configBytes), forbiddenToken)
}
