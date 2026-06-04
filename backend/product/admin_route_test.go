package main

import (
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/dao"
	"product-service/handler"
	"product-service/model"
	"product-service/service"
)

func TestAdminOrderRoutesRequireInternalToken(t *testing.T) {
	t.Setenv("INTERNAL_API_TOKEN", "internal-secret")

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	orderHandler := handler.NewOrderHandler(nil)
	registerRoutes(
		h,
		handler.NewProductHandler(nil),
		handler.NewRuleHandler(nil),
		handler.NewAuctionRuleTemplateHandler(nil),
		orderHandler,
		handler.NewStatisticsHandler(nil),
		handler.NewProductHandler(nil),
		handler.NewLiveStreamHandler(nil),
		handler.NewCategoryHandler(nil),
		handler.NewCopywritingHandler(nil),
		handler.NewInternalHandler(nil, nil),
	)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/admin/orders", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode())
}

func TestProductAdminRoutesRequireInternalToken(t *testing.T) {
	t.Setenv("INTERNAL_API_TOKEN", "internal-secret")

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	registerRoutes(
		h,
		handler.NewProductHandler(nil),
		handler.NewRuleHandler(nil),
		handler.NewAuctionRuleTemplateHandler(nil),
		handler.NewOrderHandler(nil),
		handler.NewStatisticsHandler(nil),
		handler.NewProductHandler(nil),
		handler.NewLiveStreamHandler(nil),
		handler.NewCategoryHandler(nil),
		handler.NewCopywritingHandler(nil),
		handler.NewInternalHandler(nil, nil),
	)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/admin/products", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode())
}

func TestAuctionRuleTemplateAdminRoutesRequireInternalToken(t *testing.T) {
	t.Setenv("INTERNAL_API_TOKEN", "internal-secret")

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	registerRoutes(
		h,
		handler.NewProductHandler(nil),
		handler.NewRuleHandler(nil),
		handler.NewAuctionRuleTemplateHandler(nil),
		handler.NewOrderHandler(nil),
		handler.NewStatisticsHandler(nil),
		handler.NewProductHandler(nil),
		handler.NewLiveStreamHandler(nil),
		handler.NewCategoryHandler(nil),
		handler.NewCopywritingHandler(nil),
		handler.NewInternalHandler(nil, nil),
	)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/admin/auction-rule-templates", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode())
}

func TestStatisticsRoutesRequireInternalToken(t *testing.T) {
	t.Setenv("INTERNAL_API_TOKEN", "internal-secret")
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.Order{}, &model.User{}))
	statisticsHandler := handler.NewStatisticsHandler(service.NewStatisticsService(dao.NewStatisticsDAO(db)))

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	registerRoutes(
		h,
		handler.NewProductHandler(nil),
		handler.NewRuleHandler(nil),
		handler.NewAuctionRuleTemplateHandler(nil),
		handler.NewOrderHandler(nil),
		statisticsHandler,
		handler.NewProductHandler(nil),
		handler.NewLiveStreamHandler(nil),
		handler.NewCategoryHandler(nil),
		handler.NewCopywritingHandler(nil),
		handler.NewInternalHandler(nil, nil),
	)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/overview", nil,
		ut.Header{Key: "X-User-ID", Value: "1001"},
		ut.Header{Key: "X-User-Role", Value: "admin"})

	assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode())
}
