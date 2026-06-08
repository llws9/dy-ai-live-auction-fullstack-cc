package service

import (
	"context"
	"errors"
	"fmt"
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

const heartbeatDeltaScript = `
local last = redis.call("GET", KEYS[1])
local now = tonumber(ARGV[1])
local cap = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])
local delta = cap

if last then
	local lastUnix = tonumber(last)
	if lastUnix then
		local elapsed = now - lastUnix
		if elapsed < 0 then
			elapsed = 0
		end
		if elapsed < delta then
			delta = elapsed
		end
	end
end

redis.call("SET", KEYS[1], now, "EX", ttl)
return delta
`

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

func heartbeatKey(userID int64, statDate string) string {
	return fmt.Sprintf("treasure:hb:%d:%s", userID, statDate)
}

func (s *TreasureService) Heartbeat(ctx context.Context, userID int64) (int, error) {
	if userID <= 0 {
		return 0, errors.New("invalid user_id")
	}

	now := auctionBusinessNow()
	statDate := now.In(auctionBusinessLocation).Format("2006-01-02")
	key := heartbeatKey(userID, statDate)

	delta, err := s.rdb.Eval(ctx, heartbeatDeltaScript, []string{key},
		now.Unix(),
		heartbeatCapSeconds,
		int(heartbeatTTL/time.Second),
	).Int()
	if err != nil {
		return 0, err
	}

	if delta <= 0 {
		return s.dao.GetWatchSeconds(ctx, userID, statDate)
	}
	return s.dao.AddWatchSeconds(ctx, userID, statDate, delta)
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
