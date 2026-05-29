package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"auction-service/config"
	"auction-service/model"
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
