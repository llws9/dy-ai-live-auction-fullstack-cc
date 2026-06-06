package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"

	"product-service/service"
)

type AuctionRuleTemplateHandler struct {
	service *service.AuctionRuleTemplateService
}

func NewAuctionRuleTemplateHandler(service *service.AuctionRuleTemplateService) *AuctionRuleTemplateHandler {
	return &AuctionRuleTemplateHandler{service: service}
}

func (h *AuctionRuleTemplateHandler) List(ctx context.Context, c *app.RequestContext) {
	actor, ok := requireMerchantActor(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := h.service.List(ctx, actor.UserID, page, pageSize)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取规则模板失败: " + err.Error()})
		return
	}
	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"list":      items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func (h *AuctionRuleTemplateHandler) Get(ctx context.Context, c *app.RequestContext) {
	actor, id, ok := h.requireActorAndID(c)
	if !ok {
		return
	}
	item, err := h.service.Get(ctx, actor.UserID, id)
	if err != nil {
		writeRuleTemplateError(c, err)
		return
	}
	c.JSON(200, map[string]interface{}{"code": 200, "message": "success", "data": item})
}

func (h *AuctionRuleTemplateHandler) Create(ctx context.Context, c *app.RequestContext) {
	actor, ok := requireMerchantActor(c)
	if !ok {
		return
	}
	var req service.CreateAuctionRuleTemplateRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	item, err := h.service.Create(ctx, actor.UserID, req)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(201, map[string]interface{}{"code": 201, "message": "success", "data": item})
}

func (h *AuctionRuleTemplateHandler) Update(ctx context.Context, c *app.RequestContext) {
	actor, id, ok := h.requireActorAndID(c)
	if !ok {
		return
	}
	var req service.UpdateAuctionRuleTemplateRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	item, err := h.service.Update(ctx, actor.UserID, id, req)
	if err != nil {
		writeRuleTemplateError(c, err)
		return
	}
	c.JSON(200, map[string]interface{}{"code": 200, "message": "success", "data": item})
}

func (h *AuctionRuleTemplateHandler) Delete(ctx context.Context, c *app.RequestContext) {
	actor, id, ok := h.requireActorAndID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(ctx, actor.UserID, id); err != nil {
		writeRuleTemplateError(c, err)
		return
	}
	c.JSON(200, map[string]interface{}{"code": 200, "message": "删除成功"})
}

func (h *AuctionRuleTemplateHandler) ApplyToProduct(ctx context.Context, c *app.RequestContext) {
	actor, ok := requireMerchantActor(c)
	if !ok {
		return
	}
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || productID <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的商品ID"})
		return
	}
	var req service.ApplyAuctionRuleTemplateRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	rule, err := h.service.ApplyToProduct(ctx, actor.UserID, productID, req.TemplateID)
	if err != nil {
		writeRuleTemplateError(c, err)
		return
	}
	c.JSON(200, map[string]interface{}{"code": 200, "message": "success", "data": rule})
}

func (h *AuctionRuleTemplateHandler) requireActorAndID(c *app.RequestContext) (AdminActor, int64, bool) {
	actor, ok := requireMerchantActor(c)
	if !ok {
		return AdminActor{}, 0, false
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的规则模板ID"})
		return AdminActor{}, 0, false
	}
	return actor, id, true
}

func writeRuleTemplateError(c *app.RequestContext, err error) {
	if err == gorm.ErrRecordNotFound {
		c.JSON(404, map[string]interface{}{"code": 404, "message": "规则模板不存在"})
		return
	}
	c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
}
