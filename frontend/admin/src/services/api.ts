// services/api.ts

import { Product, AuctionRule, ApiResponse, PaginatedResponse } from '../types';

const API_BASE_URL = '/api/v1';

// 通用请求方法
async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
    ...options,
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || '请求失败');
  }

  return response.json();
}

// 商品 API
export const productApi = {
  // 获取商品列表
  list: (params?: { status?: number; page?: number; page_size?: number }) => {
    const query = new URLSearchParams();
    if (params?.status !== undefined) query.set('status', String(params.status));
    if (params?.page) query.set('page', String(params.page));
    if (params?.page_size) query.set('page_size', String(params.page_size));

    return request<PaginatedResponse<Product>>(`/products?${query.toString()}`);
  },

  // 获取商品详情
  get: (id: number) => request<Product>(`/products/${id}`),

  // 创建商品
  create: (data: Partial<Product>) =>
    request<Product>('/products', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // 更新商品
  update: (id: number, data: Partial<Product>) =>
    request<Product>(`/products/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  // 删除商品
  delete: (id: number) =>
    request<void>(`/products/${id}`, {
      method: 'DELETE',
    }),
};

// 竞拍规则 API
export const ruleApi = {
  // 获取规则
  get: (productId: number) => request<AuctionRule>(`/products/${productId}/rules`),

  // 创建规则
  create: (productId: number, data: Partial<AuctionRule>) =>
    request<AuctionRule>(`/products/${productId}/rules`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
};

// 竞拍 API
export const auctionApi = {
  // 取消竞拍
  cancel: (id: number) =>
    request<void>(`/auctions/${id}/cancel`, {
      method: 'PUT',
    }),

  // 获取竞拍结果
  getResult: (id: number) => request<any>(`/auctions/${id}/result`),
};

// 订单 API
export const orderApi = {
  // 获取订单列表
  list: () => request<any>('/orders'),

  // 模拟支付
  pay: (id: number) =>
    request<any>(`/orders/${id}/pay`, {
      method: 'POST',
    }),
};
