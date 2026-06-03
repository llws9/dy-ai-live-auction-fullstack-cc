// API统一封装入口

import { get, post, put, del, buildQuery, ApiError, setToastFunction } from './request';

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

export interface CopywritingGenerateData {
  images: string[];
  category_id?: number;
  keywords?: string;
}

export interface CopywritingDraft {
  name: string;
  description: string;
  selling_points: string[];
  suggested_start_price: string;
}

// 商品API - 增加 get 方法
export const productApi = {
  list: (params?: { status?: number; page?: number; page_size?: number }) => {
    const query = buildQuery(params || {});
    return get<{ list: any[]; total: number; page: number; page_size: number }>(`/products?${query}`);
  },

  get: (id: number) => get<any>(`/products/${id}`),

  create: (data: { name: string; description: string; images: string[]; category?: string }) =>
    post<any>('/products', data),

  generateCopywriting: (data: CopywritingGenerateData) =>
    post<CopywritingDraft>('/products/ai/copywriting', data, { timeout: 70000 }),

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
  // admin 端订单列表：使用 /admin/orders（不被 X-User-ID 过滤），可透传 user_id 筛某用户。
  list: (params?: { user_id?: number; status?: number; page?: number; page_size?: number }) => {
    const query = buildQuery(params || {});
    return get<{ list: any[]; total: number; page: number; page_size: number }>(`/admin/orders?${query}`);
  },

  // admin 端订单详情：使用 /admin/orders/:id（不被 winner_id 过滤）。
  get: (id: number) => get<any>(`/admin/orders/${id}`),

  updateStatus: (id: number, status: number) => put<any>(`/orders/${id}`, { status }),

  // pay 是用户行为，仍走用户路由
  pay: (id: number) => post<any>(`/orders/${id}/pay`),

  ship: (id: number) => put<any>(`/orders/${id}/ship`),

  // 用户视角的竞拍历史，保持原路由
  getUserHistory: () => get<any[]>('/orders/history'),
};

// 直播间API
export const liveStreamApi = {
  // 获取直播列表（公开API，用于移动端关注列表）
  list: (params?: { status?: number; page?: number; page_size?: number }) => {
    const query = buildQuery(params || {});
    return get<{ list: any[]; total: number }>(`/live-streams?${query}`);
  },

  adminList: (params?: { status?: number; page?: number; page_size?: number }) => {
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

  start: (id: number) => post<any>(`/live-streams/${id}/start`),

  end: (id: number) => put<any>(`/admin/live-streams/${id}/end`),

  ban: (id: number, reason: string) => put<any>(`/admin/live-streams/${id}/ban`, { reason }),
};

export type FixedPriceAdminStatus = 'on_sale' | 'sold_out' | 'offline'

export interface FixedPriceAdminItem {
  id: number
  live_stream_id: number
  product_id: number
  product_title?: string
  product?: {
    id?: number
    title?: string
    cover_image?: string
  }
  price: string
  total_stock: number
  remaining_stock: number
  status: FixedPriceAdminStatus
  created_at?: string
}

export interface FixedPriceAdminListResponse {
  items: FixedPriceAdminItem[]
  total?: number
  page?: number
  page_size?: number
}

export const fixedPriceAdminApi = {
  list: (liveStreamId: number, params?: { page?: number; page_size?: number }) => {
    const query = buildQuery(params || {})
    const suffix = query ? `?${query}` : ''
    return get<FixedPriceAdminListResponse>(`/admin/live-streams/${liveStreamId}/fixed-price/items${suffix}`)
  },

  listItem: (liveStreamId: number, data: { product_id: number; price: string; stock: number }) =>
    post<FixedPriceAdminItem>('/fixed-price/items', {
      live_stream_id: liveStreamId,
      product_id: data.product_id,
      price: data.price,
      total_stock: data.stock,
      max_per_user: 1,
    }),

  offline: (itemId: number) => post<{ id: number; status: FixedPriceAdminStatus }>(`/fixed-price/items/${itemId}/offline`),
}

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
