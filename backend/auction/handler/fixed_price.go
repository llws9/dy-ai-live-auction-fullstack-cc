package handler

import (
	"context"
	"errors"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"auction-service/client"
	"auction-service/model"
	"auction-service/service"
)

// FixedPriceUsecase 抽象 service.FixedPriceService，便于 handler 单测注入 fake。
type FixedPriceUsecase interface {
	ListItem(ctx context.Context, r service.ListItemReq) (*model.FixedPriceItem, error)
	ListByLiveStream(ctx context.Context, r service.ListLiveItemsReq) ([]*service.LiveFixedPriceItem, error)
	ListAllByLiveStream(ctx context.Context, r service.ListLiveItemsReq) ([]*service.LiveFixedPriceItem, error)
	Purchase(ctx context.Context, r service.PurchaseReq) (*service.PurchaseResult, error)
	Offline(ctx context.Context, itemID, userID int64) error
	GetItem(ctx context.Context, itemID int64) (*model.FixedPriceItem, error)
	RemainingStock(ctx context.Context, itemID int64) (int, error)
	GetMyPurchase(ctx context.Context, itemID, userID int64) (*model.FixedPricePurchase, error)
}

// FixedPriceHandler 一口价 HTTP 入口（spec 2026-06-01 §4）。
//
// user_id 由 gateway 经 X-User-ID 注入 c.Set("user_id")；
// X-Idempotency-Key 由 gateway 透传，抢购强制校验。
// handler 仅做 HTTP shell + 错误码映射，业务编排在 FixedPriceUsecase。
type FixedPriceHandler struct {
	uc            FixedPriceUsecase
	balance       BalanceProvider
	productClient client.ProductClient
}

func NewFixedPriceHandler(uc FixedPriceUsecase, balance BalanceProvider) *FixedPriceHandler {
	return &FixedPriceHandler{uc: uc, balance: balance}
}

// SetProductClient 注入 product-service 客户端，用于回填 product_title。
func (h *FixedPriceHandler) SetProductClient(pc client.ProductClient) {
	h.productClient = pc
}

// fpErrResp 一口价统一错误响应（spec §4.3）。
type fpErrResp struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func writeFPErr(c *app.RequestContext, status int, code, msg string, details map[string]string) {
	c.JSON(status, fpErrResp{Code: code, Message: msg, Details: details})
}

// requireFPUser 从 gateway 注入的 user_id 取当前用户；缺失返回 401。
func requireFPUser(c *app.RequestContext) (int64, bool) {
	uid := c.GetInt64("user_id")
	if uid <= 0 {
		writeFPErr(c, 401, "FP_NOT_AUTHENTICATED", "未登录或无效用户", nil)
		return 0, false
	}
	return uid, true
}

func parseFPItemID(c *app.RequestContext) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		writeFPErr(c, 400, "FP_INVALID_PARAM", "item_id invalid", nil)
		return 0, false
	}
	return id, true
}

func parseFPLiveStreamID(c *app.RequestContext) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		writeFPErr(c, 400, "FP_INVALID_PARAM", "live_stream_id invalid", nil)
		return 0, false
	}
	return id, true
}

func fpStatusString(s model.FixedPriceStatus) string {
	switch s {
	case model.FixedPriceStatusOnSale:
		return "on_sale"
	case model.FixedPriceStatusSoldOut:
		return "sold_out"
	case model.FixedPriceStatusOffline:
		return "offline"
	default:
		return "unknown"
	}
}

