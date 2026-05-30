package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"auction-service/client"
	"auction-service/model"
)

type fakeFollowProvider struct {
	follows []model.UserLiveStreamFollow
	total   int64
	err     error
	gotUID  int64
	gotPage int
	gotSize int
}

func (f *fakeFollowProvider) GetUserFollows(ctx context.Context, userID int64, page, pageSize int) ([]model.UserLiveStreamFollow, int64, error) {
	f.gotUID = userID
	f.gotPage = page
	f.gotSize = pageSize
	return f.follows, f.total, f.err
}

type fakeLSFetcher struct {
	streams map[int64]client.LiveStreamSummary
	err     error
	gotIDs  []int64
}

func (f *fakeLSFetcher) BatchGetLiveStreams(ctx context.Context, ids []int64) (map[int64]client.LiveStreamSummary, error) {
	f.gotIDs = ids
	return f.streams, f.err
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func int64Ptr(i int64) *int64 { return &i }

func TestBuildFollowedLiveStreams_InvalidUserID(t *testing.T) {
	_, err := BuildFollowedLiveStreams(context.Background(),
		&fakeFollowProvider{}, &fakeLSFetcher{},
		0, 1, 20)
	if err == nil {
		t.Fatalf("expected error for userID=0")
	}
}

func TestBuildFollowedLiveStreams_DefaultsPageAndSize(t *testing.T) {
	provider := &fakeFollowProvider{follows: nil, total: 0}
	resp, err := BuildFollowedLiveStreams(context.Background(),
		provider, &fakeLSFetcher{},
		1, 0, 0)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if provider.gotPage != 1 || provider.gotSize != 20 {
		t.Fatalf("expected default page=1 size=20, got %d/%d", provider.gotPage, provider.gotSize)
	}
	if resp.Page != 1 || resp.PageSize != 20 {
		t.Fatalf("resp page/size mismatch: %+v", resp)
	}
	if resp.Items == nil {
		t.Fatalf("expected non-nil items")
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected empty items, got %d", len(resp.Items))
	}
}

func TestBuildFollowedLiveStreams_PageSizeOverLimitFallback(t *testing.T) {
	provider := &fakeFollowProvider{}
	_, err := BuildFollowedLiveStreams(context.Background(),
		provider, &fakeLSFetcher{},
		1, 2, 500)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if provider.gotSize != 20 {
		t.Fatalf("expected pageSize fallback to 20, got %d", provider.gotSize)
	}
}

func TestBuildFollowedLiveStreams_EmptyFollowsSkipsBatch(t *testing.T) {
	provider := &fakeFollowProvider{total: 0}
	ls := &fakeLSFetcher{}
	resp, err := BuildFollowedLiveStreams(context.Background(),
		provider, ls, 1, 1, 20)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if ls.gotIDs != nil {
		t.Fatalf("batch fetcher should not be called when follows empty")
	}
	if len(resp.Items) != 0 || resp.Total != 0 {
		t.Fatalf("expected empty resp, got %+v", resp)
	}
}

func TestBuildFollowedLiveStreams_FullAggregation(t *testing.T) {
	now := time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC)
	provider := &fakeFollowProvider{
		follows: []model.UserLiveStreamFollow{
			{ID: 100, UserID: 1, LiveStreamID: 11, NotificationEnabled: true, CreatedAt: now},
			{ID: 101, UserID: 1, LiveStreamID: 12, NotificationEnabled: false, CreatedAt: now.Add(time.Minute)},
		},
		total: 2,
	}
	ls := &fakeLSFetcher{streams: map[int64]client.LiveStreamSummary{
		11: {ID: 11, Name: "room-11", CoverImage: "c11.png", Status: 1, CreatorID: 9001, HostAvatar: strPtr("a9001.png"), AuctionCount: intPtr(3)},
		12: {ID: 12, Name: "room-12", CoverImage: "c12.png", Status: 0, CreatorID: 9002, HostAvatar: strPtr("a9002.png"), AuctionCount: nil},
	}}

	resp, err := BuildFollowedLiveStreams(context.Background(), provider, ls, 1, 1, 20)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if resp.Total != 2 || len(resp.Items) != 2 {
		t.Fatalf("bad resp: %+v", resp)
	}
	first := resp.Items[0]
	if first.LiveStreamID != 11 || first.LiveStreamName != "room-11" || first.CoverImage != "c11.png" ||
		first.Status != 1 || first.HostAvatar == nil || *first.HostAvatar != "a9001.png" || !first.NotificationEnabled ||
		first.AuctionCount == nil || *first.AuctionCount != 3 {
		t.Fatalf("first item mismatch: %+v", first)
	}
	if first.ViewerCount != nil {
		t.Fatalf("first viewer_count should be nil, got %d", *first.ViewerCount)
	}
	if first.FollowedAt != "2026-05-31T08:00:00Z" {
		t.Fatalf("first followed_at = %q", first.FollowedAt)
	}
	second := resp.Items[1]
	if second.LiveStreamID != 12 || second.HostAvatar == nil || *second.HostAvatar != "a9002.png" || second.AuctionCount != nil {
		t.Fatalf("second item mismatch: %+v", second)
	}
}

