// 直播间API

import { get, post, put, del, buildQuery } from './request';
import { LiveStream, PaginatedResponse } from './types';

export interface LiveStreamListParams {
  page?: number;
  page_size?: number;
}

export const liveStreamApi = {
  // 管理端获取直播间列表
  adminList: (params?: LiveStreamListParams) => {
    const query = buildQuery(params || {});
    return get<PaginatedResponse<LiveStream>>(`/admin/live-streams?${query}`);
  },

  // 获取直播间详情
  get: (id: number) => get<LiveStream>(`/live-streams/${id}`),

  // 用户关注的直播间列表
  getUserFollows: (params?: LiveStreamListParams) => {
    const query = buildQuery(params || {});
    return get<PaginatedResponse<LiveStream>>(`/user/followed-live-streams?${query}`);
  },

  // 关注直播间
  follow: (id: number) => post<void>(`/live-streams/${id}/follow`),

  // 取消关注
  unfollow: (id: number) => del<void>(`/live-streams/${id}/follow`),

  // 切换通知开关
  toggleNotification: (id: number, enabled: boolean) => put<void>(`/live-streams/${id}/notification`, { enabled }),
};
