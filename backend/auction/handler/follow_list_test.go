package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"auction-service/client"
	"auction-service/model"
)

// ---- fakes ----

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

type fakeUserFetcher struct {
	users  map[int64]*model.User
	err    error
	gotIDs []int64
}

func (f *fakeUserFetcher) GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.User, error) {
	f.gotIDs = ids
	return f.users, f.err
}

type fakeAuctionCountFetcher struct {
	counts map[int64]int64
	err    error
	gotIDs []int64
}

func (f *fakeAuctionCountFetcher) CountActiveByLiveStreamIDs(ctx context.Context, ids []int64) (map[int64]int64, error) {
	f.gotIDs = ids
	return f.counts, f.err
}

// ---- tests ----

func TestBuildFollowedLiveStreams_InvalidUserID(t *testing.T) {
	_, err := BuildFollowedLiveStreams(context.Background(),
		&fakeFollowProvider{}, &fakeLSFetcher{}, &fakeUserFetcher{}, &fakeAuctionCountFetcher{},
		0, 1, 20)
	if err == nil {
		t.Fatalf("expected error for userID=0")
	}
}

func TestBuildFollowedLiveStreams_DefaultsPageAndSize(t *testing.T) {
	provider := &fakeFollowProvider{follows: nil, total: 0}
	resp, err := BuildFollowedLiveStreams(context.Background(),
		provider, &fakeLSFetcher{}, &fakeUserFetcher{}, &fakeAuctionCountFetcher{},
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
		provider, &fakeLSFetcher{}, &fakeUserFetcher{}, &fakeAuctionCountFetcher{},
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
	uf := &fakeUserFetcher{}
	af := &fakeAuctionCountFetcher{}
	resp, err := BuildFollowedLiveStreams(context.Background(),
		provider, ls, uf, af, 1, 1, 20)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if ls.gotIDs != nil || uf.gotIDs != nil || af.gotIDs != nil {
		t.Fatalf("batch fetchers should not be called when follows empty")
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
		11: {ID: 11, Name: "room-11", CoverImage: "c11.png", Status: 1, CreatorID: 9001},
		12: {ID: 12, Name: "room-12", CoverImage: "c12.png", Status: 0, CreatorID: 9002},
	}}
	uf := &fakeUserFetcher{users: map[int64]*model.User{
		9001: {ID: 9001, Avatar: "a9001.png"},
		9002: {ID: 9002, Avatar: "a9002.png"},
	}}
	af := &fakeAuctionCountFetcher{counts: map[int64]int64{11: 3}}

	resp, err := BuildFollowedLiveStreams(context.Background(), provider, ls, uf, af, 1, 1, 20)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if resp.Total != 2 || len(resp.Items) != 2 {
		t.Fatalf("bad resp: %+v", resp)
	}
	first := resp.Items[0]
	if first.LiveStreamID != 11 || first.LiveStreamName != "room-11" || first.CoverImage != "c11.png" ||
		first.Status != 1 || first.HostAvatar != "a9001.png" || !first.NotificationEnabled ||
		first.AuctionCount != 3 || first.ViewerCount != 0 {
		t.Fatalf("first item mismatch: %+v", first)
	}
	if first.FollowedAt != "2026-05-31T08:00:00Z" {
		t.Fatalf("first followed_at = %q", first.FollowedAt)
	}
	second := resp.Items[1]
	if second.LiveStreamID != 12 || second.HostAvatar != "a9002.png" || second.AuctionCount != 0 {
		t.Fatalf("second item mismatch: %+v", second)
	}
}

