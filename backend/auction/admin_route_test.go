package main

import (
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/dao"
	"auction-service/handler"
	"auction-service/model"
	"auction-service/service"
)

func TestAuctionAdminRoutesRequireInternalToken(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))
	svc := service.NewAuctionService(dao.NewAuctionDAO(db))
	ownerID := int64(1001)
	_, err = svc.CreateAuction(t.Context(), &service.CreateAuctionRequest{
		ProductID:      1,
		CreatorID:      &ownerID,
		Duration:       3600,
		ProductOwnerID: ownerID,
		ProductStatus:  1,
		RuleBound:      true,
		LiveStreamID:   1,
	})
	require.NoError(t, err)

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	registerRoutes(
		h,
		"internal-secret",
		handler.NewAuctionHandler(svc),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/admin/auctions", nil,
		ut.Header{Key: "X-User-ID", Value: "1001"},
		ut.Header{Key: "X-User-Role", Value: "merchant"})

	require.Equal(t, http.StatusUnauthorized, w.Result().StatusCode())
}
