package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"

	"gateway-service/model"
	"gateway-service/pkg/metrics"
)

type BusinessEventStore interface {
	Create(ctx context.Context, event *model.BusinessEvent) (bool, error)
}

type BusinessEventHandler struct {
	store   BusinessEventStore
	metrics *metrics.Metrics
}

type BusinessEventRequest struct {
	EventType    string                 `json:"event_type"`
	Source       string                 `json:"source"`
	LiveStreamID int64                  `json:"live_stream_id"`
	AuctionID    int64                  `json:"auction_id"`
	ProductID    int64                  `json:"product_id"`
	Metadata     map[string]interface{} `json:"metadata"`
}

var allowedBusinessEventTypes = map[string]struct{}{
	"reminder_subscribe":  {},
	"reminder_click":      {},
	"live_room_enter":     {},
	"bid_button_click":    {},
	"fixed_price_click":   {},
	"purchase_success":    {},
	"auction_win":         {},
	"notification_expose": {},
	"notification_click":  {},
}

var allowedBusinessEventSources = map[string]struct{}{
	"home":                {},
	"live_room":           {},
	"live_reminder":       {},
	"notification_center": {},
	"product_detail":      {},
	"auction_card":        {},
	"fixed_price_card":    {},
	"unknown":             {},
}

var allowedMetadataKeys = map[string]struct{}{
	"client_event_id": {},
	"item_id":         {},
	"entry":           {},
	"result":          {},
	"trigger":         {},
}

var clientEventIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

const (
	maxMetadataKeys        = 8
	maxMetadataStringBytes = 256
)

var errInvalidMetadata = errors.New("invalid metadata")

func NewBusinessEventHandler(store BusinessEventStore, m *metrics.Metrics) *BusinessEventHandler {
	return &BusinessEventHandler{store: store, metrics: m}
}

func (h *BusinessEventHandler) Create(ctx context.Context, c *app.RequestContext) {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, map[string]interface{}{"code": 503, "message": "business event store unavailable"})
		return
	}

	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, map[string]interface{}{"code": 401, "message": "missing authenticated user"})
		return
	}

	var req BusinessEventRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": 400, "message": "invalid request"})
		return
	}
	if !isAllowed(req.EventType, allowedBusinessEventTypes) {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": 400, "message": "unsupported event_type"})
		return
	}

	source := req.Source
	if !isAllowed(source, allowedBusinessEventSources) {
		source = "unknown"
	}
	clientEventID := metadataString(req.Metadata, "client_event_id")
	if clientEventID == "" {
		clientEventID = uuid.NewString()
	}
	if !validClientEventID(clientEventID) {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": 400, "message": "invalid client_event_id"})
		return
	}
	if err := validateMetadata(req.Metadata); err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": 400, "message": err.Error()})
		return
	}
	metadataBytes, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": 400, "message": "invalid metadata"})
		return
	}

	event := &model.BusinessEvent{
		UserID:        userID,
		EventType:     req.EventType,
		Source:        source,
		LiveStreamID:  req.LiveStreamID,
		AuctionID:     req.AuctionID,
		ProductID:     req.ProductID,
		ClientEventID: clientEventID,
		Metadata:      string(metadataBytes),
		CreatedAt:     time.Now(),
	}
	inserted, err := h.store.Create(ctx, event)
	if err != nil {
		if h.metrics != nil {
			h.metrics.RecordBusinessFunnelEvent(req.EventType, source, "failed")
		}
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": 500, "message": "failed to record event"})
		return
	}

	if h.metrics != nil {
		result := "success"
		if !inserted {
			result = "duplicate"
		}
		h.metrics.RecordBusinessFunnelEvent(req.EventType, source, result)
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": 0, "message": "success"})
}

func authenticatedUserID(c *app.RequestContext) (int64, bool) {
	v, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}
	switch id := v.(type) {
	case int64:
		return id, id > 0
	case int:
		return int64(id), id > 0
	case string:
		n, err := strconv.ParseInt(id, 10, 64)
		return n, err == nil && n > 0
	default:
		return 0, false
	}
}

func isAllowed(value string, allowed map[string]struct{}) bool {
	_, ok := allowed[value]
	return ok
}

func metadataString(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	if v, ok := metadata[key].(string); ok {
		return v
	}
	return ""
}

func validClientEventID(value string) bool {
	return clientEventIDPattern.MatchString(value)
}

func validateMetadata(metadata map[string]interface{}) error {
	if len(metadata) > maxMetadataKeys {
		return errInvalidMetadata
	}
	for key, value := range metadata {
		if !isAllowed(key, allowedMetadataKeys) {
			return errInvalidMetadata
		}
		switch v := value.(type) {
		case string:
			if len(v) > maxMetadataStringBytes {
				return errInvalidMetadata
			}
		case float64, bool, nil:
		default:
			return errInvalidMetadata
		}
	}
	return nil
}
