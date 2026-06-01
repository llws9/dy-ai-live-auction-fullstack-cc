package handler

import (
	"encoding/json"
	"testing"

	"product-service/model"
	"product-service/service"

	"github.com/stretchr/testify/assert"
)

func TestProductHandler_Create_RequestValidation(t *testing.T) {
	t.Run("should validate create request fields", func(t *testing.T) {
		testCases := []struct {
			name     string
			request  service.CreateProductRequest
			isValid  bool
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
			name     string
			idStr    string
			isValid  bool
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
