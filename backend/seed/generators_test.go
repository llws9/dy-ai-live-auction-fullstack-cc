package main

import (
	"testing"

	"product-service/model"
)

func TestGenerateAuctionRules_CoversPublishedProductsOnce(t *testing.T) {
	cfg := &SeedConfig{AuctionRulesCount: 5}
	products := []model.Product{
		{ID: 101, Status: model.ProductStatusPublished},
		{ID: 102, Status: model.ProductStatusPublished},
		{ID: 103, Status: model.ProductStatusDraft},
		{ID: 104, Status: model.ProductStatusUnpublished},
	}

	rules := GenerateAuctionRules(cfg, products)

	if len(rules) != 2 {
		t.Fatalf("expected one rule per published product, got %d rules: %+v", len(rules), rules)
	}

	seen := make(map[int64]bool)
	for _, rule := range rules {
		if seen[rule.ProductID] {
			t.Fatalf("duplicate rule for product_id=%d", rule.ProductID)
		}
		seen[rule.ProductID] = true
		if rule.Increment <= 0 {
			t.Fatalf("expected positive increment for product_id=%d", rule.ProductID)
		}
		if rule.Duration <= 0 {
			t.Fatalf("expected positive duration for product_id=%d", rule.ProductID)
		}
	}

	for _, productID := range []int64{101, 102} {
		if !seen[productID] {
			t.Fatalf("missing rule for published product_id=%d", productID)
		}
	}
	if seen[103] || seen[104] {
		t.Fatalf("draft/unpublished products must not get auction rules: %+v", seen)
	}
}
