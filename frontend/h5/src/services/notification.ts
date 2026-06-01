// services/notification.ts
// 通知服务 - 提供通知相关的API调用

import { get, post, put } from './api';

// 通知项接口
export interface NotificationItem {
  id: number;
  type: string;
  title: string;
  content: string;
  data?: Record<string, unknown>;
  read_at?: string;
  created_at: string;
}

// 通知列表响应
export interface NotificationListResponse {
  items: NotificationItem[];
  total: number;
  page: number;
  page_size: number;
}

// 未读数量响应
export interface UnreadCountResponse {
  count: number;
}

// 热拉响应
export interface HotPullResponse {
  notifications: NotificationItem[];
  has_more: boolean;
}

export interface TouchpointSummary {
  unreadTotal: number;
  pendingPayment: number;
  wonNotPaid: number;
  outbid: number;
  endingSoon: number;
}

export interface PendingLiveReminderResponse {
  hasReminder: boolean;
  stream: {
    id: string | number;
    name: string;
    avatarUrl: string;
    statusText?: string;
    liveRoomId?: string | number;
    startedAt?: number;
  } | null;
}

// 通知 API
export const notificationApi = {
  // 获取通知列表
  list: (page: number = 1, pageSize: number = 20): Promise<NotificationListResponse> => {
    return get<NotificationListResponse>(`/notifications?page=${page}&page_size=${pageSize}`);
  },

  // 获取未读数量
  getUnreadCount: (): Promise<UnreadCountResponse> => {
    return get<UnreadCountResponse>('/notifications/unread-count');
  },

  // 标记单条通知已读
  markAsRead: (id: number): Promise<void> => {
    return put<void>(`/notifications/${id}/read`);
  },

  // 标记全部已读
  markAllAsRead: (): Promise<void> => {
    return put<void>('/notifications/read-all');
  },

  // 热拉通知 - 用户切换前台或登录时主动拉取
  hotPull: (): Promise<HotPullResponse> => {
    return post<HotPullResponse>('/notifications/hot-pull');
  },

  getTouchpointSummary: (): Promise<TouchpointSummary> => {
    return get<TouchpointSummary>('/notifications/summary');
  },

  markCategoryAsRead: (category: 'pendingPayment' | 'outbid' | 'endingSoon' | 'all'): Promise<void> => {
    return post<void>('/notifications/read-category', { category });
  },

  getPendingLiveReminder: (): Promise<PendingLiveReminderResponse> => {
    return get<PendingLiveReminderResponse>('/live/pending-reminder');
  },
};
