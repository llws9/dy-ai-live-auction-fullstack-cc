// 订单API

import { get, post, put, buildQuery } from './request';
import { Order, PaginatedResponse } from './types';
import { normalizeOrderListResponse, normalizeOrderText } from './orderEncoding';

export interface OrderListParams {
  user_id?: number;
  status?: number;
  page?: number;
  page_size?: number;
}

export const orderApi = {
  // 获取订单列表
  list: (params?: OrderListParams) => {
    const query = buildQuery(params || {});
    return get<PaginatedResponse<Order>>(`/orders?${query}`).then(normalizeOrderListResponse);
  },

  // 获取订单详情
  get: (id: number) => get<Order>(`/orders/${id}`).then(normalizeOrderText),

  // 更新订单状态
  updateStatus: (id: number, status: number) => put<Order>(`/orders/${id}`, { status }),

  // 模拟支付
  pay: (id: number) => post<Order>(`/orders/${id}/pay`),

  // 模拟发货
  ship: (id: number) => put<Order>(`/orders/${id}/ship`),

  // 用户订单历史
  getUserHistory: () => get<Order[]>('/orders/history'),
};
