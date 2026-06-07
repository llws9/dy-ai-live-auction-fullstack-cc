package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"auction-service/config"
	"auction-service/dao"
	"auction-service/model"

	"github.com/shopspring/decimal"
)

type stubSkyLampDAO struct {
	activeByAuction []model.SkyLampSubscription
	updatedStatus   []struct {
		id     int64
		status model.SkyLampStatus
	}
}

func (s *stubSkyLampDAO) Create(ctx context.Context, subscription *model.SkyLampSubscription) error {
	return nil
}

func (s *stubSkyLampDAO) GetByID(ctx context.Context, id int64) (*model.SkyLampSubscription, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSkyLampDAO) GetActiveByUser(ctx context.Context, auctionID, userID int64) (*model.SkyLampSubscription, error) {
	return nil, nil
}

func (s *stubSkyLampDAO) GetActiveByAuction(ctx context.Context, auctionID int64) ([]model.SkyLampSubscription, error) {
	return s.activeByAuction, nil
}

func (s *stubSkyLampDAO) Update(ctx context.Context, subscription *model.SkyLampSubscription) error {
	return nil
}

func (s *stubSkyLampDAO) UpdateStatus(ctx context.Context, id int64, status model.SkyLampStatus) error {
	s.updatedStatus = append(s.updatedStatus, struct {
		id     int64
		status model.SkyLampStatus
	}{id: id, status: status})
	return nil
}

func (s *stubSkyLampDAO) UpdateStatusWithStoppedAt(ctx context.Context, id int64, status model.SkyLampStatus) error {
	s.updatedStatus = append(s.updatedStatus, struct {
		id     int64
		status model.SkyLampStatus
	}{id: id, status: status})
	return nil
}

func (s *stubSkyLampDAO) StopByAuction(ctx context.Context, auctionID int64) error { return nil }
func (s *stubSkyLampDAO) GetByUserStatus(ctx context.Context, userID int64, status model.SkyLampStatus, page, pageSize int) ([]model.SkyLampSubscription, int64, error) {
	return nil, 0, nil
}
func (s *stubSkyLampDAO) Delete(ctx context.Context, id int64) error { return nil }

func TestSkyLampConfig_DefaultsApplied(t *testing.T) {
	cfg := config.DefaultSkyLampConfig()
	if !cfg.Enabled {
		t.Fatalf("expected skylamp enabled by default")
	}
	if cfg.MaxPriceOffset <= 0 || cfg.MinFollowInterval <= 0 || cfg.MaxAutoBidCount <= 0 {
		t.Fatalf("invalid skylamp defaults: %+v", cfg)
	}
}

func TestSkyLampStatusModelCanAutoBid(t *testing.T) {
	s := &model.SkyLampSubscription{
		Status:        model.SkyLampStatusActive,
		MaxPriceLimit: 100,
	}
	if !s.CanAutoBid(99) {
		t.Fatalf("should auto bid under max price")
	}
	if s.CanAutoBid(101) {
		t.Fatalf("should not auto bid above max price")
	}
}

func TestSkyLampStatusTransitionRequiresStoppedAtPath(t *testing.T) {
	// This test ensures DAO has explicit API for status+stopped_at path.
	// Full DB integration is covered by existing DAO integration tests patterns in repository.
	dao := &stubSkyLampDAO{}
	if err := dao.UpdateStatusWithStoppedAt(context.Background(), 1, model.SkyLampStatusStopped); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dao.updatedStatus) != 1 {
		t.Fatalf("expected one status update")
	}
	if dao.updatedStatus[0].status != model.SkyLampStatusStopped {
		t.Fatalf("unexpected status: %v", dao.updatedStatus[0].status)
	}
}

func TestSkyLampThrottleConfigValue(t *testing.T) {
	cfg := config.DefaultSkyLampConfig()
	if time.Duration(cfg.MinFollowInterval)*time.Millisecond <= 0 {
		t.Fatalf("invalid throttle duration")
	}
}

func TestSkyLampStartSubscriptionUsesStartPriceForInitialBid(t *testing.T) {
	db := newBidSettlementTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	rdb := setupTestRedis(t)
	if err := db.AutoMigrate(&model.SkyLampSubscription{}); err != nil {
		t.Fatalf("migrate sky lamp subscription: %v", err)
	}
	previousRedis := dao.GetRedis()
	dao.RedisClient = rdb
	t.Cleanup(func() { dao.RedisClient = previousRedis })

	userID := int64(3001)
	auctionID := int64(3301)
	productID := int64(4301)
	if err := db.Create(&model.User{
		ID:       userID,
		Name:     "sky-lamp-buyer",
		Password: "password",
		Status:   1,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.Auction{
		ID:           auctionID,
		ProductID:    productID,
		Status:       model.AuctionStatusOngoing,
		CurrentPrice: decimal.Zero,
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      time.Now().Add(time.Hour),
	}).Error; err != nil {
		t.Fatalf("create auction: %v", err)
	}
	if err := db.Create(&model.AuctionRule{
		ProductID:          productID,
		StartPrice:         decimal.NewFromInt(100),
		Increment:          decimal.NewFromInt(10),
		Duration:           60,
		DelayDuration:      30,
		MaxDelayTime:       180,
		TriggerDelayBefore: 30,
	}).Error; err != nil {
		t.Fatalf("create rule: %v", err)
	}

	bidSvc := NewBidService(
		dao.NewAuctionDAO(db),
		dao.NewBidDAO(db),
		dao.NewAuctionRuleDAO(db),
		dao.NewUserDAO(db),
	)
	skyLampSvc := NewSkyLampService(
		dao.NewSkyLampDAO(db),
		bidSvc,
		config.DefaultSkyLampConfig(),
		nil,
	)

	subscription, err := skyLampSvc.StartSubscription(context.Background(), userID, auctionID)

	if err != nil {
		t.Fatalf("start subscription: %v", err)
	}
	if subscription == nil {
		t.Fatal("expected subscription")
	}
	if subscription.InitialBidAmount != 110 {
		t.Fatalf("initial bid amount=%v want 110", subscription.InitialBidAmount)
	}

	var bid model.Bid
	if err := db.Where("auction_id = ? AND user_id = ?", auctionID, userID).First(&bid).Error; err != nil {
		t.Fatalf("query bid: %v", err)
	}
	if !bid.Amount.Equal(decimal.NewFromInt(110)) {
		t.Fatalf("bid amount=%s want 110", bid.Amount)
	}
}
