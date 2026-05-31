package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDefaultConfigsIncludeSharedInternalToken(t *testing.T) {
	gatewayConfigBytes, err := os.ReadFile("../../../configs/nacos/gateway-config.yaml")
	require.NoError(t, err)
	var gatewayCfg Config
	require.NoError(t, yaml.Unmarshal(gatewayConfigBytes, &gatewayCfg))
	require.NotEmpty(t, gatewayCfg.Services.InternalToken)

	auctionConfigBytes, err := os.ReadFile("../../../configs/nacos/auction-config.yaml")
	require.NoError(t, err)
	require.Contains(t, string(auctionConfigBytes), "token: \""+gatewayCfg.Services.InternalToken+"\"")

	dockerComposeBytes, err := os.ReadFile("../../../docker-compose.yml")
	require.NoError(t, err)
	dockerCompose := string(dockerComposeBytes)
	require.Contains(t, dockerCompose, "INTERNAL_API_TOKEN="+gatewayCfg.Services.InternalToken)
	require.Equal(t, 2, strings.Count(dockerCompose, "INTERNAL_API_TOKEN="+gatewayCfg.Services.InternalToken))
}