func TestBuildFollowedLiveStreams_MissingStreamAndUserDefaults(t *testing.T) {
	now := time.Now()
	provider := &fakeFollowProvider{
		follows: []model.UserLiveStreamFollow{
			{ID: 1, UserID: 1, LiveStreamID: 11, CreatedAt: now}, // 流不存在
			{ID: 2, UserID: 1, LiveStreamID: 12, CreatedAt: now}, // 流存在但 creator user 缺失
		},
		total: 2,
	}
	ls := &fakeLSFetcher{streams: map[int64]client.LiveStreamSummary{
		12: {ID: 12, Name: "room-12", CoverImage: "c12.png", Status: 1, CreatorID: 9002},
	}}
	uf := &fakeUserFetcher{users: map[int64]*model.User{}} // no creator user
	af := &fakeAuctionCountFetcher{counts: map[int64]int64{}}

	resp, err := BuildFollowedLiveStreams(context.Background(), provider, ls, uf, af, 1, 1, 20)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	missing := resp.Items[0]
	if missing.LiveStreamID != 11 || missing.LiveStreamName != "" || missing.CoverImage != "" ||
		missing.Status != 0 || missing.HostAvatar != "" || missing.AuctionCount != 0 {
		t.Fatalf("missing-stream item should be defaults, got %+v", missing)
	}
	noUser := resp.Items[1]
	if noUser.LiveStreamID != 12 || noUser.LiveStreamName != "room-12" || noUser.HostAvatar != "" {
		t.Fatalf("no-user item mismatch: %+v", noUser)
	}
}

func TestBuildFollowedLiveStreams_SkipsZeroCreatorAndDeduplicates(t *testing.T) {
	now := time.Now()
	provider := &fakeFollowProvider{
		follows: []model.UserLiveStreamFollow{
			{ID: 1, UserID: 1, LiveStreamID: 11, CreatedAt: now},
			{ID: 2, UserID: 1, LiveStreamID: 12, CreatedAt: now},
			{ID: 3, UserID: 1, LiveStreamID: 13, CreatedAt: now},
		},
		total: 3,
	}
	ls := &fakeLSFetcher{streams: map[int64]client.LiveStreamSummary{
		11: {ID: 11, CreatorID: 9001},
		12: {ID: 12, CreatorID: 0},    // 应被跳过
		13: {ID: 13, CreatorID: 9001}, // 应去重
	}}
	uf := &fakeUserFetcher{users: map[int64]*model.User{9001: {ID: 9001, Avatar: "a.png"}}}
	af := &fakeAuctionCountFetcher{counts: map[int64]int64{}}

	if _, err := BuildFollowedLiveStreams(context.Background(), provider, ls, uf, af, 1, 1, 20); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(uf.gotIDs) != 1 || uf.gotIDs[0] != 9001 {
		t.Fatalf("expected dedup creator IDs [9001], got %v", uf.gotIDs)
	}
}

func TestBuildFollowedLiveStreams_ProviderError(t *testing.T) {
	provider := &fakeFollowProvider{err: errors.New("db down")}
	_, err := BuildFollowedLiveStreams(context.Background(), provider,
		&fakeLSFetcher{}, &fakeUserFetcher{}, &fakeAuctionCountFetcher{}, 1, 1, 20)
	if err == nil {
		t.Fatalf("expected provider error")
	}
}

func TestBuildFollowedLiveStreams_LSFetcherError(t *testing.T) {
	now := time.Now()
	provider := &fakeFollowProvider{
		follows: []model.UserLiveStreamFollow{{ID: 1, UserID: 1, LiveStreamID: 11, CreatedAt: now}},
		total:   1,
	}
	_, err := BuildFollowedLiveStreams(context.Background(), provider,
		&fakeLSFetcher{err: errors.New("product down")},
		&fakeUserFetcher{}, &fakeAuctionCountFetcher{}, 1, 1, 20)
	if err == nil {
		t.Fatalf("expected ls fetcher error")
	}
}

func TestBuildFollowedLiveStreams_AuctionFetcherError(t *testing.T) {
	now := time.Now()
	provider := &fakeFollowProvider{
		follows: []model.UserLiveStreamFollow{{ID: 1, UserID: 1, LiveStreamID: 11, CreatedAt: now}},
		total:   1,
	}
	ls := &fakeLSFetcher{streams: map[int64]client.LiveStreamSummary{11: {ID: 11, CreatorID: 9001}}}
	uf := &fakeUserFetcher{users: map[int64]*model.User{9001: {ID: 9001}}}
	af := &fakeAuctionCountFetcher{err: errors.New("count failed")}
	_, err := BuildFollowedLiveStreams(context.Background(), provider, ls, uf, af, 1, 1, 20)
	if err == nil {
		t.Fatalf("expected auction fetcher error")
	}
}
