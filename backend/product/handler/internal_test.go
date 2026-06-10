package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
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
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{}))
	db.Exec("DELETE FROM products")
	db.Exec("DELETE FROM categories")
	db.Exec("DELETE FROM auction_rules")
	db.Exec("DELETE FROM live_streams")
	if seed != nil {
		seed(db)
	}
	svc := service.NewProductService(dao.NewProductDAO(db), dao.NewAuctionRuleDAO(db), dao.NewLiveStreamDAO(db))
	return NewInternalHandler(svc, nil, nil)
}

func newInternalHandlerWithSeedAndViewers(t *testing.T, seed func(db *gorm.DB), viewers service.LiveViewerCounter) *InternalHandler {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{}))
	db.Exec("DELETE FROM products")
	db.Exec("DELETE FROM categories")
	db.Exec("DELETE FROM auction_rules")
	db.Exec("DELETE FROM live_streams")
	if seed != nil {
		seed(db)
	}
	svc := service.NewProductService(dao.NewProductDAO(db), dao.NewAuctionRuleDAO(db), dao.NewLiveStreamDAO(db))
	lsSvc := service.NewLiveStreamServiceWithMetrics(dao.NewLiveStreamDAO(db), viewers)
	return NewInternalHandler(svc, dao.NewLiveStreamDAO(db), lsSvc)
}

func ptr64(v int64) *int64 { return &v }

// --- GET /internal/products/:id/auction-info -------------------------------

func TestInternalHandler_GetAuctionProductInfo(t *testing.T) {
	var productID int64
	const ownerID int64 = 1001
	h := newInternalHandlerWithSeed(t, func(db *gorm.DB) {
		product := &model.Product{OwnerID: ptr64(ownerID), Name: "schedulable", Status: model.ProductStatusPublished}
		require.NoError(t, db.Create(product).Error)
		productID = product.ID
		require.NoError(t, db.Create(&model.AuctionRule{
			ProductID:  product.ID,
			StartPrice: 100,
			Increment:  10,
			Duration:   3600,
		}).Error)
	})

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/internal/products/" + strconv.FormatInt(productID, 10) + "/auction-info")
	c.Params = append(c.Params, param.Param{Key: "id", Value: strconv.FormatInt(productID, 10)})

	h.GetAuctionProductInfo(context.Background(), c)

	require.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		Code int `json:"code"`
		Data struct {
			ID        int64 `json:"id"`
			OwnerID   int64 `json:"owner_id"`
			Status    int   `json:"status"`
			RuleBound bool  `json:"rule_bound"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, 200, body.Code)
	assert.Equal(t, productID, body.Data.ID)
	assert.Equal(t, ownerID, body.Data.OwnerID)
	assert.Equal(t, int(model.ProductStatusPublished), body.Data.Status)
	assert.True(t, body.Data.RuleBound)
}

func TestInternalHandler_GetAuctionProductInfoReturns500WhenRuleLookupFails(t *testing.T) {
	var productID int64
	const ownerID int64 = 1001
	h := newInternalHandlerWithSeed(t, func(db *gorm.DB) {
		product := &model.Product{OwnerID: ptr64(ownerID), Name: "rule lookup fails", Status: model.ProductStatusPublished}
		require.NoError(t, db.Create(product).Error)
		productID = product.ID
		require.NoError(t, db.Migrator().DropTable(&model.AuctionRule{}))
	})

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/internal/products/" + strconv.FormatInt(productID, 10) + "/auction-info")
	c.Params = append(c.Params, param.Param{Key: "id", Value: strconv.FormatInt(productID, 10)})

	h.GetAuctionProductInfo(context.Background(), c)

	require.Equal(t, 500, c.Response.StatusCode())
	var body struct {
		Code int `json:"code"`
		Data *struct {
			RuleBound bool `json:"rule_bound"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, 500, body.Code)
	assert.Nil(t, body.Data)
}

func TestInternalHandler_GetAuctionProductInfoReturns404WhenProductMissing(t *testing.T) {
	h := newInternalHandlerWithSeed(t, nil)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/internal/products/99999/auction-info")
	c.Params = append(c.Params, param.Param{Key: "id", Value: "99999"})

	h.GetAuctionProductInfo(context.Background(), c)

	require.Equal(t, 404, c.Response.StatusCode())
	var body struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, 404, body.Code)
}

// --- POST /internal/live-streams/get-or-create -----------------------------

func TestInternalHandler_GetOrCreateActiveLiveStreamCreatesNotStartedStream(t *testing.T) {
	h := newInternalHandlerWithSeed(t, nil)

	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/live-streams/get-or-create")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBodyString(`{"creator_id":1001,"creator_name":"merchant_1001"}`)

	h.GetOrCreateActiveLiveStream(context.Background(), c)

	require.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		Code int `json:"code"`
		Data struct {
			ID        int64 `json:"id"`
			CreatorID int64 `json:"creator_id"`
			Status    int   `json:"status"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, 200, body.Code)
	assert.NotZero(t, body.Data.ID)
	assert.Equal(t, int64(1001), body.Data.CreatorID)
	assert.Equal(t, int(model.LiveStreamStatusNotStarted), body.Data.Status)
}

func TestInternalHandler_GetOrCreateActiveLiveStreamAllowsExistingNotStartedStream(t *testing.T) {
	h := newInternalHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.LiveStream{
			CreatorID: 1001,
			Name:      "not-started",
			Status:    model.LiveStreamStatusNotStarted,
		}).Error)
	})

	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/live-streams/get-or-create")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBodyString(`{"creator_id":1001,"creator_name":"merchant_1001"}`)

	h.GetOrCreateActiveLiveStream(context.Background(), c)

	require.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		Data struct {
			Status int `json:"status"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, int(model.LiveStreamStatusNotStarted), body.Data.Status)
}

