package handler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewLiveStartHandlerUsesBFFTimeout(t *testing.T) {
	h := NewLiveStartHandler("http://product:8081", "http://auction:8082", "internal-token")

	require.NotNil(t, h.client)
	require.Equal(t, 2*time.Second, h.client.Timeout)
}
