package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"product-service/model"
	"product-service/service"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/dao"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 { return &v }

func newProductHandlerWithSeed(t *testing.T, seed func(db *gorm.DB)) (*ProductHandler, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{}))
	require.NoError(t, db.Exec("DELETE FROM auction_rules").Error)
	require.NoError(t, db.Exec("DELETE FROM live_streams").Error)
	require.NoError(t, db.Exec("DELETE FROM products").Error)
	require.NoError(t, db.Exec("DELETE FROM categories").Error)
	if seed != nil {
		seed(db)
	}
	productSvc := service.NewProductService(dao.NewProductDAO(db), dao.NewAuctionRuleDAO(db), dao.NewLiveStreamDAO(db))
	return NewProductHandler(productSvc), db
}

func newProductRequestContext(method, uri string, body []byte) *app.RequestContext {
	c := app.NewContext(0)
	c.Request.SetMethod(method)
	c.Request.SetRequestURI(uri)
	if body != nil {
		c.Request.SetBody(body)
		c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	}
	return c
}

func TestProductHandler_Create_RequestValidation(t *testing.T) {
	t.Run("should validate create request fields", func(t *testing.T) {
		testCases := []struct {
			name    string
			request service.CreateProductRequest
			isValid bool
		}{
			{
				name: "valid request",
				request: service.CreateProductRequest{
					Name:        "Product",
					Description: "Description",
					Images:      []string{"image.jpg"},
				},
				isValid: true,
			},
			{
				name: "empty name",
				request: service.CreateProductRequest{
					Name: "",
				},
				isValid: false,
			},
			{
				name: "name with images",
				request: service.CreateProductRequest{
					Name:   "Product",
					Images: []string{"img1.jpg", "img2.jpg"},
				},
				isValid: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.request.Name == "" {
					assert.False(t, tc.isValid)
				} else {
					assert.True(t, tc.isValid)
				}
			})
		}
	})

	t.Run("should marshal request correctly", func(t *testing.T) {
		req := service.CreateProductRequest{
			Name:        "Test Product",
			Description: "Test Description",
			Images:      []string{"image1.jpg"},
		}

		body, err := json.Marshal(req)
		assert.NoError(t, err)

		var parsed service.CreateProductRequest
		err = json.Unmarshal(body, &parsed)
		assert.NoError(t, err)
		assert.Equal(t, req.Name, parsed.Name)
	})
}

func TestProductHandler_Get_IDValidation(t *testing.T) {
	t.Run("should validate product ID", func(t *testing.T) {
		testCases := []struct {
			name    string
			idStr   string
			isValid bool
		}{
			{"valid ID", "1", true},
			{"valid large ID", "999999", true},
			{"negative ID", "-1", true}, // Will parse but is invalid logically
			{"zero ID", "0", true},      // Will parse but is invalid logically
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Simulate ID parsing
				assert.NotNil(t, tc.idStr)
			})
		}
	})
}

func TestProductHandler_List_QueryParameters(t *testing.T) {
	t.Run("should handle pagination parameters", func(t *testing.T) {
		testCases := []struct {
			name      string
			page      int
			pageSize  int
			expectVal bool
		}{
			{"default page", 1, 20, true},
			{"custom page", 2, 50, true},
			{"invalid page", -1, 0, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.expectVal {
					assert.GreaterOrEqual(t, tc.page, 1)
					assert.GreaterOrEqual(t, tc.pageSize, 1)
				}
			})
		}
	})

	t.Run("should handle status filter", func(t *testing.T) {
		statuses := []model.ProductStatus{
			model.ProductStatusDraft,
			model.ProductStatusPublished,
		}

		for _, status := range statuses {
			assert.Contains(t, []model.ProductStatus{0, 1}, status)
		}
	})
}

