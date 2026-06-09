package main

import (
	"testing"
	"time"

	"test-service/client/auction"
	"test-service/scenario/pressure"
)

func TestPressureFixtureSellerForIndexUsesDistinctMerchantsForThroughput(t *testing.T) {
	cfg := pressure.Config{Scenario: "throughput"}
	baseID := int64(910000)

	first := pressureFixtureSellerForIndex(cfg, baseID, 0)
	second := pressureFixtureSellerForIndex(cfg, baseID, 1)

	if first.Role != auction.RoleMerchant || second.Role != auction.RoleMerchant {
		t.Fatalf("pressure fixture sellers must be merchant actors: first=%+v second=%+v", first, second)
	}
	if first.UserID == second.UserID {
		t.Fatalf("throughput fixture shards must use distinct merchants, got %d", first.UserID)
	}
}

func TestPressureFixtureSellerForIndexKeepsSingleMerchantForHotAuction(t *testing.T) {
	cfg := pressure.Config{Scenario: "hot_auction"}
	baseID := int64(920000)

	first := pressureFixtureSellerForIndex(cfg, baseID, 0)
	second := pressureFixtureSellerForIndex(cfg, baseID, 1)

	if first.UserID != second.UserID {
		t.Fatalf("hot auction fixture should keep one merchant, got %d and %d", first.UserID, second.UserID)
	}
}

func TestPressureFixtureMerchantBaseIDChangesAcrossRuns(t *testing.T) {
	now := time.UnixMilli(1710000000000)
	first := pressureFixtureMerchantBaseID(now)
	second := pressureFixtureMerchantBaseID(now)

	if first == second {
		t.Fatalf("pressure fixture merchant base id must be run-scoped, got %d", first)
	}
}
