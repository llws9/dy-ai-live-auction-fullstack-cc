// API统一封装入口

import { request, get, post, put, del, buildQuery, ApiError, setToastFunction } from './request';

// 重新导出类型
export * from './types';

// 认证API
export const authApi = {
  login: (data: { email?: string; phone?: string; password: string }) =>
    post<{ token: string; user: any }>('/auth/login', data),

  register: (data: { name: string; email?: string; phone?: string; password: string }) =>
    post<{ token: string; user: any }>('/auth/register', data),

  getCurrentUser: () => get<any>('/users/me'),
};

// 商品API - 增加 get 方法
export const productApi = {
  list: (params?: { status?: number; page?: number; page_size?: number }) => {
    const query = buildQuery(params || {});
    return get<{ list: any[]; total: number; page: number; page_size: number }>(`/products?${query}`);
  },

  get: (id: number) => get<any>(`/products/${id}`),

  create: (data: { name: string; description: string; images: string[]; category?: string }) =>
    post<any>('/products', data),

  update: (id: number, data: Partial<{ name: string; description: string; images: string[]; category?: string }>) =>
    put<any>(`/products/${id}`, data),

  delete: (id: number) => del<void>(`/products/${id}`),

  publish: (id: number) => post<any>(`/products/${id}/publish`),

  unpublish: (id: number, reason?: string) => post<any>(`/products/${id}/unpublish`, { reason }),

  getRules: (productId: number) => get<any>(`/products/${productId}/rules`),

  createRules: (productId: number, data: {
    start_price: number;
    increment: number;
    cap_price: number;
    duration: number;
    delay_duration: number;
    max_delay_time: number;
    trigger_delay_before: number;
  }) => post<any>(`/products/${productId}/rules`, data),
};

// 竞拍API
export const auctionApi = {
  list: (params?: { status?: number; live_stream_id?: number; live_stream_name?: string; search?: string; page?: number; page_size?: number }) => {
    const query = buildQuery(params || {});
    return get<{ list: any[]; total: number }>(`/auctions?${query}`);
  },

  get: (id: number) => get<any>(`/auctions/${id}`),

  create: (data: { product_id: number; live_stream_id: number }) => post<any>('/auctions', data),

  getBids: (id: number) => get<any[]>(`/auctions/${id}/bids`),

  getRanking: (id: number) => get<any[]>(`/auctions/${id}/ranking`),

  placeBid: (id: number, amount: number) => post<any>(`/auctions/${id}/bids`, { amount }),

  cancel: (id: number) => put<void>(`/auctions/${id}/cancel`),

  getResult: (id: number) => get<any>(`/auctions/${id}/result`),
};

// 订单API
export const orderApi = {
  list: (params?: { user_id?: number; status?: number; page?: number; page_size?: number }) => {
    const query = buildQuery(params || {});
    return get<{ list: any[]; total: number }>(`/orders?${query}`);
  },

  get: (id: number) => get<any>(`/orders/${id}`),

  updateStatus: (id: number, status: number) => put<any>(`/orders/${id}`, { status }),

  pay: (id: number) => post<any>(`/orders/${id}/pay`),

  ship: (id: number) => put<any>(`/orders/${id}/ship`),

  getUserHistory: () => get<any[]>('/orders/history'),
};

// 直播间API
export const liveStreamApi = {
  // 获取直播列表（公开API，用于移动端关注列表）
  list: (params?: { status?: number; page?: number; page_size?: number }) => {
    const query = buildQuery(params || {});
    return get<{ list: any[]; total: number }>(`/live-streams?${query}`);
  },

  adminList: (params?: { page?: number; page_size?: number }) => {
    const query = buildQuery(params || {});
    return get<{ list: any[]; total: number }>(`/admin/live-streams?${query}`);
  },

  get: (id: number) => get<any>(`/live-streams/${id}`),

  getUserFollows: (params?: { page?: number; page_size?: number }) => {
    const query = buildQuery(params || {});
    return get<{ list: any[]; total: number }>(`/user/followed-live-streams?${query}`);
  },

  follow: (id: number) => post<void>(`/live-streams/${id}/follow`),

  unfollow: (id: number) => del<void>(`/live-streams/${id}/follow`),

  toggleNotification: (id: number, enabled: boolean) => put<void>(`/live-streams/${id}/notification`, { enabled }),
};

// 通知API
export const notificationApi = {
  list: (params?: { page?: number; page_size?: number; unread_only?: boolean }) => {
    const query = buildQuery(params || {});
    return get<{ list: any[]; total: number }>(`/notifications?${query}`);
  },

  getUnreadCount: () => get<{ count: number }>('/notifications/unread-count'),

  markAsRead: (id: number) => put<void>(`/notifications/${id}/read`),

  markAllAsRead: () => put<void>('/notifications/read-all'),
};

// 统计API
export const statisticsApi = {
  getOverview: () => get<any>('/statistics/overview'),

  getAuctionStats: (params?: { start_date?: string; end_date?: string }) => {
    const query = buildQuery(params || {});
    return get<any[]>(`/statistics/auctions?${query}`);
  },

  getRevenueStats: (params?: { start_date?: string; end_date?: string; category?: string; group_by?: string }) => {
    const query = buildQuery(params || {});
    return get<any[]>(`/statistics/revenue?${query}`);
  },

  getUserStats: (params?: { start_date?: string; end_date?: string }) => {
    const query = buildQuery(params || {});
    return get<any[]>(`/statistics/users?${query}`);
  },
};

export { ApiError, setToastFunction };