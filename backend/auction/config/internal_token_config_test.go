package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNacosAuctionConfigLoadsInternalToken(t *testing.T) {
	configBytes, err := os.ReadFile("../../../configs/nacos/auction-config.yaml")
	require.NoError(t, err)

	var cfg Config
	require.NoError(t, yaml.Unmarshal(configBytes, &cfg))
	require.Equal(t, "dev-internal-api-token", cfg.Internal.Token)
}