// Purchase POST /fixed-price/items/:id/purchase（spec §4.2 / T9）。
// 方案③：响应 order_id 即购买凭证 PurchaseID。
func (h *FixedPriceHandler) Purchase(ctx context.Context, c *app.RequestContext) {
	itemID, ok := parseFPItemID(c)
	if !ok {
		return
	}
	userID, ok := requireFPUser(c)
	if !ok {
		return
	}
	idemKey := string(c.GetHeader("X-Idempotency-Key"))
	if idemKey == "" {
		writeFPErr(c, 400, "FP_INVALID_PARAM", "missing X-Idempotency-Key", nil)
		return
	}

	res, err := h.uc.Purchase(ctx, service.PurchaseReq{ItemID: itemID, UserID: userID, IdemKey: idemKey})
	switch {
	case err == nil:
		c.JSON(200, map[string]any{
			"order_id":        res.PurchaseID,
			"item_id":         res.ItemID,
			"price":           res.Price.StringFixed(2),
			"remaining_stock": res.RemainingStock,
			"status":          "success",
		})
	case errors.Is(err, service.ErrInvalidParam):
		writeFPErr(c, 400, "FP_INVALID_PARAM", err.Error(), nil)
	case errors.Is(err, service.ErrNotOnSale):
		writeFPErr(c, 409, "FP_NOT_ON_SALE", "商品已下架", nil)
	case errors.Is(err, service.ErrSoldOut):
		writeFPErr(c, 409, "FP_SOLD_OUT", "已售罄", nil)
	case errors.Is(err, service.ErrAlreadyBought):
		writeFPErr(c, 409, "FP_ALREADY_BOUGHT", "每人限购，您已购买", nil)
	case errors.Is(err, service.ErrInsufficient):
		h.writeInsufficient(ctx, c, itemID, userID)
	default:
		writeFPErr(c, 500, "FP_INTERNAL", "服务异常", nil)
	}
}

// writeInsufficient 拼装 402 余额不足响应（required/available/shortage）。
func (h *FixedPriceHandler) writeInsufficient(ctx context.Context, c *app.RequestContext, itemID, userID int64) {
	required := decimal.Zero
	if item, err := h.uc.GetItem(ctx, itemID); err == nil {
		required = item.Price
	}
	available := decimal.Zero
	if h.balance != nil {
		if avail, _, _, hit, err := h.balance.GetByUserID(ctx, userID); err == nil && hit {
			available = avail
		}
	}
	shortage := required.Sub(available)
	if shortage.IsNegative() {
		shortage = decimal.Zero
	}
	writeFPErr(c, 402, "FP_INSUFFICIENT_BALANCE", "余额不足", map[string]string{
		"required":  required.StringFixed(2),
		"available": available.StringFixed(2),
		"shortage":  shortage.StringFixed(2),
	})
}

// listItemBody 上架请求体。price 以字符串传递（decimal 精度安全）。
type listItemBody struct {
	LiveStreamID int64  `json:"live_stream_id"`
	ProductID    int64  `json:"product_id"`
	Price        string `json:"price"`
	TotalStock   int    `json:"total_stock"`
	MaxPerUser   int    `json:"max_per_user"`
}

// List POST /fixed-price/items（spec §4.1 / T10）。CreatorID 取自登录用户，不信任请求体。
func (h *FixedPriceHandler) List(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireFPUser(c)
	if !ok {
		return
	}
	var body listItemBody
	if err := c.BindAndValidate(&body); err != nil {
		writeFPErr(c, 400, "FP_INVALID_PARAM", "invalid body", nil)
		return
	}
	price, err := decimal.NewFromString(body.Price)
	if err != nil {
		writeFPErr(c, 400, "FP_INVALID_PARAM", "price format", nil)
		return
	}

	item, err := h.uc.ListItem(ctx, service.ListItemReq{
		LiveStreamID: body.LiveStreamID,
		ProductID:    body.ProductID,
		CreatorID:    userID,
		Price:        price,
		TotalStock:   body.TotalStock,
		MaxPerUser:   body.MaxPerUser,
	})
	switch {
	case err == nil:
		c.JSON(200, map[string]any{
			"id":              item.ID,
			"live_stream_id":  item.LiveStreamID,
			"product_id":      item.ProductID,
			"price":           item.Price.StringFixed(2),
			"total_stock":     item.TotalStock,
			"remaining_stock": item.RemainingStock,
			"max_per_user":    item.MaxPerUser,
			"status":          fpStatusString(item.Status),
			"created_at":      item.CreatedAt,
		})
	case errors.Is(err, service.ErrInvalidParam):
		writeFPErr(c, 400, "FP_INVALID_PARAM", err.Error(), nil)
	case errors.Is(err, service.ErrNotStreamOwner):
		writeFPErr(c, 403, "FP_NOT_STREAM_OWNER", "非主播本人，无法上架", nil)
	case errors.Is(err, service.ErrProductNotFound):
		writeFPErr(c, 404, "FP_PRODUCT_NOT_FOUND", "商品不存在", nil)
	default:
		writeFPErr(c, 500, "FP_INTERNAL", "服务异常", nil)
	}
}

