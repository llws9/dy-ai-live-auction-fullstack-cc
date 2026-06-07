// API统一封装入口

import { get, post, put, del, buildQuery, ApiError, setToastFunction } from './request';
import { normalizeAuctionListResponse, normalizeAuctionText } from './auctionEncoding';
import { normalizeBids } from './bidEncoding';
import { normalizeOrderListResponse, normalizeOrderText } from './orderEncoding';
export { productApi } from './product';

// 重新导出类型
export * from './types';
export type {
  ProductListParams,
  ProductCreateData,
  RuleCreateData,
  CopywritingGenerateData,
  CopywritingDraft,
} from './product';

// 认证API
export const authApi = {
  login: (data: { email?: string; phone?: string; password: string }) =>
    post<{ token: string; user: any }>('/auth/login', data),

  register: (data: { name: string; email?: string; phone?: string; password: string }) =>
    post<{ token: string; user: any }>('/auth/register', data),

  getCurrentUser: () => get<any>('/users/me'),
};

// 竞拍API
export const auctionApi = {
  list: (params?: { status?: number; live_stream_id?: number; live_stream_name?: string; search?: string; page?: number; page_size?: number }) => {
    const query = buildQuery(params || {});
    return get<{ list: any[]; total: number }>(`/auctions?${query}`)
      .then(normalizeAuctionListResponse);
  },

  get: (id: number) => get<any>(`/auctions/${id}`).then(normalizeAuctionText),

  create: (data: { product_id: number; duration: number; live_stream_id?: number; start_time?: string }) => post<any>('/auctions', data).then(normalizeAuctionText),

  getBids: (id: number) => get<any[]>(`/auctions/${id}/bids`).then(normalizeBids),

  getRanking: (id: number) => get<any[]>(`/auctions/${id}/ranking`),

  placeBid: (id: number, amount: number) => post<any>(`/auctions/${id}/bids`, { amount }),

  cancel: (id: number) => put<void>(`/auctions/${id}/cancel`),

  getResult: (id: number) => get<any>(`/auctions/${id}/result`),
};

// 订单API
export const orderApi = {
  // admin 端订单列表：使用 /admin/orders（不被 X-User-ID 过滤），可透传 user_id 筛某用户。
  list: (params?: { user_id?: number; status?: number; search?: string; page?: number; page_size?: number }) => {
    const query = buildQuery(params || {});
    return get<{
      list: any[];
      total: number;
      page: number;
      page_size: number;
      summary?: {
        pending_payment_count?: number;
        paid_count?: number;
        shipped_count?: number;
        completed_count?: number;
      };
    }>(`/admin/orders?${query}`).then(normalizeOrderListResponse);
  },

  // admin 端订单详情：使用 /admin/orders/:id（不被 winner_id 过滤）。
  get: (id: number) => get<any>(`/admin/orders/${id}`).then(normalizeOrderText),

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

  adminGet: (id: number) => get<any>(`/admin/live-streams/${id}`),

  create: (data: { name: string; description?: string; cover_image?: string; video_url?: string; streamer_name?: string; streamer_avatar?: string }) =>
    post<any>('/admin/live-streams', data),

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

export interface AuctionRuleTemplate {
  id: number
  name: string
  start_price: string
  increment: string
  cap_price?: string
  duration: number
  delay_duration: number
  max_delay_time: number
  trigger_delay_before: number
  is_default: boolean
}

export type AuctionRuleTemplatePayload = Omit<AuctionRuleTemplate, 'id'>

export const auctionRuleTemplateApi = {
  list: (params?: { page?: number; page_size?: number }) => {
    const query = buildQuery(params || {})
    const suffix = query ? `?${query}` : ''
    return get<{ list: AuctionRuleTemplate[]; total: number; page: number; page_size: number }>(
      `/admin/auction-rule-templates${suffix}`
    )
  },

  get: (id: number) => get<AuctionRuleTemplate>(`/admin/auction-rule-templates/${id}`),

  create: (data: AuctionRuleTemplatePayload) =>
    post<AuctionRuleTemplate>('/admin/auction-rule-templates', data),

  update: (id: number, data: AuctionRuleTemplatePayload) =>
    put<AuctionRuleTemplate>(`/admin/auction-rule-templates/${id}`, data),

  delete: (id: number) => del<void>(`/admin/auction-rule-templates/${id}`),
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

function normalizeRevenueStatsResponse(response: any, groupBy?: string): any[] {
  if (Array.isArray(response)) {
    return response;
  }

  if (groupBy === 'category' && Array.isArray(response?.category_distribution)) {
    return response.category_distribution;
  }

  if (Array.isArray(response?.daily_revenue)) {
    return response.daily_revenue;
  }

  if (Array.isArray(response?.monthly_revenue)) {
    return response.monthly_revenue;
  }

  if (Array.isArray(response?.list)) {
    return response.list;
  }

  return [];
}

// 统计API
export const statisticsApi = {
  getOverview: () => get<any>('/statistics/overview'),

  getAuctionStats: (params?: { start_date?: string; end_date?: string; group_by?: string }) => {
    const query = buildQuery(params || {});
    return get<any[]>(`/statistics/auctions?${query}`);
  },

  getRevenueStats: (params?: { start_date?: string; end_date?: string; category?: string; group_by?: string }) => {
    const query = buildQuery(params || {});
    return get<any>(`/statistics/revenue?${query}`).then((response) =>
      normalizeRevenueStatsResponse(response, params?.group_by)
    );
  },

  getUserStats: (params?: { start_date?: string; end_date?: string }) => {
    const query = buildQuery(params || {});
    return get<any>(`/statistics/users?${query}`);
  },
};

export { ApiError, setToastFunction };
