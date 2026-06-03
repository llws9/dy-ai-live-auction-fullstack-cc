package metrics

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
)

// TrackEventRequest 埋点请求
type TrackEventRequest struct {
	EventType string                 `json:"event_type" binding:"required"` // 事件类型
	EventName string                 `json:"event_name"`                    // 事件名称
	Params    map[string]interface{} `json:"params"`                        // 事件参数
	UserID    string                 `json:"user_id"`                       // 用户ID
	Timestamp int64                  `json:"timestamp"`                     // 时间戳
}

var allowedTouchpointEvents = map[string]struct{}{
	"summary_exposed":           {},
	"entry_clicked":             {},
	"notification_list_exposed": {},
	"notification_item_clicked": {},
	"mark_read":                 {},
	"hot_pull_triggered":        {},
	"live_reminder_exposed":     {},
	"live_reminder_clicked":     {},
	"live_reminder_dismissed":   {},
}

var allowedTouchpointSources = map[string]struct{}{
	"home":                {},
	"bottom_nav":          {},
	"profile":             {},
	"notification_center": {},
	"mobile_shell":        {},
	"notification_hook":   {},
}

var allowedTouchpointEntries = map[string]struct{}{
	"notification_bell":   {},
	"profile_tab":         {},
	"auction_history":     {},
	"notification_center": {},
	"notification_item":   {},
	"mark_all_read":       {},
	"hot_pull":            {},
	"live_reminder_modal": {},
}

var allowedTouchpointTypes = map[string]struct{}{
	"all":             {},
	"pending_payment": {},
	"outbid":          {},
	"ending_soon":     {},
	"live_start":      {},
	"notification":    {},
}

var allowedTouchpointResults = map[string]struct{}{
	"success":   {},
	"failed":    {},
	"clicked":   {},
	"dismissed": {},
	"debounced": {},
}

// TrackEvent 处理前端埋点请求
func TrackEvent(m *Metrics) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		var req TrackEventRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		// 根据事件类型记录不同的指标
		switch req.EventType {
		case "live_room_enter":
			// 直播间进入
			roomID := getStringParam(req.Params, "room_id")
			userType := getStringParam(req.Params, "user_type", "normal")
			m.RecordLiveRoomEnter(roomID, userType)

		case "live_room_leave":
			// 直播间离开（可用于计算观看时长等）
			// 可扩展更多指标

		case "auction_view":
			// 竞拍浏览

		case "bid_click":
			// 出价按钮点击
			auctionID := getStringParam(req.Params, "auction_id")
			m.AuctionBidTotal.WithLabelValues(auctionID, "click").Inc()

		case "payment_start":
			// 发起支付
			method := getStringParam(req.Params, "method", "unknown")
			amount := getFloatParam(req.Params, "amount")
			m.PaymentInitiated.WithLabelValues(method).Inc()
			if amount > 0 {
				m.PaymentAmount.WithLabelValues(method).Observe(amount)
			}

		case "user_register":
			// 用户注册
			source := getStringParam(req.Params, "source", "direct")
			m.UserRegister.WithLabelValues(source).Inc()

		case "user_login":
			// 用户登录
			method := getStringParam(req.Params, "method", "password")
			m.UserLogin.WithLabelValues(method).Inc()

		case "touchpoint_event":
			recordTouchpointEvent(m, req)

		default:
			// 通用事件计数
			// 可以扩展自定义指标
		}

		c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}

func normalizeLabel(value string, allowed map[string]struct{}) string {
	if _, ok := allowed[value]; ok {
		return value
	}
	return "unknown"
}

func recordTouchpointEvent(m *Metrics, req TrackEventRequest) {
	event := normalizeLabel(req.EventName, allowedTouchpointEvents)
	source := normalizeLabel(getStringParam(req.Params, "source", "unknown"), allowedTouchpointSources)
	entry := normalizeLabel(getStringParam(req.Params, "entry", "unknown"), allowedTouchpointEntries)
	touchpointType := normalizeLabel(getStringParam(req.Params, "type", "unknown"), allowedTouchpointTypes)
	result := normalizeLabel(getStringParam(req.Params, "result", "unknown"), allowedTouchpointResults)
	m.RecordTouchpointEvent(event, source, entry, touchpointType, result)
}

// getStringParam 从参数中获取字符串
func getStringParam(params map[string]interface{}, key string, defaults ...string) string {
	if params == nil {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return ""
	}
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if len(defaults) > 0 {
		return defaults[0]
	}
	return ""
}

// getFloatParam 从参数中获取浮点数
func getFloatParam(params map[string]interface{}, key string) float64 {
	if params == nil {
		return 0
	}
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return 0
}
