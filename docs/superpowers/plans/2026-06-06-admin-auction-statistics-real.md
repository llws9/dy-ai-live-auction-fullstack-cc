# Admin Auction Statistics Real Data Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Connect the Admin `数据统计 -> 竞拍统计` page to real auction-service statistics while keeping the frontend API path unchanged.

**Architecture:** Gateway keeps `/api/v1/statistics/auctions` as the only frontend entry and routes it to auction-service instead of product-service. Auction-service owns the real aggregation from `auctions` and `bids`, enforces admin/merchant role scope from Gateway headers, and returns the array shape already expected by Admin frontend. Product-service statistics endpoints other than auction statistics remain unchanged.

**Tech Stack:** Go 1.24+, Hertz, GORM, MySQL, React, TypeScript, Recharts, existing Gateway/Product/Auction services.

---

## 0. Scope And Contract

### 0.1 External API

All frontend traffic goes through Gateway:

```http
GET /api/v1/statistics/auctions?start_date=2026-06-01&end_date=2026-06-07&group_by=day
Authorization: Bearer <admin-or-merchant-jwt>
```

Successful response:

```json
[
  {
    "date": "2026-06-01",
    "auction_count": 12,
    "bid_count": 146,
    "avg_price": 1288.5,
    "success_rate": 83.3
  }
]
```

### 0.2 Role Scope

| Role header | Scope |
|---|---|
| `X-User-Role: admin` | all auctions |
| `X-User-Role: merchant` | `auctions.creator_id = X-User-ID` |
| other or missing role | `403` |

### 0.3 Statistics Rules

| Field | Rule |
|---|---|
| `date` | `DATE(auctions.start_time)` |
| `auction_count` | `COUNT(DISTINCT auctions.id)` |
| `bid_count` | `COUNT(bids.id)` over auctions in that date bucket |
| `avg_price` | average `auctions.current_price` for successful auctions only |
| `success_rate` | successful auctions / auction_count * 100 |

Successful auction:

```go
auction.status == model.AuctionStatusEnded && auction.WinnerID != nil
```

### 0.4 Files

Gateway:

- Modify: `backend/gateway/router/router.go`
- Modify: `backend/gateway/router/admin_statistics_route_test.go`

Auction service:

- Create: `backend/auction/dao/statistics.go`
- Create: `backend/auction/dao/statistics_test.go`
- Create: `backend/auction/service/statistics.go`
- Create: `backend/auction/service/statistics_test.go`
- Create: `backend/auction/handler/statistics.go`
- Create: `backend/auction/handler/statistics_test.go`
- Modify: `backend/auction/main.go`

Admin frontend:

- Modify: `frontend/admin/src/pages-new/Stats.tsx`
- Create: `frontend/admin/src/pages-new/__tests__/Stats.auctionStats.test.tsx`

Docs:

- Already created: `docs/superpowers/specs/2026-06-06-admin-auction-statistics-real-design.md`
- Already created: `docs/superpowers/plans/2026-06-06-admin-auction-statistics-real.md`

---

## 1. Task: Gateway Routes Auction Statistics To Auction Service

**Files:**

- Modify: `backend/gateway/router/admin_statistics_route_test.go`
- Modify: `backend/gateway/router/router.go`

- [ ] **Step 1: Write the failing route test**

Replace `TestStatisticsRoutesRoleScope` in `backend/gateway/router/admin_statistics_route_test.go` with tests that distinguish product-service and auction-service calls:

