package handler

import (
	"context"
	"errors"

	"auction-service/client"
	"auction-service/model"
)

type FollowedLiveStreamItem struct {
	LiveStreamID        int64   `json:"live_stream_id"`
	LiveStreamName      string  `json:"live_stream_name"`
	CoverImage          string  `json:"cover_image"`
	Status              int     `json:"status"`
	HostAvatar          *string `json:"host_avatar"`
	NotificationEnabled bool    `json:"notification_enabled"`
	FollowedAt          string  `json:"followed_at"`
	ViewerCount         *int64  `json:"viewer_count"`
	AuctionCount        *int64  `json:"auction_count"`
}

type FollowedLiveStreamsResponse struct {
	Items    []FollowedLiveStreamItem `json:"items"`
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

type FollowListProvider interface {
	GetUserFollows(ctx context.Context, userID int64, page, pageSize int) ([]model.UserLiveStreamFollow, int64, error)
}

type LiveStreamBatchFetcher interface {
	BatchGetLiveStreams(ctx context.Context, ids []int64) (map[int64]client.LiveStreamSummary, error)
}

func BuildFollowedLiveStreams(
	ctx context.Context,
	provider FollowListProvider,
	lsFetcher LiveStreamBatchFetcher,
	userID int64,
	page, pageSize int,
) (*FollowedLiveStreamsResponse, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user_id")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	follows, total, err := provider.GetUserFollows(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}

	if len(follows) == 0 {
		return &FollowedLiveStreamsResponse{
			Items:    []FollowedLiveStreamItem{},
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	liveStreamIDs := make([]int64, 0, len(follows))
	for _, f := range follows {
		liveStreamIDs = append(liveStreamIDs, f.LiveStreamID)
	}

	streams, lsErr := lsFetcher.BatchGetLiveStreams(ctx, liveStreamIDs)
	if lsErr != nil {
		streams = map[int64]client.LiveStreamSummary{}
	}

	items := make([]FollowedLiveStreamItem, 0, len(follows))
	for _, f := range follows {
		item := FollowedLiveStreamItem{
			LiveStreamID:        f.LiveStreamID,
			NotificationEnabled: f.NotificationEnabled,
			FollowedAt:          f.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			ViewerCount:         nil,
		}
		if s, ok := streams[f.LiveStreamID]; ok {
			item.LiveStreamName = s.Name
			item.CoverImage = s.CoverImage
			item.Status = s.Status
			item.HostAvatar = s.HostAvatar
			if s.AuctionCount != nil {
				ac := int64(*s.AuctionCount)
				item.AuctionCount = &ac
			}
		}
		items = append(items, item)
	}

	return &FollowedLiveStreamsResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
