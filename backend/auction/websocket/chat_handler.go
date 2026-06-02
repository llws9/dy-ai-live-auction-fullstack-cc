package websocket

import (
	"context"
	"strings"
	"time"
)

// ChatHandler 编排弹幕发送：校验 → 频控 → 广播
type ChatHandler struct {
	hub      *Hub
	filter   *ChatFilter
	throttle *ChatThrottle
}

// NewChatHandler 构造
func NewChatHandler(hub *Hub, filter *ChatFilter, throttle *ChatThrottle) *ChatHandler {
	return &ChatHandler{hub: hub, filter: filter, throttle: throttle}
}

// Handle 处理一次弹幕发送
func (h *ChatHandler) Handle(ctx context.Context, c *Client, data *ChatSendData) {
	// 鉴权：必须登录
	if c.UserID <= 0 || !c.Authenticated {
		c.Send <- NewErrorMessage(ChatErrCodeNotAuthenticated, "login required")
		return
	}

	// 直播间一致性
	if data.LiveStreamID <= 0 || (c.LiveStreamID > 0 && data.LiveStreamID != c.LiveStreamID) {
		c.Send <- NewErrorMessage(ChatErrCodeLengthExceeded, "invalid live_stream_id")
		return
	}

	text := strings.TrimSpace(data.Text)

	// 内容校验
	if code := h.filter.Validate(text); code != 0 {
		c.Send <- NewErrorMessage(code, codeMessage(code))
		return
	}

	// 频控
	if code := h.throttle.Allow(ctx, c.UserID, data.LiveStreamID); code != 0 {
		c.Send <- NewErrorMessage(code, codeMessage(code))
		return
	}

	// 广播
	out := NewChatMessage(&ChatMessageData{
		LiveStreamID: data.LiveStreamID,
		UserID:       c.UserID,
		UserName:     c.UserName,
		Text:         text,
		SentAt:       time.Now().UnixMilli(),
		ClientMsgID:  data.ClientMsgID,
	})
	h.hub.BroadcastToLiveStream(data.LiveStreamID, out)
}

func codeMessage(code int) string {
	switch code {
	case ChatErrCodeLengthExceeded:
		return "text length invalid"
	case ChatErrCodeBlockedWord:
		return "blocked word detected"
	case ChatErrCodeRateLimited:
		return "rate limited"
	case ChatErrCodeNotAuthenticated:
		return "login required"
	default:
		return "chat error"
	}
}
