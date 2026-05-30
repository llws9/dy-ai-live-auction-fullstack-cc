package handler

import (
	"context"
	"errors"

	"auction-service/client"
	"auction-service/model"
)

// FollowedLiveStreamItem 是 GET /api/v1/user/followed-live-streams 列表项
// （T3.3 / spec B §2.3）。
type FollowedLiveStreamItem struct {
	LiveStreamID        int64  `json:"live_stream_id"`
	LiveStreamName      string `json:"live_stream_name"`
	CoverImage          string `json:"cover_image"`
	Status              int    `json:"status"`
	HostAvatar          string `json:"host_avatar"`
	NotificationEnabled bool   `json:"notification_enabled"`
	FollowedAt          string `json:"followed_at"`
	ViewerCount         int64  `json:"viewer_count"`
	AuctionCount        int64  `json:"auction_count"`
}

// FollowedLiveStreamsResponse 是 GET /api/v1/user/followed-live-streams 的 data 字段。
type FollowedLiveStreamsResponse struct {
	Items    []FollowedLiveStreamItem `json:"items"`
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

// FollowListProvider 抽象 followService.GetUserFollows，便于编排单测。
type FollowListProvider interface {
	GetUserFollows(ctx context.Context, userID int64, page, pageSize int) ([]model.UserLiveStreamFollow, int64, error)
}

// LiveStreamBatchFetcher 抽象 LiveStreamClient，便于注入 fake。
type LiveStreamBatchFetcher interface {
	BatchGetLiveStreams(ctx context.Context, ids []int64) (map[int64]client.LiveStreamSummary, error)
}

// UserAvatarFetcher 抽象按 user_id 批量取头像（本服务的 UserDAO.GetByIDs）。
type UserAvatarFetcher interface {
	GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.User, error)
}

// AuctionCountFetcher 抽象按 live_stream_id 批量取进行中竞拍数。
type AuctionCountFetcher interface {
	CountActiveByLiveStreamIDs(ctx context.Context, ids []int64) (map[int64]int64, error)
}

// BuildFollowedLiveStreams 是 T3.3 / spec B §2.3 / F-B3 的编排函数。
//
//   - 参数：userID（>0）+ page/pageSize（默认 1/20，pageSize 上限 100）
//   - 流程：
//     1) FollowListProvider 拉关注关系列表（含 follow_id / live_stream_id / notification_enabled / created_at）
//     2) LiveStreamBatchFetcher 一次取齐 live_stream 摘要（name / cover_image / status / creator_id）
//     3) UserAvatarFetcher 一次取齐 host_avatar（按 creator_id 批量）
//     4) AuctionCountFetcher 一次取齐 auction_count（按 live_stream_id 批量）
//   - 缺失字段全部走默认值（host_avatar=""、auction_count=0、viewer_count=0）。
//   - 编排不缓存、不并发：N≤pageSize≤100，三次 batch 调用即可。
func BuildFollowedLiveStreams(
	ctx context.Context,
	provider FollowListProvider,
	lsFetcher LiveStreamBatchFetcher,
	userFetcher UserAvatarFetcher,
	auctionFetcher AuctionCountFetcher,
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

	// 关注列表为空时，避免后续无意义的 batch 调用。
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

	streams, err := lsFetcher.BatchGetLiveStreams(ctx, liveStreamIDs)
	if err != nil {
		return nil, err
	}

	creatorIDs := make([]int64, 0, len(streams))
	seenCreator := make(map[int64]struct{}, len(streams))
	for _, s := range streams {
		if s.CreatorID == 0 {
			continue
		}
		if _, ok := seenCreator[s.CreatorID]; ok {
			continue
		}
		seenCreator[s.CreatorID] = struct{}{}
		creatorIDs = append(creatorIDs, s.CreatorID)
	}

	users := map[int64]*model.User{}
	if len(creatorIDs) > 0 {
		users, err = userFetcher.GetByIDs(ctx, creatorIDs)
		if err != nil {
			return nil, err
		}
	}

	counts, err := auctionFetcher.CountActiveByLiveStreamIDs(ctx, liveStreamIDs)
	if err != nil {
		return nil, err
	}

	items := make([]FollowedLiveStreamItem, 0, len(follows))
	for _, f := range follows {
		item := FollowedLiveStreamItem{
			LiveStreamID:        f.LiveStreamID,
			NotificationEnabled: f.NotificationEnabled,
			FollowedAt:          f.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			ViewerCount:         0, // spec B §2.3：本期固定 0
			AuctionCount:        counts[f.LiveStreamID],
		}
		if s, ok := streams[f.LiveStreamID]; ok {
			item.LiveStreamName = s.Name
			item.CoverImage = s.CoverImage
			item.Status = s.Status
			if u, hit := users[s.CreatorID]; hit && u != nil {
				item.HostAvatar = u.Avatar
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
