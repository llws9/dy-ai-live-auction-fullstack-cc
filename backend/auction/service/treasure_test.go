package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"auction-service/client"
	"auction-service/dao"
	"auction-service/model"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var treasureServiceDBCounter int64

type fakeLiveStreamLookup struct {
	items map[int64]client.LiveStreamSummary
	err   error
}

func (f fakeLiveStreamLookup) BatchGetLiveStreams(ctx context.Context, ids []int64) (map[int64]client.LiveStreamSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[int64]client.LiveStreamSummary, len(ids))
	for _, id := range ids {
		if item, ok := f.items[id]; ok {
			out[id] = item
		}
	}
	return out, nil
}

func setupTreasureService(t *testing.T) (*TreasureService, *dao.TreasureDAO, *redis.Client) {
	t.Helper()

	dsn := fmt.Sprintf("file:treasure_service_test_%d?mode=memory&cache=shared", atomic.AddInt64(&treasureServiceDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&model.UserCoin{},
		&model.UserWatchDuration{},
		&model.TreasureClaim{},
	))
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	d := dao.NewTreasureDAO(db)
	svc := NewTreasureService(d, rdb)
	svc.SetLiveStreamLookup(fakeLiveStreamLookup{items: map[int64]client.LiveStreamSummary{
		200: {ID: 200, Status: 1},
	}})
	return svc, d, rdb
}

func TestTreasureService_Heartbeat_FirstBeatRecords30s(t *testing.T) {
	svc, dd, _ := setupTreasureService(t)
	ctx := context.Background()

	total, err := svc.Heartbeat(ctx, 100, 200)

	require.NoError(t, err)
	assert.Equal(t, 30, total)
	secs, err := dd.GetWatchSeconds(ctx, 100, businessStatDate())
	require.NoError(t, err)
	assert.Equal(t, 30, secs)
}

func TestTreasureService_Heartbeat_ConcurrentFirstBeatDoesNotMultiplyWindow(t *testing.T) {
	svc, dd, _ := setupTreasureService(t)
	ctx := context.Background()

	const workers = 20
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, workers)
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.Heartbeat(ctx, 100, 200)
			errs <- err
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	secs, err := dd.GetWatchSeconds(ctx, 100, businessStatDate())
	require.NoError(t, err)
	assert.LessOrEqual(t, secs, heartbeatCapSeconds)
}

func TestTreasureService_HeartbeatKey_IncludesStatDate(t *testing.T) {
	const date = "2026-06-09"

	key := heartbeatKey(100, date)

	assert.Contains(t, key, date)
	assert.Equal(t, fmt.Sprintf("treasure:hb:100:%s", date), key)
}

func TestTreasureService_Heartbeat_ShortIntervalDoesNotAddFullBeat(t *testing.T) {
	svc, dd, rdb := setupTreasureService(t)
	ctx := context.Background()

	total, err := svc.Heartbeat(ctx, 100, 200)
	require.NoError(t, err)
	require.Equal(t, 30, total)
	require.NoError(t, rdb.Set(ctx, heartbeatKey(100, businessStatDate()), auctionBusinessNow().Add(10*time.Second).Unix(), 120*time.Second).Err())

	total, err = svc.Heartbeat(ctx, 100, 200)

	require.NoError(t, err)
	assert.Equal(t, 30, total)
	secs, err := dd.GetWatchSeconds(ctx, 100, businessStatDate())
	require.NoError(t, err)
	assert.Equal(t, 30, secs)
}

func TestTreasureService_Heartbeat_CapsAt30sPerBeat(t *testing.T) {
	svc, dd, rdb := setupTreasureService(t)
	ctx := context.Background()

	total, err := svc.Heartbeat(ctx, 100, 200)
	require.NoError(t, err)
	require.Equal(t, 30, total)
	require.NoError(t, rdb.Set(ctx, heartbeatKey(100, businessStatDate()), auctionBusinessNow().Add(-500*time.Second).Unix(), 120*time.Second).Err())

	total, err = svc.Heartbeat(ctx, 100, 200)

	require.NoError(t, err)
	assert.Equal(t, 60, total)
	secs, err := dd.GetWatchSeconds(ctx, 100, businessStatDate())
	require.NoError(t, err)
	assert.Equal(t, 60, secs)
}

