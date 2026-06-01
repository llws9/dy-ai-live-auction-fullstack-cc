package handler

import (
	"context"
	"errors"
)

var (
	ErrInvalidUserID     = errors.New("invalid user_id")
	ErrInvalidLiveStream = errors.New("invalid live_stream_id")
)

type FollowChecker interface {
	IsFollowing(ctx context.Context, userID, liveStreamID int64) (bool, error)
}

type FollowStatusResponse struct {
	IsFollowing bool `json:"is_following"`
}

func BuildFollowStatusResponse(ctx context.Context, fc FollowChecker, userID, liveStreamID int64) (*FollowStatusResponse, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}
	if liveStreamID <= 0 {
		return nil, ErrInvalidLiveStream
	}
	following, err := fc.IsFollowing(ctx, userID, liveStreamID)
	if err != nil {
		return nil, err
	}
	return &FollowStatusResponse{IsFollowing: following}, nil
}
