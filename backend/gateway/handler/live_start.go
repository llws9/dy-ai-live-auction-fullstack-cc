package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

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
		client:        &http.Client{},
	}
}

func (h *LiveStartHandler) StartLive(ctx context.Context, c *app.RequestContext) {
	if h.internalToken == "" {
		log.Printf("live start internal token is not configured")
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": 500, "message": "开始直播失败"})
		return
	}

	liveStreamID := c.Param("id")
	if liveStreamID == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": 400, "message": "无效的直播间ID"})
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/internal/live-streams/%s/start", h.auctionURL, liveStreamID), nil)
	if err != nil {
		log.Printf("failed to create live start request: %v", err)
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": 500, "message": "开始直播失败"})
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
		log.Printf("failed to forward live start request: %v", err)
		c.JSON(http.StatusBadGateway, map[string]interface{}{"code": 502, "message": "开始直播失败"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read live start response: %v", err)
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": 500, "message": "开始直播失败"})
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("live start upstream status %d: %s", resp.StatusCode, string(body))
		c.JSON(http.StatusBadGateway, map[string]interface{}{"code": 502, "message": "开始直播失败"})
		return
	}

	c.Response.SetStatusCode(resp.StatusCode)
	c.Response.SetBody(body)
}
