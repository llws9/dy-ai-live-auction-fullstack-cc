package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type FollowStatusResponse struct {
	IsFollowing bool `json:"is_following"`
}

type FollowersStatsResponse struct {
	Count int64 `json:"count"`
}

type AuctionClient struct {
	baseURL string
	hc      *http.Client
}

func NewAuctionClient(baseURL string, timeout time.Duration) *AuctionClient {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &AuctionClient{
		baseURL: baseURL,
		hc:      &http.Client{Timeout: timeout},
	}
}

func (c *AuctionClient) GetFollowStatus(ctx context.Context, userID, liveStreamID int64) (*FollowStatusResponse, error) {
	url := fmt.Sprintf("%s/api/v1/live-streams/%d/follow-status", c.baseURL, liveStreamID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call auction-service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auction-service returned status %d", resp.StatusCode)
	}
	var body struct {
		Code int                  `json:"code"`
		Data FollowStatusResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &body.Data, nil
}

func (c *AuctionClient) GetFollowersCount(ctx context.Context, liveStreamID int64) (int64, error) {
	url := fmt.Sprintf("%s/api/v1/live-streams/%d/followers/count", c.baseURL, liveStreamID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return 0, fmt.Errorf("call auction-service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("auction-service returned status %d", resp.StatusCode)
	}
	var body struct {
		Code int                  `json:"code"`
		Data FollowersStatsResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	return body.Data.Count, nil
}
