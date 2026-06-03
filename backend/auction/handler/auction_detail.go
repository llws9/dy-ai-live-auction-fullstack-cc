package handler

import (
	"context"

	"auction-service/model"
)

type auctionRuleFetcher interface {
	GetByProductID(ctx context.Context, productID int64) (*model.AuctionRule, error)
}

type AuctionDetailResponse struct {
	*model.Auction
	Rules *model.AuctionRule `json:"rules,omitempty"`
}

func BuildAuctionDetailResponse(ctx context.Context, ruleFetcher auctionRuleFetcher, auction *model.Auction) (*AuctionDetailResponse, error) {
	resp := &AuctionDetailResponse{Auction: auction}
	if ruleFetcher == nil || auction == nil {
		return resp, nil
	}

	rule, err := ruleFetcher.GetByProductID(ctx, auction.ProductID)
	if err != nil {
		return nil, err
	}
	resp.Rules = rule
	return resp, nil
}
