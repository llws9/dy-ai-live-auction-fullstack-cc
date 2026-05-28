// 商品API

import { get, post, put, del, buildQuery } from './request';
import { Product, AuctionRule, PaginatedResponse } from './types';

export interface ProductListParams {
  status?: number;
  page?: number;
  page_size?: number;
}

export interface ProductCreateData {
  name: string;
  description: string;
  images: string[];
  category?: string;
}

export interface RuleCreateData {
  start_price: number;
  increment: number;
  cap_price: number;
  duration: number;
  delay_duration: number;
  max_delay_time: number;
  trigger_delay_before: number;
}

export const productApi = {
  // 获取商品列表
  list: (params?: ProductListParams) => {
    const query = buildQuery(params || {});
    return get<PaginatedResponse<Product>>(`/products?${query}`);
  },

  // 获取商品详情
  get: (id: number) => get<Product>(`/products/${id}`),

  // 创建商品
  create: (data: ProductCreateData) => post<Product>('/products', data),

  // 更新商品
  update: (id: number, data: Partial<ProductCreateData>) => put<Product>(`/products/${id}`, data),

  // 删除商品
  delete: (id: number) => del<void>(`/products/${id}`),

  // 发布商品
  publish: (id: number) => post<Product>(`/products/${id}/publish`),

  // 下架商品
  unpublish: (id: number, reason?: string) => post<Product>(`/products/${id}/unpublish`, { reason }),

  // 获取竞拍规则
  getRules: (productId: number) => get<AuctionRule>(`/products/${productId}/rules`),

  // 创建竞拍规则
  createRules: (productId: number, data: RuleCreateData) => post<AuctionRule>(`/products/${productId}/rules`, data),
};