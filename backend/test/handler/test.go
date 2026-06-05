package handler

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"

	"test-service/dao"
	"test-service/model"
	"test-service/runner"
)

// TestHandler 提供测试任务的 HTTP API
type TestHandler struct {
	runner    *runner.Runner
	resultDAO *dao.ResultDAO
}

// NewTestHandler 构造
func NewTestHandler(r *runner.Runner, d *dao.ResultDAO) *TestHandler {
	return &TestHandler{runner: r, resultDAO: d}
}

// PostDummy POST /api/test/dummy 启动一个 dummy 任务
func (h *TestHandler) PostDummy(ctx context.Context, c *app.RequestContext) {
	body := c.Request.Body()
	hlog.CtxInfof(ctx, "[http] POST /api/test/dummy remote=%s body=%s",
		c.ClientIP(), truncate(string(body), 256))

	id, err := h.runner.Submit(ctx, model.TypeDummy, body)
	if err != nil {
		hlog.CtxErrorf(ctx, "[http] PostDummy submit failed: %v", err)
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}
	hlog.CtxInfof(ctx, "[http] PostDummy submitted test_id=%s", id)
	c.JSON(200, map[string]any{"test_id": id})
}

// PostPressure POST /api/test/pressure 启动一个压测任务
func (h *TestHandler) PostPressure(ctx context.Context, c *app.RequestContext) {
	body := c.Request.Body()
	hlog.CtxInfof(ctx, "[http] POST /api/test/pressure remote=%s body=%s",
		c.ClientIP(), truncate(string(body), 256))

	id, err := h.runner.Submit(ctx, model.TypePressure, body)
	if err != nil {
		hlog.CtxErrorf(ctx, "[http] PostPressure submit failed: %v", err)
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}
	hlog.CtxInfof(ctx, "[http] PostPressure submitted test_id=%s", id)
	c.JSON(200, map[string]any{"test_id": id})
}

// PostE2E POST /api/test/e2e 启动一个 E2E 全链路测试任务
func (h *TestHandler) PostE2E(ctx context.Context, c *app.RequestContext) {
	body := c.Request.Body()
	hlog.CtxInfof(ctx, "[http] POST /api/test/e2e remote=%s body=%s",
		c.ClientIP(), truncate(string(body), 256))

	id, err := h.runner.Submit(ctx, model.TypeE2E, body)
	if err != nil {
		hlog.CtxErrorf(ctx, "[http] PostE2E submit failed: %v", err)
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}
	hlog.CtxInfof(ctx, "[http] PostE2E submitted test_id=%s", id)
	c.JSON(200, map[string]any{"test_id": id})
}

// PostUserJourney POST /api/test/user-journey 启动买家验收剧本
func (h *TestHandler) PostUserJourney(ctx context.Context, c *app.RequestContext) {
	body := c.Request.Body()
	hlog.CtxInfof(ctx, "[http] POST /api/test/user-journey remote=%s body=%s",
		c.ClientIP(), truncate(string(body), 256))

	id, err := h.runner.Submit(ctx, model.TypeUserJourney, body)
	if err != nil {
		hlog.CtxErrorf(ctx, "[http] PostUserJourney submit failed: %v", err)
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}
	hlog.CtxInfof(ctx, "[http] PostUserJourney submitted test_id=%s", id)
	c.JSON(200, map[string]any{"test_id": id})
}

// PostAntiSnipe POST /api/test/antisnipe 启动防狙击延时测试任务
func (h *TestHandler) PostAntiSnipe(ctx context.Context, c *app.RequestContext) {
	body := c.Request.Body()
	hlog.CtxInfof(ctx, "[http] POST /api/test/antisnipe remote=%s body=%s",
		c.ClientIP(), truncate(string(body), 256))

	id, err := h.runner.Submit(ctx, model.TypeAntiSnipe, body)
	if err != nil {
		hlog.CtxErrorf(ctx, "[http] PostAntiSnipe submit failed: %v", err)
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}
	hlog.CtxInfof(ctx, "[http] PostAntiSnipe submitted test_id=%s", id)
	c.JSON(200, map[string]any{"test_id": id})
}

// PostCallback POST /api/test/callback 启动外部回调可靠投递测试任务
func (h *TestHandler) PostCallback(ctx context.Context, c *app.RequestContext) {
	body := c.Request.Body()
	hlog.CtxInfof(ctx, "[http] POST /api/test/callback remote=%s body=%s",
		c.ClientIP(), truncate(string(body), 256))

	id, err := h.runner.Submit(ctx, model.TypeCallback, body)
	if err != nil {
		hlog.CtxErrorf(ctx, "[http] PostCallback submit failed: %v", err)
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}
	hlog.CtxInfof(ctx, "[http] PostCallback submitted test_id=%s", id)
	c.JSON(200, map[string]any{"test_id": id})
}

