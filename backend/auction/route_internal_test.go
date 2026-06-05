package main

import (
	"context"
	"net/http"
	"testing"

	"auction-service/handler"
	"auction-service/middleware"
	"auction-service/model"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type routeReminderProvider struct{}

func (p *routeReminderProvider) GetPendingReminder(ctx context.Context, userID int64) (*model.PendingLiveReminderResponse, error) {
	return &model.PendingLiveReminderResponse{HasReminder: false}, nil
}

func TestPendingReminderPublicRouteDoesNotTrustForgedUserHeader(t *testing.T) {
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	h.Use(gatewayIdentityMiddleware())

	registerRoutes(
		h,
		"internal-secret",
		&handler.AuctionHandler{},
		&handler.BidHandler{},
		&handler.WSHandler{},
		&handler.UserHandler{},
		&handler.AuthHandler{},
		&handler.NotificationHandler{},
		&handler.FollowHandler{},
		&handler.ProductReminderHandler{},
		&handler.SkyLampHandler{},
		&handler.UserBalanceHandler{},
		&handler.UserAddressHandler{},
		handler.NewLiveReminderHandler(&routeReminderProvider{}),
		&handler.LiveStreamStatsHandler{},
	)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/api/v1/live/pending-reminder",
		nil,
		ut.Header{Key: "X-User-ID", Value: "999"},
	)

	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode())
}

func TestPendingReminderInternalRouteRequiresInternalToken(t *testing.T) {
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	h.Use(gatewayIdentityMiddleware())
	registerInternalRoutes(
		h,
		middleware.InternalAuthMiddleware("internal-secret"),
		nil,
		nil,
		handler.NewLiveReminderHandler(&routeReminderProvider{}),
		&handler.LiveStreamStatsHandler{},
		nil,
	)

	unauthorized := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/internal/live/pending-reminder",
		nil,
		ut.Header{Key: "X-User-ID", Value: "999"},
	)
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Result().StatusCode())

	authorized := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/internal/live/pending-reminder",
		nil,
		ut.Header{Key: "X-Internal-Token", Value: "internal-secret"},
		ut.Header{Key: "X-User-ID", Value: "999"},
	)
	require.Equal(t, http.StatusOK, authorized.Result().StatusCode())
}