func TestBuildFollowedLiveStreams_ViewerCountAlwaysNull(t *testing.T) {
	now := time.Now()
	provider := &fakeFollowProvider{
		follows: []model.UserLiveStreamFollow{
			{ID: 1, UserID: 1, LiveStreamID: 11, CreatedAt: now},
		},
		total: 1,
	}
	ls := &fakeLSFetcher{streams: map[int64]client.LiveStreamSummary{
		11: {ID: 11, Name: "room-11", CreatorID: 9001},
	}}
	resp, err := BuildFollowedLiveStreams(context.Background(), provider, ls, 1, 1, 20)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item")
	}
	if resp.Items[0].ViewerCount != nil {
		t.Fatalf("viewer_count should be nil, got %d", *resp.Items[0].ViewerCount)
	}
}

func TestBuildFollowedLiveStreams_LSFetcherDegradation(t *testing.T) {
	now := time.Now()
	provider := &fakeFollowProvider{
		follows: []model.UserLiveStreamFollow{
			{ID: 1, UserID: 1, LiveStreamID: 11, NotificationEnabled: true, CreatedAt: now},
			{ID: 2, UserID: 1, LiveStreamID: 12, NotificationEnabled: false, CreatedAt: now},
		},
		total: 2,
	}
	ls := &fakeLSFetcher{err: errors.New("product-service down")}

	resp, err := BuildFollowedLiveStreams(context.Background(), provider, ls, 1, 1, 20)
	if err != nil {
		t.Fatalf("should degrade, not error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	for _, item := range resp.Items {
		if item.HostAvatar != nil {
			t.Fatalf("host_avatar should be nil on degradation, got %q", *item.HostAvatar)
		}
		if item.AuctionCount != nil {
			t.Fatalf("auction_count should be nil on degradation, got %d", *item.AuctionCount)
		}
		if item.ViewerCount != nil {
			t.Fatalf("viewer_count should be nil, got %d", *item.ViewerCount)
		}
		if item.LiveStreamName != "" || item.CoverImage != "" || item.Status != 0 {
			t.Fatalf("stream fields should be defaults on degradation, got %+v", item)
		}
	}
}

func TestBuildFollowedLiveStreams_MissingStreamDefaults(t *testing.T) {
	now := time.Now()
	provider := &fakeFollowProvider{
		follows: []model.UserLiveStreamFollow{
			{ID: 1, UserID: 1, LiveStreamID: 11, CreatedAt: now},
			{ID: 2, UserID: 1, LiveStreamID: 12, CreatedAt: now},
		},
		total: 2,
	}
	ls := &fakeLSFetcher{streams: map[int64]client.LiveStreamSummary{
		12: {ID: 12, Name: "room-12", CoverImage: "c12.png", Status: 1, CreatorID: 9002, HostAvatar: strPtr("a.png"), AuctionCount: intPtr(5)},
	}}

	resp, err := BuildFollowedLiveStreams(context.Background(), provider, ls, 1, 1, 20)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	missing := resp.Items[0]
	if missing.LiveStreamID != 11 || missing.LiveStreamName != "" || missing.CoverImage != "" ||
		missing.Status != 0 || missing.HostAvatar != nil || missing.AuctionCount != nil {
		t.Fatalf("missing-stream item should be defaults, got %+v", missing)
	}
	present := resp.Items[1]
	if present.LiveStreamID != 12 || present.LiveStreamName != "room-12" || present.HostAvatar == nil || *present.HostAvatar != "a.png" {
		t.Fatalf("present item mismatch: %+v", present)
	}
	if present.AuctionCount == nil || *present.AuctionCount != 5 {
		t.Fatalf("present auction_count mismatch: %+v", present)
	}
}

func TestBuildFollowedLiveStreams_ProviderError(t *testing.T) {
	provider := &fakeFollowProvider{err: errors.New("db down")}
	_, err := BuildFollowedLiveStreams(context.Background(), provider,
		&fakeLSFetcher{}, 1, 1, 20)
	if err == nil {
		t.Fatalf("expected provider error")
	}
}
