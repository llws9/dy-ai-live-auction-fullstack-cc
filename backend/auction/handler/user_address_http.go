package handler

import (
	"context"
	"errors"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
)

// UserAddressHandler /api/v1/users/me/addresses CRUD（T3.2 / spec A F-A3）。
//
// user_id 由 gateway JWTAuth 通过 X-User-ID header 注入到 c.Set("user_id", ...)。
// 业务编排在 Build* 函数；HTTP shell 仅做参数解析、错误码映射与序列化。
type UserAddressHandler struct {
	store AddressStore
}

func NewUserAddressHandler(store AddressStore) *UserAddressHandler {
	return &UserAddressHandler{store: store}
}

func (h *UserAddressHandler) requireUserID(c *app.RequestContext) (int64, bool) {
	uid := c.GetInt64("user_id")
	if uid <= 0 {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未登录或无效用户"})
		return 0, false
	}
	return uid, true
}

func parsePathID(c *app.RequestContext) (int64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "invalid id"})
		return 0, false
	}
	return id, true
}

// writeAddressErr 把业务错误统一映射到 HTTP code（spec A F-A3 §错误码）。
func writeAddressErr(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, ErrAddressInvalid):
		c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
	case errors.Is(err, ErrAddressLimitExceeded):
		c.JSON(400, map[string]interface{}{"code": 400, "message": "地址数量已达上限（20）"})
	case errors.Is(err, ErrAddressNotFound):
		c.JSON(404, map[string]interface{}{"code": 404, "message": "地址不存在"})
	default:
		c.JSON(500, map[string]interface{}{"code": 500, "message": "服务异常"})
	}
}

func (h *UserAddressHandler) List(ctx context.Context, c *app.RequestContext) {
	uid, ok := h.requireUserID(c)
	if !ok {
		return
	}
	resp, err := BuildListAddresses(ctx, h.store, uid)
	if err != nil {
		writeAddressErr(c, err)
		return
	}
	c.JSON(200, map[string]interface{}{"code": 200, "data": resp})
}

func (h *UserAddressHandler) Create(ctx context.Context, c *app.RequestContext) {
	uid, ok := h.requireUserID(c)
	if !ok {
		return
	}
	var in AddressInput
	if err := c.BindAndValidate(&in); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "invalid body"})
		return
	}
	v, err := BuildCreateAddress(ctx, h.store, uid, in)
	if err != nil {
		writeAddressErr(c, err)
		return
	}
	c.JSON(200, map[string]interface{}{"code": 200, "data": v})
}

func (h *UserAddressHandler) Update(ctx context.Context, c *app.RequestContext) {
	uid, ok := h.requireUserID(c)
	if !ok {
		return
	}
	id, ok := parsePathID(c)
	if !ok {
		return
	}
	var in AddressInput
	if err := c.BindAndValidate(&in); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "invalid body"})
		return
	}
	if err := BuildUpdateAddress(ctx, h.store, uid, id, in); err != nil {
		writeAddressErr(c, err)
		return
	}
	c.JSON(200, map[string]interface{}{"code": 200, "message": "ok"})
}

func (h *UserAddressHandler) Delete(ctx context.Context, c *app.RequestContext) {
	uid, ok := h.requireUserID(c)
	if !ok {
		return
	}
	id, ok := parsePathID(c)
	if !ok {
		return
	}
	if err := BuildDeleteAddress(ctx, h.store, uid, id); err != nil {
		writeAddressErr(c, err)
		return
	}
	c.JSON(200, map[string]interface{}{"code": 200, "message": "ok"})
}

func (h *UserAddressHandler) SetDefault(ctx context.Context, c *app.RequestContext) {
	uid, ok := h.requireUserID(c)
	if !ok {
		return
	}
	id, ok := parsePathID(c)
	if !ok {
		return
	}
	if err := BuildSetDefaultAddress(ctx, h.store, uid, id); err != nil {
		writeAddressErr(c, err)
		return
	}
	c.JSON(200, map[string]interface{}{"code": 200, "message": "ok"})
}