func TestTreasureService_Heartbeat_RejectsMissingOrNonLiveStream(t *testing.T) {
	svc, dd, _ := setupTreasureService(t)
	ctx := context.Background()
	svc.SetLiveStreamLookup(fakeLiveStreamLookup{items: map[int64]client.LiveStreamSummary{
		201: {ID: 201, Status: 2},
	}})

	total, err := svc.Heartbeat(ctx, 100, 999)

	assert.ErrorIs(t, err, ErrLiveStreamNotLive)
	assert.Equal(t, 0, total)
	secs, err := dd.GetWatchSeconds(ctx, 100, businessStatDate())
	require.NoError(t, err)
	assert.Equal(t, 0, secs)

	total, err = svc.Heartbeat(ctx, 100, 201)

	assert.ErrorIs(t, err, ErrLiveStreamNotLive)
	assert.Equal(t, 0, total)
	secs, err = dd.GetWatchSeconds(ctx, 100, businessStatDate())
	require.NoError(t, err)
	assert.Equal(t, 0, secs)
}

func TestTreasureService_Heartbeat_LiveStreamAccumulates(t *testing.T) {
	svc, dd, _ := setupTreasureService(t)
	ctx := context.Background()

	total, err := svc.Heartbeat(ctx, 100, 200)

	require.NoError(t, err)
	assert.Equal(t, 30, total)
	secs, err := dd.GetWatchSeconds(ctx, 100, businessStatDate())
	require.NoError(t, err)
	assert.Equal(t, 30, secs)
}

func TestTreasureService_GetStatus_StateMachine(t *testing.T) {
	svc, dd, _ := setupTreasureService(t)
	ctx := context.Background()
	date := businessStatDate()

	_, err := dd.AddWatchSeconds(ctx, 100, date, 640)
	require.NoError(t, err)
	_, err = dd.ClaimTx(ctx, 100, date, 0, 100)
	require.NoError(t, err)

	st, err := svc.GetStatus(ctx, 100)

	require.NoError(t, err)
	assert.Equal(t, date, st.StatDate)
	assert.Equal(t, 640, st.WatchedSeconds)
	assert.Equal(t, int64(100), st.CoinBalance)
	require.Len(t, st.Tiers, 3)
	assert.Equal(t, "claimed", st.Tiers[0].State)
	assert.Equal(t, "unlockable", st.Tiers[1].State)
	assert.Equal(t, "locked", st.Tiers[2].State)
}

func TestTreasureService_GetStatus_JSONContract(t *testing.T) {
	svc, _, _ := setupTreasureService(t)
	ctx := context.Background()

	st, err := svc.GetStatus(ctx, 100)
	require.NoError(t, err)
	raw, err := json.Marshal(st)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Contains(t, got, "stat_date")
	assert.Contains(t, got, "watched_seconds")
	assert.Contains(t, got, "coin_balance")
	assert.Contains(t, got, "tiers")
	tier := got["tiers"].([]any)[0].(map[string]any)
	assert.Contains(t, tier, "tier")
	assert.Contains(t, tier, "threshold_seconds")
	assert.Contains(t, tier, "coins")
	assert.Contains(t, tier, "state")
}

func TestTreasureService_Claim_Success(t *testing.T) {
	svc, dd, _ := setupTreasureService(t)
	ctx := context.Background()
	_, err := dd.AddWatchSeconds(ctx, 100, businessStatDate(), 200)
	require.NoError(t, err)

	coins, balance, err := svc.Claim(ctx, 100, 0)

	require.NoError(t, err)
	assert.Equal(t, int64(100), coins)
	assert.Equal(t, int64(100), balance)
}

func TestTreasureService_Claim_RejectsWhenBelowThreshold(t *testing.T) {
	svc, _, _ := setupTreasureService(t)

	_, _, err := svc.Claim(context.Background(), 100, 0)

	assert.ErrorIs(t, err, ErrThresholdNotMet)
}

func TestTreasureService_Claim_RejectsInvalidTier(t *testing.T) {
	svc, _, _ := setupTreasureService(t)

	_, _, err := svc.Claim(context.Background(), 100, 9)

	assert.ErrorIs(t, err, ErrInvalidTier)
}

func TestTreasureService_Claim_DuplicateRejected(t *testing.T) {
	svc, dd, _ := setupTreasureService(t)
	ctx := context.Background()
	_, err := dd.AddWatchSeconds(ctx, 100, businessStatDate(), 200)
	require.NoError(t, err)

	_, _, err = svc.Claim(ctx, 100, 0)
	require.NoError(t, err)
	_, _, err = svc.Claim(ctx, 100, 0)

	assert.ErrorIs(t, err, dao.ErrAlreadyClaimed)
}
