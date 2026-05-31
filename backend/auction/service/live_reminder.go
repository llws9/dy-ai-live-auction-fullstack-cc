package service

import (
	"context"

	"auction-service/model"
)

const liveReminderCandidateLimit = 50

type LiveSessionResolver interface {
	GetActiveSession(ctx context.Context, liveStreamID int64) (*model.StreamInfo, error)
}

type LiveReminderReceiptClaimer interface {
	Claim(ctx context.Context, userID, liveStreamID, startedAt int64) (bool, error)
}

type LiveStatsSessionResolver struct {
	statsService *LiveStreamStatsService
}

func NewLiveStatsSessionResolver(statsService *LiveStreamStatsService) *LiveStatsSessionResolver {
	return &LiveStatsSessionResolver{statsService: statsService}
}

func (r *LiveStatsSessionResolver) GetActiveSession(ctx context.Context, liveStreamID int64) (*model.StreamInfo, error) {
	stats, err := r.statsService.GetStats(ctx, liveStreamID)
	if err != nil {
		return nil, err
	}
	if stats == nil || stats.Status != "live" || stats.StartedAt == nil {
		return nil, nil
	}
	return &model.StreamInfo{
		ID:         liveStreamID,
		Name:       "关注直播间",
		AvatarURL:  "",
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
	follows, err := s.followDAO.GetUserFollows(ctx, userID, 0, liveReminderCandidateLimit)
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

	return &model.PendingLiveReminderResponse{HasReminder: false, Stream: nil}, nil
}
