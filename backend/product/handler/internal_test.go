package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/dao"
	"product-service/model"
	"product-service/service"
)

// newInternalHandlerWithSeed builds a fresh in-memory product-service stack for
// the /internal/* endpoints. The unique DSN per call avoids cross-test pollution
// from the shared :memory: database used elsewhere.
func newInternalHandlerWithSeed(t *testing.T, seed func(db *gorm.DB)) *InternalHandler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.AuctionRule{}, &model.LiveStream{}))
	// Clean slate; ":memory:?cache=shared" is shared across the process so tests
	// must reset the table before seeding.
	db.Exec("DELETE FROM products")
	if seed != nil {
		seed(db)
	}
	svc := service.NewProductService(dao.NewProductDAO(db), dao.NewAuctionRuleDAO(db), dao.NewLiveStreamDAO(db))
	return NewInternalHandler(svc, nil)
}

func ptr64(v int64) *int64 { return &v }

// --- GET /internal/products?category_id= -----------------------------------

func TestInternalHandler_ListByCategory_OK(t *testing.T) {
	h := newInternalHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.Product{Name: "A", CategoryID: ptr64(12)})
		db.Create(&model.Product{Name: "B", CategoryID: ptr64(99)})
		db.Create(&model.Product{Name: "C", CategoryID: ptr64(12)})
	})

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/internal/products?category_id=12")

	h.ListByCategory(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 200, body["code"])
	data := body["data"].(map[string]interface{})
	assert.EqualValues(t, 2, data["total"])
	items := data["items"].([]interface{})
	assert.Len(t, items, 2)
}

func TestInternalHandler_ListByCategory_MissingCategoryID(t *testing.T) {
	h := newInternalHandlerWithSeed(t, nil)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/internal/products")

	h.ListByCategory(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}

func TestInternalHandler_ListByCategory_InvalidCategoryID(t *testing.T) {
	h := newInternalHandlerWithSeed(t, nil)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/internal/products?category_id=abc")

	h.ListByCategory(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}

// --- POST /internal/products/batch -----------------------------------------

func TestInternalHandler_BatchByIDs_OK(t *testing.T) {
	var p1, p2 int64
	h := newInternalHandlerWithSeed(t, func(db *gorm.DB) {
		a := &model.Product{Name: "A", CategoryID: ptr64(1), Images: model.JSONArray{"a.jpg"}}
		b := &model.Product{Name: "B", CategoryID: ptr64(2)}
		require.NoError(t, db.Create(a).Error)
		require.NoError(t, db.Create(b).Error)
		p1, p2 = a.ID, b.ID
	})

	body := map[string]interface{}{"ids": []int64{p1, p2, 99999}}
	raw, _ := json.Marshal(body)

	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/products/batch")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody(raw)

	h.BatchByIDs(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &resp))
	data := resp["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	assert.Len(t, items, 2, "missing id 99999 must not appear in items")
}

func TestInternalHandler_BatchByIDs_EmptyIDsRejected(t *testing.T) {
	h := newInternalHandlerWithSeed(t, nil)

	raw := []byte(`{"ids":[]}`)
	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/products/batch")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody(raw)

	h.BatchByIDs(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}

func TestInternalHandler_BatchByIDs_OversizedRejected(t *testing.T) {
	h := newInternalHandlerWithSeed(t, nil)

	ids := make([]int64, 201)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	raw, _ := json.Marshal(map[string]interface{}{"ids": ids})

	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/products/batch")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody(raw)

	h.BatchByIDs(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}

func TestInternalHandler_BatchByIDs_InvalidJSON(t *testing.T) {
	h := newInternalHandlerWithSeed(t, nil)

	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/products/batch")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody(bytes.NewBufferString(`not-json`).Bytes())

	h.BatchByIDs(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}