// PostChaos POST /api/test/chaos 启动场景 G 故障注入测试
func (h *TestHandler) PostChaos(ctx context.Context, c *app.RequestContext) {
	body := c.Request.Body()
	hlog.CtxInfof(ctx, "[http] POST /api/test/chaos remote=%s body=%s",
		c.ClientIP(), truncate(string(body), 256))

	id, err := h.runner.Submit(ctx, model.TypeChaos, body)
	if err != nil {
		hlog.CtxErrorf(ctx, "[http] PostChaos submit failed: %v", err)
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}
	hlog.CtxInfof(ctx, "[http] PostChaos submitted test_id=%s", id)
	c.JSON(200, map[string]any{"test_id": id})
}

// PostScript POST /api/test/script/:name 按剧本顺序执行多个 sub-scenario
func (h *TestHandler) PostScript(ctx context.Context, c *app.RequestContext) {
	name := c.Param("name")
	body := c.Request.Body()
	hlog.CtxInfof(ctx, "[http] POST /api/test/script/%s body=%s",
		name, truncate(string(body), 256))

	// 把 script_name 注入 cfg JSON：脚本场景从中读取
	cfg := injectScriptName(body, name)
	id, err := h.runner.Submit(ctx, model.TypeScript, cfg)
	if err != nil {
		hlog.CtxErrorf(ctx, "[http] PostScript submit failed: %v", err)
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]any{"test_id": id, "script": name})
}

// PostCompare POST /api/test/compare 并发跑两份配置，返回两个 test_id
func (h *TestHandler) PostCompare(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Type      string          `json:"type"`
		LeftBody  json.RawMessage `json:"left"`
		RightBody json.RawMessage `json:"right"`
	}
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(400, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	if req.Type == "" {
		c.JSON(400, map[string]any{"error": "missing type"})
		return
	}
	leftID, lErr := h.runner.Submit(ctx, req.Type, req.LeftBody)
	rightID, rErr := h.runner.Submit(ctx, req.Type, req.RightBody)
	if lErr != nil || rErr != nil {
		c.JSON(400, map[string]any{
			"error":     "submit failed",
			"left_err":  errStr(lErr),
			"right_err": errStr(rErr),
			"left_id":   leftID,
			"right_id":  rightID,
		})
		return
	}
	hlog.CtxInfof(ctx, "[http] PostCompare type=%s left=%s right=%s", req.Type, leftID, rightID)
	c.JSON(200, map[string]any{
		"type":     req.Type,
		"left_id":  leftID,
		"right_id": rightID,
	})
}

func errStr(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}

// injectScriptName 将路径参数 name 合并到 cfg JSON 的 script_name 字段
func injectScriptName(body []byte, name string) []byte {
	var m map[string]any
	if len(body) == 0 || string(body) == "null" {
		m = map[string]any{}
	} else if err := json.Unmarshal(body, &m); err != nil {
		m = map[string]any{}
	}
	m["script_name"] = name
	out, _ := json.Marshal(m)
	return out
}

// GetStatus GET /api/test/status/:id 查询状态
func (h *TestHandler) GetStatus(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")
	hlog.CtxDebugf(ctx, "[http] GET /api/test/status/%s", id)

	rec, err := h.resultDAO.GetByID(ctx, id)
	if err != nil {
		hlog.CtxWarnf(ctx, "[http] GetStatus not found id=%s err=%v", id, err)
		c.JSON(404, map[string]any{"error": "not found"})
		return
	}
	c.JSON(200, rec)
}

// GetHistory GET /api/test/history?test_type=&status=&page=&page_size=
func (h *TestHandler) GetHistory(ctx context.Context, c *app.RequestContext) {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	filters := dao.HistoryFilters{
		TestType: c.Query("test_type"),
		Status:   c.Query("status"),
		Page:     page,
		PageSize: size,
	}
	t0 := time.Now()
	list, total, err := h.resultDAO.GetHistory(ctx, filters)
	cost := time.Since(t0)

	if err != nil {
		hlog.CtxErrorf(ctx, "[http] GetHistory failed filters=%+v err=%v", filters, err)
		c.JSON(500, map[string]any{"error": err.Error()})
		return
	}
	hlog.CtxInfof(ctx, "[http] GetHistory filters=%+v total=%d returned=%d cost=%s",
		filters, total, len(list), cost)
	c.JSON(200, map[string]any{"total": total, "items": list})
}

// GetReport GET /api/test/report/:id 查询完整报告（M1 等同 status）
func (h *TestHandler) GetReport(ctx context.Context, c *app.RequestContext) {
	hlog.CtxDebugf(ctx, "[http] GET /api/test/report/%s", c.Param("id"))
	h.GetStatus(ctx, c)
}

// PostCancel POST /api/test/cancel/:id 取消任务
func (h *TestHandler) PostCancel(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")
	hlog.CtxInfof(ctx, "[http] POST /api/test/cancel/%s", id)

	if err := h.runner.Cancel(id); err != nil {
		hlog.CtxWarnf(ctx, "[http] PostCancel id=%s err=%v", id, err)
		c.JSON(404, map[string]any{"error": err.Error()})
		return
	}
	hlog.CtxInfof(ctx, "[http] PostCancel id=%s ok", id)
	c.JSON(200, map[string]any{"cancelled": id})
}

// truncate 截断字符串便于日志输出
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