```go
func TestStatisticsAuctionRouteUsesAuctionService(t *testing.T) {
	var productCalls atomic.Int64
	var auctionCalls atomic.Int64
	var auctionPath atomic.Value
	var auctionQuery atomic.Value
	var auctionRole atomic.Value
	var auctionUserID atomic.Value
	var auctionInternalToken atomic.Value
	auctionPath.Store("")
	auctionQuery.Store("")
	auctionRole.Store("")
	auctionUserID.Store("")
	auctionInternalToken.Store("")

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		productCalls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer productMock.Close()
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auctionCalls.Add(1)
		auctionPath.Store(r.URL.Path)
		auctionQuery.Store(r.URL.RawQuery)
		auctionRole.Store(r.Header.Get("X-User-Role"))
		auctionUserID.Store(r.Header.Get("X-User-ID"))
		auctionInternalToken.Store(r.Header.Get("X-Internal-Token"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			ProductURL:    productMock.URL,
			AuctionURL:    auctionMock.URL,
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "statistics-route-secret"},
	}
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)
	merchantToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9, "merchant", 1, 24)
	assert.NoError(t, err)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/auctions?start_date=2026-06-01&end_date=2026-06-07&group_by=day", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken})

	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int64(0), productCalls.Load())
	assert.Equal(t, int64(1), auctionCalls.Load())
	assert.Equal(t, "/api/v1/statistics/auctions", auctionPath.Load().(string))
	assert.Equal(t, "start_date=2026-06-01&end_date=2026-06-07&group_by=day", auctionQuery.Load().(string))
	assert.Equal(t, "merchant", auctionRole.Load().(string))
	assert.Equal(t, "9", auctionUserID.Load().(string))
	assert.Equal(t, "internal-secret", auctionInternalToken.Load().(string))
}
```

Keep an additional test for product-owned statistics:

```go
func TestStatisticsNonAuctionRoutesStillUseProductService(t *testing.T) {
	var productCalls atomic.Int64
	var auctionCalls atomic.Int64
	var lastProductPath atomic.Value
	lastProductPath.Store("")

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		productCalls.Add(1)
		lastProductPath.Store(r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer productMock.Close()
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auctionCalls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			ProductURL:    productMock.URL,
			AuctionURL:    auctionMock.URL,
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "statistics-route-secret"},
	}
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)
	adminToken, err := middleware.GenerateToken(cfg.JWT.Secret, 99, "admin", 2, 24)
	assert.NoError(t, err)

	for _, path := range []string{"/api/v1/statistics/overview", "/api/v1/statistics/revenue", "/api/v1/statistics/users"} {
		productCalls.Store(0)
		auctionCalls.Store(0)
		w := ut.PerformRequest(h.Engine, http.MethodGet, path, nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + adminToken})
		assert.Equal(t, http.StatusOK, w.Result().StatusCode(), path)
		assert.Equal(t, int64(1), productCalls.Load(), path)
		assert.Equal(t, int64(0), auctionCalls.Load(), path)
		assert.Equal(t, path, lastProductPath.Load().(string))
	}
}
```

- [ ] **Step 2: Run the failing route test**

Run:

```bash
cd backend/gateway
go test ./router -run 'TestStatistics(AuctionRouteUsesAuctionService|NonAuctionRoutesStillUseProductService)' -count=1
```

Expected: `TestStatisticsAuctionRouteUsesAuctionService` fails because `/statistics/auctions` still calls product-service.

- [ ] **Step 3: Change Gateway route**

In `backend/gateway/router/router.go`, change only this route:

```go
authGroup.GET("/statistics/auctions", middleware.RequireMerchantOrAdmin(), adminAuctionProxy.Forward)
```

Keep these routes on product-service:

```go
authGroup.GET("/statistics/overview", middleware.RequireMerchantOrAdmin(), adminProductProxy.Forward)
authGroup.GET("/statistics/revenue", middleware.RequireMerchantOrAdmin(), adminProductProxy.Forward)
authGroup.GET("/statistics/users", middleware.RequireAdmin(), adminProductProxy.Forward)
```

- [ ] **Step 4: Verify Gateway tests pass**

Run:

```bash
cd backend/gateway
go test ./router -run 'TestStatistics(AuctionRouteUsesAuctionService|NonAuctionRoutesStillUseProductService)' -count=1
```

Expected: both tests pass.

- [ ] **Step 5: Commit Gateway route change**

```bash
git add backend/gateway/router/router.go backend/gateway/router/admin_statistics_route_test.go
git commit -m "feat: route auction statistics to auction service"
```

---

## 2. Task: Add Auction Statistics DAO

**Files:**

- Create: `backend/auction/dao/statistics.go`
- Create: `backend/auction/dao/statistics_test.go`

- [ ] **Step 1: Write DAO tests**

Create `backend/auction/dao/statistics_test.go`:

