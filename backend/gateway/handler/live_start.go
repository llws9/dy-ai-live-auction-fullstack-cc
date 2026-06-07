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
	productURL    string
	auctionURL    string
	internalToken string
	client        *http.Client
}

func NewLiveStartHandler(productURL, auctionURL, internalToken string) *LiveStartHandler {
	return &LiveStartHandler{
		productURL:    strings.TrimRight(productURL, "/"),
		auctionURL:    strings.TrimRight(auctionURL, "/"),
		internalToken: internalToken,
		client:        &http.Client{Timeout: 2 * time.Second},
	}
}

func (h *LiveStartHandler) StartLive(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": 400, "message": "无效的直播间ID"})
		return
	}
	auctionStatus, auctionBody, ok := h.callInternal(ctx, c, h.auctionURL, http.MethodPost, fmt.Sprintf("/internal/live-streams/%s/start", id), "开始直播失败")
	if !ok {
		h.writeUpstreamResult(c, auctionStatus, auctionBody, "开始直播失败")
		return
	}
	productStatus, productBody, ok := h.callInternal(ctx, c, h.productURL, http.MethodPut, fmt.Sprintf("/api/v1/admin/live-streams/%s/start", id), "开始直播失败")
	if !ok {
		h.writeUpstreamResult(c, productStatus, productBody, "开始直播失败")
		return
	}
	c.Response.SetStatusCode(productStatus)
	c.Response.SetBody(productBody)
}

func (h *LiveStartHandler) EndLive(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": 400, "message": "无效的直播间ID"})
		return
	}
	productStatus, productBody, ok := h.callInternal(ctx, c, h.productURL, http.MethodPut, fmt.Sprintf("/api/v1/admin/live-streams/%s/end", id), "结束直播失败")
	if !ok {
		h.writeUpstreamResult(c, productStatus, productBody, "结束直播失败")
		return
	}
	auctionStatus, auctionBody, ok := h.callInternal(ctx, c, h.auctionURL, http.MethodPost, fmt.Sprintf("/internal/live-streams/%s/end", id), "结束直播失败")
	if !ok {
		h.writeUpstreamResult(c, auctionStatus, auctionBody, "结束直播失败")
		return
	}
	c.Response.SetStatusCode(productStatus)
	c.Response.SetBody(productBody)
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
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			c.Response.SetStatusCode(resp.StatusCode)
			c.Response.SetBody(body)
			return
		}
		c.JSON(http.StatusBadGateway, map[string]interface{}{"code": 502, "message": failureMessage})
		return
	}

	c.Response.SetStatusCode(resp.StatusCode)
	c.Response.SetBody(body)
}

func (h *LiveStartHandler) callInternal(ctx context.Context, c *app.RequestContext, baseURL, method, path, failureMessage string) (int, []byte, bool) {
	if h.internalToken == "" {
		log.Printf("live internal token is not configured")
		return http.StatusInternalServerError, []byte(fmt.Sprintf(`{"code":500,"message":%q}`, failureMessage)), false
	}
	if baseURL == "" || strings.Contains(path, "//") {
		return http.StatusBadRequest, []byte(`{"code":400,"message":"无效的直播间ID"}`), false
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, nil)
	if err != nil {
		log.Printf("failed to create live internal request: %v", err)
		return http.StatusInternalServerError, []byte(fmt.Sprintf(`{"code":500,"message":%q}`, failureMessage)), false
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
		return http.StatusBadGateway, []byte(fmt.Sprintf(`{"code":502,"message":%q}`, failureMessage)), false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read live internal response: %v", err)
		return http.StatusInternalServerError, []byte(fmt.Sprintf(`{"code":500,"message":%q}`, failureMessage)), false
	}
	return resp.StatusCode, body, resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (h *LiveStartHandler) writeUpstreamResult(c *app.RequestContext, status int, body []byte, failureMessage string) {
	if status >= 400 && status < 500 && len(body) > 0 {
		c.Response.SetStatusCode(status)
		c.Response.SetBody(body)
		return
	}
	if status > 0 && len(body) > 0 {
		c.Response.SetStatusCode(status)
		c.Response.SetBody(body)
		return
	}
	c.JSON(http.StatusBadGateway, map[string]interface{}{"code": 502, "message": failureMessage})
}
