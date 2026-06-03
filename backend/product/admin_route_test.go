package main

import (
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"

	"product-service/handler"
)

func TestAdminOrderRoutesRequireInternalToken(t *testing.T) {
	t.Setenv("INTERNAL_API_TOKEN", "internal-secret")

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	orderHandler := handler.NewOrderHandler(nil)
	registerRoutes(
		h,
		handler.NewProductHandler(nil),
		handler.NewRuleHandler(nil),
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