```go
package dao

import (
	"context"
	"testing"
	"time"

	"auction-service/model"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestStatisticsDAOListAuctionDailyStats(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Auction{}, &model.Bid{}))
	dao := NewStatisticsDAO(db)
	ctx := context.Background()
	day1 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	winnerID := int64(2001)
	merchantID := int64(9)
	otherMerchantID := int64(10)

	auctions := []model.Auction{
		{ID: 101, CreatorID: &merchantID, ProductID: 1, Status: model.AuctionStatusEnded, WinnerID: &winnerID, CurrentPrice: decimal.NewFromInt(120), StartTime: day1, EndTime: day1.Add(time.Hour)},
		{ID: 102, CreatorID: &merchantID, ProductID: 2, Status: model.AuctionStatusEnded, WinnerID: nil, CurrentPrice: decimal.NewFromInt(80), StartTime: day1, EndTime: day1.Add(time.Hour)},
		{ID: 103, CreatorID: &merchantID, ProductID: 3, Status: model.AuctionStatusOngoing, WinnerID: nil, CurrentPrice: decimal.NewFromInt(50), StartTime: day2, EndTime: day2.Add(time.Hour)},
		{ID: 104, CreatorID: &otherMerchantID, ProductID: 4, Status: model.AuctionStatusEnded, WinnerID: &winnerID, CurrentPrice: decimal.NewFromInt(500), StartTime: day1, EndTime: day1.Add(time.Hour)},
	}
	require.NoError(t, db.Create(&auctions).Error)
	bids := []model.Bid{
		{AuctionID: 101, UserID: 1, Amount: decimal.NewFromInt(100), CreatedAt: day1},
		{AuctionID: 101, UserID: 2, Amount: decimal.NewFromInt(120), CreatedAt: day1},
		{AuctionID: 102, UserID: 3, Amount: decimal.NewFromInt(80), CreatedAt: day1},
		{AuctionID: 104, UserID: 4, Amount: decimal.NewFromInt(500), CreatedAt: day1},
	}
	require.NoError(t, db.Create(&bids).Error)

	rows, err := dao.ListAuctionDailyStats(ctx, day1, day2.AddDate(0, 0, 1), &merchantID)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	require.Equal(t, "2026-06-01", rows[0].Date)
	require.Equal(t, int64(2), rows[0].AuctionCount)
	require.Equal(t, int64(3), rows[0].BidCount)
	require.Equal(t, int64(1), rows[0].SuccessCount)
	require.Equal(t, 120.0, rows[0].AvgPrice)

	require.Equal(t, "2026-06-02", rows[1].Date)
	require.Equal(t, int64(1), rows[1].AuctionCount)
	require.Equal(t, int64(0), rows[1].BidCount)
	require.Equal(t, int64(0), rows[1].SuccessCount)
	require.Equal(t, 0.0, rows[1].AvgPrice)
}
```

- [ ] **Step 2: Run the failing DAO test**

Run:

```bash
cd backend/auction
go test ./dao -run TestStatisticsDAOListAuctionDailyStats -count=1
```

Expected: FAIL because `NewStatisticsDAO` and `ListAuctionDailyStats` do not exist.

- [ ] **Step 3: Implement DAO**

Create `backend/auction/dao/statistics.go`:

```go
package dao

import (
	"context"
	"time"

	"auction-service/model"

	"gorm.io/gorm"
)

type StatisticsDAO struct {
	db *gorm.DB
}

func NewStatisticsDAO(db *gorm.DB) *StatisticsDAO {
	return &StatisticsDAO{db: db}
}

type AuctionDailyStatsRow struct {
	Date         string  `gorm:"column:date"`
	AuctionCount int64  `gorm:"column:auction_count"`
	BidCount     int64  `gorm:"column:bid_count"`
	SuccessCount int64  `gorm:"column:success_count"`
	AvgPrice     float64 `gorm:"column:avg_price"`
}

func (d *StatisticsDAO) ListAuctionDailyStats(ctx context.Context, startInclusive, endExclusive time.Time, creatorID *int64) ([]AuctionDailyStatsRow, error) {
	query := d.db.WithContext(ctx).
		Table("auctions AS a").
		Select(`
			DATE(a.start_time) AS date,
			COUNT(DISTINCT a.id) AS auction_count,
			COUNT(b.id) AS bid_count,
			SUM(CASE WHEN a.status = ? AND a.winner_id IS NOT NULL THEN 1 ELSE 0 END) AS success_count,
			COALESCE(AVG(CASE WHEN a.status = ? AND a.winner_id IS NOT NULL THEN a.current_price END), 0) AS avg_price
		`, model.AuctionStatusEnded, model.AuctionStatusEnded).
		Joins("LEFT JOIN bids AS b ON b.auction_id = a.id").
		Where("a.start_time >= ? AND a.start_time < ?", startInclusive, endExclusive)

	if creatorID != nil {
		query = query.Where("a.creator_id = ?", *creatorID)
	}

	var rows []AuctionDailyStatsRow
	err := query.Group("DATE(a.start_time)").
		Order("DATE(a.start_time) ASC").
		Scan(&rows).Error
	return rows, err
}
```