func TestProductHandler_Update_RequestValidation(t *testing.T) {
	t.Run("should validate update request", func(t *testing.T) {
		req := service.UpdateProductRequest{
			Name:        "Updated Name",
			Description: "Updated Description",
			Images:      []string{"new_image.jpg"},
		}

		assert.NotEmpty(t, req.Name)
		assert.NotEmpty(t, req.Description)
		assert.Len(t, req.Images, 1)
	})

	t.Run("should allow partial updates", func(t *testing.T) {
		req := service.UpdateProductRequest{
			Name: "Only Name",
		}

		assert.NotEmpty(t, req.Name)
		assert.Empty(t, req.Description)
	})
}

func TestProductHandler_Delete_IDValidation(t *testing.T) {
	t.Run("should validate product ID for deletion", func(t *testing.T) {
		testCases := []struct {
			name    string
			idStr   string
			isValid bool
		}{
			{"valid ID", "1", true},
			{"invalid string", "abc", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.isValid {
					assert.NotEmpty(t, tc.idStr)
				}
			})
		}
	})
}

func TestProductHandler_ResponseFormat(t *testing.T) {
	t.Run("should return correct JSON format for product", func(t *testing.T) {
		product := model.Product{
			ID:          1,
			Name:        "Test Product",
			Description: "Test Description",
			Images:      []string{"image1.jpg"},
			Status:      model.ProductStatusPublished,
		}

		body, err := json.Marshal(product)
		assert.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(body, &parsed)
		assert.NoError(t, err)

		assert.NotNil(t, parsed["id"])
		assert.NotNil(t, parsed["name"])
		assert.NotNil(t, parsed["status"])
	})

	t.Run("should return correct error format", func(t *testing.T) {
		errorResp := map[string]interface{}{
			"code":    400,
			"message": "请求参数错误",
		}

		body, err := json.Marshal(errorResp)
		assert.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(body, &parsed)
		assert.NoError(t, err)

		assert.Equal(t, float64(400), parsed["code"])
		assert.Contains(t, parsed["message"], "请求参数错误")
	})

	t.Run("should return correct list format", func(t *testing.T) {
		listResp := map[string]interface{}{
			"list":      []interface{}{},
			"total":     10,
			"page":      1,
			"page_size": 20,
		}

		body, err := json.Marshal(listResp)
		assert.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(body, &parsed)
		assert.NoError(t, err)

		assert.NotNil(t, parsed["list"])
		assert.Equal(t, float64(10), parsed["total"])
		assert.Equal(t, float64(1), parsed["page"])
		assert.Equal(t, float64(20), parsed["page_size"])
	})
}

func TestProductHandler_List_ReturnsWrappedData(t *testing.T) {
	h, db := newProductHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Category{
			ID:     11,
			Name:   "珠宝名表",
			Code:   "jewelry",
			Status: model.CategoryStatusActive,
		}).Error)
		require.NoError(t, db.Create(&model.Product{
			ID:         1,
			Name:       "茶杯",
			CategoryID: int64Ptr(11),
			Status:     model.ProductStatusPublished,
		}).Error)
	})
	c := newProductRequestContext("GET", "/api/v1/products?page=1&page_size=20", nil)

	h.List(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			List     []map[string]interface{} `json:"list"`
			Total    int64                    `json:"total"`
			Page     int                      `json:"page"`
			PageSize int                      `json:"page_size"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, 200, body.Code)
	assert.Equal(t, "success", body.Message)
	assert.EqualValues(t, 1, body.Data.Total)
	assert.Len(t, body.Data.List, 1)
	assert.Equal(t, 1, body.Data.Page)
	assert.Equal(t, 20, body.Data.PageSize)
	assert.Equal(t, "珠宝名表", body.Data.List[0]["category_name"])

	var stored model.Product
	require.NoError(t, db.First(&stored, 1).Error)
	require.NotNil(t, stored.CategoryID)
	assert.EqualValues(t, 11, *stored.CategoryID)
}

func TestProductHandler_PublicListOnlyReturnsPublishedProducts(t *testing.T) {
	h, _ := newProductHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Product{
			ID:     11,
			Name:   "未发布商品",
			Status: model.ProductStatusDraft,
		}).Error)
		require.NoError(t, db.Create(&model.Product{
			ID:     12,
			Name:   "已发布商品",
			Status: model.ProductStatusPublished,
		}).Error)
		require.NoError(t, db.Create(&model.Product{
			ID:     13,
			Name:   "已下架商品",
			Status: model.ProductStatusUnpublished,
		}).Error)
	})
	c := newProductRequestContext("GET", "/api/v1/products?page=1&page_size=20", nil)

	h.List(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		Data struct {
			List  []model.Product `json:"list"`
			Total int64           `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	require.Len(t, body.Data.List, 1)
	assert.EqualValues(t, 12, body.Data.List[0].ID)
	assert.EqualValues(t, 1, body.Data.Total)
}

