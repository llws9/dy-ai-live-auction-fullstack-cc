package nacos

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetServiceConfigInfoDefaultsToProductConfig(t *testing.T) {
	t.Setenv("NACOS_GROUP", "")
	t.Setenv("NACOS_DATA_ID", "")

	group, dataID := GetServiceConfigInfo()

	require.Equal(t, "default", group)
	require.Equal(t, "product-config.yaml", dataID)
}