- [ ] **Step 4: Verify DAO test passes**

Run:

```bash
cd backend/auction
go test ./dao -run TestStatisticsDAOListAuctionDailyStats -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit DAO**

```bash
git add backend/auction/dao/statistics.go backend/auction/dao/statistics_test.go
git commit -m "feat: add auction statistics dao"
```

---

## 3. Task: Add Auction Statistics Service

**Files:**

- Create: `backend/auction/service/statistics.go`
- Create: `backend/auction/service/statistics_test.go`

- [ ] **Step 1: Write service tests**

Create `backend/auction/service/statistics_test.go`:

```go
package service

import (
	"testing"
	"time"

	"auction-service/dao"

	"github.com/stretchr/testify/require"
)

func TestAuctionStatisticsBuildSeries(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
	rows := []dao.AuctionDailyStatsRow{
		{Date: "2026-06-01", AuctionCount: 2, BidCount: 3, SuccessCount: 1, AvgPrice: 120},
		{Date: "2026-06-03", AuctionCount: 1, BidCount: 0, SuccessCount: 0, AvgPrice: 0},
	}

	stats := buildAuctionDailySeries(start, end, rows)

	require.Len(t, stats, 3)
	require.Equal(t, "2026-06-01", stats[0].Date)
	require.Equal(t, int64(2), stats[0].AuctionCount)
	require.Equal(t, int64(3), stats[0].BidCount)
	require.Equal(t, 50.0, stats[0].SuccessRate)
	require.Equal(t, 120.0, stats[0].AvgPrice)
	require.Equal(t, "2026-06-02", stats[1].Date)
	require.Equal(t, int64(0), stats[1].AuctionCount)
	require.Equal(t, 0.0, stats[1].SuccessRate)
	require.Equal(t, "2026-06-03", stats[2].Date)
}

func TestValidateAuctionStatisticsRange(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	require.NoError(t, validateAuctionStatisticsRange(start, end))

	require.Error(t, validateAuctionStatisticsRange(end, start))
	require.Error(t, validateAuctionStatisticsRange(start, start.AddDate(0, 0, 91)))
}
```

- [ ] **Step 2: Run the failing service tests**

Run:

```bash
cd backend/auction
go test ./service -run 'TestAuctionStatisticsBuildSeries|TestValidateAuctionStatisticsRange' -count=1
```

Expected: FAIL because service functions and response type do not exist.

- [ ] **Step 3: Implement service**

Create `backend/auction/service/statistics.go`:

```go
package service

import (
	"context"
	"errors"
	"math"
	"time"

	"auction-service/dao"
)

var ErrInvalidStatisticsRange = errors.New("invalid statistics date range")

type AuctionDailyStat struct {
	Date         string  `json:"date"`
	AuctionCount int64  `json:"auction_count"`
	BidCount     int64  `json:"bid_count"`
	AvgPrice     float64 `json:"avg_price"`
	SuccessRate  float64 `json:"success_rate"`
}

type StatisticsService struct {
	statisticsDAO *dao.StatisticsDAO
}

func NewStatisticsService(statisticsDAO *dao.StatisticsDAO) *StatisticsService {
	return &StatisticsService{statisticsDAO: statisticsDAO}
}

func (s *StatisticsService) GetAuctionDailyStats(ctx context.Context, startDate, endDate time.Time, creatorID *int64) ([]AuctionDailyStat, error) {
	if err := validateAuctionStatisticsRange(startDate, endDate); err != nil {
		return nil, err
	}
	endExclusive := endDate.AddDate(0, 0, 1)
	rows, err := s.statisticsDAO.ListAuctionDailyStats(ctx, startDate, endExclusive, creatorID)
	if err != nil {
		return nil, err
	}
	return buildAuctionDailySeries(startDate, endDate, rows), nil
}

