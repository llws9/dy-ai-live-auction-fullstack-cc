package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type FollowStatusResponse struct {
	IsFollowing bool `json:"is_following"`
}

type FollowersStatsResponse struct {
	Count int64 `json:"count"`
}

type UserSummary struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

type auctionListData struct {
	Total int64 `json:"total"`
}

type AuctionClient struct {
	baseURL       string
	hc            *http.Client
	internalToken string
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

// SetInternalToken 设置服务间内部调用的鉴权 token，用于访问 auction-service /internal/* 接口。
func (c *AuctionClient) SetInternalToken(token string) {
	c.internalToken = token
}

// CurrentAuctionItem 表示某直播间当前进行中的竞拍信息。
type CurrentAuctionItem struct {
	LiveStreamID int64  `json:"live_stream_id"`
	AuctionID    int64  `json:"auction_id"`
	ProductID    int64  `json:"product_id"`
	CurrentPrice string `json:"current_price"`
	Status       int    `json:"status"`
}

type ProductAuctionState struct {
	ProductID           int64  `json:"product_id"`
	ActiveAuctionID     *int64 `json:"active_auction_id"`
	ActiveStatus        *int   `json:"active_status"`
	LatestAuctionID     *int64 `json:"latest_auction_id"`
	LatestAuctionStatus *int   `json:"latest_auction_status"`
	LatestAuctionResult string `json:"latest_auction_result"`
}

// CurrentByLiveStreamIDs 批量查询多个直播间的当前竞拍，返回 live_stream_id -> 当前竞拍 的映射。
func (c *AuctionClient) CurrentByLiveStreamIDs(ctx context.Context, ids []int64) (map[int64]CurrentAuctionItem, error) {
	reqURL := fmt.Sprintf("%s/internal/auctions/current-by-live-streams", c.baseURL)
	payload, err := json.Marshal(map[string]interface{}{"live_stream_ids": ids})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Token", c.internalToken)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call auction-service: %w", err)
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auction-service returned status %d", resp.StatusCode)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []CurrentAuctionItem `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	result := make(map[int64]CurrentAuctionItem, len(body.Data.Items))
	for _, item := range body.Data.Items {
		result[item.LiveStreamID] = item
	}
	return result, nil
}

func (c *AuctionClient) BatchProductAuctionStates(ctx context.Context, productIDs []int64) (map[int64]ProductAuctionState, error) {
	if len(productIDs) == 0 {
		return map[int64]ProductAuctionState{}, nil
	}
	reqURL := fmt.Sprintf("%s/internal/auctions/by-products", c.baseURL)
	payload, err := json.Marshal(map[string]interface{}{"product_ids": productIDs})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Token", c.internalToken)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call auction-service: %w", err)
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auction-service returned status %d", resp.StatusCode)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []ProductAuctionState `json:"items"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if body.Code != 0 && body.Code != 200 {
		return nil, fmt.Errorf("auction-service business code %d: %s", body.Code, body.Message)
	}
	result := make(map[int64]ProductAuctionState, len(body.Data.Items))
	for _, item := range body.Data.Items {
		result[item.ProductID] = item
	}
	return result, nil
}

func (c *AuctionClient) BatchCountAuctionsByLiveStreamIDs(ctx context.Context, liveStreamIDs []int64) (map[int64]int64, error) {
	if len(liveStreamIDs) == 0 {
		return map[int64]int64{}, nil
	}
	reqURL := fmt.Sprintf("%s/internal/auctions/count-by-live-streams", c.baseURL)
	payload, err := json.Marshal(map[string]interface{}{"live_stream_ids": liveStreamIDs})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Token", c.internalToken)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call auction-service: %w", err)
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auction-service returned status %d", resp.StatusCode)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Counts map[int64]int64 `json:"counts"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if body.Code != 0 && body.Code != 200 {
		return nil, fmt.Errorf("auction-service business code %d: %s", body.Code, body.Message)
	}
	if body.Data.Counts == nil {
		return map[int64]int64{}, nil
	}
	return body.Data.Counts, nil
}

func (c *AuctionClient) BatchGetUserSummaries(ctx context.Context, ids []int64) (map[int64]UserSummary, error) {
	if len(ids) == 0 {
		return map[int64]UserSummary{}, nil
	}
	reqURL := fmt.Sprintf("%s/internal/users/batch", c.baseURL)
	payload, err := json.Marshal(map[string]interface{}{"ids": ids})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Token", c.internalToken)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call auction-service: %w", err)
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auction-service returned status %d", resp.StatusCode)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []UserSummary `json:"items"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if body.Code != 0 && body.Code != 200 {
		return nil, fmt.Errorf("auction-service business code %d: %s", body.Code, body.Message)
	}
	result := make(map[int64]UserSummary, len(body.Data.Items))
	for _, item := range body.Data.Items {
		result[item.ID] = item
	}
	return result, nil
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
		Code int                    `json:"code"`
		Data FollowersStatsResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	return body.Data.Count, nil
}

func (c *AuctionClient) CountAuctionsByLiveStreamID(ctx context.Context, liveStreamID int64) (int64, error) {
	values := url.Values{}
	values.Set("live_stream_id", fmt.Sprintf("%d", liveStreamID))
	values.Set("page", "1")
	values.Set("page_size", "1")
	reqURL := fmt.Sprintf("%s/api/v1/auctions?%s", c.baseURL, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
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
		Code int             `json:"code"`
		Data auctionListData `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	return body.Data.Total, nil
}