// Offline POST /fixed-price/items/:id/offline（spec §4.1 / T10）。
func (h *FixedPriceHandler) Offline(ctx context.Context, c *app.RequestContext) {
	itemID, ok := parseFPItemID(c)
	if !ok {
		return
	}
	userID, ok := requireFPUser(c)
	if !ok {
		return
	}
	err := h.uc.Offline(ctx, itemID, userID)
	switch {
	case err == nil:
		c.JSON(200, map[string]any{"status": "offline"})
	case errors.Is(err, service.ErrNotStreamOwner):
		writeFPErr(c, 403, "FP_NOT_STREAM_OWNER", "非主播本人，无法下架", nil)
	case errors.Is(err, gorm.ErrRecordNotFound):
		writeFPErr(c, 404, "FP_NOT_FOUND", "商品不存在", nil)
	default:
		writeFPErr(c, 500, "FP_INTERNAL", "服务异常", nil)
	}
}

// Detail GET /fixed-price/items/:id（spec §4.1 / T10）。remaining_stock 以 Redis 权威为准。
func (h *FixedPriceHandler) Detail(ctx context.Context, c *app.RequestContext) {
	itemID, ok := parseFPItemID(c)
	if !ok {
		return
	}
	item, err := h.uc.GetItem(ctx, itemID)
	if err != nil {
		writeFPErr(c, 404, "FP_NOT_FOUND", "商品不存在", nil)
		return
	}
	rem := item.RemainingStock
	if live, err := h.uc.RemainingStock(ctx, itemID); err == nil {
		rem = live
	}
	c.JSON(200, map[string]any{
		"id":              item.ID,
		"live_stream_id":  item.LiveStreamID,
		"product_id":      item.ProductID,
		"price":           item.Price.StringFixed(2),
		"total_stock":     item.TotalStock,
		"remaining_stock": rem,
		"max_per_user":    item.MaxPerUser,
		"status":          fpStatusString(item.Status),
	})
}

// ListByLiveStream GET /live-streams/:id/fixed-price/items。公开读取直播间一口价列表。
func (h *FixedPriceHandler) ListByLiveStream(ctx context.Context, c *app.RequestContext) {
	liveStreamID, ok := parseFPLiveStreamID(c)
	if !ok {
		return
	}
	items, err := h.uc.ListByLiveStream(ctx, service.ListLiveItemsReq{LiveStreamID: liveStreamID})
	if err != nil {
		writeFPErr(c, 500, "FP_INTERNAL", "服务异常", nil)
		return
	}
	productSummaries := h.batchProductSummaries(ctx, items)
	resp := make([]map[string]any, 0, len(items))
	for _, it := range items {
		item := it.Item
		entry := map[string]any{
			"id":              item.ID,
			"live_stream_id":  item.LiveStreamID,
			"product_id":      item.ProductID,
			"price":           item.Price.StringFixed(2),
			"total_stock":     item.TotalStock,
			"remaining_stock": it.RemainingStock,
			"max_per_user":    item.MaxPerUser,
			"status":          fpStatusString(item.Status),
		}
		if summary, ok := productSummaries[item.ProductID]; ok {
			brief := map[string]any{
				"id":    summary.ID,
				"title": summary.Name,
			}
			if len(summary.Images) > 0 {
				brief["cover_image"] = summary.Images[0]
			}
			entry["product_brief"] = brief
		}
		resp = append(resp, entry)
	}
	c.JSON(200, map[string]any{"items": resp})
}

