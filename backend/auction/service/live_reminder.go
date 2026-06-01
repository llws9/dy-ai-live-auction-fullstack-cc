package service

import (
	"context"

	"auction-service/client"
	"auction-service/model"
)

const (
	liveReminderCandidateLimit       = 50
	liveReminderMaxScannedCandidates = 500
)

type LiveSessionResolver interface {
	GetActiveSession(ctx context.Context, liveStreamID int64) (*model.StreamInfo, error)
}

type LiveReminderReceiptClaimer interface {
	Claim(ctx context.Context, userID, liveStreamID, startedAt int64) (bool, error)
}

type LiveStatsSessionResolver struct {
	statsService LiveStreamStatsProvider
	metadata     LiveStreamMetadataProvider
}

type LiveStreamStatsProvider interface {
	GetStats(ctx context.Context, liveStreamID int64) (*LiveStreamStats, error)
}

type LiveStreamMetadataProvider interface {
	BatchGetLiveStreams(ctx context.Context, ids []int64) (map[int64]client.LiveStreamSummary, error)
}

func NewLiveStatsSessionResolver(statsService LiveStreamStatsProvider) *LiveStatsSessionResolver {
	return &LiveStatsSessionResolver{statsService: statsService}
}

func NewLiveStatsSessionResolverWithMetadata(statsService LiveStreamStatsProvider, metadata LiveStreamMetadataProvider) *LiveStatsSessionResolver {
	return &LiveStatsSessionResolver{statsService: statsService, metadata: metadata}
}

func (r *LiveStatsSessionResolver) GetActiveSession(ctx context.Context, liveStreamID int64) (*model.StreamInfo, error) {
	stats, err := r.statsService.GetStats(ctx, liveStreamID)
	if err != nil {
		return nil, err
	}
	if stats == nil || stats.Status != "live" || stats.StartedAt == nil {
		return nil, nil
	}
	name := ""
	avatarURL := ""
	if r.metadata != nil {
		items, err := r.metadata.BatchGetLiveStreams(ctx, []int64{liveStreamID})
		if err != nil {
			return nil, err
		}
		if item, ok := items[liveStreamID]; ok {
			name = item.Name
			avatarURL = item.CoverImage
		}
	}
	return &model.StreamInfo{
		ID:         liveStreamID,
		Name:       name,
		AvatarURL:  avatarURL,
		StatusText: "正在直播",
		LiveRoomID: liveStreamID,
		StartedAt:  stats.StartedAt.UnixMilli(),
	}, nil
}

type LiveReminderService struct {
	followDAO           FollowDAO
	liveSessionResolver LiveSessionResolver
	receiptClaimer      LiveReminderReceiptClaimer
}

func NewLiveReminderService(followDAO FollowDAO, liveSessionResolver LiveSessionResolver, receiptClaimer LiveReminderReceiptClaimer) *LiveReminderService {
	return &LiveReminderService{followDAO: followDAO, liveSessionResolver: liveSessionResolver, receiptClaimer: receiptClaimer}
}

func (s *LiveReminderService) GetPendingReminder(ctx context.Context, userID int64) (*model.PendingLiveReminderResponse, error) {
	for offset := 0; offset < liveReminderMaxScannedCandidates; offset += liveReminderCandidateLimit {
		remaining := liveReminderMaxScannedCandidates - offset
		limit := liveReminderCandidateLimit
		if remaining < limit {
			limit = remaining
		}
		follows, err := s.followDAO.GetUserFollows(ctx, userID, offset, limit)
		if err != nil {
			return nil, err
		}

		for _, follow := range follows {
			if !follow.NotificationEnabled {
				continue
			}

			session, err := s.liveSessionResolver.GetActiveSession(ctx, follow.LiveStreamID)
			if err != nil {
				return nil, err
			}
			if session == nil || session.StartedAt <= 0 {
				continue
			}

			claimed, err := s.receiptClaimer.Claim(ctx, userID, follow.LiveStreamID, session.StartedAt)
			if err != nil {
				return nil, err
			}
			if !claimed {
				continue
			}

			return &model.PendingLiveReminderResponse{
				HasReminder: true,
				Stream:      session,
			}, nil
		}

		if len(follows) < limit {
			break
		}
	}

	return &model.PendingLiveReminderResponse{HasReminder: false, Stream: nil}, nil
}