func validateAuctionStatisticsRange(startDate, endDate time.Time) error {
	if startDate.After(endDate) {
		return ErrInvalidStatisticsRange
	}
	if int(endDate.Sub(startDate).Hours()/24) > 90 {
		return ErrInvalidStatisticsRange
	}
	return nil
}

func buildAuctionDailySeries(startDate, endDate time.Time, rows []dao.AuctionDailyStatsRow) []AuctionDailyStat {
	byDate := make(map[string]dao.AuctionDailyStatsRow, len(rows))
	for _, row := range rows {
		byDate[row.Date] = row
	}

	stats := make([]AuctionDailyStat, 0, int(endDate.Sub(startDate).Hours()/24)+1)
	for day := startDate; !day.After(endDate); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		row := byDate[date]
		successRate := 0.0
		if row.AuctionCount > 0 {
			successRate = round1(float64(row.SuccessCount) / float64(row.AuctionCount) * 100)
		}
		stats = append(stats, AuctionDailyStat{
			Date:         date,
			AuctionCount: row.AuctionCount,
			BidCount:     row.BidCount,
			AvgPrice:     row.AvgPrice,
			SuccessRate:  successRate,
		})
	}
	return stats
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
```

- [ ] **Step 4: Verify service tests pass**

Run:

```bash
cd backend/auction
go test ./service -run 'TestAuctionStatisticsBuildSeries|TestValidateAuctionStatisticsRange' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit service**

```bash
git add backend/auction/service/statistics.go backend/auction/service/statistics_test.go
git commit -m "feat: add auction statistics service"
```

---

## 4. Task: Add Auction Statistics HTTP Handler

**Files:**

- Create: `backend/auction/handler/statistics.go`
- Create: `backend/auction/handler/statistics_test.go`
- Modify: `backend/auction/main.go`

- [ ] **Step 1: Write handler tests**

Create `backend/auction/handler/statistics_test.go`:

```go
package handler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/service"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestStatisticsHandlerGetAuctionStatistics(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Auction{}, &model.Bid{}))
	merchantID := int64(9)
	winnerID := int64(2001)
	start := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	require.NoError(t, db.Create(&model.Auction{
		ID: 101, CreatorID: &merchantID, ProductID: 1, Status: model.AuctionStatusEnded,
		WinnerID: &winnerID, CurrentPrice: decimal.NewFromInt(120), StartTime: start, EndTime: start.Add(time.Hour),
	}).Error)
	require.NoError(t, db.Create(&model.Bid{AuctionID: 101, UserID: 1, Amount: decimal.NewFromInt(120), CreatedAt: start}).Error)

	statisticsDAO := dao.NewStatisticsDAO(db)
	statisticsService := service.NewStatisticsService(statisticsDAO)
	statisticsHandler := NewStatisticsHandler(statisticsService)
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	h.GET("/api/v1/statistics/auctions", statisticsHandler.GetAuctionStatistics)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/auctions?start_date=2026-06-01&end_date=2026-06-01", nil,
		ut.Header{Key: "X-User-Role", Value: "merchant"},
		ut.Header{Key: "X-User-ID", Value: "9"})

	require.Equal(t, http.StatusOK, w.Result().StatusCode())
	require.Contains(t, string(w.Body.Bytes()), `"date":"2026-06-01"`)
	require.Contains(t, string(w.Body.Bytes()), `"auction_count":1`)
	require.Contains(t, string(w.Body.Bytes()), `"bid_count":1`)
	require.Contains(t, string(w.Body.Bytes()), `"success_rate":100`)
}

func TestStatisticsHandlerRejectsInvalidRoleAndRange(t *testing.T) {
	db := setupTestDB(t)
	statisticsHandler := NewStatisticsHandler(service.NewStatisticsService(dao.NewStatisticsDAO(db)))
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	h.GET("/api/v1/statistics/auctions", statisticsHandler.GetAuctionStatistics)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/auctions", nil,
		ut.Header{Key: "X-User-Role", Value: "user"},
		ut.Header{Key: "X-User-ID", Value: "1"})
	require.Equal(t, http.StatusForbidden, w.Result().StatusCode())

	w = ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/auctions?start_date=2026-06-07&end_date=2026-06-01", nil,
		ut.Header{Key: "X-User-Role", Value: "admin"},
		ut.Header{Key: "X-User-ID", Value: "99"})
	require.Equal(t, http.StatusBadRequest, w.Result().StatusCode())
}

func TestParseStatisticsDateDefaults(t *testing.T) {
	now := time.Date(2026, 6, 7, 15, 30, 0, 0, time.UTC)
	start, end := defaultAuctionStatisticsRange(now)
	require.Equal(t, "2026-06-01", start.Format("2006-01-02"))
	require.Equal(t, "2026-06-07", end.Format("2006-01-02"))
}
```