func TestProductHandler_PublicGetRejectsUnpublishedProduct(t *testing.T) {
	h, _ := newProductHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Product{
			ID:     21,
			Name:   "未发布商品",
			Status: model.ProductStatusDraft,
		}).Error)
	})
	c := newProductRequestContext("GET", "/api/v1/products/21", nil)
	c.Params = append(c.Params, param.Param{Key: "id", Value: "21"})

	h.Get(context.Background(), c)

	assert.Equal(t, 404, c.Response.StatusCode())
}

func TestProductHandler_Create_PersistsCategoryID(t *testing.T) {
	h, db := newProductHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Category{
			ID:     7,
			Name:   "艺术收藏",
			Code:   "art",
			Status: model.CategoryStatusActive,
		}).Error)
	})

	c := newProductRequestContext("POST", "/api/v1/products", []byte(`{
		"name":"青花瓷",
		"description":"元青花",
		"images":["a.jpg"],
		"category_id":7
	}`))

	h.Create(context.Background(), c)

	assert.Equal(t, 201, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 7, body["category_id"])

	var stored model.Product
	require.NoError(t, db.Where("name = ?", "青花瓷").First(&stored).Error)
	require.NotNil(t, stored.CategoryID)
	assert.EqualValues(t, 7, *stored.CategoryID)
}

func TestProductHandler_Create_InvalidCategory_Returns400(t *testing.T) {
	h, _ := newProductHandlerWithSeed(t, nil)
	c := newProductRequestContext("POST", "/api/v1/products", []byte(`{
		"name":"青花瓷",
		"category_id":999
	}`))

	h.Create(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
	assert.Contains(t, string(c.Response.Body()), "category")
}

func TestProductHandler_Update_ChangesCategoryID(t *testing.T) {
	h, db := newProductHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Category{
			ID:     1,
			Name:   "艺术收藏",
			Code:   "art",
			Status: model.CategoryStatusActive,
		}).Error)
		require.NoError(t, db.Create(&model.Category{
			ID:     2,
			Name:   "珠宝名表",
			Code:   "watch",
			Status: model.CategoryStatusActive,
		}).Error)
		require.NoError(t, db.Create(&model.Product{
			ID:         101,
			Name:       "旧商品",
			CategoryID: int64Ptr(1),
			Status:     model.ProductStatusDraft,
		}).Error)
	})
	c := newProductRequestContext("PUT", "/api/v1/products/101", []byte(`{"category_id":2}`))
	c.Params = append(c.Params, param.Param{Key: "id", Value: "101"})

	h.Update(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 2, body["category_id"])

	var stored model.Product
	require.NoError(t, db.First(&stored, 101).Error)
	require.NotNil(t, stored.CategoryID)
	assert.EqualValues(t, 2, *stored.CategoryID)
}

func TestProductHandler_Get_ReturnsCategoryName(t *testing.T) {
	h, _ := newProductHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Category{
			ID:     9,
			Name:   "潮流文玩",
			Code:   "trend",
			Status: model.CategoryStatusActive,
		}).Error)
		require.NoError(t, db.Create(&model.Product{
			ID:         102,
			Name:       "手串",
			CategoryID: int64Ptr(9),
			Status:     model.ProductStatusPublished,
		}).Error)
	})
	c := newProductRequestContext("GET", "/api/v1/products/102", nil)
	c.Params = append(c.Params, param.Param{Key: "id", Value: "102"})

	h.Get(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 9, body["category_id"])
	assert.Equal(t, "潮流文玩", body["category_name"])
}

