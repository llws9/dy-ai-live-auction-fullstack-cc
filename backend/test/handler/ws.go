package handler

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/gorilla/websocket"

	"test-service/ws"
)

// WSHandler 处理 /ws/test/progress 的 WebSocket 连接
type WSHandler struct {
	broker   *ws.Broker
	upgrader websocket.Upgrader

	// 全局连接计数（仅用于日志可观测）
	activeConns int64
}

// NewWSHandler 创建 WS handler
func NewWSHandler(broker *ws.Broker) *WSHandler {
	return &WSHandler{
		broker: broker,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

// HandleProgress 处理 /ws/test/progress?test_id=xxx
func (h *WSHandler) HandleProgress(w http.ResponseWriter, r *http.Request) {
	testID := r.URL.Query().Get("test_id")
	remote := r.RemoteAddr
	if testID == "" {
		hlog.Warnf("[ws] reject empty test_id remote=%s", remote)
		http.Error(w, "test_id is required", http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		hlog.Errorf("[ws] upgrade failed remote=%s test_id=%s err=%v", remote, testID, err)
		return
	}

	active := atomic.AddInt64(&h.activeConns, 1)
	connectedAt := time.Now()
	hlog.Infof("[ws] connected remote=%s test_id=%s active=%d", remote, testID, active)

	defer func() {
		active := atomic.AddInt64(&h.activeConns, -1)
		hlog.Infof("[ws] disconnected remote=%s test_id=%s active=%d duration=%s",
			remote, testID, active, time.Since(connectedAt))
		_ = conn.Close()
	}()

	ch, unsub := h.broker.Subscribe(testID)
	hlog.Debugf("[ws] subscribed broker test_id=%s", testID)
	defer func() {
		unsub()
		hlog.Debugf("[ws] unsubscribed broker test_id=%s", testID)
	}()

	// 客户端 close 时退出写循环
	closeCh := make(chan struct{})
	go func() {
		defer close(closeCh)
		for {
			if _, _, err := conn.NextReader(); err != nil {
				hlog.Debugf("[ws] read loop end test_id=%s err=%v", testID, err)
				return
			}
		}
	}()

	// 心跳：30s 发一次 ping，避免中间设备超时断开
	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	var sentCount int64
	for {
		select {
		case <-closeCh:
			hlog.Infof("[ws] client closed test_id=%s sent=%d", testID, sentCount)
			return
		case <-ping.C:
			hlog.Debugf("[ws] ping test_id=%s", testID)
			_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
		case msg, ok := <-ch:
			if !ok {
				hlog.Infof("[ws] broker channel closed test_id=%s sent=%d", testID, sentCount)
				return
			}
			data, err := json.Marshal(msg)
			if err != nil {
				hlog.Warnf("[ws] marshal msg failed test_id=%s err=%v", testID, err)
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				hlog.Warnf("[ws] write failed test_id=%s sent=%d err=%v", testID, sentCount, err)
				return
			}
			sentCount++
			hlog.Debugf("[ws] sent test_id=%s seq=%d progress=%d step=%q",
				testID, sentCount, msg.Progress, msg.Step)
		}
	}
}