- [ ] **Step 2: Run the failing handler tests**

Run:

```bash
cd backend/auction
go test ./handler -run 'TestStatisticsHandler|TestParseStatisticsDateDefaults' -count=1
```

Expected: FAIL because `StatisticsHandler` does not exist.

- [ ] **Step 3: Implement handler**

Create `backend/auction/handler/statistics.go`:

```go
package handler

import (
	"context"
	"errors"
	"strconv"
	"time"

	"auction-service/service"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	statisticsRoleAdmin    = "admin"
	statisticsRoleMerchant = "merchant"
)

type StatisticsHandler struct {
	statisticsService *service.StatisticsService
}

func NewStatisticsHandler(statisticsService *service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{statisticsService: statisticsService}
}

func (h *StatisticsHandler) GetAuctionStatistics(ctx context.Context, c *app.RequestContext) {
	creatorID, ok := readAuctionStatisticsScope(c)
	if !ok {
		return
	}
	if groupBy := c.Query("group_by"); groupBy != "" && groupBy != "day" {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "group_by only supports day"})
		return
	}

	startDate, endDate, err := parseAuctionStatisticsRange(c, time.Now())
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
		return
	}
	stats, err := h.statisticsService.GetAuctionDailyStats(ctx, startDate, endDate, creatorID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidStatisticsRange) {
			c.JSON(400, map[string]interface{}{"code": 400, "message": "invalid statistics date range"})
			return
		}
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取竞拍统计失败: " + err.Error()})
		return
	}
	c.JSON(200, stats)
}

func readAuctionStatisticsScope(c *app.RequestContext) (*int64, bool) {
	switch string(c.GetHeader("X-User-Role")) {
	case statisticsRoleAdmin:
		return nil, true
	case statisticsRoleMerchant:
		userID, err := strconv.ParseInt(string(c.GetHeader("X-User-ID")), 10, 64)
		if err != nil || userID <= 0 {
			c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
			return nil, false
		}
		return &userID, true
	default:
		c.JSON(403, map[string]interface{}{"code": 403, "message": "权限不足"})
		return nil, false
	}
}

func parseAuctionStatisticsRange(c *app.RequestContext, now time.Time) (time.Time, time.Time, error) {
	start, end := defaultAuctionStatisticsRange(now)
	if raw := c.Query("start_date"); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		start = parsed
	}
	if raw := c.Query("end_date"); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end = parsed
	}
	return start, end, nil
}

func defaultAuctionStatisticsRange(now time.Time) (time.Time, time.Time) {
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := end.AddDate(0, 0, -6)
	return start, end
}
```

- [ ] **Step 4: Register route in auction-service main**

In `backend/auction/main.go`, where DAOs/services/handlers are initialized, add:

```go
statisticsDAO := dao.NewStatisticsDAO(db)
statisticsService := service.NewStatisticsService(statisticsDAO)
statisticsHandler := handler.NewStatisticsHandler(statisticsService)
```

In route registration under `v1`, add:

```go
v1.GET("/statistics/auctions", internalAuth, statisticsHandler.GetAuctionStatistics)
```

Place the route near existing management/internal routes, not under public unauthenticated endpoints.

- [ ] **Step 5: Verify handler tests pass**

Run:

