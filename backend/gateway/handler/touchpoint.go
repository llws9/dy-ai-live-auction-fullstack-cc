package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

type TouchpointHandler struct {
	auctionURL string
	productURL string
	client     *http.Client
}

type upstreamEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type auctionSummary struct {
	UnreadTotal int64 `json:"unreadTotal"`
	Outbid      int64 `json:"outbid"`
	EndingSoon  int64 `json:"endingSoon"`
}

type orderSummary struct {
	PendingPayment int64 `json:"pendingPayment"`
	WonNotPaid     int64 `json:"wonNotPaid"`
}

type touchpointSummary struct {
	UnreadTotal    int64 `json:"unreadTotal"`
	PendingPayment int64 `json:"pendingPayment"`
	WonNotPaid     int64 `json:"wonNotPaid"`
	Outbid         int64 `json:"outbid"`
	EndingSoon     int64 `json:"endingSoon"`
}

func NewTouchpointHandler(auctionURL, productURL string) *TouchpointHandler {
	return &TouchpointHandler{
		auctionURL: strings.TrimRight(auctionURL, "/"),
		productURL: strings.TrimRight(productURL, "/"),
		client:     &http.Client{Timeout: 2 * time.Second},
	}
}

func (h *TouchpointHandler) GetNotificationSummary(ctx context.Context, c *app.RequestContext) {
	token := string(c.Request.Header.Peek("Authorization"))
	userID := toString(c.GetInt64("user_id"))

	auctionData := auctionSummary{}
	if err := h.fetch(ctx, h.auctionURL+"/api/v1/notifications/summary", token, userID, &auctionData); isAuthUpstreamError(err) {
		writeUpstreamAuthError(c, err)
		return
	}

	orderData := orderSummary{}
	if err := h.fetch(ctx, h.productURL+"/api/v1/orders/summary", token, userID, &orderData); isAuthUpstreamError(err) {
		writeUpstreamAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"code":    0,
		"message": "success",
		"data": touchpointSummary{
			UnreadTotal:    auctionData.UnreadTotal,
			PendingPayment: orderData.PendingPayment,
			WonNotPaid:     orderData.WonNotPaid,
			Outbid:         auctionData.Outbid,
			EndingSoon:     auctionData.EndingSoon,
		},
	})
}

func (h *TouchpointHandler) fetch(ctx context.Context, url, token, userID string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	if userID != "" {
		req.Header.Set("X-User-ID", userID)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return upstreamStatusError{status: resp.StatusCode}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var env upstreamEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return err
	}
	if env.Code != 0 && env.Code != http.StatusOK {
		return fmt.Errorf("upstream code %d: %s", env.Code, env.Message)
	}
	if len(env.Data) == 0 {
		return nil
	}
	return json.Unmarshal(env.Data, out)
}

type upstreamStatusError struct {
	status int
}

func (e upstreamStatusError) Error() string {
	return fmt.Sprintf("upstream status %d", e.status)
}

func isAuthUpstreamError(err error) bool {
	statusErr, ok := err.(upstreamStatusError)
	return ok && (statusErr.status == http.StatusUnauthorized || statusErr.status == http.StatusForbidden)
}

func writeUpstreamAuthError(c *app.RequestContext, err error) {
	statusErr := err.(upstreamStatusError)
	c.JSON(statusErr.status, map[string]interface{}{
		"code":    statusErr.status,
		"message": "authentication failed",
	})
}
