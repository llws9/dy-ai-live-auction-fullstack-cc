// 通知API

import { get, put, buildQuery } from './request';
import { Notification, PaginatedResponse } from './types';

export interface NotificationListParams {
  page?: number;
  page_size?: number;
  unread_only?: boolean;
}

export interface UnreadCountResponse {
  count: number;
}

export const notificationApi = {
  // 获取通知列表
  list: (params?: NotificationListParams) => {
    const query = buildQuery(params || {});
    return get<PaginatedResponse<Notification>>(`/notifications?${query}`);
  },

  // 获取未读数量
  getUnreadCount: () => get<UnreadCountResponse>('/notifications/unread-count'),

  // 标记单条已读
  markAsRead: (id: number) => put<void>(`/notifications/${id}/read`),

  // 标记全部已读
  markAllAsRead: () => put<void>('/notifications/read-all'),
};