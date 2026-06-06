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
	AuctionCount int64   `gorm:"column:auction_count"`
	BidCount     int64   `gorm:"column:bid_count"`
	SuccessCount int64   `gorm:"column:success_count"`
	AvgPrice     float64 `gorm:"column:avg_price"`
}

func (d *StatisticsDAO) ListAuctionDailyStats(ctx context.Context, startInclusive, endExclusive time.Time, creatorID *int64) ([]AuctionDailyStatsRow, error) {
	where := "a.start_time >= ? AND a.start_time < ?"
	args := []interface{}{model.AuctionStatusEnded, startInclusive, endExclusive}
	if creatorID != nil {
		where += " AND a.creator_id = ?"
		args = append(args, *creatorID)
	}

	query := `
SELECT
  auction_rows.date AS date,
  COUNT(*) AS auction_count,
  COALESCE(SUM(auction_rows.bid_count), 0) AS bid_count,
  COALESCE(SUM(auction_rows.success), 0) AS success_count,
  COALESCE(AVG(CASE WHEN auction_rows.success = 1 THEN auction_rows.current_price END), 0) AS avg_price
FROM (
  SELECT
    a.id,
    DATE(a.start_time) AS date,
    a.current_price,
    CASE WHEN a.status = ? AND a.winner_id IS NOT NULL THEN 1 ELSE 0 END AS success,
    COUNT(b.id) AS bid_count
  FROM auctions AS a
  LEFT JOIN bids AS b ON b.auction_id = a.id
  WHERE ` + where + `
  GROUP BY a.id, DATE(a.start_time), a.current_price, a.status, a.winner_id
) AS auction_rows
GROUP BY auction_rows.date
ORDER BY auction_rows.date ASC`

	var rows []AuctionDailyStatsRow
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}