func TestInternalHandler_GetOrCreateActiveLiveStreamRejectsBanned(t *testing.T) {
	h := newInternalHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.LiveStream{
			CreatorID: 1001,
			Name:      "banned",
			Status:    model.LiveStreamStatusBanned,
		}).Error)
	})

	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/internal/live-streams/get-or-create")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBodyString(`{"creator_id":1001,"creator_name":"merchant_1001"}`)

	h.GetOrCreateActiveLiveStream(context.Background(), c)

	assert.Equal(t, 409, c.Response.StatusCode())
}

// --- GET /internal/products?category_id= -----------------------------------

func TestInternalHandler_ListByCategory_OK(t *testing.T) {
	h := newInternalHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.Product{Name: "A", CategoryID: ptr64(12), Status: model.ProductStatusPublished})
		db.Create(&model.Product{Name: "B", CategoryID: ptr64(99), Status: model.ProductStatusPublished})
		db.Create(&model.Product{Name: "C", CategoryID: ptr64(12), Status: model.ProductStatusPublished})
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

func TestInternalHandler_ListByCategory_OnlyReturnsPublishedProducts(t *testing.T) {
	h := newInternalHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.Product{Name: "draft", CategoryID: ptr64(12), Status: model.ProductStatusDraft})
		db.Create(&model.Product{Name: "published", CategoryID: ptr64(12), Status: model.ProductStatusPublished})
		db.Create(&model.Product{Name: "unpublished", CategoryID: ptr64(12), Status: model.ProductStatusUnpublished})
	})

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/internal/products?category_id=12")

	h.ListByCategory(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	assert.EqualValues(t, 1, data["total"])
	items := data["items"].([]interface{})
	require.Len(t, items, 1)
	assert.Equal(t, "published", items[0].(map[string]interface{})["name"])
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
		a := &model.Product{Name: "A", CategoryID: ptr64(1), Images: model.JSONArray{"a.jpg"}, Status: model.ProductStatusPublished}
		b := &model.Product{Name: "B", CategoryID: ptr64(2), Status: model.ProductStatusPublished}
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

func TestInternalHandler_BatchByIDs_OnlyReturnsPublishedProducts(t *testing.T) {
	var draftID, publishedID, unpublishedID int64
	h := newInternalHandlerWithSeed(t, func(db *gorm.DB) {
		draft := &model.Product{Name: "draft", Status: model.ProductStatusDraft}
		published := &model.Product{Name: "published", Status: model.ProductStatusPublished}
		unpublished := &model.Product{Name: "unpublished", Status: model.ProductStatusUnpublished}
		require.NoError(t, db.Create(draft).Error)
		require.NoError(t, db.Create(published).Error)
		require.NoError(t, db.Create(unpublished).Error)
		draftID, publishedID, unpublishedID = draft.ID, published.ID, unpublished.ID
	})

	raw, _ := json.Marshal(map[string]interface{}{"ids": []int64{draftID, publishedID, unpublishedID}})
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
	require.Len(t, items, 1)
	assert.Equal(t, "published", items[0].(map[string]interface{})["name"])
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

func TestInternalHandler_BatchLiveStreams_ViewerCountRedisFirst(t *testing.T) {
	h := newInternalHandlerWithSeedAndViewers(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.LiveStream{
			ID: 101, Name: "room-a", Status: 1, CreatorID: 9, ViewerCount: 19,
		}).Error)
	}, service.StaticLiveViewerCounter{101: 42})

	body, _ := json.Marshal(map[string]interface{}{"ids": []int64{101}})
	c := app.NewContext(0)
	c.Request.SetBody(body)
	c.Request.Header.SetMethod("POST")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	h.BatchLiveStreams(context.Background(), c)

	require.Equal(t, 200, c.Response.StatusCode())
	var resp struct {
		Data struct {
			Items []struct {
				ID          int64 `json:"id"`
				ViewerCount int64 `json:"viewer_count"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &resp))
	require.Len(t, resp.Data.Items, 1)
	assert.Equal(t, int64(42), resp.Data.Items[0].ViewerCount)
}

func TestInternalHandler_BatchLiveStreams_ViewerCountDBFallback(t *testing.T) {
	h := newInternalHandlerWithSeedAndViewers(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.LiveStream{
			ID: 102, Name: "room-b", Status: 1, CreatorID: 9, ViewerCount: 7,
		}).Error)
	}, service.StaticLiveViewerCounter{})

	body, _ := json.Marshal(map[string]interface{}{"ids": []int64{102}})
	c := app.NewContext(0)
	c.Request.SetBody(body)
	c.Request.Header.SetMethod("POST")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	h.BatchLiveStreams(context.Background(), c)

	require.Equal(t, 200, c.Response.StatusCode())
	var resp struct {
		Data struct {
			Items []struct {
				ViewerCount int64 `json:"viewer_count"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &resp))
	require.Len(t, resp.Data.Items, 1)
	assert.Equal(t, int64(7), resp.Data.Items[0].ViewerCount)
}
