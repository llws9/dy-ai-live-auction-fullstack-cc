package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"auction-service/dao"

	"github.com/redis/go-redis/v9"
)

type TierConfig struct {
	Tier             int8
	ThresholdSeconds int
	Coins            int64
}

var treasureTiers = []TierConfig{
	{Tier: 0, ThresholdSeconds: 180, Coins: 100},
	{Tier: 1, ThresholdSeconds: 600, Coins: 300},
	{Tier: 2, ThresholdSeconds: 1800, Coins: 800},
}

const (
	heartbeatCapSeconds = 30
	heartbeatTTL        = 120 * time.Second
)

var (
	ErrThresholdNotMet = errors.New("watch duration below threshold")
	ErrInvalidTier     = errors.New("invalid tier")
)

type TierStatus struct {
	Tier             int8   `json:"tier"`
	ThresholdSeconds int    `json:"threshold_seconds"`
	Coins            int64  `json:"coins"`
	State            string `json:"state"`
}

type TreasureStatus struct {
	StatDate       string       `json:"stat_date"`
	WatchedSeconds int          `json:"watched_seconds"`
	CoinBalance    int64        `json:"coin_balance"`
	Tiers          []TierStatus `json:"tiers"`
}

type TreasureService struct {
	dao *dao.TreasureDAO
	rdb *redis.Client
}

func NewTreasureService(d *dao.TreasureDAO, rdb *redis.Client) *TreasureService {
	return &TreasureService{dao: d, rdb: rdb}
}

func businessStatDate() string {
	return auctionBusinessNow().Format("2006-01-02")
}

func heartbeatKey(userID int64) string {
	return fmt.Sprintf("treasure:hb:%d", userID)
}

func (s *TreasureService) Heartbeat(ctx context.Context, userID int64) (int, error) {
	if userID <= 0 {
		return 0, errors.New("invalid user_id")
	}

	now := auctionBusinessNow()
	delta := heartbeatCapSeconds
	key := heartbeatKey(userID)

	lastStr, err := s.rdb.Get(ctx, key).Result()
	switch {
	case err == nil:
		lastUnix, parseErr := strconv.ParseInt(lastStr, 10, 64)
		if parseErr == nil {
			elapsed := int(now.Unix() - lastUnix)
			if elapsed < 0 {
				elapsed = 0
			}
			if elapsed < delta {
				delta = elapsed
			}
		}
	case errors.Is(err, redis.Nil):
	default:
		return 0, err
	}

	if err := s.rdb.Set(ctx, key, now.Unix(), heartbeatTTL).Err(); err != nil {
		return 0, err
	}

	date := businessStatDate()
	if delta <= 0 {
		return s.dao.GetWatchSeconds(ctx, userID, date)
	}
	return s.dao.AddWatchSeconds(ctx, userID, date, delta)
}

func (s *TreasureService) GetStatus(ctx context.Context, userID int64) (*TreasureStatus, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user_id")
	}

	date := businessStatDate()
	secs, err := s.dao.GetWatchSeconds(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	balance, err := s.dao.GetCoinBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	claimed, err := s.dao.ListClaimedTiers(ctx, userID, date)
	if err != nil {
		return nil, err
	}

	tiers := make([]TierStatus, 0, len(treasureTiers))
	for _, tier := range treasureTiers {
		state := "locked"
		if claimed[tier.Tier] {
			state = "claimed"
		} else if secs >= tier.ThresholdSeconds {
			state = "unlockable"
		}
		tiers = append(tiers, TierStatus{
			Tier:             tier.Tier,
			ThresholdSeconds: tier.ThresholdSeconds,
			Coins:            tier.Coins,
			State:            state,
		})
	}

	return &TreasureStatus{
		StatDate:       date,
		WatchedSeconds: secs,
		CoinBalance:    balance,
		Tiers:          tiers,
	}, nil
}

func (s *TreasureService) Claim(ctx context.Context, userID int64, tier int8) (int64, int64, error) {
	if userID <= 0 {
		return 0, 0, errors.New("invalid user_id")
	}

	cfg, ok := findTreasureTier(tier)
	if !ok {
		return 0, 0, ErrInvalidTier
	}

	date := businessStatDate()
	secs, err := s.dao.GetWatchSeconds(ctx, userID, date)
	if err != nil {
		return 0, 0, err
	}
	if secs < cfg.ThresholdSeconds {
		return 0, 0, ErrThresholdNotMet
	}

	balance, err := s.dao.ClaimTx(ctx, userID, date, tier, cfg.Coins)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Coins, balance, nil
}

func findTreasureTier(tier int8) (TierConfig, bool) {
	for _, cfg := range treasureTiers {
		if cfg.Tier == tier {
			return cfg, true
		}
	}
	return TierConfig{}, false
}
