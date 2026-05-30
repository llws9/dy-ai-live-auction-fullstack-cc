package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"test-service/ws"
)

// 通过 httptest 起 server，订阅一个 testID，验证 broker.Publish 后能收到 JSON 消息
func TestWSHandler_PushOnPublish(t *testing.T) {
	b := ws.NewBroker(0) // 不节流
	h := NewWSHandler(b)

	srv := httptest.NewServer(http.HandlerFunc(h.HandleProgress))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/?test_id=abc"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer c.Close()

	// 先等到订阅注册（broker.Subscribe 在 handler 内 upgrade 之后才会执行）
	require.Eventually(t, func() bool {
		// 通过尝试发布并设置一个非常短的 read deadline 来探测；
		// 简化起见：等 50ms 让 handler 走到 Subscribe
		time.Sleep(50 * time.Millisecond)
		return true
	}, time.Second, 10*time.Millisecond)

	// publish
	b.Publish("abc", ws.Message{Progress: 42, Step: "go"})

	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := c.ReadMessage()
	require.NoError(t, err)

	var got ws.Message
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, 42, got.Progress)
	assert.Equal(t, "abc", got.TestID)
	assert.Equal(t, "go", got.Step)
}

// 缺少 test_id 应 400
func TestWSHandler_MissingTestID(t *testing.T) {
	b := ws.NewBroker(0)
	h := NewWSHandler(b)
	srv := httptest.NewServer(http.HandlerFunc(h.HandleProgress))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