// ListAllByLiveStream GET /admin/live-streams/:id/fixed-price/items。管理端读取全部状态。
// 授权由 Gateway RequireStreamer 中间件保证；handler 层仅做认证校验作为纵深防御。
func (h *FixedPriceHandler) ListAllByLiveStream(ctx context.Context, c *app.RequestContext) {
	if _, ok := requireFPUser(c); !ok {
		return
	}
	liveStreamID, ok := parseFPLiveStreamID(c)
	if !ok {
		return
	}
	items, err := h.uc.ListAllByLiveStream(ctx, service.ListLiveItemsReq{LiveStreamID: liveStreamID})
	if err != nil {
		writeFPErr(c, 500, "FP_INTERNAL", "服务异常", nil)
		return
	}

	// 批量回填 product_title
	productTitles := h.batchProductTitles(ctx, items)

	resp := make([]map[string]any, 0, len(items))
	for _, it := range items {
		item := it.Item
		entry := map[string]any{
			"id":              item.ID,
			"live_stream_id":  item.LiveStreamID,
			"product_id":      item.ProductID,
			"price":           item.Price.StringFixed(2),
			"total_stock":     item.TotalStock,
			"remaining_stock": it.RemainingStock,
			"max_per_user":    item.MaxPerUser,
			"status":          fpStatusString(item.Status),
			"created_at":      item.CreatedAt,
		}
		if title, ok := productTitles[item.ProductID]; ok {
			entry["product_title"] = title
		}
		resp = append(resp, entry)
	}
	c.JSON(200, map[string]any{"items": resp, "total": len(resp)})
}

// batchProductTitles 批量查询 product_title，失败时静默降级（返回空 map）。
func (h *FixedPriceHandler) batchProductTitles(ctx context.Context, items []*service.LiveFixedPriceItem) map[int64]string {
	summaries := h.batchProductSummaries(ctx, items)
	if len(summaries) == 0 {
		return nil
	}
	out := make(map[int64]string, len(summaries))
	for id, s := range summaries {
		out[id] = s.Name
	}
	return out
}

func (h *FixedPriceHandler) batchProductSummaries(ctx context.Context, items []*service.LiveFixedPriceItem) map[int64]client.ProductSummary {
	if h.productClient == nil || len(items) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.Item.ProductID)
	}
	summaries, err := h.productClient.BatchGetSummaries(ctx, ids)
	if err != nil {
		return nil
	}
	return summaries
}

// MyPurchase GET /fixed-price/items/:id/my-purchase（spec §4.1 / T10）。无跨域，查本服务购买记录。
func (h *FixedPriceHandler) MyPurchase(ctx context.Context, c *app.RequestContext) {
	itemID, ok := parseFPItemID(c)
	if !ok {
		return
	}
	userID, ok := requireFPUser(c)
	if !ok {
		return
	}
	p, err := h.uc.GetMyPurchase(ctx, itemID, userID)
	if err != nil {
		c.JSON(200, map[string]any{"i_bought": false})
		return
	}
	c.JSON(200, map[string]any{
		"i_bought":   true,
		"order_id":   p.ID,
		"price":      p.Price.StringFixed(2),
		"created_at": p.CreatedAt,
	})
}

// RegisterFixedPriceRoutes 在 /api/v1 组下挂载一口价路由。
func RegisterFixedPriceRoutes(g *route.RouterGroup, h *FixedPriceHandler) {
	g.GET("/live-streams/:id/fixed-price/items", h.ListByLiveStream)
	g.GET("/admin/live-streams/:id/fixed-price/items", h.ListAllByLiveStream)
	fp := g.Group("/fixed-price")
	fp.POST("/items", h.List)
	fp.POST("/items/:id/offline", h.Offline)
	fp.GET("/items/:id", h.Detail)
	fp.POST("/items/:id/purchase", h.Purchase)
	fp.GET("/items/:id/my-purchase", h.MyPurchase)
}
