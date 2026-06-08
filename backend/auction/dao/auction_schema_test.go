package dao

import (
	"testing"

	"github.com/stretchr/testify/require"

	"auction-service/model"
)

func TestEnsureAuctionActiveProductUniqueIndexSkipsSQLite(t *testing.T) {
	db := newAuctionDAOTestDB(t)

	require.NoError(t, EnsureAuctionActiveProductUniqueIndex(db))

	require.False(t, db.Migrator().HasColumn(&model.Auction{}, "active_product_key"))
	require.False(t, db.Migrator().HasIndex(&model.Auction{}, "uk_active_product"))
}

func TestEnsureAuctionLiveStreamUniqueIndexesSkipsSQLite(t *testing.T) {
	db := newAuctionDAOTestDB(t)

	require.NoError(t, EnsureAuctionLiveStreamUniqueIndexes(db))

	require.False(t, db.Migrator().HasColumn(&model.Auction{}, "active_live_stream_key"))
	require.False(t, db.Migrator().HasIndex(&model.Auction{}, "uk_active_live_stream"))
	require.False(t, db.Migrator().HasColumn(&model.Auction{}, "pending_live_stream_key"))
	require.False(t, db.Migrator().HasIndex(&model.Auction{}, "uk_pending_live_stream"))
	require.False(t, db.Migrator().HasColumn(&model.Auction{}, "running_live_stream_key"))
	require.False(t, db.Migrator().HasIndex(&model.Auction{}, "uk_running_live_stream"))
}
