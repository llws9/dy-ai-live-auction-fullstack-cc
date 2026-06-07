package handler

import (
	"context"
	"testing"

	"auction-service/model"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAuctionRuleFetcher struct {
	out *model.AuctionRule
	err error
}

func (f *fakeAuctionRuleFetcher) GetByProductID(_ context.Context, _ int64) (*model.AuctionRule, error) {
	return f.out, f.err
}

func (f *fakeAuctionRuleFetcher) GetByProductIDs(_ context.Context, _ []int64) (map[int64]*model.AuctionRule, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.out == nil {
		return map[int64]*model.AuctionRule{}, nil
	}
	return map[int64]*model.AuctionRule{f.out.ProductID: f.out}, nil
}

func TestBuildAuctionDetailResponseIncludesAuctionRule(t *testing.T) {
	ctx := context.Background()
	auction := &model.Auction{
		ID:           7,
		ProductID:    11,
		Status:       model.AuctionStatusOngoing,
		CurrentPrice: decimal.NewFromInt(3400),
	}
	ruleFetcher := &fakeAuctionRuleFetcher{
		out: &model.AuctionRule{
			ProductID:  11,
			StartPrice: decimal.NewFromInt(3000),
			Increment:  decimal.NewFromInt(200),
			Duration:   3600,
		},
	}

	got, err := BuildAuctionDetailResponse(ctx, ruleFetcher, auction)
	require.NoError(t, err)

	assert.Equal(t, int64(7), got.ID)
	require.NotNil(t, got.Rules)
	assert.True(t, got.Rules.StartPrice.Equal(decimal.NewFromInt(3000)))
	assert.True(t, got.Rules.Increment.Equal(decimal.NewFromInt(200)))
}
