package handler

import (
	"context"
	"errors"
)

// FollowChecker 抽象 FollowService.IsFollowing，方便单测。
//
// 现有 service.FollowService.IsFollowing(ctx, userID, liveStreamID) (bool, error)
// 已天然实现该接口，无需 service 改动。
type FollowChecker interface {
	IsFollowing(ctx context.Context, userID, liveStreamID int64) (bool, error)
}

// FollowStatusResponse 是 GET /live-streams/:id/follow-status 的稳定响应数据。
type FollowStatusResponse struct {
	IsFollowing bool `json:"is_following"`
}

// BuildFollowStatusResponse 是 F-B2 的纯编排函数：
//   - 校验 ids
//   - 调用 checker
//   - 错误向上冒泡（handler 转 5xx）
//
// 拆出独立函数是为了让 handler 层只负责 HTTP 解析/序列化，
// 业务编排可在 service-less 单测下覆盖。
func BuildFollowStatusResponse(ctx context.Context, fc FollowChecker, userID, liveStreamID int64) (*FollowStatusResponse, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user_id")
	}
	if liveStreamID <= 0 {
		return nil, errors.New("invalid live_stream_id")
	}
	following, err := fc.IsFollowing(ctx, userID, liveStreamID)
	if err != nil {
		return nil, err
	}
	return &FollowStatusResponse{IsFollowing: following}, nil
}