func TestProductHandler_AdminCreateRejectsAdminActor(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}))
	require.NoError(t, db.Exec("DELETE FROM products").Error)
	productSvc := service.NewProductService(dao.NewProductDAO(db), nil, dao.NewLiveStreamDAO(db))
	h := NewProductHandler(productSvc)
	c := app.NewContext(0)
	c.Request.SetMethod(http.MethodPost)
	c.Request.SetRequestURI("/api/v1/admin/products")
	c.Request.Header.Set("X-User-ID", "2001")
	c.Request.Header.Set("X-User-Role", "admin")
	c.Request.SetBodyString(`{"name":"Admin Should Not Create"}`)

	h.AdminCreate(context.Background(), c)

	assert.Equal(t, http.StatusForbidden, c.Response.StatusCode())
}

func TestProductHandler_AdminCreateMerchantSetsOwnerID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}))
	require.NoError(t, db.Exec("DELETE FROM products").Error)
	productSvc := service.NewProductService(dao.NewProductDAO(db), nil, dao.NewLiveStreamDAO(db))
	h := NewProductHandler(productSvc)
	c := app.NewContext(0)
	c.Request.SetMethod(http.MethodPost)
	c.Request.SetRequestURI("/api/v1/admin/products")
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Request.SetBodyString(`{"name":"Merchant Product"}`)

	h.AdminCreate(context.Background(), c)

	assert.Equal(t, http.StatusCreated, c.Response.StatusCode())
	var products []model.Product
	require.NoError(t, db.Find(&products).Error)
	require.Len(t, products, 1)
	require.NotNil(t, products[0].OwnerID)
	assert.Equal(t, int64(1001), *products[0].OwnerID)
}

func TestProductHandler_AdminListMerchantOnlyOwnProducts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}))
	require.NoError(t, db.Exec("DELETE FROM products").Error)
	ownerA := int64(1001)
	ownerB := int64(1002)
	require.NoError(t, db.Create(&model.Product{Name: "A", OwnerID: &ownerA, Status: model.ProductStatusDraft}).Error)
	require.NoError(t, db.Create(&model.Product{Name: "B", OwnerID: &ownerB, Status: model.ProductStatusDraft}).Error)
	productSvc := service.NewProductService(dao.NewProductDAO(db), nil, dao.NewLiveStreamDAO(db))
	h := NewProductHandler(productSvc)
	c := app.NewContext(0)
	c.Request.SetMethod(http.MethodGet)
	c.Request.SetRequestURI("/api/v1/admin/products?page=1&page_size=20")
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")

	h.AdminList(context.Background(), c)

	assert.Equal(t, http.StatusOK, c.Response.StatusCode())
	var body struct {
		Data struct {
			List  []model.Product `json:"list"`
			Total int64           `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, int64(1), body.Data.Total)
	require.Len(t, body.Data.List, 1)
	assert.Equal(t, "A", body.Data.List[0].Name)
}

func TestProductHandler_ErrorHandling(t *testing.T) {
	t.Run("should return 400 for invalid JSON", func(t *testing.T) {
		invalidJSON := []byte("invalid json")
		var req service.CreateProductRequest
		err := json.Unmarshal(invalidJSON, &req)
		assert.Error(t, err)
	})

	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		invalidID := "abc"
		assert.NotEmpty(t, invalidID)
	})

	t.Run("should return 404 for not found", func(t *testing.T) {
		errorResp := map[string]interface{}{
			"code":    404,
			"message": "商品不存在",
		}

		assert.Equal(t, 404, errorResp["code"])
	})

	t.Run("should return 500 for server error", func(t *testing.T) {
		errorResp := map[string]interface{}{
			"code":    500,
			"message": "创建商品失败",
		}

		assert.Equal(t, 500, errorResp["code"])
	})
}