```bash
cd backend/auction
go test ./handler -run 'TestStatisticsHandler|TestParseStatisticsDateDefaults' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit handler and route**

```bash
git add backend/auction/handler/statistics.go backend/auction/handler/statistics_test.go backend/auction/main.go
git commit -m "feat: expose auction statistics endpoint"
```

---

## 5. Task: Connect Admin Auction Statistics UI

**Files:**

- Modify: `frontend/admin/src/pages-new/Stats.tsx`
- Create: `frontend/admin/src/pages-new/__tests__/Stats.auctionStats.test.tsx`

- [ ] **Step 1: Write frontend tests**

Create `frontend/admin/src/pages-new/__tests__/Stats.auctionStats.test.tsx`:

```tsx
import { render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import Stats from "../Stats"
import { statisticsApi } from "@/shared/api"

vi.mock("@/shared/api", () => ({
  statisticsApi: {
    getAuctionStats: vi.fn(),
    getRevenueStats: vi.fn(),
    getUserStats: vi.fn(),
  },
}))

const mockedStatisticsApi = vi.mocked(statisticsApi)

function renderStats() {
  return render(
    <MemoryRouter initialEntries={["/stats/auction"]}>
      <Routes>
        <Route path="/stats/:tab" element={<Stats />} />
      </Routes>
    </MemoryRouter>
  )
}

describe("Stats auction statistics", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedStatisticsApi.getRevenueStats.mockResolvedValue([])
    mockedStatisticsApi.getUserStats.mockResolvedValue([])
  })

  it("renders real auction statistics returned by API", async () => {
    mockedStatisticsApi.getAuctionStats.mockResolvedValue([
      { date: "2026-06-01", auction_count: 2, bid_count: 3, avg_price: 120, success_rate: 50 },
      { date: "2026-06-02", auction_count: 1, bid_count: 1, avg_price: 80, success_rate: 100 },
    ])

    renderStats()

    await waitFor(() => expect(mockedStatisticsApi.getAuctionStats).toHaveBeenCalled())
    expect(await screen.findByText("3")).toBeInTheDocument()
    expect(screen.getByText("75.0%")).toBeInTheDocument()
    expect(screen.getByText("2.0")).toBeInTheDocument()
  })

  it("does not render static fallback values when API fails", async () => {
    mockedStatisticsApi.getAuctionStats.mockRejectedValue(new Error("network error"))

    renderStats()

    await waitFor(() => expect(mockedStatisticsApi.getAuctionStats).toHaveBeenCalled())
    expect(screen.queryByText("35")).not.toBeInTheDocument()
    expect(screen.getByText("0")).toBeInTheDocument()
    expect(screen.getByText("0.0%")).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Run the failing frontend test**

Run:

```bash
cd frontend/admin
npm test -- Stats.auctionStats.test.tsx --runInBand
```

If the project uses Vitest without `--runInBand`, run:

```bash
cd frontend/admin
npm test -- Stats.auctionStats.test.tsx
```

Expected: the failure test catches static fallback values or incorrect average calculation.

- [ ] **Step 3: Update Stats.tsx data loading**

In `frontend/admin/src/pages-new/Stats.tsx`, replace the auction section inside `fetchData` with:

```tsx
const today = new Date()
const start = new Date(today)
start.setDate(today.getDate() - 6)
const formatDate = (date: Date) => date.toISOString().slice(0, 10)

const auctionStats = await statisticsApi.getAuctionStats({
  start_date: formatDate(start),
  end_date: formatDate(today),
  group_by: "day",
})
const normalizedAuctionStats = Array.isArray(auctionStats) ? auctionStats : []
setAuctionData(normalizedAuctionStats.map((item: any) => ({
  name: new Date(item.date).toLocaleDateString("zh-CN", { weekday: "short" }),
  count: item.auction_count || 0,
  rate: item.success_rate || 0,
})))
const totalAuctionCount = normalizedAuctionStats.reduce((sum: number, item: any) => sum + (item.auction_count || 0), 0)
const totalBidCount = normalizedAuctionStats.reduce((sum: number, item: any) => sum + (item.bid_count || 0), 0)
const avgSuccessRate = normalizedAuctionStats.length > 0
  ? normalizedAuctionStats.reduce((sum: number, item: any) => sum + (item.success_rate || 0), 0) / normalizedAuctionStats.length
  : 0
const avgBids = totalAuctionCount > 0 ? totalBidCount / totalAuctionCount : 0
setIndicators(prev => ({
  ...prev,
  auction: { total: totalAuctionCount, rate: avgSuccessRate, avgBids },
}))
```

Replace the `catch` block so it no longer injects static data:

```tsx
} catch (e) {
  console.error("获取统计数据失败:", e)
  setAuctionData([])
  setRevenueData([])
  setUserData([])
  setIndicators({
    auction: { total: 0, rate: 0, avgBids: 0 },
    revenue: { total: 0, avgPrice: 0, commission: 0 },
    user: { total: 0, active: 0, rate: 0 },
  })
} finally {
  setLoading(false)
}
```

- [ ] **Step 4: Verify frontend test passes**

Run:

```bash
cd frontend/admin
npm test -- Stats.auctionStats.test.tsx
```

Expected: PASS.

- [ ] **Step 5: Commit frontend change**

```bash
git add frontend/admin/src/pages-new/Stats.tsx frontend/admin/src/pages-new/__tests__/Stats.auctionStats.test.tsx
git commit -m "feat: connect admin auction statistics"
```

---

## 6. Task: Full Verification

**Files:**

- No new files.
- Verify all files changed by Tasks 1-5.

- [ ] **Step 1: Format Go code**

Run:

```bash
gofmt -w backend/gateway/router/admin_statistics_route_test.go backend/gateway/router/router.go backend/auction/dao/statistics.go backend/auction/dao/statistics_test.go backend/auction/service/statistics.go backend/auction/service/statistics_test.go backend/auction/handler/statistics.go backend/auction/handler/statistics_test.go backend/auction/main.go
```

Expected: command exits `0`.

- [ ] **Step 2: Run Gateway route tests**

Run:

```bash
cd backend/gateway
go test ./router -run 'TestStatistics(AuctionRouteUsesAuctionService|NonAuctionRoutesStillUseProductService)' -count=1
```

Expected: PASS.

- [ ] **Step 3: Run auction-service focused tests**

Run:

```bash
cd backend/auction
go test ./dao ./service ./handler -run 'TestStatistics|TestAuctionStatistics|TestParseStatisticsDateDefaults' -count=1
```

Expected: PASS.

- [ ] **Step 4: Run admin frontend focused test**

Run:

```bash
cd frontend/admin
npm test -- Stats.auctionStats.test.tsx
```

Expected: PASS.

- [ ] **Step 5: Build admin frontend**

Run:

```bash
cd frontend/admin
npm run build
```

Expected: PASS.

- [ ] **Step 6: Check diff hygiene**

Run:

```bash
git diff --check
git status --short
```

Expected: no whitespace errors. `git status --short` should only contain intentional task changes and any pre-existing user changes.

- [ ] **Step 7: Final commit if verification changed generated files**

If formatting or build-safe source edits changed tracked files after the previous commits:

```bash
git add backend/gateway/router/admin_statistics_route_test.go backend/gateway/router/router.go backend/auction/dao/statistics.go backend/auction/dao/statistics_test.go backend/auction/service/statistics.go backend/auction/service/statistics_test.go backend/auction/handler/statistics.go backend/auction/handler/statistics_test.go backend/auction/main.go frontend/admin/src/pages-new/Stats.tsx frontend/admin/src/pages-new/__tests__/Stats.auctionStats.test.tsx
git commit -m "test: verify admin auction statistics"
```

Expected: commit succeeds only when there are new intentional changes.

---

## 7. Self-Review

### 7.1 Spec Coverage

| Spec requirement | Plan coverage |
|---|---|
| Keep `/api/v1/statistics/auctions` unchanged | Task 1 |
| Route auction statistics to auction-service | Task 1 |
| Aggregate from `auctions` and `bids` | Task 2 |
| Role-scoped admin vs merchant data | Tasks 2 and 4 |
| Return `AuctionStatistics[]` | Tasks 3 and 4 |
| Remove frontend static mock fallback | Task 5 |
| Tests for Gateway, backend, frontend | Tasks 1-6 |

### 7.2 Placeholder Scan

This plan contains no open-ended implementation placeholders. Each code-producing task includes concrete files, code, commands, and expected outcomes.

### 7.3 Type Consistency

The response type is consistently named `AuctionDailyStat` in auction-service and serialized to the frontend's existing `AuctionStatistics` shape:

```ts
{
  date: string
  auction_count: number
  bid_count: number
  avg_price: number
  success_rate: number
}
```
