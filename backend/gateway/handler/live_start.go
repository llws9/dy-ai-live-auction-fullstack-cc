package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

type LiveStartHandler struct {
	auctionURL    string
	internalToken string
	client        *http.Client
}

func NewLiveStartHandler(auctionURL, internalToken string) *LiveStartHandler {
	return &LiveStartHandler{
		auctionURL:    strings.TrimRight(auctionURL, "/"),
		internalToken: internalToken,
		client:        &http.Client{Timeout: 2 * time.Second},
	}
}

func (h *LiveStartHandler) StartLive(ctx context.Context, c *app.RequestContext) {
	h.forwardInternal(ctx, c, http.MethodPost, fmt.Sprintf("/internal/live-streams/%s/start", c.Param("id")), "开始直播失败")
}

func (h *LiveStartHandler) GetPendingReminder(ctx context.Context, c *app.RequestContext) {
	h.forwardInternal(ctx, c, http.MethodGet, "/internal/live/pending-reminder", "获取开播提醒失败")
}

func (h *LiveStartHandler) forwardInternal(ctx context.Context, c *app.RequestContext, method, path, failureMessage string) {
	if h.internalToken == "" {
		log.Printf("live internal token is not configured")
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": 500, "message": failureMessage})
		return
	}

	if strings.Contains(path, "//") || strings.HasSuffix(path, "/start") && c.Param("id") == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": 400, "message": "无效的直播间ID"})
		return
	}

	req, err := http.NewRequestWithContext(ctx, method, h.auctionURL+path, nil)
	if err != nil {
		log.Printf("failed to create live internal request: %v", err)
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": 500, "message": failureMessage})
		return
	}
	req.Header.Set("X-Internal-Token", h.internalToken)
	if userID, exists := c.Get("user_id"); exists {
		req.Header.Set("X-User-ID", toString(userID))
	}
	if username, exists := c.Get("username"); exists {
		req.Header.Set("X-Username", toString(username))
	}
	if role, exists := c.Get("user_role"); exists {
		req.Header.Set("X-User-Role", toRoleString(role))
	}

	resp, err := h.client.Do(req)
	if err != nil {
		log.Printf("failed to forward live internal request: %v", err)
		c.JSON(http.StatusBadGateway, map[string]interface{}{"code": 502, "message": failureMessage})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read live internal response: %v", err)
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": 500, "message": failureMessage})
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("live internal upstream status %d: %s", resp.StatusCode, string(body))
		c.JSON(http.StatusBadGateway, map[string]interface{}{"code": 502, "message": failureMessage})
		return
	}

	c.Response.SetStatusCode(resp.StatusCode)
	c.Response.SetBody(body)
}
