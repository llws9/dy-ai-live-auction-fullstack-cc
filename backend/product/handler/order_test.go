package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/dao"
	"product-service/model"
	"product-service/service"
)

func newOrderHandlerWithDB(t *testing.T) (*OrderHandler, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.Order{}, &model.Product{}, &model.Category{}, &model.LiveStream{}, &model.User{}))
	productDAO := dao.NewProductDAO(db)
	svc := service.NewOrderService(dao.NewOrderDAO(db), dao.NewHistoryDAO(db))
	svc.SetProductDAO(productDAO)
	svc.SetAdminOrderDAO(dao.NewOrderAdminDAO(db))
	return NewOrderHandler(svc), db
}

func TestOrderHandler_CreateFromAuctionResult(t *testing.T) {
	h, db := newOrderHandlerWithDB(t)
	ownerID := int64(3001)
	assert.NoError(t, db.Create(&model.Product{
		ID:      11,
		OwnerID: &ownerID,
		Name:    "auction product",
		Status:  model.ProductStatusPublished,
	}).Error)

	c := app.NewContext(0)
	c.Request.SetMethod(consts.MethodPost)
	c.Request.SetRequestURI("/internal/orders/from-auction-result")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBodyString(`{"auction_id":101,"product_id":11,"winner_id":2001,"final_price":"110.00"}`)

	h.CreateFromAuctionResult(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		Code int `json:"code"`
		Data struct {
			ID         int64  `json:"id"`
			AuctionID  int64  `json:"auction_id"`
			ProductID  int64  `json:"product_id"`
			WinnerID   int64  `json:"winner_id"`
			FinalPrice string `json:"final_price"`
			Status     int    `json:"status"`
		} `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, 0, body.Code)
	assert.NotZero(t, body.Data.ID)
	assert.Equal(t, int64(101), body.Data.AuctionID)
	assert.Equal(t, int64(11), body.Data.ProductID)
	assert.Equal(t, int64(2001), body.Data.WinnerID)
	assert.Equal(t, "110.00", body.Data.FinalPrice)
	assert.Equal(t, int(model.OrderStatusPending), body.Data.Status)
}

func TestOrderHandler_CreateFromAuctionResultRejectsInvalidPrice(t *testing.T) {
	h, _ := newOrderHandlerWithDB(t)

	c := app.NewContext(0)
	c.Request.SetMethod(consts.MethodPost)
	c.Request.SetRequestURI("/internal/orders/from-auction-result")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBodyString(`{"auction_id":101,"product_id":11,"winner_id":2001,"final_price":"abc"}`)

	h.CreateFromAuctionResult(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}

// TestOrderHandler_GetUserHistory_AuthContract 验证 spec C / F-C3 安全契约：
//   - 未携带 X-User-ID（即 Gateway JWT 中间件未放行） → 401；
//   - X-User-ID 解析失败 / 非正整数 → 401；
//   - X-User-ID 合法 → 200，且仅以 header 用户身份查询，query user_id 不再生效。
func TestOrderHandler_GetUserHistory_AuthContract(t *testing.T) {
	// historyDAO=nil 时 OrderService.GetUserHistory 返回空集合，足够覆盖 handler 鉴权分支。
	h := NewOrderHandler(service.NewOrderService(nil, nil))

	t.Run("missing X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/history")

		h.GetUserHistory(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
		var body map[string]interface{}
		assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
		assert.EqualValues(t, 401, body["code"])
	})

	t.Run("non-numeric X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/history")
		c.Request.Header.Set("X-User-ID", "not-a-number")

		h.GetUserHistory(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("non-positive X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/history")
		c.Request.Header.Set("X-User-ID", "0")

		h.GetUserHistory(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("valid X-User-ID returns 200 and query user_id is ignored", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		// query 中故意塞一个不同的 user_id：必须被忽略，不能借此越权。
		c.Request.SetRequestURI("/api/v1/orders/history?user_id=999&page=1&page_size=20")
		c.Request.Header.Set("X-User-ID", "123")

		h.GetUserHistory(context.Background(), c)

		assert.Equal(t, 200, c.Response.StatusCode())
		var body map[string]interface{}
		assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
		assert.EqualValues(t, 0, body["total"])
		assert.EqualValues(t, 1, body["page"])
		assert.EqualValues(t, 20, body["page_size"])
		assert.NotNil(t, body["list"])
	})
}

func TestOrderHandler_GetUserHistory_ReturnsProductImage(t *testing.T) {
	h, db := newOrderHandlerWithDB(t)
	assert.NoError(t, db.Exec(`
                CREATE TABLE IF NOT EXISTS auctions (
                        id INTEGER PRIMARY KEY,
                        product_id INTEGER NOT NULL,
                        status INTEGER NOT NULL,
                        created_at DATETIME NOT NULL
                )
        `).Error)
	assert.NoError(t, db.Exec(`
                CREATE TABLE IF NOT EXISTS bids (
                        id INTEGER PRIMARY KEY,
                        auction_id INTEGER NOT NULL,
                        user_id INTEGER NOT NULL
                )
        `).Error)

	ownerID := int64(3001)
	userID := int64(1991)
	productID := int64(995204)
	auctionID := int64(995304)
	assert.NoError(t, db.Exec("DELETE FROM bids WHERE id = ? OR auction_id = ?", 995001, auctionID).Error)
	assert.NoError(t, db.Exec("DELETE FROM orders WHERE id = ? OR auction_id = ?", 995056, auctionID).Error)
	assert.NoError(t, db.Exec("DELETE FROM products WHERE id = ?", productID).Error)
	assert.NoError(t, db.Exec("DELETE FROM auctions WHERE id = ?", auctionID).Error)
	assert.NoError(t, db.Create(&model.Product{
		ID:      productID,
		OwnerID: &ownerID,
		Name:    "山海鎏金香炉",
		Images:  model.JSONArray{"https://cdn.example.com/products/incense-burner.jpg"},
		Status:  model.ProductStatusPublished,
	}).Error)
	assert.NoError(t, db.Exec(
		"INSERT INTO auctions (id, product_id, status, created_at) VALUES (?, ?, ?, ?)",
		auctionID, productID, dao.AuctionStatusEnded, "2026-05-29T12:00:00Z",
	).Error)
	assert.NoError(t, db.Exec(
		"INSERT INTO bids (id, auction_id, user_id) VALUES (?, ?, ?)",
		995001, auctionID, userID,
	).Error)
	assert.NoError(t, db.Create(&model.Order{
		ID:         995056,
		AuctionID:  auctionID,
		ProductID:  productID,
		SellerID:   &ownerID,
		WinnerID:   userID,
		FinalPrice: decimal.NewFromInt(6800),
		Status:     model.OrderStatusPending,
	}).Error)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/orders/history?page=1&page_size=20")
	c.Request.Header.Set("X-User-ID", "1991")

	h.GetUserHistory(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		List []map[string]interface{} `json:"list"`
	}
	assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Len(t, body.List, 1)
	assert.Equal(t, "山海鎏金香炉", body.List[0]["product_name"])
	assert.Equal(t, "https://cdn.example.com/products/incense-burner.jpg", body.List[0]["product_image"])
}

func TestOrderHandler_Summary_XUserIDContract(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.Order{}))

	h := NewOrderHandler(service.NewOrderService(dao.NewOrderDAO(db), nil))

	t.Run("missing X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/summary")

		h.Summary(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
		var body map[string]interface{}
		assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
		assert.EqualValues(t, 401, body["code"])
	})

	t.Run("non-numeric X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/summary")
		c.Request.Header.Set("X-User-ID", "not-a-number")

		h.Summary(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("non-positive X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/summary")
		c.Request.Header.Set("X-User-ID", "0")

		h.Summary(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("valid X-User-ID returns header user's summary and ignores query user_id", func(t *testing.T) {
		assert.NoError(t, db.Exec("DELETE FROM orders").Error)
		ctx := context.Background()
		svc := service.NewOrderService(dao.NewOrderDAO(db), nil)
		_, err := svc.CreateOrder(ctx, 1, 1, 123, decimal.NewFromInt(100))
		assert.NoError(t, err)
		_, err = svc.CreateOrder(ctx, 2, 1, 999, decimal.NewFromInt(200))
		assert.NoError(t, err)
		_, err = svc.CreateOrder(ctx, 3, 1, 999, decimal.NewFromInt(300))
		assert.NoError(t, err)

		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		// query 中伪造不同 user_id：Summary 必须忽略 query，只信 Gateway 透传的 X-User-ID。
		c.Request.SetRequestURI("/api/v1/orders/summary?user_id=999")
		c.Request.Header.Set("X-User-ID", "123")

		h.Summary(context.Background(), c)

		assert.Equal(t, 200, c.Response.StatusCode())
		var body struct {
			Code int `json:"code"`
			Data struct {
				PendingPayment int64 `json:"pendingPayment"`
				WonNotPaid     int64 `json:"wonNotPaid"`
			} `json:"data"`
		}
		assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
		assert.Equal(t, 0, body.Code)
		assert.Equal(t, int64(1), body.Data.PendingPayment)
		assert.Equal(t, int64(1), body.Data.WonNotPaid)
	})
}

// TestOrderHandler_List_AuthContract 验证 GET /orders 的安全契约（与 GetUserHistory 对齐）：
//   - 未携带 X-User-ID → 401；
//   - X-User-ID 非法 → 401；
//   - 合法 X-User-ID → 200，query user_id 被忽略。
func TestOrderHandler_List_AuthContract(t *testing.T) {
	h := NewOrderHandler(service.NewOrderService(nil, nil))

	t.Run("missing X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders")

		h.List(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("non-numeric X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders")
		c.Request.Header.Set("X-User-ID", "abc")

		h.List(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("valid X-User-ID returns 200", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders?page=1&page_size=20")
		c.Request.Header.Set("X-User-ID", "42")

		h.List(context.Background(), c)

		assert.Equal(t, 200, c.Response.StatusCode())
	})
}

func TestOrderHandler_List_ReturnsProductDisplayFields(t *testing.T) {
	h, db := newOrderHandlerWithDB(t)
	ownerID := int64(9001)
	assert.NoError(t, db.Create(&model.User{
		ID:     ownerID,
		Name:   "山海商家",
		Role:   int(model.RoleStreamer),
		Status: 1,
	}).Error)
	assert.NoError(t, db.Create(&model.Product{
		ID:      992204,
		OwnerID: &ownerID,
		Name:    "山海鎏金香炉",
		Images:  model.JSONArray{"https://cdn.example.com/products/incense-burner.jpg"},
		Status:  model.ProductStatusPublished,
	}).Error)
	assert.NoError(t, db.Create(&model.Order{
		ID:         56,
		AuctionID:  992304,
		ProductID:  992204,
		SellerID:   &ownerID,
		WinnerID:   1001,
		FinalPrice: decimal.NewFromInt(6800),
		Status:     model.OrderStatusPending,
	}).Error)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/orders?page=1&page_size=20")
	c.Request.Header.Set("X-User-ID", "1001")

	h.List(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		List []map[string]interface{} `json:"list"`
	}
	assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Len(t, body.List, 1)
	assert.Equal(t, "山海鎏金香炉", body.List[0]["product_name"])
	assert.Equal(t, "https://cdn.example.com/products/incense-burner.jpg", body.List[0]["product_image"])
	assert.Equal(t, "山海商家", body.List[0]["seller_name"])
}

type stubSummaryGetter struct {
	err error
}

func (s *stubSummaryGetter) GetSummary(_ context.Context, _ int64) (*model.OrderSummaryResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &model.OrderSummaryResponse{PendingPayment: 0, WonNotPaid: 0}, nil
}

func TestOrderHandler_Summary_ErrorNoLeak(t *testing.T) {
	internalErr := errors.New("db connection refused: password=secret")
	h := &OrderHandler{
		orderService:   service.NewOrderService(nil, nil),
		summaryService: &stubSummaryGetter{err: internalErr},
	}

	writer := &testLogWriter{}
	log.SetOutput(writer)
	defer log.SetOutput(os.Stderr)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/orders/summary")
	c.Request.Header.Set("X-User-ID", "1")

	h.Summary(context.Background(), c)

	assert.Equal(t, 500, c.Response.StatusCode())
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 500, body["code"])
	assert.Equal(t, "获取订单汇总失败", body["message"])
	assert.NotContains(t, body["message"], "secret")
	assert.NotContains(t, body["message"], "db connection")
	assert.Contains(t, writer.String(), "db connection refused")
}

type testLogWriter struct {
	data []byte
}

func (w *testLogWriter) Write(p []byte) (n int, err error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func (w *testLogWriter) String() string {
	return string(w.data)
}

func TestOrderHandler_Get_AuthContract(t *testing.T) {
	h := NewOrderHandler(service.NewOrderService(nil, nil))

	t.Run("missing X-User-ID returns 401 before reading order", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/123")

		h.Get(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("non-numeric X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/123")
		c.Request.Header.Set("X-User-ID", "abc")

		h.Get(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})
}

func TestOrderHandler_Pay_AuthContract(t *testing.T) {
	h := NewOrderHandler(service.NewOrderService(nil, nil))

	t.Run("missing X-User-ID returns 401 before paying order", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.SetRequestURI("/api/v1/orders/123/pay")

		h.Pay(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("non-positive X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.SetRequestURI("/api/v1/orders/123/pay")
		c.Request.Header.Set("X-User-ID", "0")

		h.Pay(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})
}

func TestOrderHandler_Ship_AdminRoleRejected(t *testing.T) {
	h := NewOrderHandler(service.NewOrderService(nil, nil))
	c := app.NewContext(0)
	c.Request.SetMethod("PUT")
	c.Request.SetRequestURI("/api/v1/orders/123/ship")
	c.Request.Header.Set("X-User-ID", "2001")
	c.Request.Header.Set("X-User-Role", "admin")

	h.Ship(context.Background(), c)

	assert.Equal(t, 403, c.Response.StatusCode())
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 403, body["code"])
}

func TestOrderHandler_Ship_MerchantOtherSellerRejected(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.Order{}))
	assert.NoError(t, db.Exec("DELETE FROM orders").Error)
	seller := int64(1001)
	order := &model.Order{ID: 123, AuctionID: 1, ProductID: 1, SellerID: &seller, WinnerID: 3001, FinalPrice: decimal.NewFromInt(100), Status: model.OrderStatusPaid}
	assert.NoError(t, db.Create(order).Error)
	h := NewOrderHandler(service.NewOrderService(dao.NewOrderDAO(db), nil))

	c := app.NewContext(0)
	c.Request.SetMethod("PUT")
	c.Request.SetRequestURI("/api/v1/orders/123/ship")
	c.Request.Header.Set("X-User-ID", "1002")
	c.Request.Header.Set("X-User-Role", "merchant")

	h.Ship(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
	var saved model.Order
	assert.NoError(t, db.First(&saved, 123).Error)
	assert.Equal(t, model.OrderStatusPaid, saved.Status)
}
